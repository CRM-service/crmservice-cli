package cmd

import (
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
