package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"crmservice/internal/api"
)

type Options struct {
	Format  string
	Fields  []string
	Full    bool
	Columns []string
}

func outputJSONL(v interface{}, opts Options) error {
	data := cleanJSONLRecords(v, opts)
	if data == nil {
		fmt.Println("No data found")
		return nil
	}
	return StreamJSONLRecords(data, opts)
}

type CSVStreamWriter struct {
	headerWritten bool
	writer        *csv.Writer
}

func NewCSVStreamWriter() *CSVStreamWriter {
	return &CSVStreamWriter{writer: csv.NewWriter(os.Stdout)}
}

func (w *CSVStreamWriter) WritePage(data interface{}, opts Options) error {
	if data == nil {
		return nil
	}

	records, err := extractCSVRecords(data, opts.Fields, opts.Full, opts.Columns)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}

	start := 1
	if !w.headerWritten {
		if err := w.writer.Write(records[0]); err != nil {
			return fmt.Errorf("failed to write CSV: %w", err)
		}
		w.headerWritten = true
	}

	for i := start; i < len(records); i++ {
		if err := w.writer.Write(records[i]); err != nil {
			return fmt.Errorf("failed to write CSV: %w", err)
		}
	}
	w.writer.Flush()
	return w.writer.Error()
}

func StreamJSONLRecords(data interface{}, opts Options) error {
	if data == nil {
		return nil
	}

	encoder := json.NewEncoder(os.Stdout)
	dataVal := reflect.ValueOf(data)
	if dataVal.Kind() == reflect.Slice {
		for i := 0; i < dataVal.Len(); i++ {
			record := dataVal.Index(i).Interface()
			if !opts.Full {
				record = flattenResource(record)
			}
			if err := encoder.Encode(record); err != nil {
				return err
			}
		}
		return nil
	}

	record := data
	if !opts.Full {
		record = flattenResource(data)
	}
	return encoder.Encode(record)
}

func cleanJSONLRecords(v interface{}, opts Options) interface{} {
	if resp, ok := v.(*api.Response); ok {
		if opts.Full {
			return resp.Data
		}
		return flattenResourceList(resp.Data)
	}
	if single, ok := v.(*api.SingleResponse); ok {
		if opts.Full {
			return single.Data
		}
		return flattenResource(single.Data)
	}
	return v
}

func cleanStructuredOutput(v interface{}, opts Options) interface{} {
	if opts.Full {
		return v
	}

	if resp, ok := v.(*api.Response); ok {
		return flattenResourceList(resp.Data)
	}
	if single, ok := v.(*api.SingleResponse); ok {
		return flattenResource(single.Data)
	}

	return v
}

func flattenResourceList(data interface{}) interface{} {
	if data == nil {
		return nil
	}

	dataVal := reflect.ValueOf(data)
	if dataVal.Kind() != reflect.Slice {
		return flattenResource(data)
	}

	items := make([]interface{}, 0, dataVal.Len())
	for i := 0; i < dataVal.Len(); i++ {
		items = append(items, flattenResource(dataVal.Index(i).Interface()))
	}
	return items
}

func flattenResource(item interface{}) interface{} {
	itemMap, ok := item.(map[string]interface{})
	if !ok {
		return item
	}

	attrs, ok := itemMap["attributes"].(map[string]interface{})
	if !ok {
		return item
	}

	record := make(map[string]interface{}, len(attrs)+1)
	for key, value := range attrs {
		record[key] = value
	}
	if id, ok := itemMap["id"]; ok {
		record["id"] = id
	}
	return record
}

func outputCSV(v interface{}, opts Options) error {
	var records [][]string

	if resp, ok := v.(*api.Response); ok {
		if resp.Data == nil {
			fmt.Println("No data found")
			return nil
		}

		recs, err := extractCSVRecords(resp.Data, opts.Fields, opts.Full, opts.Columns)
		if err != nil {
			return err
		}
		records = recs
	} else if single, ok := v.(*api.SingleResponse); ok {
		if single.Data == nil {
			fmt.Println("No data found")
			return nil
		}

		recs, err := extractCSVRecords(single.Data, opts.Fields, opts.Full, opts.Columns)
		if err != nil {
			return err
		}
		records = recs
	}

	if len(records) > 0 {
		writer := csv.NewWriter(os.Stdout)
		if err := writer.WriteAll(records); err != nil {
			return fmt.Errorf("failed to write CSV: %w", err)
		}
		return writer.Error()
	}

	return nil
}

func extractAttributes(item interface{}) map[string]interface{} {
	itemVal := reflect.ValueOf(item)

	if itemVal.Kind() != reflect.Map {
		return nil
	}

	for _, key := range itemVal.MapKeys() {
		if key.String() == "attributes" {
			attr := itemVal.MapIndex(key).Interface()
			if attrMap, ok := attr.(map[string]interface{}); ok {
				return attrMap
			}
		}
	}
	return nil
}

func printTable(headers []string, rows [][]string) {
	colWidths := make([]int, len(headers))

	for i, header := range headers {
		colWidths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	printRow(headers, colWidths)
	printSeparator(colWidths)

	for _, row := range rows {
		printRow(row, colWidths)
	}
}

func printRow(row []string, colWidths []int) {
	for i, cell := range row {
		fmt.Printf("%-*s", colWidths[i], cell)
		if i < len(row)-1 {
			fmt.Print(" | ")
		}
	}
	fmt.Println()
}

func printSeparator(colWidths []int) {
	for i, width := range colWidths {
		fmt.Print(strings.Repeat("-", width))
		if i < len(colWidths)-1 {
			fmt.Print("-+-")
		}
	}
	fmt.Println()
}

func ErrorResponse(err error) error {
	if writeErr := StderrError(activeFormat, err); writeErr != nil {
		return &ReportedError{Err: writeErr}
	}
	return &ReportedError{Err: err}
}

func TruncationStatus(format string, returned, maxResults int) error {
	status := map[string]interface{}{
		"status":      "truncated",
		"max_results": maxResults,
		"returned":    returned,
		"message":     fmt.Sprintf("Returned %d records (limit %d); more records may exist", returned, maxResults),
	}
	if format == "table" {
		fmt.Fprintf(os.Stderr, "Truncated: %s\n", status["message"])
		return nil
	}
	return WriteStderr(format, status)
}
