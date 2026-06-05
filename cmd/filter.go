package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
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

Preferred forms:
  {"$eq":["account_type","Customer"]}
  {"$and":[{"$eq":["account_type","Customer"]},{"$cts":["name","Acme"]}]}

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

	cmd.AddCommand(&cobra.Command{
		Use:   "validate <filter-json>",
		Short: "Validate filter JSON syntax and common operator shapes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateFilterJSON(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "filter OK")
			return nil
		},
	})

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
