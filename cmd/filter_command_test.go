package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"crmservice/internal/config"
)

func setupFilterCommandTestConfig(t *testing.T) {
	t.Helper()
	oldCfg := cfg
	cfg = &config.Config{
		Auth: config.AuthConfig{
			Token: "token",
		},
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLDays:     1,
			AutoRefresh: true,
		},
		API: config.APIConfig{Timeout: 30},
	}
	t.Cleanup(func() { cfg = oldCfg })
}

func TestValidateModuleFilterSkipsEmptyFilter(t *testing.T) {
	if err := validateModuleFilter("accounts", "", "https://example.com/api/v1", "token", 0); err != nil {
		t.Fatalf("validateModuleFilter() error = %v, expected nil", err)
	}
}

func TestValidateModuleFilterRejectsUnknownField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestResponse(t, w, `{"attributes":{"name":{"type":"string"}}}`)
	}))
	defer server.Close()

	setupFilterCommandTestConfig(t)

	err := validateModuleFilter("accounts", `{"$eq":["missing_field","x"]}`, server.URL+"/api/v1", "token", 0)
	if err == nil {
		t.Fatal("validateModuleFilter() error = nil, expected unknown field error")
	}
	if !strings.Contains(err.Error(), "missing_field") {
		t.Fatalf("validateModuleFilter() error = %v, expected missing_field", err)
	}
}

func TestValidateModuleFilterRejectsUnknownBareField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestResponse(t, w, `{"attributes":{"name":{"type":"string"}}}`)
	}))
	defer server.Close()

	setupFilterCommandTestConfig(t)

	err := validateModuleFilter("accounts", `{"missing_field":"x"}`, server.URL+"/api/v1", "token", 0)
	if err == nil {
		t.Fatal("validateModuleFilter() error = nil, expected unknown field error")
	}
	if !strings.Contains(err.Error(), "missing_field") {
		t.Fatalf("validateModuleFilter() error = %v, expected missing_field", err)
	}
}

func TestValidateModuleFilterAcceptsKnownBareField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestResponse(t, w, `{"attributes":{"account_type":{"type":"string"}}}`)
	}))
	defer server.Close()

	setupFilterCommandTestConfig(t)

	err := validateModuleFilter("accounts", `{"account_type":"Customer"}`, server.URL+"/api/v1", "token", 0)
	if err != nil {
		t.Fatalf("validateModuleFilter() error = %v, expected nil", err)
	}
}

func TestRunListCommandRejectsUnknownBareField(t *testing.T) {
	listRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/accounts") {
			listRequests++
		}
		writeTestResponse(t, w, `{"data":[]}`)
	}))
	defer server.Close()

	setupFilterCommandTestConfig(t)

	cmd := listCmd()
	cmd.SetContext(context.Background())
	if err := cmd.Flags().Set("filter", `{"missing_field":"x"}`); err != nil {
		t.Fatalf("Set(filter) error: %v", err)
	}

	err := runListCommand(cmd, []string{"accounts"}, "")
	if err == nil {
		t.Fatal("runListCommand() error = nil, expected filter validation error")
	}
	if listRequests != 0 {
		t.Errorf("list requests = %d, want 0", listRequests)
	}
}

func TestRunListCommandRejectsInvalidFilterSyntax(t *testing.T) {
	listRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/accounts") {
			listRequests++
		}
		writeTestResponse(t, w, `{"data":[]}`)
	}))
	defer server.Close()

	setupFilterCommandTestConfig(t)

	cmd := listCmd()
	cmd.SetContext(context.Background())
	if err := cmd.Flags().Set("filter", `{"$unknown":["name","Acme"]}`); err != nil {
		t.Fatalf("Set(filter) error: %v", err)
	}

	err := runListCommand(cmd, []string{"accounts"}, "")
	if err == nil {
		t.Fatal("runListCommand() error = nil, expected filter validation error")
	}
	if listRequests != 0 {
		t.Errorf("list requests = %d, want 0", listRequests)
	}
}

func TestListCommandFilterValidationOmitsUsage(t *testing.T) {
	setupFilterCommandTestConfig(t)

	cmd := listCmd()
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{"accounts", "--filter", `{"$unknown":["name","Acme"]}`, "-o", "json"})

	stderr := captureStderr(t, func() {
		if err := cmd.Execute(); err == nil {
			t.Fatal("Execute() error = nil, expected filter validation error")
		}
	})
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr contains usage: %q", stderr)
	}
	if strings.Contains(stderr, "Flags:") {
		t.Fatalf("stderr contains flags help: %q", stderr)
	}
}

func TestRunCountCommandRejectsInvalidFilterSyntax(t *testing.T) {
	countRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/reporting") {
			countRequests++
		}
		writeTestResponse(t, w, `{"data":[{"count_value":1}]}`)
	}))
	defer server.Close()

	setupFilterCommandTestConfig(t)

	cmd := countCmd()
	cmd.SetContext(context.Background())

	err := runCountCommand(cmd, []string{"accounts", `{"$unknown":["name","Acme"]}`})
	if err == nil {
		t.Fatal("runCountCommand() error = nil, expected filter validation error")
	}
	if countRequests != 0 {
		t.Errorf("count requests = %d, want 0", countRequests)
	}
}
