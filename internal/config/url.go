package config

import (
	"errors"
	"strings"
)

// ErrAPIURLMissing is returned when a required API URL is not configured.
var ErrAPIURLMissing = errors.New("API URL not provided. Set CRMSERVICE_API_URL environment variable, config api.url, or use --url flag")

// ResolveAPIURL normalizes a CRM hostname or API base URL to a canonical JSON:API v1 base.
// When requireValue is true, an empty input returns an error.
func ResolveAPIURL(raw string, requireValue bool) (string, error) {
	url := strings.TrimSpace(raw)
	if url == "" {
		if requireValue {
			return "", ErrAPIURLMissing
		}
		return "", nil
	}

	url = strings.TrimSuffix(url, "/")
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	if !strings.HasSuffix(url, "/api/v1") {
		url += "/api/v1"
	}
	return url, nil
}

// DisplayAPIURL returns the API base URL without a trailing slash for display output.
func DisplayAPIURL(raw string) string {
	url, err := ResolveAPIURL(raw, false)
	if err != nil || url == "" {
		return strings.TrimSuffix(strings.TrimSpace(raw), "/")
	}
	return strings.TrimSuffix(url, "/")
}