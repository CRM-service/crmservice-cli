package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"crmservice/internal/cache"
	"crmservice/internal/output"
)

func getSchemaBody(module, url, token string, verbose int, force bool) ([]byte, error) {
	schemaCache := cache.NewSchemaCache(getSchemaCacheDir(url))
	if cfg != nil {
		schemaCache.TTLSeconds = cfg.Cache.TTLSeconds
		schemaCache.AutoRefresh = cfg.Cache.AutoRefresh
	}

	if !force {
		cachedBody, err := schemaCache.Load(module)
		if err == nil {
			if verbose >= 1 {
				fmt.Fprintf(os.Stderr, "[CACHE] HIT %s\n", schemaCache.GetPath(module))
			}
			return cachedBody, nil
		}
		if !schemaCache.AutoRefresh {
			return nil, fmt.Errorf("schema cache unavailable for %s and auto refresh is disabled: %w", module, err)
		}
		if verbose >= 1 {
			fmt.Fprintf(os.Stderr, "[CACHE] MISS %s: %v\n", schemaCache.GetPath(module), err)
		}
	} else if verbose >= 1 {
		fmt.Fprintf(os.Stderr, "[CACHE] FORCE REFRESH %s\n", schemaCache.GetPath(module))
	}

	body, err := fetchSchemaBody(module, url, token, verbose)
	if err != nil {
		return nil, err
	}
	if err := schemaCache.Save(module, body); err != nil {
		if verbose >= 1 {
			fmt.Fprintf(os.Stderr, "[CACHE] SAVE FAILED %s: %v\n", schemaCache.GetPath(module), err)
		}
		return body, nil
	}
	return body, nil
}

func getSchemaCacheDir(url string) string {
	cacheDir := ""
	if cfg != nil {
		cacheDir = cfg.Cache.SchemaDir
	}
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "crmservice", "schema")
	}
	return filepath.Join(cacheDir, cacheKey(url))
}

func cacheKey(value string) string {
	value = strings.TrimSuffix(value, "/")
	if value == "" {
		return "default"
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func fetchSchemaBody(module, url, token string, verbose int) ([]byte, error) {
	body, err := newAPIClient(url, token, verbose).GetSchema(context.Background(), module)
	if err != nil {
		return nil, output.ErrorResponse(err)
	}
	return body, nil
}