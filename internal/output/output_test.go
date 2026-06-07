package output

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"crmservice/internal/api"
)

func TestListResponseJSON(t *testing.T) {
	resp := &api.Response{
		Data: []interface{}{
			map[string]interface{}{
				"id":   "1",
				"type": "test",
				"attributes": map[string]interface{}{
					"name": "test1",
				},
			},
		},
	}

	opts := Options{Format: "json"}

	err := ListResponse(resp, opts)
	if err != nil {
		t.Errorf("ListResponse failed: %v", err)
	}
}

func TestListResponseYAML(t *testing.T) {
	resp := &api.Response{
		Data: []interface{}{
			map[string]interface{}{
				"id":   "1",
				"type": "test",
				"attributes": map[string]interface{}{
					"name": "test1",
				},
			},
		},
	}

	opts := Options{Format: "yaml"}

	err := ListResponse(resp, opts)
	if err != nil {
		t.Errorf("ListResponse failed: %v", err)
	}
}

func TestListResponseCSV(t *testing.T) {
	resp := &api.Response{
		Data: []interface{}{
			map[string]interface{}{
				"id":         "1",
				"type":       "test",
				"attributes": map[string]interface{}{"name": "test1"},
			},
		},
	}

	opts := Options{Format: "csv"}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to capture stdout: %v", err)
	}
	os.Stdout = w

	err = ListResponse(resp, opts)
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ListResponse failed: %v", err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}
	reader := csv.NewReader(strings.NewReader(string(out)))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("Expected 2 CSV records (header + data), got %d", len(records))
	}
}

func TestListResponseTable(t *testing.T) {
	resp := &api.Response{
		Data: []interface{}{
			map[string]interface{}{
				"id":         "1",
				"type":       "test",
				"attributes": map[string]interface{}{"name": "test1"},
			},
		},
	}

	opts := Options{Format: "table"}

	err := ListResponse(resp, opts)
	if err != nil {
		t.Errorf("ListResponse failed: %v", err)
	}
}

func TestListResponseInvalidFormat(t *testing.T) {
	resp := &api.Response{Data: []interface{}{}}
	opts := Options{Format: "invalid"}

	err := ListResponse(resp, opts)
	if err == nil {
		t.Error("Expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid output format") {
		t.Errorf("Expected 'invalid output format' error, got: %v", err)
	}
}

func TestListResponseEmptyData(t *testing.T) {
	resp := &api.Response{Data: nil}
	opts := Options{Format: "table"}

	err := ListResponse(resp, opts)
	if err != nil {
		t.Errorf("ListResponse failed: %v", err)
	}
}

func TestItemResponseJSON(t *testing.T) {
	resp := &api.SingleResponse{
		Data: map[string]interface{}{
			"id":         "1",
			"type":       "test",
			"attributes": map[string]interface{}{"name": "test1"},
		},
	}

	opts := Options{Format: "json"}

	err := ItemResponse(resp, opts)
	if err != nil {
		t.Errorf("ItemResponse failed: %v", err)
	}
}

func TestItemResponseYAML(t *testing.T) {
	resp := &api.SingleResponse{
		Data: map[string]interface{}{
			"id":         "1",
			"type":       "test",
			"attributes": map[string]interface{}{"name": "test1"},
		},
	}

	opts := Options{Format: "yaml"}

	err := ItemResponse(resp, opts)
	if err != nil {
		t.Errorf("ItemResponse failed: %v", err)
	}
}

func TestItemResponseCSV(t *testing.T) {
	resp := &api.SingleResponse{
		Data: []interface{}{
			map[string]interface{}{
				"id":         "1",
				"type":       "test",
				"attributes": map[string]interface{}{"name": "test1"},
			},
		},
	}

	opts := Options{Format: "csv"}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to capture stdout: %v", err)
	}
	os.Stdout = w

	err = ItemResponse(resp, opts)
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("ItemResponse failed: %v", err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}
	reader := csv.NewReader(strings.NewReader(string(out)))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("Expected 2 CSV records (header + data), got %d", len(records))
	}
}

func TestItemResponseTable(t *testing.T) {
	resp := &api.SingleResponse{
		Data: map[string]interface{}{
			"id":         "1",
			"type":       "test",
			"attributes": map[string]interface{}{"name": "test1"},
		},
	}

	opts := Options{Format: "table"}

	err := ItemResponse(resp, opts)
	if err != nil {
		t.Errorf("ItemResponse failed: %v", err)
	}
}

func TestItemResponseEmptyData(t *testing.T) {
	resp := &api.SingleResponse{Data: nil}
	opts := Options{Format: "table"}

	err := ItemResponse(resp, opts)
	if err != nil {
		t.Errorf("ItemResponse failed: %v", err)
	}
}

func TestItemResponseInvalidFormat(t *testing.T) {
	resp := &api.SingleResponse{Data: map[string]interface{}{}}
	opts := Options{Format: "invalid"}

	err := ItemResponse(resp, opts)
	if err == nil {
		t.Error("Expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid output format") {
		t.Errorf("Expected 'invalid output format' error, got: %v", err)
	}
}

func TestValidOutputFormat(t *testing.T) {
	tests := []struct {
		format string
		valid  bool
	}{
		{"table", true},
		{"json", true},
		{"yaml", true},
		{"jsonl", true},
		{"csv", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := ValidOutputFormat(tt.format)
			if result != tt.valid {
				t.Errorf("ValidOutputFormat(%q) = %v, want %v", tt.format, result, tt.valid)
			}
		})
	}
}

func TestOutputJSON(t *testing.T) {
	testData := map[string]interface{}{"key": "value"}

	err := outputJSON(testData, Options{})
	if err != nil {
		t.Errorf("outputJSON failed: %v", err)
	}
}

func TestOutputYAMLDataNil(t *testing.T) {
	resp := &api.Response{Data: nil}

	err := outputYAML(resp, Options{})
	if err != nil {
		t.Errorf("outputYAML failed: %v", err)
	}
}

func TestOutputYAMLNonFull(t *testing.T) {
	resp := &api.Response{
		Data: []interface{}{
			map[string]interface{}{
				"id":         "1",
				"type":       "test",
				"attributes": map[string]interface{}{"name": "test1"},
			},
		},
	}

	opts := Options{Format: "yaml", Full: false}

	err := outputYAML(resp, opts)
	if err != nil {
		t.Errorf("outputYAML failed: %v", err)
	}
}

func TestOutputCSVDataNil(t *testing.T) {
	resp := &api.Response{Data: nil}

	err := outputCSV(resp, Options{})
	if err != nil {
		t.Errorf("outputCSV failed: %v", err)
	}
}

func TestOutputCSVEmptyRecords(t *testing.T) {
	resp := &api.Response{Data: []interface{}{}}

	err := outputCSV(resp, Options{})
	if err != nil {
		t.Errorf("outputCSV failed: %v", err)
	}
}

func TestOutputTableDataNil(t *testing.T) {
	resp := &api.Response{Data: nil}

	err := outputTable(resp, Options{})
	if err != nil {
		t.Errorf("outputTable failed: %v", err)
	}
}

func TestOutputTableEmptyData(t *testing.T) {
	resp := &api.Response{Data: []interface{}{}}

	err := outputTable(resp, Options{})
	if err != nil {
		t.Errorf("outputTable failed: %v", err)
	}
}

func TestExtractAttributes(t *testing.T) {
	item := map[string]interface{}{
		"id":         "1",
		"type":       "test",
		"attributes": map[string]interface{}{"name": "test1"},
	}

	result := extractAttributes(item)
	if result == nil {
		t.Error("Expected attributes map, got nil")
	}
	if result["name"] != "test1" {
		t.Errorf("Expected name=test1, got %v", result["name"])
	}
}

func TestExtractAttributesNoAttributes(t *testing.T) {
	item := map[string]interface{}{
		"id":   "1",
		"type": "test",
	}

	result := extractAttributes(item)
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

func TestExtractAttributesNonMap(t *testing.T) {
	result := extractAttributes("not a map")
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

func TestPrintTable(t *testing.T) {
	headers := []string{"Name", "Value"}
	rows := [][]string{{"test", "123"}}

	printTable(headers, rows)
}

func TestPrintRow(t *testing.T) {
	row := []string{"test", "value"}
	colWidths := []int{4, 5}

	printRow(row, colWidths)
}

func TestPrintSeparator(t *testing.T) {
	colWidths := []int{4, 5}

	printSeparator(colWidths)
}

func TestExtractCSVRecords(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{
			"id":         "1",
			"type":       "test",
			"attributes": map[string]interface{}{"name": "test1"},
		},
	}

	records, err := extractCSVRecords(data, []string{"id", "name"}, false, nil)
	if err != nil {
		t.Errorf("extractCSVRecords failed: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("Expected 2 records (header + data), got %d", len(records))
	}
}

func TestExtractCSVRecordsNonSlice(t *testing.T) {
	records, err := extractCSVRecords("not a slice", nil, false, nil)
	if err != nil {
		t.Errorf("extractCSVRecords failed: %v", err)
	}
	if records != nil {
		t.Errorf("Expected nil records, got %v", records)
	}
}

func TestExtractCSVRecordsEmpty(t *testing.T) {
	data := []interface{}{}
	records, err := extractCSVRecords(data, nil, false, nil)
	if err != nil {
		t.Errorf("extractCSVRecords failed: %v", err)
	}
	if records != nil {
		t.Errorf("Expected nil records, got %v", records)
	}
}

func TestOutputTableDataNonSlice(t *testing.T) {
	err := outputTableData("not a slice", nil, false, nil)
	if err != nil {
		t.Errorf("outputTableData failed: %v", err)
	}
}

func TestOutputTableDataEmpty(t *testing.T) {
	err := outputTableData([]interface{}{}, nil, false, nil)
	if err != nil {
		t.Errorf("outputTableData failed: %v", err)
	}
}

func TestOutputTableDataFull(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{
			"id":         "1",
			"type":       "test",
			"attributes": map[string]interface{}{"name": "test1"},
		},
	}

	err := outputTableData(data, nil, true, nil)
	if err != nil {
		t.Errorf("outputTableData failed: %v", err)
	}
}

func TestOutputTableDataWithFields(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{
			"id":         "1",
			"type":       "test",
			"attributes": map[string]interface{}{"name": "test1", "value": "123"},
		},
	}

	err := outputTableData(data, []string{"id", "name"}, false, nil)
	if err != nil {
		t.Errorf("outputTableData failed: %v", err)
	}
}

func TestOutputTableItemDataNil(t *testing.T) {
	err := outputTableItemData(nil, nil, false, nil)
	if err != nil {
		t.Errorf("outputTableItemData failed: %v", err)
	}
}

func TestOutputTableItemDataNonMap(t *testing.T) {
	err := outputTableItemData("not a map", nil, false, nil)
	if err != nil {
		t.Errorf("outputTableItemData failed: %v", err)
	}
}

func TestOutputTableItemDataWithAttributes(t *testing.T) {
	data := map[string]interface{}{
		"id":         "1",
		"type":       "test",
		"attributes": map[string]interface{}{"name": "test1", "value": "123"},
	}

	err := outputTableItemData(data, nil, false, nil)
	if err != nil {
		t.Errorf("outputTableItemData failed: %v", err)
	}
}

func TestErrorResponseAPIError(t *testing.T) {
	apiErr := &api.Error{
		Status:  404,
		Body:    []byte("Not Found"),
		Message: "Resource not found",
	}

	_ = captureStderr(t, func() {
		err := ErrorResponse(apiErr)
		if err == nil {
			t.Error("Expected error to be returned")
		}
	})
}

func TestErrorResponseRegularError(t *testing.T) {
	_ = captureStderr(t, func() {
		err := errors.New("regular error")
		result := ErrorResponse(err)
		if result == nil {
			t.Error("Expected error to be returned")
		}
	})
}

func TestLongValues(t *testing.T) {
	longValue := strings.Repeat("x", 1000)

	resp := &api.Response{
		Data: []interface{}{
			map[string]interface{}{
				"id":         "1",
				"type":       "test",
				"attributes": map[string]interface{}{"long": longValue},
			},
		},
	}

	opts := Options{Format: "table"}

	err := ListResponse(resp, opts)
	if err != nil {
		t.Errorf("ListResponse failed: %v", err)
	}
}

func TestMixedTypes(t *testing.T) {
	resp := &api.Response{
		Data: []interface{}{
			map[string]interface{}{
				"id":   "1",
				"type": "test",
				"attributes": map[string]interface{}{
					"string": "text",
					"int":    123,
					"bool":   true,
					"float":  1.23,
				},
			},
			nil,
		},
	}

	opts := Options{Format: "table"}

	err := ListResponse(resp, opts)
	if err != nil {
		t.Errorf("ListResponse failed: %v", err)
	}
}

func TestOutputJSONNonFullFlattensResourceList(t *testing.T) {
	resp := &api.Response{
		Data: []interface{}{
			map[string]interface{}{
				"id":   "297603",
				"type": "accounts",
				"attributes": map[string]interface{}{
					"entity_no":  "ACC-1",
					"name":       "Acme Corp",
					"created_at": "2026-01-01 12:00:00",
				},
			},
		},
		Meta: &api.Meta{Total: 1},
	}

	out := captureStdout(t, func() {
		if err := outputJSON(resp, Options{Format: "json"}); err != nil {
			t.Fatalf("outputJSON() returned error: %v", err)
		}
	})

	var records []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &records); err != nil {
		t.Fatalf("json.Unmarshal() returned error: %v\noutput: %s", err, out)
	}
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, expected 1", len(records))
	}
	if records[0]["id"] != "297603" {
		t.Errorf("id = %v, expected 297603", records[0]["id"])
	}
	if records[0]["name"] != "Acme Corp" {
		t.Errorf("name = %v, expected Acme Corp", records[0]["name"])
	}
	if _, ok := records[0]["attributes"]; ok {
		t.Error("flattened record should not contain attributes")
	}
	if _, ok := records[0]["type"]; ok {
		t.Error("flattened record should not contain type")
	}
}

func TestOutputJSONFullPreservesEnvelope(t *testing.T) {
	resp := &api.Response{
		Data: []interface{}{
			map[string]interface{}{
				"id":         "297603",
				"type":       "accounts",
				"attributes": map[string]interface{}{"name": "Acme Corp"},
			},
		},
		Meta: &api.Meta{Total: 1},
	}

	out := captureStdout(t, func() {
		if err := outputJSON(resp, Options{Format: "json", Full: true}); err != nil {
			t.Fatalf("outputJSON() returned error: %v", err)
		}
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("json.Unmarshal() returned error: %v", err)
	}
	if _, ok := envelope["data"]; !ok {
		t.Error("full JSON output should contain data envelope")
	}
	if _, ok := envelope["meta"]; !ok {
		t.Error("full JSON output should contain meta")
	}
}

func TestOutputYAMLNonFullFlattensResourceList(t *testing.T) {
	resp := &api.Response{
		Data: []interface{}{
			map[string]interface{}{
				"id":         "297603",
				"type":       "accounts",
				"attributes": map[string]interface{}{"name": "Acme Corp"},
			},
		},
	}

	out := captureStdout(t, func() {
		if err := outputYAML(resp, Options{Format: "yaml"}); err != nil {
			t.Fatalf("outputYAML() returned error: %v", err)
		}
	})

	if !strings.Contains(out, "id: \"297603\"") && !strings.Contains(out, "id: 297603") {
		t.Errorf("YAML output should contain flattened id, got: %s", out)
	}
	if !strings.Contains(out, "name: Acme Corp") {
		t.Errorf("YAML output should contain flattened name, got: %s", out)
	}
	if strings.Contains(out, "attributes:") {
		t.Errorf("YAML output should not contain attributes, got: %s", out)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned error: %v", err)
	}
	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("writer.Close() returned error: %v", err)
	}
	os.Stdout = oldStdout
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("io.ReadAll() returned error: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("reader.Close() returned error: %v", err)
	}
	return string(out)
}

func TestCSVStreamWriterWritesHeaderOnceAcrossPages(t *testing.T) {
	dataPage1 := []interface{}{
		map[string]interface{}{
			"id":         "1",
			"type":       "accounts",
			"attributes": map[string]interface{}{"name": "A"},
		},
	}
	dataPage2 := []interface{}{
		map[string]interface{}{
			"id":         "2",
			"type":       "accounts",
			"attributes": map[string]interface{}{"name": "B"},
		},
	}

	opts := Options{Format: "csv"}

	out := captureStdout(t, func() {
		writer := NewCSVStreamWriter()
		if err := writer.WritePage(dataPage1, opts); err != nil {
			t.Fatalf("WritePage(page1) error: %v", err)
		}
		if err := writer.WritePage(dataPage2, opts); err != nil {
			t.Fatalf("WritePage(page2) error: %v", err)
		}
	})

	reader := csv.NewReader(strings.NewReader(out))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("CSV records = %d, want header + 2 rows: %q", len(records), out)
	}
	if records[0][0] != "id" || records[0][1] != "type" {
		t.Fatalf("CSV header = %v, want [id type]", records[0])
	}
}

func TestOutputJSONLNonFullWritesOneFlatRecordPerLine(t *testing.T) {
	resp := &api.Response{
		Data: []interface{}{
			map[string]interface{}{
				"id":         "297603",
				"type":       "accounts",
				"attributes": map[string]interface{}{"name": "Acme Corp"},
			},
			map[string]interface{}{
				"id":         "297604",
				"type":       "accounts",
				"attributes": map[string]interface{}{"name": "Example Inc"},
			},
		},
	}

	out := captureStdout(t, func() {
		if err := outputJSONL(resp, Options{Format: "jsonl"}); err != nil {
			t.Fatalf("outputJSONL() returned error: %v", err)
		}
	})

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("JSONL line count = %d, expected 2; output: %s", len(lines), out)
	}
	for i, line := range lines {
		var record map[string]interface{}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("line %d is not valid JSON: %v", i, err)
		}
		if _, ok := record["attributes"]; ok {
			t.Errorf("line %d should contain a flat record, got: %s", i, line)
		}
	}
}
