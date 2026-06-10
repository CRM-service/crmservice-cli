package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// StructuredOptions configures map-based CLI output.
type StructuredOptions struct {
	Columns       []string
	ExtraColumns  []string
	OmitEmpty     bool
	Destination   io.Writer
}

func structuredWriter(opts StructuredOptions) io.Writer {
	if opts.Destination != nil {
		return opts.Destination
	}
	return os.Stdout
}

// WriteStructured writes a map to stdout (or opts.Destination) in the requested format.
func WriteStructured(format string, data map[string]interface{}, opts StructuredOptions) error {
	if !ValidOutputFormat(format) {
		return ValidateOutputFormat(format)
	}

	w := structuredWriter(opts)

	switch format {
	case "json":
		return encodeJSON(w, data, true)
	case "jsonl":
		return json.NewEncoder(w).Encode(data)
	case "yaml":
		return encodeYAML(w, data)
	case "csv":
		return writeStructuredCSV(w, opts.Columns, data)
	case "table":
		return writeStructuredTable(w, opts, data)
	default:
		return ValidateOutputFormat(format)
	}
}

func encodeJSON(w io.Writer, data interface{}, indent bool) error {
	encoder := json.NewEncoder(w)
	if indent {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(data)
}

func encodeYAML(w io.Writer, data interface{}) error {
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	return encoder.Encode(data)
}

func writeStructuredCSV(w io.Writer, columns []string, data map[string]interface{}) error {
	writer := csv.NewWriter(w)
	if err := writer.Write(columns); err != nil {
		return err
	}
	row := make([]string, len(columns))
	for i, column := range columns {
		if value, ok := data[column]; ok && value != nil {
			row[i] = fmt.Sprintf("%v", value)
		}
	}
	if err := writer.Write(row); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

func writeStructuredTable(w io.Writer, opts StructuredOptions, data map[string]interface{}) error {
	for _, key := range opts.Columns {
		value, ok := data[key]
		if opts.OmitEmpty && (!ok || value == nil || value == "") {
			continue
		}
		fmt.Fprintf(w, "%-20s | %v\n", key, value)
	}
	for _, key := range opts.ExtraColumns {
		value, ok := data[key]
		if !ok || value == nil {
			continue
		}
		if key == "issues" {
			if issues, ok := value.([]string); !ok || len(issues) == 0 {
				continue
			}
		}
		fmt.Fprintf(w, "%-20s | %v\n", key, value)
	}
	return nil
}