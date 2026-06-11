package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
)

const (
	reportingCountSelect = "$count.distinct(id) as count_value"
	reportingCountField  = "count_value"
)

type ReportingResponse struct {
	Meta ReportingMeta            `json:"meta"`
	Data []map[string]interface{} `json:"data"`
}

type ReportingMeta struct {
	Results int      `json:"results"`
	Fields  []string `json:"fields"`
}

type CountOptions struct {
	Filter  map[string]interface{}
	Include []string
}

func (c *Client) Count(ctx context.Context, module string, opts *CountOptions) (int, error) {
	if opts == nil {
		opts = &CountOptions{}
	}

	body := map[string]interface{}{
		"from":   module,
		"select": reportingCountSelect,
		"format": "json",
		"where":  reportingWhere(opts.Filter),
		"options": map[string]interface{}{
			"kpi": true,
		},
	}
	if len(opts.Include) > 0 {
		body["include"] = strings.Join(opts.Include, ",")
	}

	var resp ReportingResponse
	if err := c.doReporting(ctx, body, &resp); err != nil {
		return 0, err
	}

	return parseReportingCount(&resp)
}

func reportingWhere(filter map[string]interface{}) interface{} {
	if len(filter) == 0 {
		return []interface{}{}
	}
	return filter
}

func (c *Client) doReporting(ctx context.Context, body interface{}, result interface{}) error {
	if c.Verbose >= 2 {
		payload, err := json.Marshal(body)
		if err == nil {
			fmt.Fprintf(os.Stderr, "[REQUEST BODY]\n%s\n", string(payload))
		}
	}
	return c.PostJSON(ctx, "/reporting", body, result)
}

func parseReportingCount(resp *ReportingResponse) (int, error) {
	if resp == nil {
		return 0, fmt.Errorf("reporting response is nil")
	}
	if len(resp.Data) == 0 {
		if resp.Meta.Results > 0 {
			return 0, fmt.Errorf("reporting response missing data rows")
		}
		return 0, nil
	}

	value, ok := resp.Data[0][reportingCountField]
	if !ok {
		return 0, fmt.Errorf("reporting response missing %q field", reportingCountField)
	}

	return reportingValueToInt(value)
}

func reportingValueToInt(value interface{}) (int, error) {
	switch typed := value.(type) {
	case int:
		return typed, nil
	case int8:
		return int(typed), nil
	case int16:
		return int(typed), nil
	case int32:
		return int(typed), nil
	case int64:
		if typed > int64(math.MaxInt) || typed < int64(math.MinInt) {
			return 0, fmt.Errorf("count value out of range: %d", typed)
		}
		return int(typed), nil
	case uint64:
		if typed > math.MaxInt {
			return 0, fmt.Errorf("count value out of range: %d", typed)
		}
		return int(typed), nil
	case uint:
		if uint64(typed) > math.MaxInt {
			return 0, fmt.Errorf("count value out of range: %d", typed)
		}
		return int(typed), nil //nolint:gosec // bounded above
	case uint32:
		return int(typed), nil
	case uint16:
		return int(typed), nil
	case uint8:
		return int(typed), nil
	case float32:
		if math.Trunc(float64(typed)) != float64(typed) {
			return 0, fmt.Errorf("count value is not an integer: %v", typed)
		}
		return int(typed), nil
	case float64:
		if math.Trunc(typed) != typed {
			return 0, fmt.Errorf("count value is not an integer: %v", typed)
		}
		return int(typed), nil
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, err
		}
		if parsed > int64(math.MaxInt) || parsed < int64(math.MinInt) {
			return 0, fmt.Errorf("count value out of range: %d", parsed)
		}
		return int(parsed), nil
	case string:
		var parsed int64
		if _, err := fmt.Sscan(typed, &parsed); err != nil {
			return 0, err
		}
		if parsed > int64(math.MaxInt) || parsed < int64(math.MinInt) {
			return 0, fmt.Errorf("count value out of range: %d", parsed)
		}
		return int(parsed), nil
	default:
		return 0, fmt.Errorf("unsupported count value type %T", value)
	}
}
