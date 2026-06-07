package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"crmservice/internal/api"
	"crmservice/internal/config"
)

func TestBulkCommands(t *testing.T) {
	create := bulkCreateCmd()
	if create.Use != "bulk-create <module>" {
		t.Errorf("bulk-create Use = %q", create.Use)
	}
	update := bulkUpdateCmd()
	if update.Use != "bulk-update <module>" {
		t.Errorf("bulk-update Use = %q", update.Use)
	}

	for _, cmd := range []string{"continue-on-error", "dry-run", "concurrency", "skip-empty", "summary", "output", "full", "verbose"} {
		if create.Flags().Lookup(cmd) == nil {
			t.Errorf("bulk-create missing flag %s", cmd)
		}
		if update.Flags().Lookup(cmd) == nil {
			t.Errorf("bulk-update missing flag %s", cmd)
		}
	}
}

func TestParseBulkJSONL(t *testing.T) {
	records, err := parseBulkJSONL([]byte("{\"id\":\"1\",\"name\":\"Acme\"}\n{\"id\":\"2\",\"name\":\"Example\"}\n"))
	if err != nil {
		t.Fatalf("parseBulkJSONL() returned error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, expected 2", len(records))
	}
	if records[1]["name"] != "Example" {
		t.Errorf("records[1][name] = %v", records[1]["name"])
	}
}

func TestParseBulkJSONArray(t *testing.T) {
	records, err := parseBulkJSONArray([]byte(`[{"id":"1","name":"Acme"},{"id":"2","name":"Example"}]`))
	if err != nil {
		t.Fatalf("parseBulkJSONArray() returned error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, expected 2", len(records))
	}
}

func TestParseBulkJSONObjectRejectsJSONL(t *testing.T) {
	_, err := parseBulkJSONObject([]byte("{\"id\":\"1\"}\n{\"id\":\"2\"}"))
	if err == nil {
		t.Fatal("parseBulkJSONObject() error = nil, expected error")
	}
}

func TestBulkRequestBodyCreateDropsID(t *testing.T) {
	body, id, err := bulkRequestBody("accounts", "create", bulkRecord{"id": "297603", "name": "Acme"})
	if err != nil {
		t.Fatalf("bulkRequestBody() returned error: %v", err)
	}
	if id != "297603" {
		t.Errorf("id = %q, expected 297603", id)
	}
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("body[data] = %T, expected map", body["data"])
	}
	if _, ok := data["id"]; ok {
		t.Error("create body must not include client-provided id")
	}
	attrs, ok := data["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("data[attributes] = %T, expected map", data["attributes"])
	}
	if attrs["name"] != "Acme" {
		t.Errorf("attrs[name] = %v", attrs["name"])
	}
}

func TestBulkRequestBodyUpdateRequiresID(t *testing.T) {
	_, _, err := bulkRequestBody("accounts", "update", bulkRecord{"name": "Acme"})
	if err == nil {
		t.Fatal("bulkRequestBody() error = nil, expected error")
	}
}

func TestBulkRequestBodyUpdateJSONAPINumericID(t *testing.T) {
	record := bulkRecord{
		"data": map[string]interface{}{
			"type": "accounts",
			"id":   json.Number("297603"),
			"attributes": map[string]interface{}{
				"name": "Acme",
			},
		},
	}

	body, id, err := bulkRequestBody("accounts", "update", record)
	if err != nil {
		t.Fatalf("bulkRequestBody() returned error: %v", err)
	}
	if id != "297603" {
		t.Errorf("id = %q, expected 297603", id)
	}

	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("body[data] = %T, expected map", body["data"])
	}
	if data["id"] != "297603" {
		t.Errorf("data[id] = %v, expected string 297603", data["id"])
	}
}

func TestBulkRequestBodyUpdateFlatNumericID(t *testing.T) {
	body, id, err := bulkRequestBody("accounts", "update", bulkRecord{"id": json.Number("42"), "name": "Acme"})
	if err != nil {
		t.Fatalf("bulkRequestBody() returned error: %v", err)
	}
	if id != "42" {
		t.Errorf("id = %q, expected 42", id)
	}
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("body[data] = %T, expected map", body["data"])
	}
	if data["id"] != "42" {
		t.Errorf("data[id] = %v, expected 42", data["id"])
	}
}

func TestProcessBulkUpdateJSONAPINumericIDPath(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, expected PATCH", r.Method)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if _, err := w.Write([]byte(`{"data":{"id":"297603","type":"accounts","attributes":{"name":"Acme"}}}`)); err != nil {
			t.Errorf("Write() returned error: %v", err)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "token")
	record := bulkRecord{
		"data": map[string]interface{}{
			"type":       "accounts",
			"id":         json.Number("297603"),
			"attributes": map[string]interface{}{"name": "Acme"},
		},
	}
	results, summary, err := processBulkRecords(context.Background(), client, "accounts", "update", []bulkRecord{record}, bulkOptions{Concurrency: 1})
	if err != nil {
		t.Fatalf("processBulkRecords() returned error: %v", err)
	}
	if summary.Succeeded != 1 || len(results) != 1 {
		t.Fatalf("summary/results = %+v/%d", summary, len(results))
	}
	if len(paths) != 1 || paths[0] != "/accounts/297603" {
		t.Errorf("paths = %v, expected [/accounts/297603]", paths)
	}
}

func TestProcessBulkCreate(t *testing.T) {
	var bodies []map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, expected POST", r.Method)
		}
		if r.URL.Path != "/accounts" {
			t.Errorf("path = %s, expected /accounts", r.URL.Path)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write([]byte(`{"data":{"id":"new-id","type":"accounts","attributes":{"name":"Acme"}}}`)); err != nil {
			t.Errorf("Write() returned error: %v", err)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "token")
	results, summary, err := processBulkRecords(context.Background(), client, "accounts", "create", []bulkRecord{{"id": "old-id", "name": "Acme"}}, bulkOptions{Concurrency: 1})
	if err != nil {
		t.Fatalf("processBulkRecords() returned error: %v", err)
	}
	if summary.Succeeded != 1 || len(results) != 1 {
		t.Fatalf("summary/results = %+v/%d", summary, len(results))
	}
	data, ok := bodies[0]["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("bodies[0][data] = %T, expected map", bodies[0]["data"])
	}
	if _, ok := data["id"]; ok {
		t.Error("POST body must not include source id")
	}
}

func TestRunBulkCommandContinueOnErrorAllFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`{"errors":[{"detail":"bad request"}]}`)); err != nil {
			t.Errorf("Write() returned error: %v", err)
		}
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		API:  config.APIConfig{URL: server.URL + "/api/v1", Timeout: 5},
		Auth: config.AuthConfig{Token: "token"},
	}
	t.Cleanup(func() { cfg = oldCfg })

	cmd := bulkCreateCmd()
	cmd.SetArgs([]string{"accounts", "--continue-on-error", "--summary", "-o", "json"})
	cmd.SetIn(strings.NewReader("{\"name\":\"Acme\"}\n"))

	stdout := captureBulkStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() returned error: %v", err)
		}
	})
	if strings.TrimSpace(stdout) == "" {
		t.Fatal("bulk summary output is empty")
	}
}

func TestOutputBulkSummaryJSONL(t *testing.T) {
	out := captureBulkStdout(t, func() {
		if err := outputBulkSummary(bulkSummary{Operation: "create", Module: "accounts", Total: 2, Succeeded: 2}, "jsonl"); err != nil {
			t.Fatalf("outputBulkSummary() returned error: %v", err)
		}
	})
	if bytes.Count([]byte(out), []byte("\n")) != 1 {
		t.Errorf("jsonl summary should be one line, got: %q", out)
	}
}

func captureBulkStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned error: %v", err)
	}
	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("writer.Close() returned error: %v", err)
	}
	os.Stdout = oldStdout
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("io.ReadAll() returned error: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("reader.Close() returned error: %v", err)
	}
	return string(out)
}
