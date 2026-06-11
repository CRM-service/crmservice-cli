package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"crmservice/internal/api"
)

var activeFormat = "table"

type ReportedError struct {
	Err error
}

func (e *ReportedError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ReportedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func SetActiveFormat(format string) {
	if ValidOutputFormat(format) {
		activeFormat = format
	}
}

func EmitError(err error) {
	if err == nil {
		return
	}
	if writeErr := WriteStderr(activeFormat, errorPayload(err)); writeErr != nil && activeFormat == "table" {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

func errorPayload(err error) map[string]interface{} {
	payload := map[string]interface{}{
		"error":   true,
		"message": err.Error(),
	}
	if apiErr, ok := err.(*api.Error); ok {
		payload["status"] = apiErr.Status
		payload["message"] = apiErr.Message
	}
	return payload
}

func WriteStderr(format string, data interface{}) error {
	switch format {
	case "json":
		return encodeJSON(os.Stderr, data, true)
	case "jsonl":
		return json.NewEncoder(os.Stderr).Encode(data)
	case "yaml":
		return encodeYAML(os.Stderr, data)
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
	return WriteStructured("table", record, StructuredOptions{
		Destination: os.Stderr,
		Columns:     keys,
		OmitEmpty:   true,
	})
}

func StderrError(format string, err error) error {
	if err == nil {
		return nil
	}
	if format == "table" {
		if apiErr, ok := err.(*api.Error); ok {
			fmt.Fprintf(os.Stderr, "Error: %s\n", apiErr.Message)
			return err
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}
	if writeErr := WriteStderr(format, errorPayload(err)); writeErr != nil {
		return writeErr
	}
	return err
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
