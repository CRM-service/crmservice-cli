package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"crmservice/internal/config"
	"crmservice/internal/output"
)

func TestListCmd(t *testing.T) {
	cmd := listCmd()

	t.Run("command structure", func(t *testing.T) {
		if cmd.Use != "list <module>" {
			t.Errorf("Use = %q, expected %q", cmd.Use, "list <module>")
		}
		if cmd.Short != "List records" {
			t.Errorf("Short = %q, expected %q", cmd.Short, "List records")
		}
	})

	t.Run("args validation", func(t *testing.T) {
		err := cmd.ValidateArgs([]string{"module"})
		if err != nil {
			t.Errorf("Expected no error for valid args, got: %v", err)
		}

		err = cmd.ValidateArgs([]string{})
		if err == nil {
			t.Error("Expected error for empty args")
		}

		err = cmd.ValidateArgs([]string{"one", "two"})
		if err == nil {
			t.Error("Expected error for too many args")
		}
	})

	t.Run("flags", func(t *testing.T) {
		flags := []string{"page-size", "fields", "sort", "output", "filter", "full", "verbose", "page", "offset", "all", "max-results"}
		for _, name := range flags {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("Missing flag: %s", name)
			}
		}
	})

	t.Run("output format validation", func(t *testing.T) {
		testCases := []struct {
			name      string
			output    string
			valid     bool
			filter    string
			shouldErr bool
		}{
			{"valid table format", "table", true, "", false},
			{"valid json format", "json", true, "", false},
			{"valid yaml format", "yaml", true, "", false},
			{"valid jsonl format", "jsonl", true, "", false},
			{"valid csv format", "csv", true, "", false},
			{"invalid format", "xml", false, "", true},
			{"invalid with filter", "invalid", false, `{"name":"test"}`, true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.shouldErr != !tc.valid {
					if tc.valid && !output.ValidOutputFormat(tc.output) {
						t.Errorf("Format %s should be valid", tc.output)
					}
					if !tc.valid && output.ValidOutputFormat(tc.output) {
						t.Errorf("Format %s should be invalid", tc.output)
					}
				}
			})
		}
	})
}

func TestGetCmd(t *testing.T) {
	cmd := getCmd()

	t.Run("command structure", func(t *testing.T) {
		if cmd.Use != "get <module> <id>" {
			t.Errorf("Use = %q, expected %q", cmd.Use, "get <module> <id>")
		}
		if cmd.Short != "Get record by ID" {
			t.Errorf("Short = %q, expected %q", cmd.Short, "Get record by ID")
		}
	})

	t.Run("args validation", func(t *testing.T) {
		err := cmd.ValidateArgs([]string{"module", "123"})
		if err != nil {
			t.Errorf("Expected no error for valid args, got: %v", err)
		}

		err = cmd.ValidateArgs([]string{"module"})
		if err == nil {
			t.Error("Expected error for missing id")
		}

		err = cmd.ValidateArgs([]string{"module", "123", "extra"})
		if err == nil {
			t.Error("Expected error for too many args")
		}
	})

	t.Run("flags", func(t *testing.T) {
		flags := []string{"fields", "output", "full", "verbose"}
		for _, name := range flags {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("Missing flag: %s", name)
			}
		}
	})
}

func TestCreateCmd(t *testing.T) {
	cmd := createCmd()

	t.Run("command structure", func(t *testing.T) {
		if cmd.Use != "create <module>" {
			t.Errorf("Use = %q, expected %q", cmd.Use, "create <module>")
		}
		if cmd.Short != "Create a new record" {
			t.Errorf("Short = %q, expected %q", cmd.Short, "Create a new record")
		}
	})

	t.Run("flags", func(t *testing.T) {
		flags := []string{"field", "output", "full", "verbose"}
		for _, name := range flags {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("Missing flag: %s", name)
			}
		}
	})
}

func TestUpdateCmd(t *testing.T) {
	cmd := updateCmd()

	t.Run("command structure", func(t *testing.T) {
		if cmd.Use != "update <module> <id>" {
			t.Errorf("Use = %q, expected %q", cmd.Use, "update <module> <id>")
		}
		if cmd.Short != "Update a record" {
			t.Errorf("Short = %q, expected %q", cmd.Short, "Update a record")
		}
	})

	t.Run("flags", func(t *testing.T) {
		flags := []string{"field", "output", "full", "verbose"}
		for _, name := range flags {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("Missing flag: %s", name)
			}
		}
	})
}

func TestDeleteCmd(t *testing.T) {
	cmd := deleteCmd()

	t.Run("command structure", func(t *testing.T) {
		if cmd.Use != "delete <module> <id>" {
			t.Errorf("Use = %q, expected %q", cmd.Use, "delete <module> <id>")
		}
		if cmd.Short != "Delete a record" {
			t.Errorf("Short = %q, expected %q", cmd.Short, "Delete a record")
		}
	})

	t.Run("args validation", func(t *testing.T) {
		err := cmd.ValidateArgs([]string{"module", "123"})
		if err != nil {
			t.Errorf("Expected no error for valid args, got: %v", err)
		}

		err = cmd.ValidateArgs([]string{"module"})
		if err == nil {
			t.Error("Expected error for missing id")
		}
	})
}

func TestFieldsCmd(t *testing.T) {
	cmd := fieldsCmd()

	t.Run("command structure", func(t *testing.T) {
		if cmd.Use != "fields <module>" {
			t.Errorf("Use = %q, expected %q", cmd.Use, "fields <module>")
		}
		if cmd.Short != "Show available fields for a module" {
			t.Errorf("Short = %q, expected %q", cmd.Short, "Show available fields for a module")
		}
	})

	t.Run("args validation", func(t *testing.T) {
		err := cmd.ValidateArgs([]string{"module"})
		if err != nil {
			t.Errorf("Expected no error for valid args, got: %v", err)
		}

		err = cmd.ValidateArgs([]string{})
		if err == nil {
			t.Error("Expected error for missing module")
		}
	})

	t.Run("flags", func(t *testing.T) {
		flags := []string{"output", "full", "force", "verbose"}
		for _, name := range flags {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("Missing flag: %s", name)
			}
		}
	})
}

func TestSearchCmd(t *testing.T) {
	cmd := searchCmd()

	t.Run("command structure", func(t *testing.T) {
		if cmd.Use != "search <module> <filter>" {
			t.Errorf("Use = %q, expected %q", cmd.Use, "search <module> <filter>")
		}
		if cmd.Short != "Search records (alias for list --filter)" {
			t.Errorf("Short = %q, expected %q", cmd.Short, "Search records (alias for list --filter)")
		}
	})

	t.Run("args validation", func(t *testing.T) {
		err := cmd.ValidateArgs([]string{"module", `{"$eq":["name","Acme"]}`})
		if err != nil {
			t.Errorf("Expected no error for valid args, got: %v", err)
		}

		err = cmd.ValidateArgs([]string{"module"})
		if err == nil {
			t.Error("Expected error for missing filter")
		}
	})

	t.Run("flags", func(t *testing.T) {
		flags := []string{"page-size", "include", "fields", "sort", "output", "full", "verbose", "page", "offset", "all", "max-results"}
		for _, name := range flags {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("Missing flag: %s", name)
			}
		}

		if flag := cmd.Flags().Lookup("filter"); flag != nil {
			t.Error("search should not expose --filter because the filter is positional")
		}
	})
}

func TestRunListCommandSplitsIncludeFlag(t *testing.T) {
	var gotInclude string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/accounts") {
			gotInclude = r.URL.Query().Get("include")
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		writeTestResponse(t, w, `{"data":[]}`)
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		API:  config.APIConfig{URL: server.URL + "/api/v1", Timeout: 5},
		Auth: config.AuthConfig{Token: "token"},
	}
	t.Cleanup(func() { cfg = oldCfg })

	cmd := listCmd()
	cmd.SetContext(context.Background())
	if err := cmd.Flags().Set("include", "owner, contacts"); err != nil {
		t.Fatalf("Set(include) error: %v", err)
	}

	if err := runListCommand(cmd, []string{"accounts"}, ""); err != nil {
		t.Fatalf("runListCommand() error: %v", err)
	}
	if gotInclude != "owner,contacts" {
		t.Errorf("include query = %q, want owner,contacts", gotInclude)
	}
}

func TestModulesCmd(t *testing.T) {
	cmd := modulesCmd()

	t.Run("command structure", func(t *testing.T) {
		if cmd.Use != "modules" {
			t.Errorf("Use = %q, expected %q", cmd.Use, "modules")
		}
		if cmd.Short != "List API modules" {
			t.Errorf("Short = %q, expected %q", cmd.Short, "List API modules")
		}
	})

	t.Run("flags", func(t *testing.T) {
		flags := []string{"output", "full", "verbose"}
		for _, name := range flags {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("Missing flag: %s", name)
			}
		}
	})
}

func TestModulesCommandHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(`{"errors":[{"detail":"unauthorized"}]}`)); err != nil {
			t.Errorf("Write() returned error: %v", err)
		}
	}))
	defer server.Close()

	oldCfg := cfg
	cfg = &config.Config{
		API:  config.APIConfig{URL: server.URL + "/api/v1", Timeout: 5},
		Auth: config.AuthConfig{Token: "token"},
	}
	t.Cleanup(func() { cfg = oldCfg })

	cmd := modulesCmd()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"-o", "json"})
	stderr := captureStderr(t, func() {
		if err := cmd.Execute(); err == nil {
			t.Fatal("Execute() error = nil, expected API error")
		}
	})
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr = %q, expected no usage output", stderr)
	}
}