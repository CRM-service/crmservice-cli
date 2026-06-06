package cmd

import (
	"encoding/json"
	"fmt"
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
		relMod, ok := attrMap["relationModule"].(string)
		if !ok || relMod == "" {
			continue
		}
		schema.addRelationTarget(name, relMod)
	}

	return schema, nil
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
