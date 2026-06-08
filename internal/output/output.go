package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"

	"crmservice/internal/api"
)

type Options struct {
	Format  string
	Fields  []string
	Full    bool
	Columns []string
}

func ListResponse(resp *api.Response, opts Options) error {
	if opts.Format == "yaml" {
		return outputYAML(resp, opts)
	}
	if opts.Format == "csv" {
		return outputCSV(resp, opts)
	}
	if opts.Format == "jsonl" {
		return outputJSONL(resp, opts)
	}
	if opts.Format == "json" {
		return outputJSON(resp, opts)
	}
	if opts.Format == "table" {
		return outputTable(resp, opts)
	}
	return ValidateOutputFormat(opts.Format)
}

func ItemResponse(resp *api.SingleResponse, opts Options) error {
	if opts.Format == "yaml" {
		return outputYAML(resp, opts)
	}
	if opts.Format == "csv" {
		return outputCSV(resp, opts)
	}
	if opts.Format == "jsonl" {
		return outputJSONL(resp, opts)
	}
	if opts.Format == "json" {
		return outputJSON(resp, opts)
	}
	if opts.Format == "table" {
		return outputTableItem(resp, opts)
	}
	return ValidateOutputFormat(opts.Format)
}

func outputJSON(v interface{}, opts Options) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(cleanStructuredOutput(v, opts))
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

func outputYAML(v interface{}, opts Options) error {
	data := cleanStructuredOutput(v, opts)

	if data == nil {
		fmt.Println("No data found")
		return nil
	}

	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	return encoder.Encode(data)
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

func outputTable(v interface{}, opts Options) error {
	data := v

	if resp, ok := data.(*api.Response); ok {
		if resp.Data == nil {
			fmt.Println("No data found")
			return nil
		}

		if err := outputTableData(resp.Data, opts.Fields, opts.Full, opts.Columns); err != nil {
			return err
		}
	} else if single, ok := data.(*api.SingleResponse); ok {
		if single.Data == nil {
			fmt.Println("No data found")
			return nil
		}

		if err := outputTableItemData(single.Data, opts.Fields, opts.Full, opts.Columns); err != nil {
			return err
		}
	}

	return nil
}

func outputTableItem(v interface{}, opts Options) error {
	return outputTable(v, opts)
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

func outputTableData(data interface{}, fields []string, full bool, columns []string) error {
	dataVal := reflect.ValueOf(data)

	if dataVal.Kind() != reflect.Slice {
		fmt.Println("No data found")
		return nil
	}

	if dataVal.Len() == 0 {
		fmt.Println("No data found")
		return nil
	}

	// If full=true, just show the raw data structure
	if full {
		for i := 0; i < dataVal.Len(); i++ {
			item := dataVal.Index(i).Interface()
			itemVal := reflect.ValueOf(item)

			if itemVal.Kind() == reflect.Map {
				for _, key := range itemVal.MapKeys() {
					val := itemVal.MapIndex(key)
					fmt.Printf("%s: %v\n", strings.ToUpper(key.String()), val.Interface())
				}
				if i < dataVal.Len()-1 {
					fmt.Println("---")
				}
			}
		}
		return nil
	}

	var columnHeaders []string

	switch {
	case len(fields) > 0:
		columnHeaders = fields
	case len(columns) > 0:
		columnHeaders = columns
	default:
		columnHeaders = []string{"id", "type"}
	}

	rows := make([][]string, 0)

	for i := 0; i < dataVal.Len(); i++ {
		item := dataVal.Index(i).Interface()
		itemVal := reflect.ValueOf(item)

		if itemVal.Kind() == reflect.Map {
			row := make([]string, len(columnHeaders))

			for j, field := range columnHeaders {
				var val reflect.Value
				if attr := extractAttributes(item); attr != nil {
					val = reflect.ValueOf(attr).MapIndex(reflect.ValueOf(field))
				}

				if !val.IsValid() {
					val = itemVal.MapIndex(reflect.ValueOf(field))
				}

				if val.IsValid() {
					row[j] = fmt.Sprintf("%v", val.Interface())
				} else {
					row[j] = ""
				}
			}
			rows = append(rows, row)
		}
	}

	if len(rows) > 0 {
		printTable(columnHeaders, rows)
	}

	return nil
}

func outputTableItemData(data interface{}, fields []string, full bool, columns []string) error {
	dataVal := reflect.ValueOf(data)

	if dataVal.Kind() != reflect.Map {
		fmt.Println("No data found")
		return nil
	}

	// Check if this is an item with attributes
	if attr := extractAttributes(data); attr != nil {
		// Print attributes in attribute|value format
		attrVal := reflect.ValueOf(attr)
		if attrVal.Kind() == reflect.Map {
			fmt.Printf("%-20s | %s\n", "ATTRIBUTE", "VALUE")

			maxLen := 0
			for _, key := range attrVal.MapKeys() {
				if len(key.String()) > maxLen {
					maxLen = len(key.String())
				}
			}

			separatorLen := maxLen + 3 + 5 // maxLen + " | " + "VALUE"
			fmt.Println(strings.Repeat("-", separatorLen))

			for _, key := range attrVal.MapKeys() {
				val := attrVal.MapIndex(key)
				fmt.Printf("%-*s | %v\n", maxLen, key.String(), val.Interface())
			}
			return nil
		}
	}

	// Fall back to showing map keys
	fmt.Printf("%-20s | %s\n", "KEY", "VALUE")

	maxLen := 0
	for _, key := range dataVal.MapKeys() {
		if len(key.String()) > maxLen {
			maxLen = len(key.String())
		}
	}

	separatorLen := maxLen + 3 + 5 // maxLen + " | " + "VALUE"
	fmt.Println(strings.Repeat("-", separatorLen))

	for _, key := range dataVal.MapKeys() {
		val := dataVal.MapIndex(key)
		fmt.Printf("%-*s | %v\n", maxLen, key.String(), val.Interface())
	}

	return nil
}

func extractCSVRecords(data interface{}, fields []string, full bool, columns []string) ([][]string, error) {
	dataVal := reflect.ValueOf(data)

	switch dataVal.Kind() {
	case reflect.Map:
		return extractCSVRecords([]interface{}{data}, fields, full, columns)
	case reflect.Slice:
	default:
		return nil, nil
	}

	if dataVal.Len() == 0 {
		return nil, nil
	}

	var columnHeaders []string

	switch {
	case len(fields) > 0:
		columnHeaders = fields
	case len(columns) > 0:
		columnHeaders = columns
	default:
		columnHeaders = []string{"id", "type"}
	}

	records := make([][]string, 0)

	for i := 0; i < dataVal.Len(); i++ {
		item := dataVal.Index(i).Interface()
		itemVal := reflect.ValueOf(item)

		if itemVal.Kind() == reflect.Map {
			row := make([]string, len(columnHeaders))
			for j, field := range columnHeaders {
				var val reflect.Value
				if attr := extractAttributes(item); attr != nil {
					val = reflect.ValueOf(attr).MapIndex(reflect.ValueOf(field))
				}
				if !val.IsValid() {
					val = itemVal.MapIndex(reflect.ValueOf(field))
				}
				if val.IsValid() {
					row[j] = fmt.Sprintf("%v", val.Interface())
				} else {
					row[j] = ""
				}
			}
			records = append(records, row)
		}
	}

	if len(records) > 0 {
		return append([][]string{columnHeaders}, records...), nil
	}

	return nil, nil
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
