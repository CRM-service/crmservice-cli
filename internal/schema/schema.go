package schema

import (
	"encoding/json"
	"fmt"
	"strings"
)

type moduleSchema struct {
	attributeNames  map[string]struct{}
	relationTargets map[string]string
}

func ParseModuleSchema(body []byte) (*moduleSchema, error) {
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

type Validator struct {
	SchemaLoader func(module string) ([]byte, error)
}

func (v *Validator) ValidateModuleAttributes(module string, fields []string) error {
	if len(fields) == 0 {
		return nil
	}

	body, err := v.SchemaLoader(module)
	if err != nil {
		return fmt.Errorf("failed to load schema for module %q: %w", module, err)
	}

	schema, err := ParseModuleSchema(body)
	if err != nil {
		return err
	}

	relatedCache := make(map[string]*moduleSchema)
	loadRelated := func(relationModule string) (*moduleSchema, error) {
		if cached, ok := relatedCache[relationModule]; ok {
			return cached, nil
		}
		relatedBody, err := v.SchemaLoader(relationModule)
		if err != nil {
			return nil, err
		}
		relatedSchema, err := ParseModuleSchema(relatedBody)
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

func (v *Validator) ValidateFilterFields(module string, fields []string) error {
	return v.ValidateModuleAttributes(module, fields)
}
