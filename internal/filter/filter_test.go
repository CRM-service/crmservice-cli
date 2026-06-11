package filter

import (
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
		{name: "bare equality shorthand", filter: `{"account_type":"Customer"}`},
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
			err := ValidateFilterJSON(tt.filter)
			if tt.wantErr && err == nil {
				t.Fatal("ValidateFilterJSON() error = nil, expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateFilterJSON() returned error: %v", err)
			}
		})
	}
}

func TestCollectFilterFieldsIncludesBareFieldKeys(t *testing.T) {
	fields := CollectFilterFields(map[string]interface{}{"account_type": "Customer"})
	if len(fields) != 1 {
		t.Fatalf("len(fields) = %d, expected 1", len(fields))
	}
	if fields[0] != "account_type" {
		t.Errorf("fields[0] = %q, expected account_type", fields[0])
	}
}

func TestCollectFilterFields(t *testing.T) {
	filterJSON := `{"$and":[{"$eq":["account_type","Customer"]},{"$cts":["name","Acme"]}]}`
	if err := ValidateFilterJSON(filterJSON); err != nil {
		t.Fatalf("ValidateFilterJSON() returned error: %v", err)
	}

	parsed, err := ParseFilterJSON(filterJSON)
	if err != nil {
		t.Fatalf("ParseFilterJSON() returned error: %v", err)
	}

	fields := CollectFilterFields(parsed)
	if len(fields) != 2 {
		t.Fatalf("len(fields) = %d, expected 2", len(fields))
	}
}

func TestParseToMap(t *testing.T) {
	filterObj, err := ParseToMap(`{"$eq":["account_type","Customer"]}`)
	if err != nil {
		t.Fatalf("ParseToMap() returned error: %v", err)
	}
	if _, ok := filterObj["$eq"]; !ok {
		t.Fatal("expected $eq key in filter object")
	}
}

func TestValidateFilterJSONIncludesFormatHint(t *testing.T) {
	err := ValidateFilterJSON("not-json")
	if err == nil {
		t.Fatal("ValidateFilterJSON() error = nil, expected syntax error")
	}
	if !strings.Contains(err.Error(), filterFormatHint) {
		t.Fatalf("ValidateFilterJSON() error = %q, expected format hint", err.Error())
	}
}

func TestParseToMapIncludesFormatHint(t *testing.T) {
	_, err := ParseToMap("not-json")
	if err == nil {
		t.Fatal("ParseToMap() error = nil, expected syntax error")
	}
	if !strings.Contains(err.Error(), filterFormatHint) {
		t.Fatalf("ParseToMap() error = %q, expected format hint", err.Error())
	}
}
