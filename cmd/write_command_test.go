package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"crmservice/internal/config"
)

func TestCreateCmdRejectsUnknownFieldAgainstSchema(t *testing.T) {
	createRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/schema/"):
			writeTestResponse(t, w, `{"attributes":{"name":{"type":"string"}}}`)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/accounts"):
			createRequests++
			writeTestResponse(t, w, `{"data":{"id":"1","type":"accounts","attributes":{"name":"Acme"}}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		API:  config.APIConfig{URL: server.URL + "/api/v1", Timeout: 5},
		Auth: config.AuthConfig{Token: "token"},
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLDays:     1,
			AutoRefresh: true,
		},
	}
	t.Cleanup(func() { cfg = oldCfg })

	cmd := createCmd()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"accounts", "--field", `abc=new value`, "-o", "json"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute() error = nil, expected unknown field error")
	}
	if createRequests != 0 {
		t.Fatalf("create requests = %d, want 0", createRequests)
	}
}

func TestUpdateCmdRejectsUnknownFieldAgainstSchema(t *testing.T) {
	updateRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/schema/"):
			writeTestResponse(t, w, `{"attributes":{"name":{"type":"string"}}}`)
		case r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/accounts/123"):
			updateRequests++
			writeTestResponse(t, w, `{"data":{"id":"123","type":"accounts","attributes":{"name":"Acme"}}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		API:  config.APIConfig{URL: server.URL + "/api/v1", Timeout: 5},
		Auth: config.AuthConfig{Token: "token"},
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLDays:     1,
			AutoRefresh: true,
		},
	}
	t.Cleanup(func() { cfg = oldCfg })

	cmd := updateCmd()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"accounts", "123", "--field", `abc=new value`, "-o", "json"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute() error = nil, expected unknown field error")
	}
	if updateRequests != 0 {
		t.Fatalf("update requests = %d, want 0", updateRequests)
	}
}
