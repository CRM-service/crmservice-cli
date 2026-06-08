package cache

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestSchemaCacheSaveLoadPersistsTTL(t *testing.T) {
	cache := NewSchemaCache(t.TempDir())

	body := []byte(`{"attributes":{"name":{"type":"string"}}}`)
	if err := cache.Save("accounts", body); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	loaded, err := cache.Load("accounts")
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if string(loaded) != string(body) {
		t.Errorf("Load() = %s, expected %s", loaded, body)
	}
}

func TestSchemaCacheLoadRejectsExpiredCache(t *testing.T) {
	cache := NewSchemaCache(t.TempDir())
	cache.TTLSeconds = DefaultTTLSeconds

	cached := CachedSchema{
		Module:    "accounts",
		Data:      []byte(`{}`),
		FetchedAt: time.Now().Add(-25 * time.Hour),
	}
	data, err := json.Marshal(cached)
	if err != nil {
		t.Fatalf("Marshal() returned error: %v", err)
	}
	if err := os.MkdirAll(cache.CacheDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() returned error: %v", err)
	}
	if err := os.WriteFile(cache.GetPath("accounts"), data, 0o600); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}

	if _, err := cache.Load("accounts"); err == nil {
		t.Fatal("Load() error = nil, expected expired cache error")
	}
}

func TestSchemaCacheLoadAcceptsExpiredCacheWhenTTLDisabled(t *testing.T) {
	cache := NewSchemaCache(t.TempDir())
	cache.TTLSeconds = 0

	cached := CachedSchema{
		Module:    "accounts",
		Data:      []byte(`{}`),
		FetchedAt: time.Now().Add(-365 * 24 * time.Hour),
	}
	data, err := json.Marshal(cached)
	if err != nil {
		t.Fatalf("Marshal() returned error: %v", err)
	}
	if err := os.MkdirAll(cache.CacheDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() returned error: %v", err)
	}
	if err := os.WriteFile(cache.GetPath("accounts"), data, 0o600); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}

	if _, err := cache.Load("accounts"); err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
}
