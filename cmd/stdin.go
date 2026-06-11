package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type bodyInput struct {
	body interface{}
	raw  bool
}

func getBodyInput(cmd *cobra.Command, id, operation string) (*bodyInput, error) {
	fields, err := cmd.Flags().GetStringArray("field")
	if err != nil {
		return nil, err
	}

	stdinBody, hasStdinBody, err := readStdinBodyInput(os.Stdin, id, operation)
	if err != nil {
		return nil, err
	}
	if hasStdinBody {
		if len(fields) > 0 {
			return nil, fmt.Errorf("cannot use --field with request body from stdin")
		}
		return stdinBody, nil
	}

	data := make(map[string]interface{})
	for _, f := range fields {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, fmt.Errorf("invalid --field %q: expected name=value", f)
		}
		data[strings.TrimSpace(parts[0])] = parseFieldValue(parts[1])
	}
	return &bodyInput{body: data}, nil
}

func parseFieldValue(value string) interface{} {
	var parsed interface{}
	if err := json.Unmarshal([]byte(value), &parsed); err == nil {
		return parsed
	}
	return value
}

func readStdinBodyInput(stdin *os.File, id, operation string) (*bodyInput, bool, error) {
	info, err := stdin.Stat()
	if err != nil {
		return nil, false, err
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return nil, false, nil
	}

	body, err := io.ReadAll(stdin)
	if err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(string(body)) == "" {
		return nil, false, nil
	}

	var request map[string]interface{}
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, true, fmt.Errorf("invalid request body from stdin: %w", err)
	}

	if _, ok := request["data"]; ok {
		if err := validateJSONAPIRequestBody(request, id, operation); err != nil {
			return nil, true, err
		}
		return &bodyInput{body: request, raw: true}, true, nil
	}

	flatBody, err := flatRecordAttributes(request, id, operation)
	if err != nil {
		return nil, true, err
	}
	return &bodyInput{body: flatBody}, true, nil
}

func validateJSONAPIRequestBody(request map[string]interface{}, id, operation string) error {
	data, ok := request["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid JSON:API request body from stdin: data must be an object")
	}
	if operation == "create" {
		delete(data, "id")
	}
	if operation == "update" {
		if bodyID, ok := data["id"]; ok && fmt.Sprintf("%v", bodyID) != id {
			return fmt.Errorf("request body id %q does not match argument id %q", fmt.Sprintf("%v", bodyID), id)
		}
	}
	return nil
}

func flatRecordAttributes(record map[string]interface{}, id, operation string) (map[string]interface{}, error) {
	attrs := make(map[string]interface{}, len(record))
	for key, value := range record {
		if key == "id" {
			if operation == "update" && value != nil && fmt.Sprintf("%v", value) != id {
				return nil, fmt.Errorf("request body id %q does not match argument id %q", fmt.Sprintf("%v", value), id)
			}
			continue
		}
		attrs[key] = value
	}
	return attrs, nil
}

func bodyInputHasEmptyAttributes(input *bodyInput) bool {
	if input == nil {
		return true
	}
	if !input.raw {
		attrs, ok := input.body.(map[string]interface{})
		return !ok || len(attrs) == 0
	}
	body, ok := input.body.(map[string]interface{})
	if !ok {
		return true
	}
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		return true
	}
	attrs, ok := data["attributes"].(map[string]interface{})
	if !ok {
		return true
	}
	return len(attrs) == 0
}
