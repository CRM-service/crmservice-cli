package cmd

import (
	"encoding/json"
	"testing"
)

func TestSingleRecordRequestBodyCreate(t *testing.T) {
	input := &bodyInput{
		body: map[string]interface{}{
			"name":         "Acme Corp",
			"account_type": "Customer",
		},
	}

	body, err := singleRecordRequestBody("accounts", "", "create", input)
	if err != nil {
		t.Fatalf("singleRecordRequestBody() returned error: %v", err)
	}

	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("body[data] = %T, expected map", body["data"])
	}
	if data["type"] != "accounts" {
		t.Errorf("data[type] = %v", data["type"])
	}
	attrs, ok := data["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("data[attributes] = %T, expected map", data["attributes"])
	}
	if attrs["name"] != "Acme Corp" {
		t.Errorf("attrs[name] = %v", attrs["name"])
	}
}

func TestSingleRecordRequestBodyCreateRequiresFields(t *testing.T) {
	input := &bodyInput{body: map[string]interface{}{}}
	_, err := singleRecordRequestBody("accounts", "", "create", input)
	if err == nil {
		t.Fatal("singleRecordRequestBody() error = nil, expected error")
	}
}

func TestSingleRecordRequestBodyUpdateRequiresFields(t *testing.T) {
	input := &bodyInput{body: map[string]interface{}{}}
	_, err := singleRecordRequestBody("accounts", "123", "update", input)
	if err == nil {
		t.Fatal("singleRecordRequestBody() error = nil, expected error")
	}
}

func TestOutputDryRunRequestJSON(t *testing.T) {
	body := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "accounts",
			"attributes": map[string]interface{}{
				"name": "Acme Corp",
			},
		},
	}
	out := captureStdout(t, func() {
		if err := outputDryRunRequest("create", "accounts", "", body, "json"); err != nil {
			t.Fatalf("outputDryRunRequest() returned error: %v", err)
		}
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("json.Unmarshal() returned error: %v\noutput: %s", err, out)
	}
	if result["dry_run"] != true {
		t.Errorf("dry_run = %v", result["dry_run"])
	}
	if result["operation"] != "create" {
		t.Errorf("operation = %v", result["operation"])
	}
}

func TestOutputDeleteResultJSON(t *testing.T) {
	out := captureStdout(t, func() {
		if err := outputDeleteResult("accounts", "123", "json"); err != nil {
			t.Fatalf("outputDeleteResult() returned error: %v", err)
		}
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("json.Unmarshal() returned error: %v\noutput: %s", err, out)
	}
	if result["deleted"] != true {
		t.Errorf("deleted = %v", result["deleted"])
	}
	if result["module"] != "accounts" {
		t.Errorf("module = %v", result["module"])
	}
	if result["id"] != "123" {
		t.Errorf("id = %v", result["id"])
	}
}
