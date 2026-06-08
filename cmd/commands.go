package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"crmservice/internal/api"
	"crmservice/internal/cache"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

func ValidOutputFormat(format string) bool {
	switch format {
	case "table", "json", "yaml", "jsonl", "csv":
		return true
	default:
		return false
	}
}

func normalizeAPIURL(url string) string {
	url = strings.TrimSuffix(url, "/")
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	if !strings.HasSuffix(url, "/api/v1") {
		url += "/api/v1"
	}

	return url
}

func getURLFromFlagOrEnv(cmd *cobra.Command) (string, error) {
	url := ""
	if cmd.Flags().Lookup("url") != nil {
		var err error
		url, err = cmd.Flags().GetString("url")
		if err != nil {
			return "", err
		}
	}
	if url == "" {
		root := cmd.Root()
		if root != nil && root.PersistentFlags().Lookup("url") != nil {
			var err error
			url, err = root.PersistentFlags().GetString("url")
			if err != nil {
				return "", err
			}
		}
	}
	if url == "" {
		url = os.Getenv("CRMSERVICE_API_URL")
	}
	if url == "" && cfg != nil {
		url = cfg.API.URL
	}
	if url == "" {
		return "", fmt.Errorf("API URL not provided. Set CRMSERVICE_API_URL environment variable, config api.url, or use --url flag")
	}

	return normalizeAPIURL(url), nil
}

func getTokenFromFlagEnvConfig(cmd *cobra.Command) (string, error) {
	token := ""
	if cmd.Flags().Lookup("token") != nil {
		var err error
		token, err = cmd.Flags().GetString("token")
		if err != nil {
			return "", err
		}
	}
	if token == "" {
		token = os.Getenv("CRMSERVICE_AUTH_TOKEN")
	}
	if token == "" && cfg != nil {
		token = cfg.Auth.Token
	}
	return token, nil
}

func getRequiredTokenFromFlagEnvConfig(cmd *cobra.Command) (string, error) {
	token, err := getTokenFromFlagEnvConfig(cmd)
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", fmt.Errorf("API token not provided. Set CRMSERVICE_AUTH_TOKEN environment variable, config auth.token, or use --token flag")
	}
	return token, nil
}

func getOutputFormatFromFlagConfig(cmd *cobra.Command) (string, error) {
	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return "", err
	}
	if !cmd.Flags().Changed("output") && cfg != nil && cfg.Output.Format != "" {
		outputFormat = cfg.Output.Format
	}
	output.SetActiveFormat(outputFormat)
	return outputFormat, nil
}

func getPageSizeFromFlagConfig(cmd *cobra.Command) (int, error) {
	pageSize, err := cmd.Flags().GetInt("page-size")
	if err != nil {
		return 0, err
	}
	if !cmd.Flags().Changed("page-size") && cfg != nil && cfg.Output.PageSize > 0 {
		pageSize = cfg.Output.PageSize
	}
	return pageSize, nil
}

func getTimeoutFromConfig() time.Duration {
	if cfg != nil && cfg.API.Timeout > 0 {
		return time.Duration(cfg.API.Timeout) * time.Second
	}
	return 30 * time.Second
}

func splitCommaSeparated(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return items
}

type bodyInput struct {
	body interface{}
	raw  bool
}

func getBodyInput(cmd *cobra.Command, id, operation string) (*bodyInput, error) {
	fields, err := cmd.Flags().GetStringArray("field")
	if err != nil {
		return nil, err
	}

	stdinBody, hasStdinBody, err := readStdinBodyInput(os.Stdin, id, operation)
	if err != nil {
		return nil, err
	}
	if hasStdinBody {
		if len(fields) > 0 {
			return nil, fmt.Errorf("cannot use --field with request body from stdin")
		}
		return stdinBody, nil
	}

	data := make(map[string]interface{})
	for _, f := range fields {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) == 2 {
			data[parts[0]] = parts[1]
		}
	}
	return &bodyInput{body: data}, nil
}

func readStdinBodyInput(stdin *os.File, id, operation string) (*bodyInput, bool, error) {
	info, err := stdin.Stat()
	if err != nil {
		return nil, false, err
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return nil, false, nil
	}

	body, err := io.ReadAll(stdin)
	if err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(string(body)) == "" {
		return nil, false, nil
	}

	var request map[string]interface{}
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, true, fmt.Errorf("invalid request body from stdin: %w", err)
	}

	if _, ok := request["data"]; ok {
		if err := validateJSONAPIRequestBody(request, id, operation); err != nil {
			return nil, true, err
		}
		return &bodyInput{body: request, raw: true}, true, nil
	}

	flatBody, err := flatRecordAttributes(request, id, operation)
	if err != nil {
		return nil, true, err
	}
	return &bodyInput{body: flatBody}, true, nil
}

func validateJSONAPIRequestBody(request map[string]interface{}, id, operation string) error {
	data, ok := request["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid JSON:API request body from stdin: data must be an object")
	}
	if operation == "create" {
		delete(data, "id")
	}
	if operation == "update" {
		if bodyID, ok := data["id"]; ok && fmt.Sprintf("%v", bodyID) != id {
			return fmt.Errorf("request body id %q does not match argument id %q", fmt.Sprintf("%v", bodyID), id)
		}
	}
	return nil
}

func flatRecordAttributes(record map[string]interface{}, id, operation string) (map[string]interface{}, error) {
	attrs := make(map[string]interface{}, len(record))
	for key, value := range record {
		if key == "id" {
			if operation == "update" && value != nil && fmt.Sprintf("%v", value) != id {
				return nil, fmt.Errorf("request body id %q does not match argument id %q", fmt.Sprintf("%v", value), id)
			}
			continue
		}
		attrs[key] = value
	}
	return attrs, nil
}

func bodyInputHasEmptyAttributes(input *bodyInput) bool {
	if input == nil {
		return true
	}
	if !input.raw {
		attrs, ok := input.body.(map[string]interface{})
		return !ok || len(attrs) == 0
	}
	body, ok := input.body.(map[string]interface{})
	if !ok {
		return true
	}
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		return true
	}
	attrs, ok := data["attributes"].(map[string]interface{})
	return ok && len(attrs) == 0
}

func readStdinJSONAPIRequest(stdin *os.File) (map[string]interface{}, bool, error) {
	input, ok, err := readStdinBodyInput(stdin, "", "create")
	if !ok || err != nil {
		return nil, ok, err
	}
	request, ok := input.body.(map[string]interface{})
	if !ok || !input.raw {
		return nil, true, fmt.Errorf("invalid JSON:API request body from stdin: missing data member")
	}
	return request, true, nil
}

func getSchemaBody(module, url, token string, verbose int, force bool) ([]byte, error) {
	schemaCache := cache.NewSchemaCache(getSchemaCacheDir(url))
	if cfg != nil {
		schemaCache.TTLDays = cfg.Cache.TTLDays
		schemaCache.AutoRefresh = cfg.Cache.AutoRefresh
	}

	if !force {
		cachedBody, err := schemaCache.Load(module)
		if err == nil {
			if verbose >= 1 {
				fmt.Fprintf(os.Stderr, "[CACHE] HIT %s\n", schemaCache.GetPath(module))
			}
			return cachedBody, nil
		}
		if !schemaCache.AutoRefresh {
			return nil, fmt.Errorf("schema cache unavailable for %s and auto refresh is disabled: %w", module, err)
		}
		if verbose >= 1 {
			fmt.Fprintf(os.Stderr, "[CACHE] MISS %s: %v\n", schemaCache.GetPath(module), err)
		}
	} else if verbose >= 1 {
		fmt.Fprintf(os.Stderr, "[CACHE] FORCE REFRESH %s\n", schemaCache.GetPath(module))
	}

	body, err := fetchSchemaBody(module, url, token, verbose)
	if err != nil {
		return nil, err
	}
	if err := schemaCache.Save(module, body); err != nil {
		return nil, fmt.Errorf("failed to save schema cache: %w", err)
	}
	return body, nil
}

func getSchemaCacheDir(url string) string {
	cacheDir := ""
	if cfg != nil {
		cacheDir = cfg.Cache.SchemaDir
	}
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "crmservice", "schema")
	}
	return filepath.Join(cacheDir, cacheKey(url))
}

func cacheKey(value string) string {
	value = strings.TrimSuffix(value, "/")
	if value == "" {
		return "default"
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func fetchSchemaBody(module, url, token string, verbose int) ([]byte, error) {
	client := &http.Client{Timeout: getTimeoutFromConfig()}
	reqURL := strings.TrimSuffix(url, "/") + "/schema/" + module

	if verbose >= 1 {
		fmt.Fprintf(os.Stderr, "[REQUEST] GET %s\n", reqURL)
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Content-Type", "application/vnd.api+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, output.ErrorResponse(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if verbose >= 1 {
		fmt.Fprintf(os.Stderr, "[RESPONSE] Status: %d\n", resp.StatusCode)
	}
	if verbose >= 2 {
		fmt.Fprintf(os.Stderr, "[RESPONSE BODY]\n%s\n", string(body))
	}

	if resp.StatusCode >= 400 {
		return nil, output.ErrorResponse(&api.Error{
			Status:  resp.StatusCode,
			Body:    body,
			Message: string(body),
		})
	}

	return body, nil
}

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <module>",
		Short: "List records",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListCommand(cmd, args, "")
		},
	}

	addListSearchFlags(cmd)
	cmd.Flags().String("filter", "", "Filter in JSON format")

	return cmd
}

func runListCommand(cmd *cobra.Command, args []string, filterOverride string) error {
	module := args[0]
	token, err := getRequiredTokenFromFlagEnvConfig(cmd)
	if err != nil {
		return err
	}
	outputFormat, err := getOutputFormatFromFlagConfig(cmd)
	if err != nil {
		return err
	}

	if !ValidOutputFormat(outputFormat) {
		return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, jsonl, csv", outputFormat)
	}
	full, err := cmd.Flags().GetBool("full")
	if err != nil {
		return err
	}

	all, maxResults, err := validateListAllFlags(cmd)
	if err != nil {
		return err
	}

	pageSize, err := pageSizeForList(cmd, all)
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
	sort, err := cmd.Flags().GetString("sort")
	if err != nil {
		return err
	}
	filter := filterOverride
	if filter == "" {
		filter, err = cmd.Flags().GetString("filter")
		if err != nil {
			return err
		}
	}
	include, err := cmd.Flags().GetString("include")
	if err != nil {
		return err
	}
	verbose, err := cmd.Flags().GetInt("verbose")
	if err != nil {
		return err
	}

	url, err := getURLFromFlagOrEnv(cmd)
	if err != nil {
		return err
	}

	if filter != "" {
		if err := validateModuleFilter(module, filter, url, token, verbose); err != nil {
			return output.ErrorResponse(err)
		}
	}

	apiClient := api.NewClient(url, token)
	apiClient.Verbose = verbose
	apiClient.HTTPClient.Timeout = getTimeoutFromConfig()

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
		opts.SetFields(splitCommaSeparated(fields))
	}

	if sort != "" {
		opts.SetSort(splitCommaSeparated(sort))
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
		for _, rel := range splitCommaSeparated(include) {
			opts.AddInclude(rel)
		}
	}

	var outputFields []string
	if fields != "" {
		outputFields = splitCommaSeparated(fields)
	}

	if all {
		return runListAll(cmd, apiClient, module, opts, pageSize, maxResults, verbose, full, outputFormat, outputFields)
	}

	resp, err := apiClient.List(cmd.Context(), module, opts)
	if err != nil {
		return output.ErrorResponse(err)
	}

	return output.ListResponse(resp, output.Options{
		Format: outputFormat,
		Fields: outputFields,
		Full:   full,
	})
}

func getCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <module> <id>",
		Short: "Get record by ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			module := args[0]
			id := args[1]
			token, err := getRequiredTokenFromFlagEnvConfig(cmd)
			if err != nil {
				return err
			}
			outputFormat, err := getOutputFormatFromFlagConfig(cmd)
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, jsonl, csv", outputFormat)
			}
			full, err := cmd.Flags().GetBool("full")
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, jsonl, csv", outputFormat)
			}
			verbose, err := cmd.Flags().GetInt("verbose")
			if err != nil {
				return err
			}

			url, err := getURLFromFlagOrEnv(cmd)
			if err != nil {
				return err
			}

			apiClient := api.NewClient(url, token)
			apiClient.Verbose = verbose
			apiClient.HTTPClient.Timeout = getTimeoutFromConfig()

			opts := &api.ListOptions{}

			if fields, err := cmd.Flags().GetString("fields"); err == nil && fields != "" {
				opts.SetFields(splitCommaSeparated(fields))
			}

			resp, err := apiClient.Get(cmd.Context(), module, id, opts)
			if err != nil {
				return output.ErrorResponse(err)
			}

			var outputFields []string
			if fields, err := cmd.Flags().GetString("fields"); err == nil && fields != "" {
				outputFields = splitCommaSeparated(fields)
			}

			return output.ItemResponse(resp, output.Options{
				Format: outputFormat,
				Fields: outputFields,
				Full:   full,
			})
		},
	}

	cmd.Flags().String("fields", "", "Comma-separated field names to include (from 'attributes' branch)")
	cmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, jsonl, or csv")
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
			outputFormat, err := getOutputFormatFromFlagConfig(cmd)
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, jsonl, csv", outputFormat)
			}
			full, err := cmd.Flags().GetBool("full")
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, jsonl, csv", outputFormat)
			}
			verbose, err := cmd.Flags().GetInt("verbose")
			if err != nil {
				return err
			}
			dryRun, err := cmd.Flags().GetBool("dry-run")
			if err != nil {
				return err
			}

			input, err := getBodyInput(cmd, "", "create")
			if err != nil {
				return err
			}
			if bodyInputHasEmptyAttributes(input) {
				return fmt.Errorf("no fields provided; use --field or pass JSON via stdin")
			}

			if dryRun {
				body, err := singleRecordRequestBody(module, "", "create", input)
				if err != nil {
					return err
				}
				return outputDryRunRequest("create", module, "", body, outputFormat)
			}

			token, err := getRequiredTokenFromFlagEnvConfig(cmd)
			if err != nil {
				return err
			}

			url, err := getURLFromFlagOrEnv(cmd)
			if err != nil {
				return err
			}

			apiClient := api.NewClient(url, token)
			apiClient.Verbose = verbose
			apiClient.HTTPClient.Timeout = getTimeoutFromConfig()

			var resp *api.SingleResponse
			if input.raw {
				resp = &api.SingleResponse{}
				err = apiClient.Do(cmd.Context(), http.MethodPost, "/"+module, input.body, resp)
			} else {
				resp, err = apiClient.Create(cmd.Context(), module, input.body)
			}
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
	cmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, jsonl, or csv")
	cmd.Flags().Bool("dry-run", false, "Build the request body without sending it to the API")
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
			outputFormat, err := getOutputFormatFromFlagConfig(cmd)
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, jsonl, csv", outputFormat)
			}
			full, err := cmd.Flags().GetBool("full")
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, jsonl, csv", outputFormat)
			}
			verbose, err := cmd.Flags().GetInt("verbose")
			if err != nil {
				return err
			}
			dryRun, err := cmd.Flags().GetBool("dry-run")
			if err != nil {
				return err
			}

			input, err := getBodyInput(cmd, id, "update")
			if err != nil {
				return err
			}
			if bodyInputHasEmptyAttributes(input) {
				return fmt.Errorf("no fields provided; use --field or pass JSON via stdin")
			}

			if dryRun {
				body, err := singleRecordRequestBody(module, id, "update", input)
				if err != nil {
					return err
				}
				return outputDryRunRequest("update", module, id, body, outputFormat)
			}

			token, err := getRequiredTokenFromFlagEnvConfig(cmd)
			if err != nil {
				return err
			}

			url, err := getURLFromFlagOrEnv(cmd)
			if err != nil {
				return err
			}

			apiClient := api.NewClient(url, token)
			apiClient.Verbose = verbose
			apiClient.HTTPClient.Timeout = getTimeoutFromConfig()

			var resp *api.SingleResponse
			if input.raw {
				resp = &api.SingleResponse{}
				err = apiClient.Do(cmd.Context(), http.MethodPatch, fmt.Sprintf("/%s/%s", module, id), input.body, resp)
			} else {
				resp, err = apiClient.Update(cmd.Context(), module, id, input.body)
			}
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
	cmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, jsonl, or csv")
	cmd.Flags().Bool("dry-run", false, "Build the request body without sending it to the API")
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
			token, err := getRequiredTokenFromFlagEnvConfig(cmd)
			if err != nil {
				return err
			}
			outputFormat, err := getOutputFormatFromFlagConfig(cmd)
			if err != nil {
				return err
			}
			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, jsonl, csv", outputFormat)
			}
			verbose, err := cmd.Flags().GetInt("verbose")
			if err != nil {
				return err
			}

			url, err := getURLFromFlagOrEnv(cmd)
			if err != nil {
				return err
			}

			apiClient := api.NewClient(url, token)
			apiClient.Verbose = verbose
			apiClient.HTTPClient.Timeout = getTimeoutFromConfig()

			errDelete := apiClient.Delete(cmd.Context(), module, id)
			if errDelete != nil {
				return output.ErrorResponse(errDelete)
			}

			return outputDeleteResult(module, id, outputFormat)
		},
	}

	cmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, jsonl, or csv")
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
			token, err := getRequiredTokenFromFlagEnvConfig(cmd)
			if err != nil {
				return err
			}
			outputFormat, err := getOutputFormatFromFlagConfig(cmd)
			if err != nil {
				return err
			}

			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, jsonl, csv", outputFormat)
			}
			verbose, err := cmd.Flags().GetInt("verbose")
			if err != nil {
				return err
			}
			force, err := cmd.Flags().GetBool("force")
			if err != nil {
				return err
			}
			full, err := cmd.Flags().GetBool("full")
			if err != nil {
				return err
			}

			url, err := getURLFromFlagOrEnv(cmd)
			if err != nil {
				return err
			}

			body, err := getSchemaBody(module, url, token, verbose, force)
			if err != nil {
				return err
			}

			var rawResp map[string]interface{}
			if err := json.Unmarshal(body, &rawResp); err != nil {
				return output.ErrorResponse(err)
			}

			var primaryKey []string
			if pkRaw, ok := rawResp["primary-key"].([]interface{}); ok {
				for _, pk := range pkRaw {
					if pkStr, ok := pk.(string); ok {
						primaryKey = append(primaryKey, pkStr)
					}
				}
			}

			var attrsObj map[string]interface{}
			if rawAttrs, ok := rawResp["attributes"].(map[string]interface{}); ok {
				attrsObj = rawAttrs
			}
			fieldList := fieldListFromSchemaAttributes(attrsObj, primaryKey)

			// Only print the human-readable "Primary Key: ..." line for table output.
			// For json/jsonl/yaml/csv (used heavily by agents and scripts) we keep
			// the output clean and include the info via the "primary" annotations above.
			if outputFormat == "table" && len(primaryKey) > 0 {
				fmt.Printf("Primary Key: %v\n", primaryKey)
			}

			// Choose what to emit as the list data.
			data := interface{}(fieldList)
			if full {
				// When --full is requested, return the original complete schema
				// document from the backend (contains "primary-key", all attribute
				// details the server knows about, etc.). This only affects
				// structured output formats; table still gets a nice view.
				if outputFormat != "table" {
					data = schemaResponseWithSortedAttributes(rawResp)
				}
			}

			return output.ListResponse(&api.Response{
				Data:  data,
				Meta:  nil,
				Links: nil,
			}, output.Options{
				Format:  outputFormat,
				Columns: []string{"name", "type", "size", "scale", "nullable", "defaultValue", "label"},
				Full:    full,
			})
		},
	}

	cmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, jsonl, or csv")
	cmd.Flags().Bool("full", false, "Include full schema response under the data key")
	cmd.Flags().Bool("force", false, "Force refresh schema cache")
	cmd.Flags().Int("verbose", 0, "Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)")

	return cmd
}

func searchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <module> <filter>",
		Short: "Search records (alias for list --filter)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			filter := args[1]
			return runListCommand(cmd, args[:1], filter)
		},
	}

	addListSearchFlags(cmd)

	return cmd
}
