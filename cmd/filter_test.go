package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestValidateFilterJSON(t *testing.T) {
	tests := []struct {
		name    string
		filter  string
		wantErr bool
	}{
		{name: "eq", filter: `{"$eq":["account_type","Customer"]}`},
		{name: "and", filter: `{"$and":[{"$eq":["account_type","Customer"]},{"$cts":["name","Acme"]}]}`},
		{name: "in", filter: `{"$in":["id",["1","2"]]}`},
		{name: "between", filter: `{"$between":["created_at","2026-01-01","2026-01-31"]}`},
		{name: "is null", filter: `{"$is.null":["email"]}`},
		{name: "invalid json", filter: `{`, wantErr: true},
		{name: "unknown operator", filter: `{"$unknown":["name","Acme"]}`, wantErr: true},
		{name: "wrong arity", filter: `{"$eq":["name"]}`, wantErr: true},
		{name: "in without array", filter: `{"$in":["id","1"]}`, wantErr: true},
		{name: "empty logical", filter: `{"$and":[]}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilterJSON(tt.filter)
			if tt.wantErr && err == nil {
				t.Fatal("validateFilterJSON() error = nil, expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validateFilterJSON() returned error: %v", err)
			}
		})
	}
}

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
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if !strings.Contains(out.String(), "filter OK") {
		t.Errorf("output = %q, expected filter OK", out.String())
	}
}
