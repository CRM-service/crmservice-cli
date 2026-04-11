package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SchemaCache struct {
	CacheDir    string
	TTLDays     int
	AutoRefresh bool
}

type CachedSchema struct {
	Module    string    `json:"module"`
	Data      []byte    `json:"data"`
	FetchedAt time.Time `json:"fetched_at"`
	ETag      string    `json:"etag,omitempty"`
}

func NewSchemaCache(cacheDir string) *SchemaCache {
	return &SchemaCache{
		CacheDir:    cacheDir,
		TTLDays:     24,
		AutoRefresh: true,
	}
}

func (c *SchemaCache) GetPath(module string) string {
	safeName := strings.ReplaceAll(module, "/", "_")
	return filepath.Join(c.CacheDir, safeName+".json")
}

func (c *SchemaCache) Exists(module string) bool {
	_, err := os.Stat(c.GetPath(module))
	return err == nil
}

func (c *SchemaCache) Load(module string) ([]byte, error) {
	path := c.GetPath(module)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cached CachedSchema
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}

	if !c.IsFresh(cached.FetchedAt) {
		return nil, fmt.Errorf("schema cache expired")
	}

	return cached.Data, nil
}

func (c *SchemaCache) Save(module string, data []byte) error {
	if err := os.MkdirAll(c.CacheDir, 0755); err != nil {
		return err
	}

	cached := CachedSchema{
		Module:    module,
		Data:      data,
		FetchedAt: time.Now(),
	}

	cachedData, err := json.Marshal(cached)
	if err != nil {
		return err
	}

	return os.WriteFile(c.GetPath(module), cachedData, 0600)
}

func (c *SchemaCache) Delete(module string) error {
	return os.Remove(c.GetPath(module))
}

func (c *SchemaCache) IsFresh(fetched time.Time) bool {
	if c.TTLDays <= 0 {
		return true
	}
	return time.Since(fetched).Hours() < float64(c.TTLDays*24)
}
