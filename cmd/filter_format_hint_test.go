package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const filterFormatHintSubstring = "Use JSON syntax: filter="

func setupFilterFormatHintTest(t *testing.T, server *httptest.Server) {
	t.Helper()
	setupFilterCommandTestConfig(t)
	t.Setenv("CRMSERVICE_API_URL", server.URL+"/api/v1")
}

func assertFilterFormatHint(t *testing.T, got string) {
	t.Helper()
	if !strings.Contains(got, filterFormatHintSubstring) {
		t.Fatalf("output = %q, expected filter format hint", got)
	}
}

func assertNoFilterFormatHint(t *testing.T, got string) {
	t.Helper()
	if strings.Contains(got, filterFormatHintSubstring) {
		t.Fatalf("output = %q, did not expect filter format hint", got)
	}
}

func TestFilterFormatHintOnAllCommandPaths(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestResponse(t, w, `{"attributes":{"account_type":{"type":"string"}}}`)
	}))
	defer server.Close()
	setupFilterFormatHintTest(t, server)

	t.Run("count positional", func(t *testing.T) {
		cmd := countCmd()
		cmd.SetContext(context.Background())
		err := runCountCommand(cmd, []string{"accounts", "not-json"})
		if err == nil {
			t.Fatal("runCountCommand() error = nil, expected syntax error")
		}
		assertFilterFormatHint(t, err.Error())
	})

	t.Run("count --filter", func(t *testing.T) {
		cmd := countCmd()
		cmd.SetContext(context.Background())
		if err := cmd.Flags().Set("filter", "not-json"); err != nil {
			t.Fatalf("Set(filter) error: %v", err)
		}
		err := runCountCommand(cmd, []string{"accounts"})
		if err == nil {
			t.Fatal("runCountCommand() error = nil, expected syntax error")
		}
		assertFilterFormatHint(t, err.Error())
	})

	t.Run("list --filter", func(t *testing.T) {
		cmd := listCmd()
		cmd.SetContext(context.Background())
		if err := cmd.Flags().Set("filter", "not-json"); err != nil {
			t.Fatalf("Set(filter) error: %v", err)
		}
		err := runListCommand(cmd, []string{"accounts"}, "")
		if err == nil {
			t.Fatal("runListCommand() error = nil, expected syntax error")
		}
		assertFilterFormatHint(t, err.Error())
	})

	t.Run("search positional", func(t *testing.T) {
		cmd := searchCmd()
		cmd.SetContext(context.Background())
		err := runListCommand(cmd, []string{"accounts"}, "not-json")
		if err == nil {
			t.Fatal("runListCommand() error = nil, expected syntax error")
		}
		assertFilterFormatHint(t, err.Error())
	})

	t.Run("filter validate syntax only", func(t *testing.T) {
		cmd := filterCmd()
		cmd.SetArgs([]string{"validate", "-o", "json", "not-json"})
		stdout := captureStdout(t, func() {
			if err := cmd.Execute(); err == nil {
				t.Fatal("Execute() error = nil, expected syntax error")
			}
		})
		assertFilterFormatHint(t, stdout)
	})

	t.Run("filter validate with module", func(t *testing.T) {
		cmd := filterCmd()
		cmd.SetArgs([]string{"validate", "-o", "json", "accounts", "not-json"})
		stdout := captureStdout(t, func() {
			if err := cmd.Execute(); err == nil {
				t.Fatal("Execute() error = nil, expected syntax error")
			}
		})
		assertFilterFormatHint(t, stdout)
	})

	t.Run("multiple JSON values", func(t *testing.T) {
		cmd := countCmd()
		cmd.SetContext(context.Background())
		err := runCountCommand(cmd, []string{"accounts", `{"a":1}{"b":2}`})
		if err == nil {
			t.Fatal("runCountCommand() error = nil, expected syntax error")
		}
		assertFilterFormatHint(t, err.Error())
	})

	t.Run("top-level array reaches ParseToMap", func(t *testing.T) {
		cmd := countCmd()
		cmd.SetContext(context.Background())
		err := runCountCommand(cmd, []string{"accounts", `[{"$eq":["account_type","Customer"]}]`})
		if err == nil {
			t.Fatal("runCountCommand() error = nil, expected non-object error")
		}
		assertFilterFormatHint(t, err.Error())
	})
}

func TestFilterFormatHintNotUsedForSemanticErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestResponse(t, w, `{"attributes":{"account_type":{"type":"string"}}}`)
	}))
	defer server.Close()
	setupFilterFormatHintTest(t, server)

	t.Run("unsupported operator", func(t *testing.T) {
		cmd := countCmd()
		cmd.SetContext(context.Background())
		err := runCountCommand(cmd, []string{"accounts", `{"$unknown":["name","x"]}`})
		if err == nil {
			t.Fatal("runCountCommand() error = nil, expected operator error")
		}
		assertNoFilterFormatHint(t, err.Error())
		if !strings.Contains(err.Error(), `$unknown`) {
			t.Fatalf("error = %q, expected unsupported operator message", err.Error())
		}
	})

	t.Run("unknown field", func(t *testing.T) {
		cmd := countCmd()
		cmd.SetContext(context.Background())
		err := runCountCommand(cmd, []string{"accounts", `{"$eq":["missing_field","x"]}`})
		if err == nil {
			t.Fatal("runCountCommand() error = nil, expected unknown field error")
		}
		assertNoFilterFormatHint(t, err.Error())
		if !strings.Contains(err.Error(), "missing_field") {
			t.Fatalf("error = %q, expected unknown field message", err.Error())
		}
	})
}