package cmd

import (
	"encoding/json"
	"testing"
)

func TestModulesDataFromLinksOrder(t *testing.T) {
	links := map[string]interface{}{
		"self":     map[string]interface{}{"href": "/"},
		"zebra":    map[string]interface{}{"href": "/zebra", "type": "modules"},
		"accounts": map[string]interface{}{"href": "/accounts", "type": "modules"},
		"contacts": map[string]interface{}{"href": "/contacts", "type": "modules"},
		"meta":     map[string]interface{}{"href": "/meta"},
	}

	t.Run("default", func(t *testing.T) {
		data := modulesDataFromLinks(links, false)
		names := moduleNamesFromData(data)
		want := []string{"accounts", "contacts", "meta", "zebra"}
		if len(names) != len(want) {
			t.Fatalf("names = %v, want %v", names, want)
		}
		for i, name := range names {
			if name != want[i] {
				t.Fatalf("names[%d] = %q, want %q (full=%v)", i, name, want[i], names)
			}
		}
	})

	t.Run("full", func(t *testing.T) {
		data := modulesDataFromLinks(links, true)
		names := moduleNamesFromData(data)
		want := []string{"accounts", "contacts", "zebra"}
		if len(names) != len(want) {
			t.Fatalf("names = %v, want %v", names, want)
		}
		for i, name := range names {
			if name != want[i] {
				t.Fatalf("names[%d] = %q, want %q (full=%v)", i, name, want[i], names)
			}
		}
	})
}

func TestFieldListFromSchemaAttributesOrder(t *testing.T) {
	attrs := map[string]interface{}{
		"zebra_field": map[string]interface{}{"type": "string"},
		"alpha_field": map[string]interface{}{"type": "integer"},
		"middle_field": map[string]interface{}{
			"type":           "relation",
			"relationModule": "accounts",
		},
	}

	fieldList := fieldListFromSchemaAttributes(attrs, []string{"alpha_field"})
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
}

func TestSchemaResponseWithSortedAttributesMarshalJSON(t *testing.T) {
	raw := map[string]interface{}{
		"primary-key": []interface{}{"id"},
		"attributes": map[string]interface{}{
			"zebra": map[string]interface{}{"type": "string"},
			"alpha": map[string]interface{}{"type": "integer"},
			"mike":  map[string]interface{}{"type": "boolean"},
		},
	}

	sorted := schemaResponseWithSortedAttributes(raw)
	attrs, ok := sorted["attributes"].(sortedStringMap)
	if !ok {
		t.Fatalf("attributes type = %T, want sortedStringMap", sorted["attributes"])
	}

	attrsJSON, err := json.Marshal(attrs)
	if err != nil {
		t.Fatalf("json.Marshal(attrs) returned error: %v", err)
	}
	wantAttrsJSON := `{"alpha":{"type":"integer"},"mike":{"type":"boolean"},"zebra":{"type":"string"}}`
	if string(attrsJSON) != wantAttrsJSON {
		t.Fatalf("attributes JSON = %s, want %s", attrsJSON, wantAttrsJSON)
	}

	// Marshal twice to confirm stable output.
	again, err := json.Marshal(attrs)
	if err != nil {
		t.Fatalf("json.Marshal(attrs) second call returned error: %v", err)
	}
	if string(again) != wantAttrsJSON {
		t.Fatalf("second attributes JSON = %s, want %s", again, wantAttrsJSON)
	}
}

func moduleNamesFromData(data interface{}) []string {
	items, ok := data.([]map[string]interface{})
	if !ok {
		return nil
	}

	names := make([]string, 0, len(items))
	for _, item := range items {
		attrs, ok := item["attributes"].(map[string]interface{})
		if !ok {
			continue
		}
		name, ok := attrs["name"].(string)
		if !ok {
			continue
		}
		names = append(names, name)
	}
	return names
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
