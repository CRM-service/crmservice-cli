package output

import (
	"encoding/csv"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"crmservice/internal/api"
)

func TestListResponseJSON(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	resp := &api.Response{Data: nil}
	opts := Options{Format: "table"}

	err := ListResponse(resp, opts)
	if err != nil {
		t.Errorf("ListResponse failed: %v", err)
	}
}

func TestItemResponseJSON(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	resp := &api.SingleResponse{Data: nil}
	opts := Options{Format: "table"}

	err := ItemResponse(resp, opts)
	if err != nil {
		t.Errorf("ItemResponse failed: %v", err)
	}
}

func TestItemResponseInvalidFormat(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	tests := []struct {
		format string
		valid  bool
	}{
		{"table", true},
		{"json", true},
		{"yaml", true},
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
	t.Parallel()

	testData := map[string]interface{}{"key": "value"}

	err := outputJSON(testData, Options{})
	if err != nil {
		t.Errorf("outputJSON failed: %v", err)
	}
}

func TestOutputYAMLDataNil(t *testing.T) {
	t.Parallel()

	resp := &api.Response{Data: nil}

	err := outputYAML(resp, Options{})
	if err != nil {
		t.Errorf("outputYAML failed: %v", err)
	}
}

func TestOutputYAMLNonFull(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	resp := &api.Response{Data: nil}

	err := outputCSV(resp, Options{})
	if err != nil {
		t.Errorf("outputCSV failed: %v", err)
	}
}

func TestOutputCSVEmptyRecords(t *testing.T) {
	t.Parallel()

	resp := &api.Response{Data: []interface{}{}}

	err := outputCSV(resp, Options{})
	if err != nil {
		t.Errorf("outputCSV failed: %v", err)
	}
}

func TestOutputTableDataNil(t *testing.T) {
	t.Parallel()

	resp := &api.Response{Data: nil}

	err := outputTable(resp, Options{})
	if err != nil {
		t.Errorf("outputTable failed: %v", err)
	}
}

func TestOutputTableEmptyData(t *testing.T) {
	t.Parallel()

	resp := &api.Response{Data: []interface{}{}}

	err := outputTable(resp, Options{})
	if err != nil {
		t.Errorf("outputTable failed: %v", err)
	}
}

func TestExtractAttributes(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	result := extractAttributes("not a map")
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

func TestPrintTable(t *testing.T) {
	t.Parallel()

	headers := []string{"Name", "Value"}
	rows := [][]string{{"test", "123"}}

	printTable(headers, rows)
}

func TestPrintRow(t *testing.T) {
	t.Parallel()

	row := []string{"test", "value"}
	colWidths := []int{4, 5}

	printRow(row, colWidths)
}

func TestPrintSeparator(t *testing.T) {
	t.Parallel()

	colWidths := []int{4, 5}

	printSeparator(colWidths)
}

func TestExtractCSVRecords(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	records, err := extractCSVRecords("not a slice", nil, false, nil)
	if err != nil {
		t.Errorf("extractCSVRecords failed: %v", err)
	}
	if records != nil {
		t.Errorf("Expected nil records, got %v", records)
	}
}

func TestExtractCSVRecordsEmpty(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	err := outputTableData("not a slice", nil, false, nil)
	if err != nil {
		t.Errorf("outputTableData failed: %v", err)
	}
}

func TestOutputTableDataEmpty(t *testing.T) {
	t.Parallel()

	err := outputTableData([]interface{}{}, nil, false, nil)
	if err != nil {
		t.Errorf("outputTableData failed: %v", err)
	}
}

func TestOutputTableDataFull(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	err := outputTableItemData(nil, nil, false, nil)
	if err != nil {
		t.Errorf("outputTableItemData failed: %v", err)
	}
}

func TestOutputTableItemDataNonMap(t *testing.T) {
	t.Parallel()

	err := outputTableItemData("not a map", nil, false, nil)
	if err != nil {
		t.Errorf("outputTableItemData failed: %v", err)
	}
}

func TestOutputTableItemDataWithAttributes(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	apiErr := &api.Error{
		Status:  404,
		Body:    []byte("Not Found"),
		Message: "Resource not found",
	}

	err := ErrorResponse(apiErr)
	if err == nil {
		t.Error("Expected error to be returned")
	}
}

func TestErrorResponseRegularError(t *testing.T) {
	t.Parallel()

	err := errors.New("regular error")

	result := ErrorResponse(err)
	if result == nil {
		t.Error("Expected error to be returned")
	}
}

func TestLongValues(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
