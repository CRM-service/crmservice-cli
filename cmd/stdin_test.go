package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestReadStdinBodyInputJSONAPIRequest(t *testing.T) {
	stdin := pipeWithContent(t, `{"data":{"type":"accounts","attributes":{"name":"Test Corp"}}}`)

	input, ok, err := readStdinBodyInput(stdin, "", "create")
	if err != nil {
		t.Fatalf("readStdinBodyInput() returned error: %v", err)
	}
	if !ok {
		t.Fatal("readStdinBodyInput() ok = false, expected true")
	}
	if !input.raw {
		t.Fatal("readStdinBodyInput() raw = false, expected true")
	}
	body, ok := input.body.(map[string]interface{})
	if !ok {
		t.Fatalf("input.body = %T, expected object", input.body)
	}
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("body[data] = %T, expected object", body["data"])
	}
	if data["type"] != "accounts" {
		t.Errorf("data[type] = %q, expected accounts", data["type"])
	}
}

func TestReadStdinBodyInputRejectsInvalidJSONAPIBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{`},
		{name: "data not object", body: `{"data":[]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdin := pipeWithContent(t, tt.body)
			_, ok, err := readStdinBodyInput(stdin, "", "create")
			if !ok {
				t.Fatal("readStdinBodyInput() ok = false, expected true")
			}
			if err == nil {
				t.Fatal("readStdinBodyInput() error = nil, expected error")
			}
		})
	}
}

func TestReadStdinBodyInputIgnoresEmptyBody(t *testing.T) {
	stdin := pipeWithContent(t, "\n  \t")

	_, ok, err := readStdinBodyInput(stdin, "", "create")
	if err != nil {
		t.Fatalf("readStdinBodyInput() returned error: %v", err)
	}
	if ok {
		t.Fatal("readStdinBodyInput() ok = true, expected false")
	}
}

func TestGetBodyInputRejectsInvalidFieldFlag(t *testing.T) {
	cmd := createCmd()
	if err := cmd.Flags().Set("field", "name"); err != nil {
		t.Fatalf("Set(field) error: %v", err)
	}

	_, err := getBodyInput(cmd, "", "create")
	if err == nil {
		t.Fatal("getBodyInput() error = nil, expected invalid field error")
	}
	if !strings.Contains(err.Error(), "invalid --field") {
		t.Fatalf("getBodyInput() error = %v, expected invalid --field", err)
	}
}

func TestParseFieldValue(t *testing.T) {
	if got := parseFieldValue("Acme"); got != "Acme" {
		t.Fatalf("parseFieldValue(string) = %v, want Acme", got)
	}
	if got := parseFieldValue("true"); got != true {
		t.Fatalf("parseFieldValue(bool) = %v, want true", got)
	}
	if got := parseFieldValue("42"); got != float64(42) {
		t.Fatalf("parseFieldValue(number) = %v, want 42", got)
	}
	if got := parseFieldValue(`["a","b"]`); got == nil {
		t.Fatal("parseFieldValue(array) = nil, want slice")
	}
}

func TestReadStdinBodyInputAcceptsFlatCreate(t *testing.T) {
	stdin := pipeWithContent(t, `{"id":"source-id","name":"Test Corp","account_type":"Customer"}`)

	input, ok, err := readStdinBodyInput(stdin, "", "create")
	if err != nil {
		t.Fatalf("readStdinBodyInput() returned error: %v", err)
	}
	if !ok {
		t.Fatal("readStdinBodyInput() ok = false, expected true")
	}
	if input.raw {
		t.Fatal("flat input should not be raw JSON:API")
	}
	attrs, ok := input.body.(map[string]interface{})
	if !ok {
		t.Fatalf("input.body = %T, expected map", input.body)
	}
	if _, ok := attrs["id"]; ok {
		t.Error("flat create attributes should not include id")
	}
	if attrs["name"] != "Test Corp" {
		t.Errorf("attrs[name] = %v", attrs["name"])
	}
}

func TestReadStdinBodyInputValidatesFlatUpdateID(t *testing.T) {
	stdin := pipeWithContent(t, `{"id":"123","name":"Test Corp"}`)

	input, ok, err := readStdinBodyInput(stdin, "456", "update")
	if !ok {
		t.Fatal("readStdinBodyInput() ok = false, expected true")
	}
	if err == nil {
		t.Fatal("readStdinBodyInput() error = nil, expected id mismatch error")
	}
	if input != nil {
		t.Fatalf("input = %#v, expected nil", input)
	}
}

func TestReadStdinBodyInputValidatesJSONAPIUpdateID(t *testing.T) {
	stdin := pipeWithContent(t, `{"data":{"type":"accounts","id":"123","attributes":{"name":"Test Corp"}}}`)

	_, ok, err := readStdinBodyInput(stdin, "456", "update")
	if !ok {
		t.Fatal("readStdinBodyInput() ok = false, expected true")
	}
	if err == nil {
		t.Fatal("readStdinBodyInput() error = nil, expected id mismatch error")
	}
}

func TestBodyInputHasEmptyAttributes(t *testing.T) {
	if !bodyInputHasEmptyAttributes(&bodyInput{body: map[string]interface{}{}}) {
		t.Fatal("empty flat attributes should be empty")
	}
	if bodyInputHasEmptyAttributes(&bodyInput{body: map[string]interface{}{"name": "Acme"}}) {
		t.Fatal("non-empty flat attributes should not be empty")
	}
	if !bodyInputHasEmptyAttributes(&bodyInput{raw: true, body: map[string]interface{}{"data": map[string]interface{}{"attributes": map[string]interface{}{}}}}) {
		t.Fatal("empty JSON:API attributes should be empty")
	}
	if bodyInputHasEmptyAttributes(&bodyInput{raw: true, body: map[string]interface{}{"data": map[string]interface{}{"attributes": map[string]interface{}{"name": "Acme"}}}}) {
		t.Fatal("non-empty JSON:API attributes should not be empty")
	}
	if !bodyInputHasEmptyAttributes(&bodyInput{raw: true, body: map[string]interface{}{"data": map[string]interface{}{"type": "accounts"}}}) {
		t.Fatal("missing JSON:API attributes should be empty")
	}
	if !bodyInputHasEmptyAttributes(&bodyInput{raw: true, body: map[string]interface{}{"data": map[string]interface{}{"type": "accounts", "attributes": nil}}}) {
		t.Fatal("null JSON:API attributes should be empty")
	}
}

func TestReadStdinBodyInputStripsJSONAPICreateID(t *testing.T) {
	stdin := pipeWithContent(t, `{"data":{"type":"accounts","id":"client-id","attributes":{"name":"Test Corp"}}}`)

	input, ok, err := readStdinBodyInput(stdin, "", "create")
	if err != nil {
		t.Fatalf("readStdinBodyInput() returned error: %v", err)
	}
	if !ok {
		t.Fatal("readStdinBodyInput() ok = false, expected true")
	}
	body, ok := input.body.(map[string]interface{})
	if !ok {
		t.Fatalf("input.body = %T, expected map", input.body)
	}
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("body[data] = %T, expected map", body["data"])
	}
	if _, ok := data["id"]; ok {
		t.Fatal("JSON:API create body should not keep client-provided id")
	}
}

func pipeWithContent(t *testing.T, content string) *os.File {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned error: %v", err)
	}
	if _, err := writer.WriteString(content); err != nil {
		t.Fatalf("writer.WriteString() returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() returned error: %v", err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	return reader
}
