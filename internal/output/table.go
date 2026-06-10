package output

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

type TableMode int

const (
	TableModeList TableMode = iota
	TableModeItem
)

func resolveColumnHeaders(fields, columns []string) []string {
	switch {
	case len(fields) > 0:
		return fields
	case len(columns) > 0:
		return columns
	default:
		return []string{"id", "type"}
	}
}

func buildTabularRow(item interface{}, columnHeaders []string) []string {
	itemVal := reflect.ValueOf(item)
	if itemVal.Kind() != reflect.Map {
		return nil
	}

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
		}
	}
	return row
}

func buildTabularRows(data interface{}, fields, columns []string) (headers []string, rows [][]string) {
	dataVal := reflect.ValueOf(data)
	if dataVal.Kind() == reflect.Map {
		headers, rows = buildTabularRows([]interface{}{data}, fields, columns)
		return headers, rows
	}
	if dataVal.Kind() != reflect.Slice || dataVal.Len() == 0 {
		return nil, nil
	}

	headers = resolveColumnHeaders(fields, columns)
	for i := 0; i < dataVal.Len(); i++ {
		if row := buildTabularRow(dataVal.Index(i).Interface(), headers); row != nil {
			rows = append(rows, row)
		}
	}
	return headers, rows
}

func extractCSVRecords(data interface{}, fields []string, _ bool, columns []string) ([][]string, error) {
	headers, rows := buildTabularRows(data, fields, columns)
	if len(rows) == 0 {
		return nil, nil
	}
	return append([][]string{headers}, rows...), nil
}

func printKeyValueMap(w io.Writer, titleKey, titleValue string, data map[string]interface{}) {
	fmt.Fprintf(w, "%-20s | %s\n", titleKey, titleValue)

	maxLen := 0
	for key := range data {
		if len(key) > maxLen {
			maxLen = len(key)
		}
	}
	fmt.Fprintln(w, strings.Repeat("-", maxLen+8))

	for key, value := range data {
		fmt.Fprintf(w, "%-*s | %v\n", maxLen, key, value)
	}
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

	headers, rows := buildTabularRows(data, fields, columns)
	if len(rows) > 0 {
		printTable(headers, rows)
	}
	return nil
}

func outputTableItemData(data interface{}, fields []string, full bool, columns []string) error {
	dataVal := reflect.ValueOf(data)
	if dataVal.Kind() != reflect.Map {
		fmt.Println("No data found")
		return nil
	}

	if attr := extractAttributes(data); attr != nil {
		printKeyValueMap(os.Stdout, "ATTRIBUTE", "VALUE", attr)
		return nil
	}

	fallback := make(map[string]interface{}, dataVal.Len())
	for _, key := range dataVal.MapKeys() {
		fallback[key.String()] = dataVal.MapIndex(key).Interface()
	}
	printKeyValueMap(os.Stdout, "KEY", "VALUE", fallback)
	return nil
}