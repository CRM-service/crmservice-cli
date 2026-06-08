package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"crmservice/internal/config"
)

func TestDoctorCmdStructure(t *testing.T) {
	cmd := doctorCmd()
	if cmd.Use != "doctor" {
		t.Errorf("Use = %q, expected doctor", cmd.Use)
	}
}

func TestRunDoctorChecksAllOK(t *testing.T) {
	oldCheck := doctorCheckAPIReachable
	oldFetch := doctorFetchCurrentUser
	doctorCheckAPIReachable = func(context.Context, string) bool { return true }
	doctorFetchCurrentUser = func(context.Context, string, string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"crm_url": "https://example.com/api/v1",
			"id":      "1",
			"name":    "Agent",
			"email":   "agent@example.com",
		}, nil
	}
	t.Cleanup(func() {
		doctorCheckAPIReachable = oldCheck
		doctorFetchCurrentUser = oldFetch
	})

	oldCfg := cfg
	cacheDir := t.TempDir()
	cfg = &config.Config{
		API: config.APIConfig{
			URL:     "https://example.com",
			Timeout: 30,
		},
		Auth: config.AuthConfig{Token: "token"},
		Cache: config.CacheConfig{
			SchemaDir:   cacheDir,
			TTLDays:     1,
			AutoRefresh: true,
		},
	}
	t.Setenv("CRMSERVICE_API_URL", "https://example.com")
	t.Setenv("CRMSERVICE_AUTH_TOKEN", "token")
	t.Cleanup(func() { cfg = oldCfg })

	cmd := doctorCmd()
	result := runDoctorChecks(cmd)
	if result["ok"] != true {
		t.Fatalf("ok = %v, issues = %v", result["ok"], result["issues"])
	}
	if result["authenticated"] != true {
		t.Errorf("authenticated = %v", result["authenticated"])
	}
}

func TestDoctorCmdFailsWhenChecksFail(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	t.Setenv("CRMSERVICE_API_URL", "")
	t.Setenv("CRMSERVICE_AUTH_TOKEN", "")
	t.Cleanup(func() { cfg = oldCfg })

	cmd := doctorCmd()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"-o", "json"})

	out := captureStdout(t, func() {
		stderr := captureStderr(t, func() {
			if err := cmd.Execute(); err == nil {
				t.Fatal("Execute() error = nil, expected error")
			}
		})
		if strings.Contains(stderr, "Usage:") {
			t.Fatalf("stderr = %q, expected no usage output", stderr)
		}
		if strings.Count(stderr, "doctor:") > 1 {
			t.Fatalf("stderr = %q, expected a single doctor failure message", stderr)
		}
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("json.Unmarshal() returned error: %v\noutput: %s", err, out)
	}
	if result["ok"] != false {
		t.Errorf("ok = %v, expected false", result["ok"])
	}
}
