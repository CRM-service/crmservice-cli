package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"crmservice/internal/api"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var filterOperators = map[string]filterOperatorSpec{
	"$eq":          {MinArgs: 2, MaxArgs: 2, FieldFirst: true},
	"$ne":          {MinArgs: 2, MaxArgs: 2, FieldFirst: true},
	"$gt":          {MinArgs: 2, MaxArgs: 2, FieldFirst: true},
	"$gte":         {MinArgs: 2, MaxArgs: 2, FieldFirst: true},
	"$lt":          {MinArgs: 2, MaxArgs: 2, FieldFirst: true},
	"$lte":         {MinArgs: 2, MaxArgs: 2, FieldFirst: true},
	"$beg":         {MinArgs: 2, MaxArgs: 2, FieldFirst: true},
	"$end":         {MinArgs: 2, MaxArgs: 2, FieldFirst: true},
	"$cts":         {MinArgs: 2, MaxArgs: 2, FieldFirst: true},
	"$not.cts":     {MinArgs: 2, MaxArgs: 2, FieldFirst: true},
	"$like":        {MinArgs: 2, MaxArgs: 2, FieldFirst: true},
	"$regex":       {MinArgs: 2, MaxArgs: 2, FieldFirst: true},
	"$in":          {MinArgs: 2, MaxArgs: 2, FieldFirst: true, SecondArgArray: true},
	"$nin":         {MinArgs: 2, MaxArgs: 2, FieldFirst: true, SecondArgArray: true},
	"$between":     {MinArgs: 3, MaxArgs: 3, FieldFirst: true},
	"$not.between": {MinArgs: 3, MaxArgs: 3, FieldFirst: true},
	"$is.null":     {MinArgs: 1, MaxArgs: 1, FieldFirst: true},
	"$not.null":    {MinArgs: 1, MaxArgs: 1, FieldFirst: true},
}

var logicalFilterOperators = map[string]bool{
	"$and": true,
	"$or":  true,
	"$nor": true,
}

type filterOperatorSpec struct {
	MinArgs        int
	MaxArgs        int
	FieldFirst     bool
	SecondArgArray bool
}

const filterReference = `CRM-service filter language

Filters are JSON expressions passed to list --filter or as the positional argument to search.

Preferred forms (use these in scripts and agent workflows):
  {"$eq":["account_type","Customer"]}
  {"$and":[{"$eq":["account_type","Customer"]},{"$cts":["name","Acme"]}]}

Equality shorthand (human convenience only; same as $eq for a single field):
  {"account_type":"Customer"}

Bare field keys are checked against the module schema when list, search, count,
or filter validate <module> runs. Prefer explicit operators for anything beyond
a single equals comparison.

Logical operators:
  $and, $or, $nor    array of filter expressions
  $not               one filter expression

Comparison operators:
  $eq, $ne, $gt, $gte, $lt, $lte
  $in, $nin          field plus array of values
  $between, $not.between
  $is.null, $not.null

String operators:
  $beg               begins with
  $end               ends with
  $cts               contains
  $not.cts           does not contain
  $like              SQL LIKE pattern; use % wildcards
  $regex             SQL REGEXP pattern

Fields:
  Use API field names from: crmservice fields <module>
  Related fields can be addressed as relation.field when the backend exposes that relation.

Date/time values:
  Use ISO 8601-formatted date/time strings accepted by the API, or $now expressions in date comparisons:
  $now, $now.date, $now.time, $now -7 days, $now.date +1 month

`

func filterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "filter",
		Short: "Show and validate filter language expressions",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "reference",
		Short: "Show filter language reference",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStdout(), filterReference)
		},
	})

	validateCmd := &cobra.Command{
		Use:           "validate [module] <filter-json>",
		Short:         "Validate filter JSON syntax and optionally check fields against module schema",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 || len(args) > 2 {
				return fmt.Errorf("requires 1 or 2 arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := getOutputFormatFromFlagConfig(cmd)
			if err != nil {
				return err
			}
			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, jsonl, csv", outputFormat)
			}

			module := ""
			filterJSON := args[0]
			if len(args) == 2 {
				module = args[0]
				filterJSON = args[1]
			}

			if module == "" {
				if err := validateFilterJSON(filterJSON); err != nil {
					return outputFilterValidateResult(outputFormat, filterValidateFailure("", filterJSON, false, err))
				}
				return outputFilterValidateResult(outputFormat, filterValidateSuccess("", filterJSON, false))
			}

			url, err := getURLFromFlagOrEnv(cmd)
			if err != nil {
				return err
			}
			token, err := getRequiredTokenFromFlagEnvConfig(cmd)
			if err != nil {
				return err
			}
			verbose, err := cmd.Flags().GetInt("verbose")
			if err != nil {
				return err
			}
			if err := validateFilterJSONAgainstModule(module, filterJSON, url, token, verbose); err != nil {
				return outputFilterValidateResult(outputFormat, filterValidateFailure(module, filterJSON, true, err))
			}
			return outputFilterValidateResult(outputFormat, filterValidateSuccess(module, filterJSON, true))
		},
	}
	validateCmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, jsonl, or csv")
	validateCmd.Flags().Int("verbose", 0, "Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)")
	cmd.AddCommand(validateCmd)

	return cmd
}

func validateFilterJSON(input string) error {
	var filter interface{}
	decoder := json.NewDecoder(strings.NewReader(input))
	decoder.UseNumber()
	if err := decoder.Decode(&filter); err != nil {
		return fmt.Errorf("invalid filter JSON: %w", err)
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err == nil {
		return fmt.Errorf("invalid filter JSON: multiple JSON values")
	} else if err != io.EOF {
		return fmt.Errorf("invalid filter JSON: %w", err)
	}
	return validateFilterExpression(filter, "$")
}

func validateFilterExpression(expr interface{}, path string) error {
	switch value := expr.(type) {
	case map[string]interface{}:
		if len(value) == 0 {
			return fmt.Errorf("%s: filter object must not be empty", path)
		}
		for key, node := range value {
			if strings.HasPrefix(key, "$") {
				if err := validateFilterOperator(key, node, path+"."+key); err != nil {
					return err
				}
			} else {
				// Bare field keys are equality shorthand, e.g. {"account_type":"Customer"}.
				// Field names are validated against the module schema separately.
				if strings.TrimSpace(key) == "" {
					return fmt.Errorf("%s: field name must not be empty", path)
				}
			}
		}
		return nil
	case []interface{}:
		if len(value) == 0 {
			return fmt.Errorf("%s: filter array must not be empty", path)
		}
		for i, node := range value {
			if err := validateFilterExpression(node, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("%s: filter must be an object or array", path)
	}
}

func validateFilterOperator(operator string, node interface{}, path string) error {
	if logicalFilterOperators[operator] {
		items, ok := node.([]interface{})
		if !ok || len(items) == 0 {
			return fmt.Errorf("%s: %s expects a non-empty array of expressions", path, operator)
		}
		for i, item := range items {
			if err := validateFilterExpression(item, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
		return nil
	}

	if operator == "$not" {
		return validateFilterExpression(node, path)
	}

	spec, ok := filterOperators[operator]
	if !ok {
		return fmt.Errorf("%s: unsupported filter operator %q", path, operator)
	}
	args, ok := node.([]interface{})
	if !ok {
		return fmt.Errorf("%s: %s expects an array", path, operator)
	}
	if len(args) < spec.MinArgs || len(args) > spec.MaxArgs {
		return fmt.Errorf("%s: %s expects %d argument(s), got %d", path, operator, spec.MinArgs, len(args))
	}
	if spec.FieldFirst {
		field, ok := args[0].(string)
		if !ok || strings.TrimSpace(field) == "" {
			return fmt.Errorf("%s: first argument must be a field name", path)
		}
	}
	if spec.SecondArgArray {
		if _, ok := args[1].([]interface{}); !ok {
			return fmt.Errorf("%s: second argument must be an array", path)
		}
	}
	return nil
}

func collectFilterFields(expr interface{}) []string {
	fields := []string{}
	collectFilterFieldsWalk(expr, &fields)
	return fields
}

func collectFilterFieldsWalk(expr interface{}, fields *[]string) {
	switch value := expr.(type) {
	case map[string]interface{}:
		for key, node := range value {
			if strings.HasPrefix(key, "$") {
				collectFilterFieldsFromOperator(key, node, fields)
				continue
			}
			if strings.TrimSpace(key) != "" {
				*fields = append(*fields, key)
			}
		}
	case []interface{}:
		for _, node := range value {
			collectFilterFieldsWalk(node, fields)
		}
	}
}

func filterValidateFailure(module, filterJSON string, schemaChecked bool, validationErr error) map[string]interface{} {
	result := map[string]interface{}{
		"valid":          false,
		"schema_checked": schemaChecked,
		"message":        validationErr.Error(),
	}
	if module != "" {
		result["module"] = module
	}
	if parsed, err := parseFilterJSONValue(filterJSON); err == nil {
		result["filter"] = parsed
	}
	return result
}

func filterValidateSuccess(module, filterJSON string, schemaChecked bool) map[string]interface{} {
	result := map[string]interface{}{
		"valid":          true,
		"schema_checked": schemaChecked,
	}
	if module != "" {
		result["module"] = module
		result["message"] = fmt.Sprintf("filter OK for module %s", module)
	} else {
		result["message"] = "filter OK"
	}
	if parsed, err := parseFilterJSONValue(filterJSON); err == nil {
		result["filter"] = parsed
	}
	return result
}

func parseFilterJSONValue(input string) (interface{}, error) {
	var filter interface{}
	decoder := json.NewDecoder(strings.NewReader(input))
	decoder.UseNumber()
	if err := decoder.Decode(&filter); err != nil {
		return nil, err
	}
	return filter, nil
}

func outputFilterValidateResult(format string, data map[string]interface{}) error {
	if err := outputFilterValidate(format, data); err != nil {
		return err
	}
	if valid, ok := data["valid"].(bool); ok && !valid {
		message := "filter validation failed"
		if msg, ok := data["message"].(string); ok && msg != "" {
			message = msg
		}
		return &output.ReportedError{Err: fmt.Errorf("%s", message)}
	}
	return nil
}

func outputFilterValidate(format string, data map[string]interface{}) error {
	switch format {
	case "table":
		keys := []string{"valid", "schema_checked", "module", "message"}
		for _, key := range keys {
			if value, ok := data[key]; ok && value != nil && value != "" {
				fmt.Printf("%-20s | %v\n", key, value)
			}
		}
		return nil
	case "csv":
		columns := []string{"valid", "schema_checked", "module", "message"}
		writer := csv.NewWriter(os.Stdout)
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
	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		encoder.SetIndent(2)
		return encoder.Encode(data)
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(data)
	case "jsonl":
		return json.NewEncoder(os.Stdout).Encode(data)
	default:
		return output.ItemResponse(&api.SingleResponse{Data: data}, output.Options{Format: format, Full: false})
	}
}

func collectFilterFieldsFromOperator(operator string, node interface{}, fields *[]string) {
	if logicalFilterOperators[operator] {
		items, ok := node.([]interface{})
		if !ok {
			return
		}
		for _, item := range items {
			collectFilterFieldsWalk(item, fields)
		}
		return
	}

	if operator == "$not" {
		collectFilterFieldsWalk(node, fields)
		return
	}

	spec, ok := filterOperators[operator]
	if !ok || !spec.FieldFirst {
		return
	}

	args, ok := node.([]interface{})
	if !ok || len(args) == 0 {
		return
	}
	if field, ok := args[0].(string); ok && strings.TrimSpace(field) != "" {
		*fields = append(*fields, field)
	}
}
