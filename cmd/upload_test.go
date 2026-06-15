package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"crmservice/internal/config"
)

func TestUploadCmd(t *testing.T) {
	cmd := uploadCmd()

	t.Run("command structure", func(t *testing.T) {
		if cmd.Use != "upload <module> <id> <file>" {
			t.Errorf("Use = %q, expected upload <module> <id> <file>", cmd.Use)
		}
		if cmd.Short != "Upload a file and link it to an entity" {
			t.Errorf("Short = %q", cmd.Short)
		}
	})

	t.Run("args validation", func(t *testing.T) {
		if err := cmd.ValidateArgs([]string{"accounts", "123", "file.pdf"}); err != nil {
			t.Errorf("Expected no error for valid args, got: %v", err)
		}
		if err := cmd.ValidateArgs([]string{"accounts", "123"}); err == nil {
			t.Error("Expected error for missing file path")
		}
	})

	t.Run("flags", func(t *testing.T) {
		flags := []string{"field", "dry-run", "output", "full", "verbose"}
		for _, name := range flags {
			if cmd.Flags().Lookup(name) == nil {
				t.Errorf("Missing flag: %s", name)
			}
		}
	})
}

func TestUploadCmdUploadsFileToEntity(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "entity-doc.txt")
	if err := os.WriteFile(tempFile, []byte("entity attachment"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("Content-Type = %q, want multipart/form-data", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		writeTestResponse(t, w, `{"data":{"id":"55","type":"files","attributes":{"file_name":"entity-doc.txt"}}}`)
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		API:  config.APIConfig{URL: server.URL + "/api/v1", Timeout: 5},
		Auth: config.AuthConfig{Token: "token"},
	}
	t.Cleanup(func() { cfg = oldCfg })

	cmd := uploadCmd()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"entities", "123", tempFile, "-o", "json"})

	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() error: %v", err)
		}
	})

	if gotPath != "/api/v1/entities/123/files" {
		t.Errorf("request path = %q, want /api/v1/entities/123/files", gotPath)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("json.Unmarshal() error: %v\noutput: %s", err, out)
	}
	if result["id"] != "55" {
		t.Errorf("id = %v, want 55", result["id"])
	}
}

func TestUploadCmdDryRun(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "dry-run.txt")
	if err := os.WriteFile(tempFile, []byte("dry run"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	uploadRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/schema/"):
			writeTestResponse(t, w, `{"attributes":{"file_usage_type":{"type":"string"}}}`)
		default:
			uploadRequests++
		}
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		API:  config.APIConfig{URL: server.URL + "/api/v1", Timeout: 5},
		Auth: config.AuthConfig{Token: "token"},
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLSeconds:  1,
			AutoRefresh: true,
		},
	}
	t.Cleanup(func() { cfg = oldCfg })

	cmd := uploadCmd()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{
		"accounts", "999", tempFile,
		"--field", `file_usage_type=Entity Attachment`,
		"--dry-run", "-o", "json",
	})

	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() error: %v", err)
		}
	})

	if uploadRequests != 0 {
		t.Fatalf("upload requests = %d, want 0", uploadRequests)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("json.Unmarshal() error: %v\noutput: %q", err, out)
	}
	if result["dry_run"] != true {
		t.Errorf("dry_run = %v, want true", result["dry_run"])
	}
	if result["path"] != "/accounts/999/files" {
		t.Errorf("path = %v, want /accounts/999/files", result["path"])
	}
	attrs, ok := result["file_attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("file_attributes = %T, want map", result["file_attributes"])
	}
	if attrs["file_usage_type"] != "Entity Attachment" {
		t.Errorf("file_usage_type = %v", attrs["file_usage_type"])
	}
}

func TestUploadCmdRejectsUnknownFieldAgainstSchema(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "reject.txt")
	if err := os.WriteFile(tempFile, []byte("reject"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	uploadRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/schema/"):
			writeTestResponse(t, w, `{"attributes":{"file_name":{"type":"string"}}}`)
		default:
			uploadRequests++
		}
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		API:  config.APIConfig{URL: server.URL + "/api/v1", Timeout: 5},
		Auth: config.AuthConfig{Token: "token"},
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLSeconds:  1,
			AutoRefresh: true,
		},
	}
	t.Cleanup(func() { cfg = oldCfg })

	cmd := uploadCmd()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"accounts", "123", tempFile, "--field", `unknown_field=value`, "-o", "json"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute() error = nil, expected unknown field error")
	}
	if uploadRequests != 0 {
		t.Fatalf("upload requests = %d, want 0", uploadRequests)
	}
}

func TestUploadCmdRejectsMissingFile(t *testing.T) {
	oldCfg := cfg
	cfg = &config.Config{
		API:  config.APIConfig{URL: "https://example.com/api/v1", Timeout: 5},
		Auth: config.AuthConfig{Token: "token"},
	}
	t.Cleanup(func() { cfg = oldCfg })

	cmd := uploadCmd()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"accounts", "123", filepath.Join(t.TempDir(), "missing.txt"), "-o", "json"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute() error = nil, expected missing file error")
	}
}