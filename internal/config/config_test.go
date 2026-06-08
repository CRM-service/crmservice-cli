package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	clearEnv(t)
	cacheDir := t.TempDir()
	setConfigDir(t, t.TempDir())
	setCacheDir(t, cacheDir)

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}

	if cfg.API.Timeout != 30 {
		t.Errorf("API.Timeout = %d, expected 30", cfg.API.Timeout)
	}
	if cfg.Output.Format != "table" {
		t.Errorf("Output.Format = %q, expected table", cfg.Output.Format)
	}
	if cfg.Output.PageSize != 20 {
		t.Errorf("Output.PageSize = %d, expected 20", cfg.Output.PageSize)
	}
	if cfg.Cache.TTLDays != 24 {
		t.Errorf("Cache.TTLDays = %d, expected 24", cfg.Cache.TTLDays)
	}
	if cfg.Cache.SchemaDir != filepath.Join(cacheDir, "crmservice", "schema") {
		t.Errorf("Cache.SchemaDir = %q", cfg.Cache.SchemaDir)
	}
	if !cfg.Cache.AutoRefresh {
		t.Error("Cache.AutoRefresh = false, expected true")
	}
}

func TestLoadConfigYAML(t *testing.T) {
	clearEnv(t)

	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`api:
  url: "https://example.com/api/v1"
  timeout: 10
output:
  format: "json"
  page_size: 50
cache:
  schema_dir: "/tmp/schema"
  ttl_days: 7
  auto_refresh: false
auth:
  token: "token-from-file"
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}

	if cfg.API.URL != "https://example.com/api/v1" {
		t.Errorf("API.URL = %q", cfg.API.URL)
	}
	if cfg.API.Timeout != 10 {
		t.Errorf("API.Timeout = %d", cfg.API.Timeout)
	}
	if cfg.Output.Format != "json" || cfg.Output.PageSize != 50 {
		t.Errorf("Output = %+v", cfg.Output)
	}
	if cfg.Cache.SchemaDir != "/tmp/schema" || cfg.Cache.TTLDays != 7 || cfg.Cache.AutoRefresh {
		t.Errorf("Cache = %+v", cfg.Cache)
	}
	if cfg.Auth.Token != "token-from-file" {
		t.Errorf("Auth.Token = %q", cfg.Auth.Token)
	}
}

func TestLoadConfigDefaultPath(t *testing.T) {
	clearEnv(t)

	configHome := t.TempDir()
	setConfigDir(t, configHome)
	configDir := filepath.Join(configHome, "crmservice")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("MkdirAll() returned error: %v", err)
	}
	path := filepath.Join(configDir, "config.yaml")
	content := []byte(`api:
  url: "https://default.example.com/api/v1"
auth:
  token: "token-from-default-path"
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}

	if cfg.API.URL != "https://default.example.com/api/v1" {
		t.Errorf("API.URL = %q", cfg.API.URL)
	}
	if cfg.Auth.Token != "token-from-default-path" {
		t.Errorf("Auth.Token = %q", cfg.Auth.Token)
	}
}

func TestLoadConfigExplicitPathOverridesDefault(t *testing.T) {
	clearEnv(t)

	configHome := t.TempDir()
	setConfigDir(t, configHome)
	defaultDir := filepath.Join(configHome, "crmservice")
	if err := os.MkdirAll(defaultDir, 0o700); err != nil {
		t.Fatalf("MkdirAll() returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(defaultDir, "config.yaml"), []byte(`api:
  url: "https://default.example.com/api/v1"
`), 0o600); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}

	explicitPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(explicitPath, []byte(`api:
  url: "https://explicit.example.com/api/v1"
`), 0o600); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}

	cfg, err := LoadConfig(explicitPath)
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}

	if cfg.API.URL != "https://explicit.example.com/api/v1" {
		t.Errorf("API.URL = %q", cfg.API.URL)
	}
}

func TestLoadConfigExpandsCacheSchemaDir(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	setHomeDir(t, home)

	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`cache:
  schema_dir: "~/.cache/crmservice/schema"
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}

	expected := filepath.Join(home, ".cache", "crmservice", "schema")
	if cfg.Cache.SchemaDir != expected {
		t.Errorf("Cache.SchemaDir = %q, expected %q", cfg.Cache.SchemaDir, expected)
	}
}

func TestLoadConfigEnvOverrides(t *testing.T) {
	clearEnv(t)
	setConfigDir(t, t.TempDir())
	t.Setenv("CRMSERVICE_API_URL", "https://env.example.com")
	t.Setenv("CRMSERVICE_AUTH_TOKEN", "env-token")
	t.Setenv("CRMSERVICE_OUTPUT_FORMAT", "yaml")
	t.Setenv("CRMSERVICE_PAGE_SIZE", "99")
	t.Setenv("CRMSERVICE_TIMEOUT", "45")
	t.Setenv("CRMSERVICE_CACHE_DIR", "/tmp/env-cache")
	t.Setenv("CRMSERVICE_CACHE_TTL_DAYS", "3")
	t.Setenv("CRMSERVICE_CACHE_AUTO_REFRESH", "false")

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}

	if cfg.API.URL != "https://env.example.com" {
		t.Errorf("API.URL = %q", cfg.API.URL)
	}
	if cfg.Auth.Token != "env-token" {
		t.Errorf("Auth.Token = %q", cfg.Auth.Token)
	}
	if cfg.Output.Format != "yaml" || cfg.Output.PageSize != 99 {
		t.Errorf("Output = %+v", cfg.Output)
	}
	if cfg.API.Timeout != 45 {
		t.Errorf("API.Timeout = %d", cfg.API.Timeout)
	}
	if cfg.Cache.SchemaDir != "/tmp/env-cache" {
		t.Errorf("Cache.SchemaDir = %q", cfg.Cache.SchemaDir)
	}
	if cfg.Cache.TTLDays != 3 {
		t.Errorf("Cache.TTLDays = %d", cfg.Cache.TTLDays)
	}
	if cfg.Cache.AutoRefresh {
		t.Error("Cache.AutoRefresh = true, expected false")
	}
}

func TestLoadConfigRejectsInvalidPageSizeFromEnv(t *testing.T) {
	testCases := []struct {
		name  string
		value string
	}{
		{"negative", "-5"},
		{"zero", "0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clearEnv(t)
			setConfigDir(t, t.TempDir())
			t.Setenv("CRMSERVICE_PAGE_SIZE", tc.value)

			_, err := LoadConfig("")
			if err == nil {
				t.Fatalf("LoadConfig() error = nil, expected invalid CRMSERVICE_PAGE_SIZE")
			}
			if !strings.Contains(err.Error(), "CRMSERVICE_PAGE_SIZE") {
				t.Fatalf("error = %v, expected CRMSERVICE_PAGE_SIZE mention", err)
			}
		})
	}
}

func TestLoadConfigRejectsInvalidPageSizeFromYAML(t *testing.T) {
	clearEnv(t)

	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`output:
  page_size: -1
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("LoadConfig() error = nil, expected invalid output.page_size")
	}
	if !strings.Contains(err.Error(), "output.page_size") {
		t.Fatalf("error = %v, expected output.page_size mention", err)
	}
}

func TestLoadConfigRejectsInvalidEnvVars(t *testing.T) {
	testCases := []struct {
		name  string
		key   string
		value string
	}{
		{"output format", "CRMSERVICE_OUTPUT_FORMAT", "xml"},
		{"page size", "CRMSERVICE_PAGE_SIZE", "not-a-number"},
		{"timeout", "CRMSERVICE_TIMEOUT", "abc"},
		{"cache ttl days", "CRMSERVICE_CACHE_TTL_DAYS", "many"},
		{"cache auto refresh", "CRMSERVICE_CACHE_AUTO_REFRESH", "maybe"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clearEnv(t)
			setConfigDir(t, t.TempDir())
			t.Setenv(tc.key, tc.value)

			_, err := LoadConfig("")
			if err == nil {
				t.Fatalf("LoadConfig() error = nil, expected invalid %s", tc.key)
			}
		})
	}
}

func setConfigDir(t *testing.T, configDir string) {
	t.Helper()

	oldUserConfigDir := userConfigDir
	userConfigDir = func() (string, error) { return configDir, nil }
	t.Cleanup(func() { userConfigDir = oldUserConfigDir })
}

func setCacheDir(t *testing.T, cacheDir string) {
	t.Helper()

	oldUserCacheDir := userCacheDir
	userCacheDir = func() (string, error) { return cacheDir, nil }
	t.Cleanup(func() { userCacheDir = oldUserCacheDir })
}

func setHomeDir(t *testing.T, home string) {
	t.Helper()

	oldUserHomeDir := userHomeDir
	userHomeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeDir = oldUserHomeDir })
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"CRMSERVICE_API_URL",
		"CRMSERVICE_AUTH_TOKEN",
		"CRMSERVICE_OUTPUT_FORMAT",
		"CRMSERVICE_PAGE_SIZE",
		"CRMSERVICE_TIMEOUT",
		"CRMSERVICE_CACHE_DIR",
		"CRMSERVICE_CACHE_TTL_DAYS",
		"CRMSERVICE_CACHE_AUTO_REFRESH",
	} {
		t.Setenv(key, "")
	}
}
