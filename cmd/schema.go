package cmd

import (
	"fmt"
	"sort"

	"crmservice/internal/filter"
	"crmservice/internal/schema"
)

func newSchemaValidator(url, token string, verbose int) *schema.Validator {
	return &schema.Validator{
		SchemaLoader: func(module string) ([]byte, error) {
			return getSchemaBody(module, url, token, verbose, false)
		},
	}
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
	return newSchemaValidator(url, token, verbose).ValidateModuleAttributes(module, fields)
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
	if err := filter.ValidateFilterJSON(input); err != nil {
		return err
	}

	parsed, err := filter.ParseFilterJSON(input)
	if err != nil {
		return fmt.Errorf("invalid filter JSON: %w", err)
	}

	fields := filter.CollectFilterFields(parsed)
	return newSchemaValidator(url, token, verbose).ValidateFilterFields(module, fields)
}