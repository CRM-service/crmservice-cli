package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"crmservice/internal/cache"
	"crmservice/internal/config"
)

func TestGetSchemaBodyUsesPersistedCache(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/api/v1/schema/accounts" {
			t.Errorf("request path = %q, expected /api/v1/schema/accounts", r.URL.Path)
		}
		writeTestResponse(t, w, `{"attributes":{"name":{"type":"string"}}}`)
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLSeconds:  1,
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

func TestGetSchemaBodyReturnsDataWhenCacheSaveFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestResponse(t, w, `{"attributes":{"name":{"type":"string"}}}`)
	}))
	defer server.Close()

	oldCfg := cfg
	cacheRoot := t.TempDir()
	schemaDir := filepath.Join(cacheRoot, "schema")
	if err := os.WriteFile(schemaDir, []byte("not-a-dir"), 0o600); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}
	cfg = &config.Config{
		Cache: config.CacheConfig{
			SchemaDir:   schemaDir,
			TTLSeconds:  1,
			AutoRefresh: true,
		},
		API: config.APIConfig{Timeout: 30},
	}
	t.Cleanup(func() { cfg = oldCfg })

	url := server.URL + "/api/v1"
	stderr := captureStderr(t, func() {
		body, err := getSchemaBody("accounts", url, "token", 1, false)
		if err != nil {
			t.Fatalf("getSchemaBody() returned error: %v", err)
		}
		if !strings.Contains(string(body), `"name"`) {
			t.Fatalf("body = %s, expected fetched schema", body)
		}
	})
	if !strings.Contains(stderr, "[CACHE] SAVE FAILED") {
		t.Fatalf("stderr = %q, expected cache save failure warning", stderr)
	}
}

func TestGetSchemaBodyRefreshesExpiredCache(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		writeTestResponse(t, w, `{"attributes":{"name":{"type":"string"}}}`)
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLSeconds:  1,
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
		writeTestResponse(t, w, fmt.Sprintf(`{"attributes":{"request_%d":{"type":"string"}}}`, requests))
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		Cache: config.CacheConfig{
			SchemaDir:   t.TempDir(),
			TTLSeconds:  1,
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
			TTLSeconds:  1,
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
