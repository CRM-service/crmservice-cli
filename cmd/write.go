package cmd

import (
	"fmt"

	"crmservice/internal/api"
	"crmservice/internal/output"
)

func singleRecordRequestBody(module, id, operation string, input *bodyInput) (map[string]interface{}, error) {
	if input.raw {
		body, ok := input.body.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid JSON:API request body")
		}
		return body, nil
	}

	attrs, ok := input.body.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid request attributes")
	}
	if len(attrs) == 0 {
		return nil, fmt.Errorf("no fields provided; use --field or pass JSON via stdin")
	}

	return api.BuildSingleWriteBody(module, id, operation, attrs), nil
}

func outputDryRunRequest(operation, module, id string, body map[string]interface{}, outputFormat string) error {
	result := map[string]interface{}{
		"dry_run":   true,
		"operation": operation,
		"module":    module,
		"body":      body,
	}
	if id != "" {
		result["id"] = id
	}
	return output.ItemResponse(&api.SingleResponse{Data: result}, output.Options{
		Format: outputFormat,
		Full:   false,
	})
}

func outputDeleteResult(module, id, outputFormat string) error {
	result := map[string]interface{}{
		"deleted": true,
		"module":  module,
		"id":      id,
	}
	if outputFormat == "table" {
		fmt.Printf("Successfully deleted %s %s\n", module, id)
		return nil
	}
	return output.ItemResponse(&api.SingleResponse{Data: result}, output.Options{
		Format: outputFormat,
		Full:   false,
	})
}
