package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

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
	if !cfg.Cache.AutoRefresh {
		t.Error("Cache.AutoRefresh = false, expected true")
	}
	if cfg.Auth.Type != "bearer" {
		t.Errorf("Auth.Type = %q, expected bearer", cfg.Auth.Type)
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
  type: "bearer"
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
	t.Setenv("XDG_CONFIG_HOME", configHome)
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
	t.Setenv("XDG_CONFIG_HOME", configHome)
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

func TestLoadConfigEnvOverrides(t *testing.T) {
	clearEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CRMSERVICE_API_URL", "https://env.example.com")
	t.Setenv("CRMSERVICE_AUTH_TOKEN", "env-token")
	t.Setenv("CRMSERVICE_OUTPUT_FORMAT", "yaml")
	t.Setenv("CRMSERVICE_PAGE_SIZE", "99")
	t.Setenv("CRMSERVICE_TIMEOUT", "45")
	t.Setenv("CRMSERVICE_CACHE_DIR", "/tmp/env-cache")

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
	} {
		t.Setenv(key, "")
	}
}
