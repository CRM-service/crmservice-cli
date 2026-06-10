package schema

import (
	"encoding/json"
	"testing"
)

func TestSchemaResponseWithSortedAttributesMarshalJSON(t *testing.T) {
	raw := map[string]interface{}{
		"primary-key": []interface{}{"id"},
		"attributes": map[string]interface{}{
			"zebra": map[string]interface{}{"type": "string"},
			"alpha": map[string]interface{}{"type": "integer"},
			"mike":  map[string]interface{}{"type": "boolean"},
		},
	}

	sorted := ResponseWithSortedAttributes(raw)
	attrs, ok := sorted["attributes"].(SortedStringMap)
	if !ok {
		t.Fatalf("attributes type = %T, want SortedStringMap", sorted["attributes"])
	}

	attrsJSON, err := json.Marshal(attrs)
	if err != nil {
		t.Fatalf("json.Marshal(attrs) returned error: %v", err)
	}
	wantAttrsJSON := `{"alpha":{"type":"integer"},"mike":{"type":"boolean"},"zebra":{"type":"string"}}`
	if string(attrsJSON) != wantAttrsJSON {
		t.Fatalf("attributes JSON = %s, want %s", attrsJSON, wantAttrsJSON)
	}

	again, err := json.Marshal(attrs)
	if err != nil {
		t.Fatalf("json.Marshal(attrs) second call returned error: %v", err)
	}
	if string(again) != wantAttrsJSON {
		t.Fatalf("second attributes JSON = %s, want %s", again, wantAttrsJSON)
	}
}

func TestSortedMapKeys(t *testing.T) {
	keys := SortedMapKeys(map[string]interface{}{
		"zebra": "z",
		"alpha": "a",
		"self":  "s",
	}, "self")
	want := []string{"alpha", "zebra"}
	if len(keys) != len(want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	for i, key := range keys {
		if key != want[i] {
			t.Fatalf("keys[%d] = %q, want %q", i, key, want[i])
		}
	}
}