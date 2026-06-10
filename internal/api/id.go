package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// NormalizeRecordID coerces record IDs from JSON values into stable string form.
func NormalizeRecordID(value interface{}) (string, bool) {
	if value == nil {
		return "", false
	}

	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return "", false
		}
		return v, true
	case json.Number:
		s := strings.TrimSpace(v.String())
		if s == "" {
			return "", false
		}
		return s, true
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10), true
		}
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case int:
		return strconv.Itoa(v), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case uint64:
		return strconv.FormatUint(v, 10), true
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", value))
		if s == "" {
			return "", false
		}
		return s, true
	}
}

func normalizeWriteRecordID(value interface{}) (string, bool) {
	return NormalizeRecordID(value)
}