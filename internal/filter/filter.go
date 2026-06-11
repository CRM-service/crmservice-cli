package filter

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

var filterOperators = map[string]operatorSpec{
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

type operatorSpec struct {
	MinArgs        int
	MaxArgs        int
	FieldFirst     bool
	SecondArgArray bool
}

const Reference = `CRM-service filter language

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

const filterFormatHint = "invalid filter format. Use JSON syntax: filter={$and:[{$eq:[\"field\",\"value\"]}]}."

func filterFormatDecodeError(err error) error {
	return fmt.Errorf("%s Error: %v", filterFormatHint, err)
}

func filterFormatDetailError(detail string) error {
	return fmt.Errorf("%s Error: %s", filterFormatHint, detail)
}

func decodeFilterJSON(input string) (interface{}, error) {
	var filter interface{}
	decoder := json.NewDecoder(strings.NewReader(input))
	decoder.UseNumber()
	if err := decoder.Decode(&filter); err != nil {
		return nil, filterFormatDecodeError(err)
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err == nil {
		return nil, filterFormatDetailError("multiple JSON values")
	} else if err != io.EOF {
		return nil, filterFormatDecodeError(err)
	}
	return filter, nil
}

func ValidateFilterJSON(input string) error {
	_, err := ParsedFilterJSON(input)
	return err
}

func ParsedFilterJSON(input string) (interface{}, error) {
	filter, err := decodeFilterJSON(input)
	if err != nil {
		return nil, err
	}
	if err := validateFilterExpression(filter, "$"); err != nil {
		return nil, err
	}
	return filter, nil
}

func ParseFilterJSON(input string) (interface{}, error) {
	return decodeFilterJSON(input)
}

func ParseToMap(filterJSON string) (map[string]interface{}, error) {
	parsed, err := ParseFilterJSON(filterJSON)
	if err != nil {
		return nil, err
	}
	filterObj, ok := parsed.(map[string]interface{})
	if !ok {
		return nil, filterFormatDetailError("filter must be a JSON object")
	}
	return filterObj, nil
}

func CollectFilterFields(expr interface{}) []string {
	fields := []string{}
	collectFilterFieldsWalk(expr, &fields)
	return fields
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
