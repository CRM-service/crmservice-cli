package cmd

import (
	"bytes"
	"encoding/json"
	"sort"
)

func sortedMapKeys(m map[string]interface{}, skip ...string) []string {
	skipSet := make(map[string]struct{}, len(skip))
	for _, s := range skip {
		skipSet[s] = struct{}{}
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		if _, ok := skipSet[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

// sortedStringMap marshals object keys in sorted order for stable JSON output.
type sortedStringMap map[string]interface{}

func (m sortedStringMap) MarshalJSON() ([]byte, error) {
	keys := sortedMapKeys(map[string]interface{}(m))

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyBytes, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		valBytes, err := json.Marshal(m[k])
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)
		buf.WriteByte(':')
		buf.Write(valBytes)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func schemaResponseWithSortedAttributes(raw map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(raw))
	for k, v := range raw {
		out[k] = v
	}

	attrs, ok := raw["attributes"].(map[string]interface{})
	if !ok {
		return out
	}

	sorted := make(sortedStringMap, len(attrs))
	for k, v := range attrs {
		sorted[k] = v
	}
	out["attributes"] = sorted
	return out
}
