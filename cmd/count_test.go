package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCountCmdStructure(t *testing.T) {
	cmd := countCmd()
	if cmd.Use != "count <module> [filter]" {
		t.Errorf("Use = %q, expected count <module> [filter]", cmd.Use)
	}
}

func TestOutputCountResultJSON(t *testing.T) {
	filter := map[string]interface{}{
		"$eq": []interface{}{"account_type", "Customer"},
	}
	out := captureStdout(t, func() {
		if err := outputCountResult("accounts", 3, filter, "json"); err != nil {
			t.Fatalf("outputCountResult() returned error: %v", err)
		}
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("json.Unmarshal() returned error: %v\noutput: %s", err, out)
	}
	if result["module"] != "accounts" {
		t.Errorf("module = %v, expected accounts", result["module"])
	}
	if result["total"] != float64(3) {
		t.Errorf("total = %v, expected 3", result["total"])
	}
	if _, ok := result["filter"]; !ok {
		t.Error("expected filter in JSON output")
	}
}

func TestRunCountCommandRejectsBothFilterSources(t *testing.T) {
	cmd := countCmd()
	args := []string{"accounts", `{"$eq":["name","Acme"]}`}
	if err := cmd.Flags().Set("filter", `{"$eq":["name","Acme"]}`); err != nil {
		t.Fatalf("Set(filter) returned error: %v", err)
	}

	err := runCountCommand(cmd, args)
	if err == nil {
		t.Fatal("runCountCommand() error = nil, expected filter conflict error")
	}
	if !strings.Contains(err.Error(), "positional filter and --filter") {
		t.Fatalf("runCountCommand() error = %v, expected filter conflict", err)
	}
}
