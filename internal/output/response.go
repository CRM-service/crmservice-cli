package output

import (
	"fmt"
	"os"

	"crmservice/internal/api"
)

func writeResponse(v interface{}, opts Options, mode TableMode) error {
	switch opts.Format {
	case "yaml":
		return outputYAML(v, opts)
	case "csv":
		return outputCSV(v, opts)
	case "jsonl":
		return outputJSONL(v, opts)
	case "json":
		return outputJSON(v, opts)
	case "table":
		return outputTable(v, opts, mode)
	default:
		return ValidateOutputFormat(opts.Format)
	}
}

// ListResponse formats a JSON:API list response.
func ListResponse(resp *api.Response, opts Options) error {
	return writeResponse(resp, opts, TableModeList)
}

// ItemResponse formats a JSON:API single-resource response.
func ItemResponse(resp *api.SingleResponse, opts Options) error {
	return writeResponse(resp, opts, TableModeItem)
}

func outputTable(v interface{}, opts Options, mode TableMode) error {
	data, err := responseData(v)
	if err != nil {
		return err
	}
	if data == nil {
		fmt.Println("No data found")
		return nil
	}

	if mode == TableModeItem {
		return outputTableItemData(data, opts.Fields, opts.Full, opts.Columns)
	}
	return outputTableData(data, opts.Fields, opts.Full, opts.Columns)
}

func responseData(v interface{}) (interface{}, error) {
	if resp, ok := v.(*api.Response); ok {
		return resp.Data, nil
	}
	if single, ok := v.(*api.SingleResponse); ok {
		return single.Data, nil
	}
	return nil, fmt.Errorf("unsupported response type %T", v)
}

func outputJSON(v interface{}, opts Options) error {
	return encodeJSON(os.Stdout, cleanStructuredOutput(v, opts), true)
}

func outputYAML(v interface{}, opts Options) error {
	data := cleanStructuredOutput(v, opts)
	if data == nil {
		fmt.Println("No data found")
		return nil
	}
	return encodeYAML(os.Stdout, data)
}