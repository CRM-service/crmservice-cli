package api

// BuildWriteBody constructs a JSON:API write request body from flat attributes or a raw data object.
func BuildWriteBody(module, id, operation string, record map[string]interface{}) (map[string]interface{}, string, error) {
	if data, ok := record["data"].(map[string]interface{}); ok {
		recordID, hasID := normalizeWriteRecordID(data["id"])
		switch operation {
		case "create":
			delete(data, "id")
		case "update":
			if !hasID {
				return nil, "", ErrBulkUpdateMissingID
			}
			data["id"] = recordID
		}
		return map[string]interface{}{"data": data}, recordID, nil
	}

	attrs := make(map[string]interface{}, len(record))
	var recordID string
	for key, value := range record {
		if key == "id" {
			if normalized, ok := normalizeWriteRecordID(value); ok {
				recordID = normalized
			}
			continue
		}
		attrs[key] = value
	}
	if len(attrs) == 0 {
		return nil, recordID, ErrEmptyRecord
	}
	if operation == "update" && recordID == "" {
		return nil, "", ErrBulkUpdateMissingID
	}

	data := map[string]interface{}{
		"type":       module,
		"attributes": attrs,
	}
	if operation == "update" {
		data["id"] = recordID
	}
	return map[string]interface{}{"data": data}, recordID, nil
}

// BuildSingleWriteBody constructs a JSON:API request body for create or update from flat attributes.
func BuildSingleWriteBody(module, id, operation string, attrs map[string]interface{}) map[string]interface{} {
	data := map[string]interface{}{
		"type":       module,
		"attributes": attrs,
	}
	if operation == "update" {
		data["id"] = id
	}
	return map[string]interface{}{"data": data}
}