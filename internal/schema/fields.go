package schema

func FieldListFromSchemaAttributes(attrsObj map[string]interface{}, primaryKey []string) []map[string]interface{} {
	if attrsObj == nil {
		return nil
	}

	pkSet := make(map[string]bool, len(primaryKey))
	for _, pk := range primaryKey {
		pkSet[pk] = true
	}

	fieldList := make([]map[string]interface{}, 0, len(attrsObj))
	for _, name := range SortedMapKeys(attrsObj) {
		attrData, ok := attrsObj[name].(map[string]interface{})
		if !ok {
			continue
		}

		field := map[string]interface{}{
			"name": name,
		}
		if typeVal, ok := attrData["type"].(string); ok {
			field["type"] = typeVal
		}
		if sizeVal, ok := attrData["size"].(float64); ok && sizeVal > 0 {
			field["size"] = sizeVal
		}
		if scaleVal, ok := attrData["scale"].(float64); ok && scaleVal > 0 {
			field["scale"] = scaleVal
		}
		if nullableVal, ok := attrData["nullable"].(bool); ok {
			field["nullable"] = nullableVal
		}
		if defaultValueVal, ok := attrData["defaultValue"]; ok && defaultValueVal != nil {
			field["defaultValue"] = defaultValueVal
		}
		if label, ok := attrData["label"].(string); ok && label != "" {
			field["label"] = label
		}
		if relMod, ok := attrData["relationModule"].(string); ok && relMod != "" {
			field["relationModule"] = relMod
		} else if relType, ok := attrData["relationType"].(string); ok && relType != "" {
			field["relationModule"] = relType
		}
		if pkSet[name] {
			field["primary"] = true
		}
		fieldList = append(fieldList, field)
	}

	return fieldList
}
