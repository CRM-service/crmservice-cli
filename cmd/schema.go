package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type moduleSchema struct {
	attributeNames  map[string]struct{}
	relationTargets map[string]string
}

func parseModuleSchema(body []byte) (*moduleSchema, error) {
	var rawResp map[string]interface{}
	if err := json.Unmarshal(body, &rawResp); err != nil {
		return nil, fmt.Errorf("invalid schema response: %w", err)
	}

	schema := &moduleSchema{
		attributeNames:  make(map[string]struct{}),
		relationTargets: make(map[string]string),
	}

	attrsObj, ok := rawResp["attributes"].(map[string]interface{})
	if !ok {
		return schema, nil
	}

	for name, attrData := range attrsObj {
		schema.attributeNames[name] = struct{}{}
		attrMap, ok := attrData.(map[string]interface{})
		if !ok {
			continue
		}
		schema.registerAttributeRelation(name, attrMap)
	}

	if relationsObj, ok := rawResp["relations"].(map[string]interface{}); ok {
		for relName, relData := range relationsObj {
			relMap, ok := relData.(map[string]interface{})
			if !ok {
				continue
			}
			if class, ok := relMap["class"].(string); ok && class != "" {
				schema.relationTargets[relName] = class
			}
		}
	}

	return schema, nil
}

func (s *moduleSchema) registerAttributeRelation(attributeName string, attrMap map[string]interface{}) {
	if relMod, ok := attrMap["relationModule"].(string); ok && relMod != "" {
		s.addRelationTarget(attributeName, relMod)
	}

	if relType, ok := attrMap["relationType"].(string); ok && relType != "" {
		s.addRelationTarget(attributeName, relType)
		if relName, ok := attrMap["relationName"].(string); ok && relName != "" {
			s.relationTargets[relName] = relType
		}
	}

	if relMods, ok := attrMap["relationModules"].([]interface{}); ok {
		for _, item := range relMods {
			if mod, ok := item.(string); ok && mod != "" {
				s.addRelationTarget(attributeName, mod)
			}
		}
	}
}

func (s *moduleSchema) addRelationTarget(attributeName, relationModule string) {
	s.relationTargets[attributeName] = relationModule
	s.relationTargets[relationModule] = relationModule

	prefix := strings.TrimSuffix(attributeName, "_id")
	if prefix != attributeName {
		s.relationTargets[prefix] = relationModule
	}
	if strings.HasSuffix(relationModule, "s") && len(relationModule) > 1 {
		s.relationTargets[strings.TrimSuffix(relationModule, "s")] = relationModule
	}
}

func (s *moduleSchema) validateField(field string, loadRelated func(module string) (*moduleSchema, error)) error {
	if _, ok := s.attributeNames[field]; ok {
		return nil
	}

	prefix, subfield, hasSubfield := strings.Cut(field, ".")
	if !hasSubfield {
		return fmt.Errorf("unknown field %q", field)
	}

	if relationModule, ok := s.relationTargets[prefix]; ok {
		related, err := loadRelated(relationModule)
		if err != nil {
			return fmt.Errorf("field %q: cannot load related module %q schema: %w", field, relationModule, err)
		}
		if _, ok := related.attributeNames[subfield]; ok {
			return nil
		}
		return fmt.Errorf("unknown field %q in related module %q", subfield, relationModule)
	}

	return fmt.Errorf("unknown field %q", field)
}

func collectBodyAttributeFields(input *bodyInput) ([]string, error) {
	if input == nil {
		return nil, nil
	}

	if input.raw {
		body, ok := input.body.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid JSON:API request body")
		}
		data, ok := body["data"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid JSON:API request body: data must be an object")
		}
		attrs, ok := data["attributes"].(map[string]interface{})
		if !ok {
			return nil, nil
		}
		fields := make([]string, 0, len(attrs))
		for name := range attrs {
			fields = append(fields, name)
		}
		sort.Strings(fields)
		return fields, nil
	}

	attrs, ok := input.body.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid request attributes")
	}
	fields := make([]string, 0, len(attrs))
	for name := range attrs {
		fields = append(fields, name)
	}
	sort.Strings(fields)
	return fields, nil
}

func validateModuleAttributes(module string, fields []string, url, token string, verbose int) error {
	if len(fields) == 0 {
		return nil
	}

	body, err := getSchemaBody(module, url, token, verbose, false)
	if err != nil {
		return fmt.Errorf("failed to load schema for module %q: %w", module, err)
	}

	schema, err := parseModuleSchema(body)
	if err != nil {
		return err
	}

	relatedCache := make(map[string]*moduleSchema)
	loadRelated := func(relationModule string) (*moduleSchema, error) {
		if cached, ok := relatedCache[relationModule]; ok {
			return cached, nil
		}
		relatedBody, err := getSchemaBody(relationModule, url, token, verbose, false)
		if err != nil {
			return nil, err
		}
		relatedSchema, err := parseModuleSchema(relatedBody)
		if err != nil {
			return nil, err
		}
		relatedCache[relationModule] = relatedSchema
		return relatedSchema, nil
	}

	for _, field := range fields {
		if err := schema.validateField(field, loadRelated); err != nil {
			return fmt.Errorf("module %q: %w", module, err)
		}
	}

	return nil
}

func validateBodyInputAgainstModule(module string, input *bodyInput, url, token string, verbose int) error {
	fields, err := collectBodyAttributeFields(input)
	if err != nil {
		return err
	}
	return validateModuleAttributes(module, fields, url, token, verbose)
}

func validateModuleFilter(module, filterJSON, url, token string, verbose int) error {
	if filterJSON == "" {
		return nil
	}
	return validateFilterJSONAgainstModule(module, filterJSON, url, token, verbose)
}

func validateFilterJSONAgainstModule(module, input, url, token string, verbose int) error {
	if err := validateFilterJSON(input); err != nil {
		return err
	}

	var filter interface{}
	decoder := json.NewDecoder(strings.NewReader(input))
	decoder.UseNumber()
	if err := decoder.Decode(&filter); err != nil {
		return fmt.Errorf("invalid filter JSON: %w", err)
	}

	body, err := getSchemaBody(module, url, token, verbose, false)
	if err != nil {
		return fmt.Errorf("failed to load schema for module %q: %w", module, err)
	}

	schema, err := parseModuleSchema(body)
	if err != nil {
		return err
	}

	relatedCache := make(map[string]*moduleSchema)
	loadRelated := func(relationModule string) (*moduleSchema, error) {
		if cached, ok := relatedCache[relationModule]; ok {
			return cached, nil
		}
		relatedBody, err := getSchemaBody(relationModule, url, token, verbose, false)
		if err != nil {
			return nil, err
		}
		relatedSchema, err := parseModuleSchema(relatedBody)
		if err != nil {
			return nil, err
		}
		relatedCache[relationModule] = relatedSchema
		return relatedSchema, nil
	}

	for _, field := range collectFilterFields(filter) {
		if err := schema.validateField(field, loadRelated); err != nil {
			return fmt.Errorf("module %q: %w", module, err)
		}
	}

	return nil
}
