package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"crmservice/internal/api"
	"crmservice/internal/config"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

func TestSplitCommaSeparated(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{name: "single value", input: "owner", expected: []string{"owner"}},
		{name: "comma separated", input: "owner,contacts", expected: []string{"owner", "contacts"}},
		{name: "trims spaces", input: "owner, contacts", expected: []string{"owner", "contacts"}},
		{name: "drops empty parts", input: "owner,,contacts", expected: []string{"owner", "contacts"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := splitCommaSeparated(tc.input)
			if len(got) != len(tc.expected) {
				t.Fatalf("splitCommaSeparated(%q) = %v, want %v", tc.input, got, tc.expected)
			}
			for i := range tc.expected {
				if got[i] != tc.expected[i] {
					t.Fatalf("splitCommaSeparated(%q) = %v, want %v", tc.input, got, tc.expected)
				}
			}
		})
	}
}

func TestGetURLFromFlagOrEnv(t *testing.T) {
	t.Setenv("CRMSERVICE_API_URL", "https://example.com")

	testCases := []struct {
		name     string
		flagURL  string
		envURL   string
		expected string
	}{
		{
			name:     "flag takes precedence",
			flagURL:  "https://flag.example.com",
			envURL:   "https://env.example.com",
			expected: "https://flag.example.com/api/v1",
		},
		{
			name:     "environment variable used",
			flagURL:  "",
			envURL:   "https://env.example.com",
			expected: "https://env.example.com/api/v1",
		},
		{
			name:     "http scheme preserved",
			flagURL:  "http://example.com",
			envURL:   "",
			expected: "http://example.com/api/v1",
		},
		{
			name:     "bare hostname uses https",
			flagURL:  "customer.crmservice.fi",
			envURL:   "",
			expected: "https://customer.crmservice.fi/api/v1",
		},
		{
			name:     "api/v1 suffix added",
			flagURL:  "https://example.com",
			envURL:   "",
			expected: "https://example.com/api/v1",
		},
		{
			name:     "trailing slash removed",
			flagURL:  "https://example.com/",
			envURL:   "",
			expected: "https://example.com/api/v1",
		},
		{
			name:     "api/v1 with trailing slash preserved without duplicate suffix",
			flagURL:  "https://example.com/api/v1/",
			envURL:   "",
			expected: "https://example.com/api/v1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CRMSERVICE_API_URL", tc.envURL)
			cmd := &cobra.Command{}
			cmd.Flags().String("url", "", "")
			if err := cmd.Flags().Set("url", tc.flagURL); err != nil {
				t.Errorf("Failed to set flag: %v", err)
			}

			result, err := getURLFromFlagOrEnv(cmd)
			if err != nil {
				t.Fatalf("getURLFromFlagOrEnv() returned error: %v", err)
			}
			if !strings.HasPrefix(result, tc.expected) {
				t.Errorf("getURLFromFlagOrEnv() = %q, expected %q", result, tc.expected)
			}
		})
	}
}

func TestGetURLFromFlagOrEnvMissingURL(t *testing.T) {
	oldCfg := cfg
	cfg = &config.Config{}
	t.Cleanup(func() { cfg = oldCfg })
	t.Setenv("CRMSERVICE_API_URL", "")

	cmd := &cobra.Command{}
	cmd.Flags().String("url", "", "")
	if _, err := getURLFromFlagOrEnv(cmd); err == nil {
		t.Fatal("getURLFromFlagOrEnv() error = nil, expected error")
	}
}

func TestConfigFallbacks(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = &config.Config{
		API: config.APIConfig{
			URL:     "https://config.example.com",
			Timeout: 42,
		},
		Auth: config.AuthConfig{
			Token: "config-token",
		},
		Output: config.OutputConfig{
			Format:   "json",
			PageSize: 75,
		},
	}
	t.Setenv("CRMSERVICE_API_URL", "")
	t.Setenv("CRMSERVICE_AUTH_TOKEN", "")

	cmd := &cobra.Command{}
	cmd.Flags().String("url", "", "")
	cmd.Flags().String("token", "", "")
	cmd.Flags().StringP("output", "o", "table", "")
	cmd.Flags().Int("page-size", 20, "")

	got, err := getURLFromFlagOrEnv(cmd)
	if err != nil {
		t.Fatalf("getURLFromFlagOrEnv() returned error: %v", err)
	}
	if got != "https://config.example.com/api/v1" {
		t.Errorf("getURLFromFlagOrEnv() = %q", got)
	}

	token, err := getTokenFromFlagEnvConfig(cmd)
	if err != nil {
		t.Fatalf("getTokenFromFlagEnvConfig() returned error: %v", err)
	}
	if token != "config-token" {
		t.Errorf("token = %q", token)
	}

	outputFormat, err := getOutputFormatFromFlagConfig(cmd)
	if err != nil {
		t.Fatalf("getOutputFormatFromFlagConfig() returned error: %v", err)
	}
	if outputFormat != "json" {
		t.Errorf("outputFormat = %q", outputFormat)
	}

	pageSize, err := getPageSizeFromFlagConfig(cmd)
	if err != nil {
		t.Fatalf("getPageSizeFromFlagConfig() returned error: %v", err)
	}
	if pageSize != 75 {
		t.Errorf("pageSize = %d", pageSize)
	}

	if timeout := getTimeoutFromConfig(); timeout != 42*time.Second {
		t.Errorf("timeout = %v", timeout)
	}
}

func TestGetOutputFormatFromFlagConfigRejectsInvalidFormat(t *testing.T) {
	cmd := listCmd()
	if err := cmd.Flags().Set("output", "xml"); err != nil {
		t.Fatalf("Set(output) error: %v", err)
	}

	_, err := getOutputFormatFromFlagConfig(cmd)
	if err == nil {
		t.Fatal("getOutputFormatFromFlagConfig() error = nil, expected invalid format error")
	}
	if !strings.Contains(err.Error(), "invalid output format") {
		t.Fatalf("error = %v, expected invalid output format", err)
	}
}

func TestRequiredTokenValidation(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = nil
	t.Setenv("CRMSERVICE_AUTH_TOKEN", "")

	cmd := &cobra.Command{}
	cmd.Flags().String("token", "", "")

	_, err := getRequiredTokenFromFlagEnvConfig(cmd)
	if err == nil {
		t.Fatal("expected error for missing API token")
	}
	if !strings.Contains(err.Error(), "API token not provided") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestErrorResponse(t *testing.T) {
	t.Run("API error", func(t *testing.T) {
		output.SetActiveFormat("table")
		apiErr := &api.Error{
			Status:  404,
			Body:    []byte(`{"error":"Not found"}`),
			Message: "Not found",
		}

		origStderr := os.Stderr
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		os.Stderr = w
		defer func() { os.Stderr = origStderr }()

		err = output.ErrorResponse(apiErr)
		if !errors.Is(err, apiErr) {
			t.Error("ErrorResponse should wrap the same error")
		}

		w.Close()
		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("Failed to read from pipe: %v", err)
		}

		if !strings.Contains(string(out), "Error: Not found") {
			t.Errorf("Expected error message, got: %s", string(out))
		}
	})

	t.Run("regular error", func(t *testing.T) {
		output.SetActiveFormat("table")
		origStderr := os.Stderr
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		os.Stderr = w
		defer func() { os.Stderr = origStderr }()

		regErr := fmt.Errorf("test error")

		err = output.ErrorResponse(regErr)
		if !errors.Is(err, regErr) {
			t.Error("ErrorResponse should wrap the same error")
		}

		w.Close()
		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("Failed to read from pipe: %v", err)
		}

		if !strings.Contains(string(out), "Error: test error") {
			t.Errorf("Expected error message, got: %s", string(out))
		}
	})
}