package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func WriteStderr(format string, data interface{}) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stderr)
		encoder.SetIndent("", "  ")
		return encoder.Encode(data)
	case "jsonl":
		return json.NewEncoder(os.Stderr).Encode(data)
	case "yaml":
		encoder := yaml.NewEncoder(os.Stderr)
		encoder.SetIndent(2)
		return encoder.Encode(data)
	case "csv":
		return writeStderrCSV(data)
	default:
		return writeStderrTable(data)
	}
}

func writeStderrTable(data interface{}) error {
	record, ok := data.(map[string]interface{})
	if !ok {
		fmt.Fprintf(os.Stderr, "%v\n", data)
		return nil
	}
	keys := []string{"valid", "schema_checked", "module", "message", "error", "status", "returned", "max_results"}
	for _, key := range keys {
		if value, ok := record[key]; ok && value != nil && value != "" {
			fmt.Fprintf(os.Stderr, "%-20s | %v\n", key, value)
		}
	}
	return nil
}

func writeStderrCSV(data interface{}) error {
	record, ok := data.(map[string]interface{})
	if !ok {
		fmt.Fprintf(os.Stderr, "%v\n", data)
		return nil
	}
	columns := []string{"valid", "schema_checked", "module", "message", "error", "status", "returned", "max_results"}
	writer := csv.NewWriter(os.Stderr)
	if err := writer.Write(columns); err != nil {
		return err
	}
	row := make([]string, len(columns))
	for i, column := range columns {
		if value, ok := record[column]; ok && value != nil {
			row[i] = fmt.Sprintf("%v", value)
		}
	}
	if err := writer.Write(row); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}