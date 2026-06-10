package schema

import (
	"testing"
)

func TestFieldListFromSchemaAttributesOrder(t *testing.T) {
	attrs := map[string]interface{}{
		"zebra_field": map[string]interface{}{"type": "string"},
		"alpha_field": map[string]interface{}{"type": "integer"},
		"middle_field": map[string]interface{}{
			"type":           "relation",
			"relationModule": "accounts",
		},
	}

	fieldList := FieldListFromSchemaAttributes(attrs, []string{"alpha_field"})
	names := fieldNamesFromList(fieldList)
	want := []string{"alpha_field", "middle_field", "zebra_field"}
	if len(names) != len(want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	for i, name := range names {
		if name != want[i] {
			t.Fatalf("names[%d] = %q, want %q", i, name, want[i])
		}
	}

	primary, ok := fieldList[0]["primary"].(bool)
	if !ok || !primary {
		t.Fatalf("fieldList[0][primary] = %v, want true", fieldList[0]["primary"])
	}

	relMod, ok := fieldList[1]["relationModule"].(string)
	if !ok || relMod != "accounts" {
		t.Fatalf("fieldList[1][relationModule] = %v, want accounts", fieldList[1]["relationModule"])
	}
	if _, hasExtras := fieldList[1]["extras"]; hasExtras {
		t.Fatal("fieldList[1] should not have extras key")
	}
}

func fieldNamesFromList(fieldList []map[string]interface{}) []string {
	names := make([]string, 0, len(fieldList))
	for _, field := range fieldList {
		name, ok := field["name"].(string)
		if !ok {
			continue
		}
		names = append(names, name)
	}
	return names
}
