package cmd

type schemaAttributeData struct {
	Type           string
	Size           float64
	Scale          float64
	Nullable       bool
	DefaultValue   interface{}
	RelationModule string
	Label          string
}

func fieldListFromSchemaAttributes(attrsObj map[string]interface{}, primaryKey []string) []map[string]interface{} {
	if attrsObj == nil {
		return nil
	}

	schemaAttrs := make([]map[string]interface{}, 0, len(attrsObj))
	for _, name := range sortedMapKeys(attrsObj) {
		attrData, ok := attrsObj[name].(map[string]interface{})
		if !ok {
			continue
		}

		parsed := schemaAttributeData{}
		if typeVal, ok := attrData["type"].(string); ok {
			parsed.Type = typeVal
		}
		if sizeVal, ok := attrData["size"].(float64); ok {
			parsed.Size = sizeVal
		}
		if scaleVal, ok := attrData["scale"].(float64); ok {
			parsed.Scale = scaleVal
		}
		if nullableVal, ok := attrData["nullable"].(bool); ok {
			parsed.Nullable = nullableVal
		}
		if defaultValueVal, ok := attrData["defaultValue"]; ok {
			parsed.DefaultValue = defaultValueVal
		}
		if relMod, ok := attrData["relationModule"].(string); ok {
			parsed.RelationModule = relMod
		}
		if label, ok := attrData["label"].(string); ok {
			parsed.Label = label
		}

		attr := map[string]interface{}{
			"name":         name,
			"type":         parsed.Type,
			"size":         parsed.Size,
			"scale":        parsed.Scale,
			"nullable":     parsed.Nullable,
			"defaultValue": parsed.DefaultValue,
			"label":        parsed.Label,
		}
		if parsed.RelationModule != "" {
			attr["relationModule"] = parsed.RelationModule
		}
		schemaAttrs = append(schemaAttrs, attr)
	}

	fieldList := make([]map[string]interface{}, 0, len(schemaAttrs))
	for _, attr := range schemaAttrs {
		field := map[string]interface{}{
			"name": attr["name"],
			"type": attr["type"],
		}
		if size, ok := attr["size"].(float64); ok && size > 0 {
			field["size"] = size
		}
		if scale, ok := attr["scale"].(float64); ok && scale > 0 {
			field["scale"] = scale
		}
		if nullable, ok := attr["nullable"].(bool); ok {
			field["nullable"] = nullable
		}
		if defaultValue, ok := attr["defaultValue"]; ok && defaultValue != nil {
			field["defaultValue"] = defaultValue
		}
		if label, ok := attr["label"].(string); ok && label != "" {
			field["label"] = label
		}
		if relMod, ok := attr["relationModule"].(string); ok && relMod != "" {
			field["extras"] = relMod
		}
		fieldList = append(fieldList, field)
	}

	pkSet := make(map[string]bool, len(primaryKey))
	for _, pk := range primaryKey {
		pkSet[pk] = true
	}
	for _, f := range fieldList {
		if name, ok := f["name"].(string); ok && pkSet[name] {
			f["primary"] = true
		}
	}

	return fieldList
}
