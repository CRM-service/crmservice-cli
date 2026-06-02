package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"crmservice/internal/api"
	"crmservice/internal/cache"
	"crmservice/internal/config"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

func TestValidOutputFormat(t *testing.T) {
	t.Run("valid formats", func(t *testing.T) {
		t.Parallel()
		formats := []string{"table", "json", "yaml", "csv"}
		for _, format := range formats {
			if !ValidOutputFormat(format) {
				t.Errorf("ValidOutputFormat(%q) = false, expected true", format)
			}
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		t.Parallel()
		if ValidOutputFormat("invalid") {
			t.Error("ValidOutputFormat(\"invalid\") = true, expected false")
		}
		if ValidOutputFormat("") {
			t.Error("ValidOutputFormat(\"\") = true, expected false")
		}
		if ValidOutputFormat("xml") {
			t.Error("ValidOutputFormat(\"xml\") = true, expected false")
		}
	})
}

func TestListCmd(t *testing.T) {
	cmd := listCmd()

	t.Run("command structure", func(t *testing.T) {
		if cmd.Use != "list <module>" {
			t.Errorf("Use = %q, expected %q", cmd.Use, "list <module>")
		}
		if cmd.Short != "List records" {
			t.Errorf("Short = %q, expected %q", cmd.Short, "List records")
		}
	})

	t.Run("args validation", func(t *testing.T) {
		err := cmd.ValidateArgs([]string{"module"})
		if err != nil {
			t.Errorf("Expected no error for valid args, got: %v", err)
		}

		err = cmd.ValidateArgs([]string{})
		if err == nil {
			t.Error("Expected error for empty args")
		}

		err = cmd.ValidateArgs([]string{"one", "two"})
		if err == nil {
			t.Error("Expected error for too many args")
		}
	})

	t.Run("flags", func(t *testing.T) {
		flags := []string{"page-size", "fields", "output", "filter", "full", "verbose", "page", "offset"}
		for _, name := range flags {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("Missing flag: %s", name)
			}
		}
	})

	t.Run("output format validation", func(t *testing.T) {
		testCases := []struct {
			name      string
			output    string
			valid     bool
			filter    string
			shouldErr bool
		}{
			{"valid table format", "table", true, "", false},
			{"valid json format", "json", true, "", false},
			{"valid yaml format", "yaml", true, "", false},
			{"valid csv format", "csv", true, "", false},
			{"invalid format", "xml", false, "", true},
			{"invalid with filter", "invalid", false, `{"name":"test"}`, true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.shouldErr != !tc.valid {
					if tc.valid && !ValidOutputFormat(tc.output) {
						t.Errorf("Format %s should be valid", tc.output)
					}
					if !tc.valid && ValidOutputFormat(tc.output) {
						t.Errorf("Format %s should be invalid", tc.output)
					}
				}
			})
		}
	})
}

func TestGetCmd(t *testing.T) {
	cmd := getCmd()

	t.Run("command structure", func(t *testing.T) {
		if cmd.Use != "get <module> <id>" {
			t.Errorf("Use = %q, expected %q", cmd.Use, "get <module> <id>")
		}
		if cmd.Short != "Get record by ID" {
			t.Errorf("Short = %q, expected %q", cmd.Short, "Get record by ID")
		}
	})

	t.Run("args validation", func(t *testing.T) {
		err := cmd.ValidateArgs([]string{"module", "123"})
		if err != nil {
			t.Errorf("Expected no error for valid args, got: %v", err)
		}

		err = cmd.ValidateArgs([]string{"module"})
		if err == nil {
			t.Error("Expected error for missing id")
		}

		err = cmd.ValidateArgs([]string{"module", "123", "extra"})
		if err == nil {
			t.Error("Expected error for too many args")
		}
	})

	t.Run("flags", func(t *testing.T) {
		flags := []string{"fields", "output", "full", "verbose"}
		for _, name := range flags {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("Missing flag: %s", name)
			}
		}
	})
}

func TestReadStdinJSONAPIRequest(t *testing.T) {
	stdin := pipeWithContent(t, `{"data":{"type":"accounts","attributes":{"name":"Test Corp"}}}`)

	body, ok, err := readStdinJSONAPIRequest(stdin)
	if err != nil {
		t.Fatalf("readStdinJSONAPIRequest() returned error: %v", err)
	}
	if !ok {
		t.Fatal("readStdinJSONAPIRequest() ok = false, expected true")
	}
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("body[data] = %T, expected object", body["data"])
	}
	if data["type"] != "accounts" {
		t.Errorf("data[type] = %q, expected accounts", data["type"])
	}
}

func TestReadStdinJSONAPIRequestRejectsInvalidBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{`},
		{name: "missing data", body: `{"attributes":{"name":"Test Corp"}}`},
		{name: "data not object", body: `{"data":[]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdin := pipeWithContent(t, tt.body)
			_, ok, err := readStdinJSONAPIRequest(stdin)
			if !ok {
				t.Fatal("readStdinJSONAPIRequest() ok = false, expected true")
			}
			if err == nil {
				t.Fatal("readStdinJSONAPIRequest() error = nil, expected error")
			}
		})
	}
}

func TestReadStdinJSONAPIRequestIgnoresEmptyBody(t *testing.T) {
	stdin := pipeWithContent(t, "\n  \t")

	_, ok, err := readStdinJSONAPIRequest(stdin)
	if err != nil {
		t.Fatalf("readStdinJSONAPIRequest() returned error: %v", err)
	}
	if ok {
		t.Fatal("readStdinJSONAPIRequest() ok = true, expected false")
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

func TestGetSchemaBodyUsesPersistedCache(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/api/v1/schema/accounts" {
			t.Errorf("request path = %q, expected /api/v1/schema/accounts", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"attributes":{"name":{"type":"string"}}}`))
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLDays:     1,
			AutoRefresh: true,
		},
		API: config.APIConfig{Timeout: 30},
	}
	t.Cleanup(func() { cfg = oldCfg })

	url := server.URL + "/api/v1"
	first, err := getSchemaBody("accounts", url, "token", 0, false)
	if err != nil {
		t.Fatalf("getSchemaBody() first call returned error: %v", err)
	}
	second, err := getSchemaBody("accounts", url, "token", 0, false)
	if err != nil {
		t.Fatalf("getSchemaBody() second call returned error: %v", err)
	}
	if string(first) != string(second) {
		t.Errorf("cached body = %s, expected %s", second, first)
	}
	if requests != 1 {
		t.Errorf("requests = %d, expected 1", requests)
	}
}

func TestGetSchemaBodyRefreshesExpiredCache(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte(`{"attributes":{"name":{"type":"string"}}}`))
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLDays:     1,
			AutoRefresh: true,
		},
		API: config.APIConfig{Timeout: 30},
	}
	t.Cleanup(func() { cfg = oldCfg })

	url := server.URL + "/api/v1"
	if _, err := getSchemaBody("accounts", url, "token", 0, false); err != nil {
		t.Fatalf("getSchemaBody() first call returned error: %v", err)
	}

	schemaCache := cache.NewSchemaCache(getSchemaCacheDir(url))
	cached := cache.CachedSchema{
		Module:    "accounts",
		Data:      []byte(`{"attributes":{"old":{"type":"string"}}}`),
		FetchedAt: time.Now().Add(-48 * time.Hour),
	}
	cachedData, err := json.Marshal(cached)
	if err != nil {
		t.Fatalf("Marshal() returned error: %v", err)
	}
	if err := os.WriteFile(schemaCache.GetPath("accounts"), cachedData, 0o600); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}

	if _, err := getSchemaBody("accounts", url, "token", 0, false); err != nil {
		t.Fatalf("getSchemaBody() second call returned error: %v", err)
	}
	if requests != 2 {
		t.Errorf("requests = %d, expected 2", requests)
	}
}

func TestGetSchemaBodyForceRefreshesCache(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte(fmt.Sprintf(`{"attributes":{"request_%d":{"type":"string"}}}`, requests)))
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLDays:     1,
			AutoRefresh: true,
		},
		API: config.APIConfig{Timeout: 30},
	}
	t.Cleanup(func() { cfg = oldCfg })

	url := server.URL + "/api/v1"
	if _, err := getSchemaBody("accounts", url, "token", 0, false); err != nil {
		t.Fatalf("getSchemaBody() first call returned error: %v", err)
	}
	body, err := getSchemaBody("accounts", url, "token", 0, true)
	if err != nil {
		t.Fatalf("getSchemaBody() force call returned error: %v", err)
	}
	if requests != 2 {
		t.Errorf("requests = %d, expected 2", requests)
	}
	if !strings.Contains(string(body), "request_2") {
		t.Errorf("body = %s, expected refreshed response", body)
	}
}

func TestGetSchemaBodyDoesNotRefreshWhenDisabled(t *testing.T) {
	oldCfg := cfg
	cfg = &config.Config{
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLDays:     1,
			AutoRefresh: false,
		},
		API: config.APIConfig{Timeout: 30},
	}
	t.Cleanup(func() { cfg = oldCfg })

	_, err := getSchemaBody("accounts", "https://example.com/api/v1", "token", 0, false)
	if err == nil {
		t.Fatal("getSchemaBody() error = nil, expected cache miss error")
	}
}

func TestCreateCmd(t *testing.T) {
	cmd := createCmd()

	t.Run("command structure", func(t *testing.T) {
		if cmd.Use != "create <module>" {
			t.Errorf("Use = %q, expected %q", cmd.Use, "create <module>")
		}
		if cmd.Short != "Create a new record" {
			t.Errorf("Short = %q, expected %q", cmd.Short, "Create a new record")
		}
	})

	t.Run("flags", func(t *testing.T) {
		flags := []string{"field", "output", "full", "verbose"}
		for _, name := range flags {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("Missing flag: %s", name)
			}
		}
	})
}

func TestUpdateCmd(t *testing.T) {
	cmd := updateCmd()

	t.Run("command structure", func(t *testing.T) {
		if cmd.Use != "update <module> <id>" {
			t.Errorf("Use = %q, expected %q", cmd.Use, "update <module> <id>")
		}
		if cmd.Short != "Update a record" {
			t.Errorf("Short = %q, expected %q", cmd.Short, "Update a record")
		}
	})

	t.Run("flags", func(t *testing.T) {
		flags := []string{"field", "output", "full", "verbose"}
		for _, name := range flags {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("Missing flag: %s", name)
			}
		}
	})
}

func TestDeleteCmd(t *testing.T) {
	cmd := deleteCmd()

	t.Run("command structure", func(t *testing.T) {
		if cmd.Use != "delete <module> <id>" {
			t.Errorf("Use = %q, expected %q", cmd.Use, "delete <module> <id>")
		}
		if cmd.Short != "Delete a record" {
			t.Errorf("Short = %q, expected %q", cmd.Short, "Delete a record")
		}
	})

	t.Run("args validation", func(t *testing.T) {
		err := cmd.ValidateArgs([]string{"module", "123"})
		if err != nil {
			t.Errorf("Expected no error for valid args, got: %v", err)
		}

		err = cmd.ValidateArgs([]string{"module"})
		if err == nil {
			t.Error("Expected error for missing id")
		}
	})
}

func TestFieldsCmd(t *testing.T) {
	cmd := fieldsCmd()

	t.Run("command structure", func(t *testing.T) {
		if cmd.Use != "fields <module>" {
			t.Errorf("Use = %q, expected %q", cmd.Use, "fields <module>")
		}
		if cmd.Short != "Show available fields for a module" {
			t.Errorf("Short = %q, expected %q", cmd.Short, "Show available fields for a module")
		}
	})

	t.Run("args validation", func(t *testing.T) {
		err := cmd.ValidateArgs([]string{"module"})
		if err != nil {
			t.Errorf("Expected no error for valid args, got: %v", err)
		}

		err = cmd.ValidateArgs([]string{})
		if err == nil {
			t.Error("Expected error for missing module")
		}
	})

	t.Run("flags", func(t *testing.T) {
		flags := []string{"fields", "output", "force", "verbose"}
		for _, name := range flags {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("Missing flag: %s", name)
			}
		}
	})
}

func TestSearchCmd(t *testing.T) {
	cmd := searchCmd()

	t.Run("command structure", func(t *testing.T) {
		if cmd.Use != "search <module> <filter>" {
			t.Errorf("Use = %q, expected %q", cmd.Use, "search <module> <filter>")
		}
		if cmd.Short != "Search records (alias for list --filter)" {
			t.Errorf("Short = %q, expected %q", cmd.Short, "Search records (alias for list --filter)")
		}
	})

	t.Run("args validation", func(t *testing.T) {
		err := cmd.ValidateArgs([]string{"module", `{"$eq":["name","Acme"]}`})
		if err != nil {
			t.Errorf("Expected no error for valid args, got: %v", err)
		}

		err = cmd.ValidateArgs([]string{"module"})
		if err == nil {
			t.Error("Expected error for missing filter")
		}
	})

	t.Run("flags", func(t *testing.T) {
		flags := []string{"page-size", "include", "fields", "output", "full", "verbose", "page", "offset"}
		for _, name := range flags {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("Missing flag: %s", name)
			}
		}

		if flag := cmd.Flags().Lookup("filter"); flag != nil {
			t.Error("search should not expose --filter because the filter is positional")
		}
	})
}

func TestModulesCmd(t *testing.T) {
	cmd := modulesCmd()

	t.Run("command structure", func(t *testing.T) {
		if cmd.Use != "modules" {
			t.Errorf("Use = %q, expected %q", cmd.Use, "modules")
		}
		if cmd.Short != "List API modules" {
			t.Errorf("Short = %q, expected %q", cmd.Short, "List API modules")
		}
	})

	t.Run("flags", func(t *testing.T) {
		flags := []string{"url", "output", "full", "verbose"}
		for _, name := range flags {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("Missing flag: %s", name)
			}
		}
	})
}

func TestGetURLFromFlagOrEnv(t *testing.T) {
	t.Setenv("CRMSERVICE_API_URL", "https://example.com")

	testCases := []struct {
		name     string
		flagURL  string
		envURL   string
		expected string
	}{
		{
			name:     "flag takes precedence",
			flagURL:  "https://flag.example.com",
			envURL:   "https://env.example.com",
			expected: "https://flag.example.com/api/v1",
		},
		{
			name:     "environment variable used",
			flagURL:  "",
			envURL:   "https://env.example.com",
			expected: "https://env.example.com/api/v1",
		},
		{
			name:     "http converted to https",
			flagURL:  "http://example.com",
			envURL:   "",
			expected: "https://example.com/api/v1",
		},
		{
			name:     "api/v1 suffix added",
			flagURL:  "https://example.com",
			envURL:   "",
			expected: "https://example.com/api/v1",
		},
		{
			name:     "trailing slash removed",
			flagURL:  "https://example.com/",
			envURL:   "",
			expected: "https://example.com/api/v1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CRMSERVICE_API_URL", tc.envURL)
			cmd := &cobra.Command{}
			cmd.Flags().String("url", "", "")
			if err := cmd.Flags().Set("url", tc.flagURL); err != nil {
				t.Errorf("Failed to set flag: %v", err)
			}

			result := getURLFromFlagOrEnv(cmd)
			if !strings.HasPrefix(result, tc.expected) {
				t.Errorf("getURLFromFlagOrEnv() = %q, expected %q", result, tc.expected)
			}
		})
	}
}

func TestConfigFallbacks(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = &config.Config{
		API: config.APIConfig{
			URL:     "https://config.example.com",
			Timeout: 42,
		},
		Auth: config.AuthConfig{
			Token: "config-token",
		},
		Output: config.OutputConfig{
			Format:   "json",
			PageSize: 75,
		},
	}
	t.Setenv("CRMSERVICE_API_URL", "")
	t.Setenv("CRMSERVICE_AUTH_TOKEN", "")

	cmd := &cobra.Command{}
	cmd.Flags().String("url", "", "")
	cmd.Flags().String("token", "", "")
	cmd.Flags().StringP("output", "o", "table", "")
	cmd.Flags().Int("page-size", 20, "")

	if got := getURLFromFlagOrEnv(cmd); got != "https://config.example.com/api/v1" {
		t.Errorf("getURLFromFlagOrEnv() = %q", got)
	}

	token, err := getTokenFromFlagEnvConfig(cmd)
	if err != nil {
		t.Fatalf("getTokenFromFlagEnvConfig() returned error: %v", err)
	}
	if token != "config-token" {
		t.Errorf("token = %q", token)
	}

	outputFormat, err := getOutputFormatFromFlagConfig(cmd)
	if err != nil {
		t.Fatalf("getOutputFormatFromFlagConfig() returned error: %v", err)
	}
	if outputFormat != "json" {
		t.Errorf("outputFormat = %q", outputFormat)
	}

	pageSize, err := getPageSizeFromFlagConfig(cmd)
	if err != nil {
		t.Fatalf("getPageSizeFromFlagConfig() returned error: %v", err)
	}
	if pageSize != 75 {
		t.Errorf("pageSize = %d", pageSize)
	}

	if timeout := getTimeoutFromConfig(); timeout != 42*time.Second {
		t.Errorf("timeout = %v", timeout)
	}
}

func TestRequiredTokenValidation(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = nil
	t.Setenv("CRMSERVICE_AUTH_TOKEN", "")

	cmd := &cobra.Command{}
	cmd.Flags().String("token", "", "")

	_, err := getRequiredTokenFromFlagEnvConfig(cmd)
	if err == nil {
		t.Fatal("expected error for missing API token")
	}
	if !strings.Contains(err.Error(), "API token not provided") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestErrorResponse(t *testing.T) {
	t.Run("API error", func(t *testing.T) {
		apiErr := &api.Error{
			Status:  404,
			Body:    []byte(`{"error":"Not found"}`),
			Message: "Not found",
		}

		origStderr := os.Stderr
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		os.Stderr = w
		defer func() { os.Stderr = origStderr }()

		err = output.ErrorResponse(apiErr)
		if err != apiErr {
			t.Error("ErrorResponse should return the same error")
		}

		w.Close()
		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("Failed to read from pipe: %v", err)
		}

		if !strings.Contains(string(out), "Error: Not found") {
			t.Errorf("Expected error message, got: %s", string(out))
		}
	})

	t.Run("regular error", func(t *testing.T) {
		origStderr := os.Stderr
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		os.Stderr = w
		defer func() { os.Stderr = origStderr }()

		regErr := fmt.Errorf("test error")

		err = output.ErrorResponse(regErr)
		if err != regErr {
			t.Error("ErrorResponse should return the same error")
		}

		w.Close()
		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("Failed to read from pipe: %v", err)
		}

		if !strings.Contains(string(out), "Error: test error") {
			t.Errorf("Expected error message, got: %s", string(out))
		}
	})
}
