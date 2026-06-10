package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"crmservice/internal/api"
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
	client := &http.Client{Timeout: getTimeoutFromConfig()}
	reqURL := strings.TrimSuffix(url, "/") + "/schema/" + module

	if verbose >= 1 {
		fmt.Fprintf(os.Stderr, "[REQUEST] GET %s\n", reqURL)
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Content-Type", "application/vnd.api+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, output.ErrorResponse(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if verbose >= 1 {
		fmt.Fprintf(os.Stderr, "[RESPONSE] Status: %d\n", resp.StatusCode)
	}
	if verbose >= 2 {
		fmt.Fprintf(os.Stderr, "[RESPONSE BODY]\n%s\n", string(body))
	}

	if resp.StatusCode >= 400 {
		return nil, output.ErrorResponse(&api.Error{
			Status:  resp.StatusCode,
			Body:    body,
			Message: string(body),
		})
	}

	return body, nil
}