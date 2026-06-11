package cmd

import "crmservice/internal/schema"

func modulesDataFromLinks(links map[string]interface{}, full bool) interface{} {
	if links == nil {
		return nil
	}

	if full {
		keys := schema.SortedMapKeys(links, "self", "meta")
		linksAsSlice := make([]map[string]interface{}, 0, len(keys))
		for _, key := range keys {
			value := links[key]
			link, ok := value.(map[string]interface{})
			if !ok {
				continue
			}
			linksAsSlice = append(linksAsSlice, map[string]interface{}{
				"id":   key,
				"type": "modules",
				"attributes": map[string]interface{}{
					"name": key,
					"href": link["href"],
					"type": link["type"],
				},
			})
		}
		return linksAsSlice
	}

	keys := schema.SortedMapKeys(links, "self")
	moduleNames := make([]map[string]interface{}, 0, len(keys))
	for _, key := range keys {
		moduleNames = append(moduleNames, map[string]interface{}{
			"id":   key,
			"type": "modules",
			"attributes": map[string]interface{}{
				"name": key,
			},
		})
	}
	return moduleNames
}
