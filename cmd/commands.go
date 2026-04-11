package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"crmservice/internal/api"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func ValidOutputFormat(format string) bool {
	switch format {
	case "table", "json", "yaml", "csv":
		return true
	default:
		return false
	}
}

func getAPIURL() string {
	v := viper.New()
	v.SetEnvPrefix("CRMSERVICE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	url := v.GetString("api.url")
	if url == "" {
		fmt.Fprintf(os.Stderr, "Error: API URL not provided. Set CRMSERVICE_API_URL environment variable or use --url flag\n")
		os.Exit(1)
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	if strings.HasPrefix(url, "http://") {
		url = "https://" + strings.TrimPrefix(url, "http://")
	}

	if !strings.HasSuffix(url, "/api/v1") {
		url = strings.TrimSuffix(url, "/") + "/api/v1"
	}

	return strings.TrimSuffix(url, "/")
}

func getURLFromFlagOrEnv(cmd *cobra.Command) string {
	url, err := cmd.Flags().GetString("url")
	if err != nil {
		return ""
	}
	if url == "" {
		url = os.Getenv("CRMSERVICE_API_URL")
	}
	if url == "" {
		url = getAPIURL()
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	if strings.HasPrefix(url, "http://") {
		url = "https://" + strings.TrimPrefix(url, "http://")
	}

	if !strings.HasSuffix(url, "/api/v1") {
		url = strings.TrimSuffix(url, "/") + "/api/v1"
	}

	return strings.TrimSuffix(url, "/")
}

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <module>",
		Short: "List records",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			module := args[0]
			token, err := cmd.Flags().GetString("token")
			if err != nil {
				return err
			}
			if token == "" {
				token = os.Getenv("CRMSERVICE_AUTH_TOKEN")
			}
			outputFormat, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, csv", outputFormat)
			}
			full, err := cmd.Flags().GetBool("full")
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, csv", outputFormat)
			}
			pageSize, err := cmd.Flags().GetInt("page-size")
			if err != nil {
				return err
			}
			page, err := cmd.Flags().GetInt("page")
			if err != nil {
				return err
			}
			offset, err := cmd.Flags().GetInt("offset")
			if err != nil {
				return err
			}
			fields, err := cmd.Flags().GetString("fields")
			if err != nil {
				return err
			}
			filter, err := cmd.Flags().GetString("filter")
			if err != nil {
				return err
			}
			include, err := cmd.Flags().GetString("include")
			if err != nil {
				return err
			}
			verbose, err := cmd.Flags().GetInt("verbose")
			if err != nil {
				return err
			}

			url := getURLFromFlagOrEnv(cmd)

			apiClient := api.NewClient(url, token)
			apiClient.Verbose = verbose

			opts := &api.ListOptions{
				PageSize: pageSize,
			}

			if page > 0 {
				opts.SetPage(page)
			}

			if offset > 0 {
				opts.SetOffset(offset)
			}

			if fields != "" {
				opts.SetFields(strings.Split(fields, ","))
			}

			if filter != "" {
				var filterObj map[string]interface{}
				if err := json.Unmarshal([]byte(filter), &filterObj); err == nil {
					opts.SetFilterObj(filterObj)
				} else {
					return fmt.Errorf("invalid filter format. Use JSON syntax: filter={$and:[{$eq:[\"field\",\"value\"]}]}. Error: %v", err)
				}
			}

			if include != "" {
				opts.AddInclude(include)
			}

			resp, err := apiClient.List(cmd.Context(), module, opts)
			if err != nil {
				return output.ErrorResponse(err)
			}

			var outputFields []string
			if fields != "" {
				outputFields = strings.Split(fields, ",")
			}

			return output.ListResponse(resp, output.Options{
				Format: outputFormat,
				Fields: outputFields,
				Full:   full,
			})
		},
	}

	cmd.Flags().Int("page-size", 20, "Items per page")
	cmd.Flags().String("include", "", "Comma-separated relation names to include")
	cmd.Flags().String("fields", "", "Comma-separated field names to include")
	cmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, or csv")
	cmd.Flags().String("filter", "", "Filter in JSON format")
	cmd.Flags().Bool("full", false, "Include full response (not just attributes)")
	cmd.Flags().Int("verbose", 0, "Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)")
	cmd.Flags().Int("page", 1, "Page number")
	cmd.Flags().Int("offset", 0, "Offset for pagination")

	return cmd
}

func getCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <module> <id>",
		Short: "Get record by ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			module := args[0]
			id := args[1]
			token, err := cmd.Flags().GetString("token")
			if err != nil {
				return err
			}
			if token == "" {
				token = os.Getenv("CRMSERVICE_AUTH_TOKEN")
			}
			outputFormat, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, csv", outputFormat)
			}
			full, err := cmd.Flags().GetBool("full")
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, csv", outputFormat)
			}
			verbose, err := cmd.Flags().GetInt("verbose")
			if err != nil {
				return err
			}

			url := getURLFromFlagOrEnv(cmd)

			apiClient := api.NewClient(url, token)
			apiClient.Verbose = verbose

			opts := &api.ListOptions{}

			if fields, err := cmd.Flags().GetString("fields"); err == nil && fields != "" {
				opts.SetFields(strings.Split(fields, ","))
			}

			resp, err := apiClient.Get(cmd.Context(), module, id, opts)
			if err != nil {
				return output.ErrorResponse(err)
			}

			var outputFields []string
			if fields, err := cmd.Flags().GetString("fields"); err == nil && fields != "" {
				outputFields = strings.Split(fields, ",")
			}

			return output.ItemResponse(resp, output.Options{
				Format: outputFormat,
				Fields: outputFields,
				Full:   full,
			})
		},
	}

	cmd.Flags().String("fields", "", "Comma-separated field names to include (from 'attributes' branch)")
	cmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, or csv")
	cmd.Flags().Bool("full", false, "Include full response (not just attributes)")
	cmd.Flags().Int("verbose", 0, "Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)")

	return cmd
}

func createCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <module>",
		Short: "Create a new record",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			module := args[0]
			token, err := cmd.Flags().GetString("token")
			if err != nil {
				return err
			}
			if token == "" {
				token = os.Getenv("CRMSERVICE_AUTH_TOKEN")
			}
			outputFormat, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, csv", outputFormat)
			}
			full, err := cmd.Flags().GetBool("full")
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, csv", outputFormat)
			}
			verbose, err := cmd.Flags().GetInt("verbose")
			if err != nil {
				return err
			}

			url := getURLFromFlagOrEnv(cmd)

			apiClient := api.NewClient(url, token)
			apiClient.Verbose = verbose

			data := make(map[string]interface{})
			if fields, err := cmd.Flags().GetStringArray("field"); err == nil && len(fields) > 0 {
				for _, f := range fields {
					parts := strings.SplitN(f, "=", 2)
					if len(parts) == 2 {
						data[parts[0]] = parts[1]
					}
				}
			}

			resp, err := apiClient.Create(cmd.Context(), module, data)
			if err != nil {
				return output.ErrorResponse(err)
			}

			return output.ItemResponse(resp, output.Options{
				Format: outputFormat,
				Full:   full,
			})
		},
	}

	cmd.Flags().StringArray("field", []string{}, "Field values to set")
	cmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, or csv")
	cmd.Flags().Bool("full", false, "Include full response (not just attributes)")
	cmd.Flags().Int("verbose", 0, "Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)")

	return cmd
}

func updateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <module> <id>",
		Short: "Update a record",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			module := args[0]
			id := args[1]
			token, err := cmd.Flags().GetString("token")
			if err != nil {
				return err
			}
			if token == "" {
				token = os.Getenv("CRMSERVICE_AUTH_TOKEN")
			}
			outputFormat, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, csv", outputFormat)
			}
			full, err := cmd.Flags().GetBool("full")
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, csv", outputFormat)
			}
			verbose, err := cmd.Flags().GetInt("verbose")
			if err != nil {
				return err
			}

			url := getURLFromFlagOrEnv(cmd)

			apiClient := api.NewClient(url, token)
			apiClient.Verbose = verbose

			data := make(map[string]interface{})
			if fields, err := cmd.Flags().GetStringArray("field"); err == nil && len(fields) > 0 {
				for _, f := range fields {
					parts := strings.SplitN(f, "=", 2)
					if len(parts) == 2 {
						data[parts[0]] = parts[1]
					}
				}
			}

			resp, err := apiClient.Update(cmd.Context(), module, id, data)
			if err != nil {
				return output.ErrorResponse(err)
			}

			return output.ItemResponse(resp, output.Options{
				Format: outputFormat,
				Full:   full,
			})
		},
	}

	cmd.Flags().StringArray("field", []string{}, "Field values to update")
	cmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, or csv")
	cmd.Flags().Bool("full", false, "Include full response (not just attributes)")
	cmd.Flags().Int("verbose", 0, "Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)")

	return cmd
}

func deleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <module> <id>",
		Short: "Delete a record",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			module := args[0]
			id := args[1]
			token, err := cmd.Flags().GetString("token")
			if err != nil {
				return err
			}
			if token == "" {
				token = os.Getenv("CRMSERVICE_AUTH_TOKEN")
			}
			verbose, err := cmd.Flags().GetInt("verbose")
			if err != nil {
				return err
			}

			url := getURLFromFlagOrEnv(cmd)

			apiClient := api.NewClient(url, token)
			apiClient.Verbose = verbose

			errDelete := apiClient.Delete(cmd.Context(), module, id)
			if errDelete != nil {
				return output.ErrorResponse(errDelete)
			}

			fmt.Printf("Successfully deleted %s %s\n", module, id)
			return nil
		},
	}

	cmd.Flags().Int("verbose", 0, "Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)")

	return cmd
}

func fieldsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fields <module>",
		Short: "Show available fields for a module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			module := args[0]
			token, err := cmd.Flags().GetString("token")
			if err != nil {
				return err
			}
			if token == "" {
				token = os.Getenv("CRMSERVICE_AUTH_TOKEN")
			}
			outputFormat, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, csv", outputFormat)
			}
			verbose, err := cmd.Flags().GetInt("verbose")
			if err != nil {
				return err
			}

			url := getURLFromFlagOrEnv(cmd)

			client := &http.Client{Timeout: 30 * time.Second}

			reqURL := strings.TrimSuffix(url, "/") + "/schema/" + module

			if verbose >= 1 {
				fmt.Printf("[REQUEST] GET %s\n", reqURL)
			}

			req, err := http.NewRequest("GET", reqURL, nil)
			if err != nil {
				return err
			}

			req.Header.Set("Accept", "application/vnd.api+json")
			req.Header.Set("Content-Type", "application/vnd.api+json")
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}

			resp, err := client.Do(req)
			if err != nil {
				return output.ErrorResponse(err)
			}
			defer resp.Body.Close()

			if verbose >= 1 {
				fmt.Printf("[RESPONSE] Status: %d\n", resp.StatusCode)
			}
			if verbose >= 2 {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return err
				}
				fmt.Printf("[RESPONSE BODY]\n%s\n", string(body))
				resp.Body = io.NopCloser(bytes.NewReader(body))
			}

			type RawAttributeData struct {
				Type           string      `json:"type"`
				Size           float64     `json:"size,omitempty"`
				Scale          float64     `json:"scale,omitempty"`
				Nullable       bool        `json:"nullable,omitempty"`
				DefaultValue   interface{} `json:"defaultValue,omitempty"`
				RelationModule string      `json:"relationModule,omitempty"`
				Label          string      `json:"label,omitempty"`
			}

			var rawResp map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
				return output.ErrorResponse(err)
			}

			var schemaAttrs []map[string]interface{}

			var primaryKey []string
			if pkRaw, ok := rawResp["primary-key"].([]interface{}); ok {
				for _, pk := range pkRaw {
					if pkStr, ok := pk.(string); ok {
						primaryKey = append(primaryKey, pkStr)
					}
				}
			}

			if attrsObj, ok := rawResp["attributes"].(map[string]interface{}); ok {
				for name, attrData := range attrsObj {
					if attrMap, ok := attrData.(map[string]interface{}); ok {
						attrData := RawAttributeData{}
						if typeVal, ok := attrMap["type"].(string); ok {
							attrData.Type = typeVal
						}
						if sizeVal, ok := attrMap["size"].(float64); ok {
							attrData.Size = sizeVal
						}
						if scaleVal, ok := attrMap["scale"].(float64); ok {
							attrData.Scale = scaleVal
						}
						if nullableVal, ok := attrMap["nullable"].(bool); ok {
							attrData.Nullable = nullableVal
						}
						if defaultValueVal, ok := attrMap["defaultValue"]; ok {
							attrData.DefaultValue = defaultValueVal
						}
						if relMod, ok := attrMap["relationModule"].(string); ok {
							attrData.RelationModule = relMod
						}
						if label, ok := attrMap["label"].(string); ok {
							attrData.Label = label
						}

						attr := map[string]interface{}{
							"name":         name,
							"type":         attrData.Type,
							"size":         attrData.Size,
							"scale":        attrData.Scale,
							"nullable":     attrData.Nullable,
							"defaultValue": attrData.DefaultValue,
							"label":        attrData.Label,
						}
						if attrData.RelationModule != "" {
							attr["relationModule"] = attrData.RelationModule
						}
						schemaAttrs = append(schemaAttrs, attr)
					}
				}
			}

			var fieldList []map[string]interface{}
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

			if len(primaryKey) > 0 {
				fmt.Printf("Primary Key: %v\n", primaryKey)
			}

			return output.ListResponse(&api.Response{
				Data:  fieldList,
				Meta:  nil,
				Links: nil,
			}, output.Options{
				Format:  outputFormat,
				Columns: []string{"name", "type", "size", "scale", "nullable", "defaultValue", "label"},
				Full:    false,
			})
		},
	}

	cmd.Flags().String("fields", "", "Comma-separated field names to include")
	cmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, or csv")
	cmd.Flags().Int("verbose", 0, "Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)")

	return cmd
}

func searchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <module> <query>",
		Short: "Search records",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			module := args[0]
			query := args[1]
			token, err := cmd.Flags().GetString("token")
			if err != nil {
				return err
			}
			if token == "" {
				token = os.Getenv("CRMSERVICE_AUTH_TOKEN")
			}
			outputFormat, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, csv", outputFormat)
			}
			full, err := cmd.Flags().GetBool("full")
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, csv", outputFormat)
			}
			verbose, err := cmd.Flags().GetInt("verbose")
			if err != nil {
				return err
			}

			url := getURLFromFlagOrEnv(cmd)

			apiClient := api.NewClient(url, token)
			apiClient.Verbose = verbose

			opts := &api.ListOptions{
				PageSize: 20,
			}
			opts.AddFilter("search", query)

			resp, err := apiClient.List(cmd.Context(), module, opts)
			if err != nil {
				return output.ErrorResponse(err)
			}

			return output.ListResponse(resp, output.Options{
				Format: outputFormat,
				Full:   full,
			})
		},
	}

	cmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, or csv")
	cmd.Flags().Bool("full", false, "Include full response (not just attributes)")
	cmd.Flags().Int("verbose", 0, "Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)")

	return cmd
}
