package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"crmservice/internal/api"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

type bulkRecord map[string]interface{}

type bulkOptions struct {
	ContinueOnError bool
	DryRun          bool
	Concurrency     int
	SkipEmpty       bool
	Summary         bool
	OutputFormat    string
	Full            bool
	Verbose         int
}

type bulkResult struct {
	Index  int
	ID     string
	Data   interface{}
	Body   map[string]interface{}
	Error  string
	Status string
}

type bulkSummary struct {
	Operation string `json:"operation" yaml:"operation"`
	Module    string `json:"module" yaml:"module"`
	Total     int    `json:"total" yaml:"total"`
	Succeeded int    `json:"succeeded" yaml:"succeeded"`
	Failed    int    `json:"failed" yaml:"failed"`
	Skipped   int    `json:"skipped" yaml:"skipped"`
	DryRun    bool   `json:"dry_run" yaml:"dry_run"`
}

func bulkCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bulk-create <module>",
		Short: "Create multiple records from JSONL or a JSON array",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBulkCommand(cmd, args[0], "create")
		},
	}
	addBulkFlags(cmd)
	return cmd
}

func bulkUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bulk-update <module>",
		Short: "Update multiple records from JSONL or a JSON array",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBulkCommand(cmd, args[0], "update")
		},
	}
	addBulkFlags(cmd)
	return cmd
}

func addBulkFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("continue-on-error", false, "Continue processing records after an error")
	cmd.Flags().Bool("dry-run", false, "Build requests without sending them to the API")
	cmd.Flags().Int("concurrency", 1, "Number of concurrent API requests")
	cmd.Flags().Bool("skip-empty", false, "Skip empty records instead of failing")
	cmd.Flags().Bool("summary", false, "Output only an operation summary")
	cmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, jsonl, or csv")
	cmd.Flags().Bool("full", false, "Include full response (not just attributes)")
	cmd.Flags().Int("verbose", 0, "Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)")
}

func runBulkCommand(cmd *cobra.Command, module, operation string) error {
	opts, err := getBulkOptions(cmd)
	if err != nil {
		return err
	}
	if !ValidOutputFormat(opts.OutputFormat) {
		return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, jsonl, csv", opts.OutputFormat)
	}

	records, err := readBulkRecords(os.Stdin)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return fmt.Errorf("no input records found; provide JSONL or a JSON array on stdin")
	}

	client := api.NewClient("", "")
	if !opts.DryRun {
		url := getURLFromFlagOrEnv(cmd)
		token, err := getRequiredTokenFromFlagEnvConfig(cmd)
		if err != nil {
			return err
		}
		client = api.NewClient(url, token)
		client.Verbose = opts.Verbose
		client.HTTPClient.Timeout = getTimeoutFromConfig()
	}

	results, summary, err := processBulkRecords(cmd.Context(), client, module, operation, records, opts)
	if err != nil && !opts.ContinueOnError {
		return err
	}

	if opts.Summary {
		return outputBulkSummary(summary, opts.OutputFormat)
	}
	if opts.DryRun {
		return outputBulkDryRun(results, opts.OutputFormat)
	}
	if len(results) == 0 || summary.Succeeded == 0 {
		if err != nil {
			return err
		}
		return outputBulkSummary(summary, opts.OutputFormat)
	}
	return outputBulkResults(results, opts)
}

func getBulkOptions(cmd *cobra.Command) (bulkOptions, error) {
	continueOnError, err := cmd.Flags().GetBool("continue-on-error")
	if err != nil {
		return bulkOptions{}, err
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return bulkOptions{}, err
	}
	concurrency, err := cmd.Flags().GetInt("concurrency")
	if err != nil {
		return bulkOptions{}, err
	}
	if concurrency < 1 {
		return bulkOptions{}, fmt.Errorf("--concurrency must be at least 1")
	}
	skipEmpty, err := cmd.Flags().GetBool("skip-empty")
	if err != nil {
		return bulkOptions{}, err
	}
	summary, err := cmd.Flags().GetBool("summary")
	if err != nil {
		return bulkOptions{}, err
	}
	outputFormat, err := getOutputFormatFromFlagConfig(cmd)
	if err != nil {
		return bulkOptions{}, err
	}
	full, err := cmd.Flags().GetBool("full")
	if err != nil {
		return bulkOptions{}, err
	}
	verbose, err := cmd.Flags().GetInt("verbose")
	if err != nil {
		return bulkOptions{}, err
	}
	return bulkOptions{ContinueOnError: continueOnError, DryRun: dryRun, Concurrency: concurrency, SkipEmpty: skipEmpty, Summary: summary, OutputFormat: outputFormat, Full: full, Verbose: verbose}, nil
}

func readBulkRecords(stdin *os.File) ([]bulkRecord, error) {
	info, err := stdin.Stat()
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return nil, nil
	}

	body, err := io.ReadAll(stdin)
	if err != nil {
		return nil, err
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, nil
	}

	if records, err := parseBulkJSONArray(body); err == nil {
		return records, nil
	}
	if record, err := parseBulkJSONObject(body); err == nil {
		return []bulkRecord{record}, nil
	}
	return parseBulkJSONL(body)
}

func parseBulkJSONArray(body []byte) ([]bulkRecord, error) {
	var raw []map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	if err := ensureNoExtraJSON(decoder); err != nil {
		return nil, err
	}
	records := make([]bulkRecord, 0, len(raw))
	for _, item := range raw {
		records = append(records, bulkRecord(item))
	}
	return records, nil
}

func parseBulkJSONObject(body []byte) (bulkRecord, error) {
	var raw map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	if err := ensureNoExtraJSON(decoder); err != nil {
		return nil, err
	}
	return bulkRecord(raw), nil
}

func ensureNoExtraJSON(decoder *json.Decoder) error {
	var extra interface{}
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("multiple JSON values")
}

func parseBulkJSONL(body []byte) ([]bulkRecord, error) {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	records := []bulkRecord{}
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record map[string]interface{}
		decoder := json.NewDecoder(strings.NewReader(line))
		decoder.UseNumber()
		if err := decoder.Decode(&record); err != nil {
			return nil, fmt.Errorf("invalid JSONL on line %d: %w", lineNo, err)
		}
		records = append(records, bulkRecord(record))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func processBulkRecords(ctx context.Context, client *api.Client, module, operation string, records []bulkRecord, opts bulkOptions) ([]bulkResult, bulkSummary, error) {
	summary := bulkSummary{Operation: operation, Module: module, Total: len(records), DryRun: opts.DryRun}
	results := make([]bulkResult, len(records))

	if opts.Concurrency == 1 || !opts.ContinueOnError {
		var firstErr error
		for i, record := range records {
			result, err := processBulkRecord(ctx, client, module, operation, i, record, opts)
			results[i] = result
			applyBulkResultToSummary(&summary, result)
			if err != nil {
				firstErr = err
				if !opts.ContinueOnError {
					return compactBulkResults(results), summary, err
				}
			}
		}
		return compactBulkResults(results), summary, firstErr
	}

	jobs := make(chan int)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for worker := 0; worker < opts.Concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				result, err := processBulkRecord(ctx, client, module, operation, i, records[i], opts)
				mu.Lock()
				results[i] = result
				applyBulkResultToSummary(&summary, result)
				if err != nil && firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}()
	}
	for i := range records {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	return compactBulkResults(results), summary, firstErr
}

func processBulkRecord(ctx context.Context, client *api.Client, module, operation string, index int, record bulkRecord, opts bulkOptions) (bulkResult, error) {
	result := bulkResult{Index: index, Status: "pending"}
	body, id, err := bulkRequestBody(module, operation, record)
	result.ID = id
	result.Body = body
	if err != nil {
		if opts.SkipEmpty && isEmptyRecordError(err) {
			result.Status = "skipped"
			return result, nil
		}
		result.Status = "failed"
		result.Error = err.Error()
		return result, fmt.Errorf("record %d: %w", index+1, err)
	}

	if opts.DryRun {
		result.Status = "dry-run"
		result.Data = body
		return result, nil
	}

	var resp api.SingleResponse
	method := http.MethodPost
	path := "/" + module
	if operation == "update" {
		method = http.MethodPatch
		path = fmt.Sprintf("/%s/%s", module, id)
	}
	if err := client.Do(ctx, method, path, body, &resp); err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, fmt.Errorf("record %d: %w", index+1, err)
	}
	result.Status = "succeeded"
	result.Data = resp.Data
	return result, nil
}

func bulkRequestBody(module, operation string, record bulkRecord) (map[string]interface{}, string, error) {
	if data, ok := record["data"].(map[string]interface{}); ok {
		id, _ := data["id"].(string)
		if operation == "create" {
			delete(data, "id")
		} else if id == "" {
			return nil, "", fmt.Errorf("bulk-update requires id in each record")
		}
		return map[string]interface{}{"data": data}, id, nil
	}

	attrs := make(map[string]interface{}, len(record))
	var id string
	for key, value := range record {
		if key == "id" {
			if value != nil {
				id = fmt.Sprintf("%v", value)
			}
			continue
		}
		attrs[key] = value
	}
	if len(attrs) == 0 {
		return nil, id, fmt.Errorf("empty record")
	}
	if operation == "update" && id == "" {
		return nil, "", fmt.Errorf("bulk-update requires id in each record")
	}

	data := map[string]interface{}{
		"type":       module,
		"attributes": attrs,
	}
	if operation == "update" {
		data["id"] = id
	}
	return map[string]interface{}{"data": data}, id, nil
}

func isEmptyRecordError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "empty record")
}

func applyBulkResultToSummary(summary *bulkSummary, result bulkResult) {
	switch result.Status {
	case "succeeded", "dry-run":
		summary.Succeeded++
	case "failed":
		summary.Failed++
	case "skipped":
		summary.Skipped++
	}
}

func compactBulkResults(results []bulkResult) []bulkResult {
	compact := make([]bulkResult, 0, len(results))
	for _, result := range results {
		if result.Status != "" {
			compact = append(compact, result)
		}
	}
	return compact
}

func outputBulkResults(results []bulkResult, opts bulkOptions) error {
	items := make([]interface{}, 0, len(results))
	for _, result := range results {
		if result.Status == "succeeded" {
			items = append(items, result.Data)
		}
	}
	return output.ListResponse(&api.Response{Data: items}, output.Options{Format: opts.OutputFormat, Full: opts.Full})
}

func outputBulkDryRun(results []bulkResult, format string) error {
	items := make([]interface{}, 0, len(results))
	for _, result := range results {
		if result.Status == "dry-run" {
			items = append(items, result.Body)
		}
	}
	return output.ListResponse(&api.Response{Data: items}, output.Options{Format: format, Full: true})
}

func outputBulkSummary(summary bulkSummary, format string) error {
	data := map[string]interface{}{
		"operation": summary.Operation,
		"module":    summary.Module,
		"total":     summary.Total,
		"succeeded": summary.Succeeded,
		"failed":    summary.Failed,
		"skipped":   summary.Skipped,
		"dry_run":   summary.DryRun,
	}
	return output.ItemResponse(&api.SingleResponse{Data: data}, output.Options{Format: format, Full: true})
}
