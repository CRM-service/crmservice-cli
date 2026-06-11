package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFilterCmd(t *testing.T) {
	cmd := filterCmd()
	if cmd.Use != "filter" {
		t.Errorf("Use = %q, expected filter", cmd.Use)
	}
	if cmd.Short != "Show and validate filter language expressions" {
		t.Errorf("Short = %q", cmd.Short)
	}

	if cmd.Commands()[0].Use != "reference" {
		t.Errorf("first subcommand = %q, expected reference", cmd.Commands()[0].Use)
	}
}

func TestFilterValidateCmd(t *testing.T) {
	cmd := filterCmd()
	cmd.SetArgs([]string{"validate", `{"$eq":["name","Acme"]}`})

	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() returned error: %v", err)
		}
	})
	if !strings.Contains(out, "filter OK") {
		t.Errorf("output = %q, expected filter OK", out)
	}
}

func TestFilterValidateCmdRequiresOneOrTwoArgs(t *testing.T) {
	cmd := filterCmd()
	cmd.SetArgs([]string{"validate"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute() error = nil, expected error")
	}
}

func TestFilterValidateCmdJSONOutput(t *testing.T) {
	cmd := filterCmd()
	cmd.SetArgs([]string{"validate", "-o", "json", `{"$eq":["name","Acme"]}`})

	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() returned error: %v", err)
		}
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("json.Unmarshal() returned error: %v\noutput: %s", err, out)
	}
	if result["valid"] != true {
		t.Errorf("valid = %v, expected true", result["valid"])
	}
	if result["schema_checked"] != false {
		t.Errorf("schema_checked = %v, expected false", result["schema_checked"])
	}
	if _, ok := result["filter"]; !ok {
		t.Error("expected filter in JSON output")
	}
}

func TestFilterValidateCmdJSONFailureOutput(t *testing.T) {
	cmd := filterCmd()
	cmd.SetArgs([]string{"validate", "-o", "json", `{`})

	var stdout string
	stderr := captureStderr(t, func() {
		stdout = captureStdout(t, func() {
			if err := cmd.Execute(); err == nil {
				t.Fatal("Execute() error = nil, expected validation failure")
			}
		})
	})
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, expected empty", stderr)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("json.Unmarshal(stdout) returned error: %v\nstdout: %s", err, stdout)
	}
	if result["valid"] != false {
		t.Errorf("valid = %v, expected false", result["valid"])
	}
	if result["message"] == nil {
		t.Error("expected message in stdout JSON output")
	}
}

func TestFilterValidateCmdHasOutputFlag(t *testing.T) {
	cmd := filterCmd()
	validate := cmd.Commands()[1]
	if validate.Flags().Lookup("output") == nil {
		t.Fatal("validate subcommand missing output flag")
	}
}
