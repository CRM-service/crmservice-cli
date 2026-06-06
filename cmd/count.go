package cmd

import (
	"encoding/json"
	"fmt"

	"crmservice/internal/api"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

func countCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "count <module> [filter]",
		Short: "Count records matching a filter",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  runCountCommand,
	}
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.Flags().String("filter", "", "Filter in JSON format")
	cmd.Flags().String("include", "", "Comma-separated relation names required by the filter")
	cmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, jsonl, or csv")
	cmd.Flags().Int("verbose", 0, "Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)")

	return cmd
}

func runCountCommand(cmd *cobra.Command, args []string) error {
	module := args[0]

	outputFormat, err := getOutputFormatFromFlagConfig(cmd)
	if err != nil {
		return err
	}
	if !ValidOutputFormat(outputFormat) {
		return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, jsonl, csv", outputFormat)
	}

	filterJSON := ""
	if len(args) == 2 {
		filterJSON = args[1]
	}
	flagFilter, err := cmd.Flags().GetString("filter")
	if err != nil {
		return err
	}
	if filterJSON != "" && flagFilter != "" {
		return fmt.Errorf("cannot use both positional filter and --filter")
	}
	if filterJSON == "" {
		filterJSON = flagFilter
	}

	token, err := getRequiredTokenFromFlagEnvConfig(cmd)
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

	if filterJSON != "" {
		if err := validateModuleFilter(module, filterJSON, url, token, verbose); err != nil {
			return output.ErrorResponse(err)
		}
	}

	var filterObj map[string]interface{}
	if filterJSON != "" {
		if err := json.Unmarshal([]byte(filterJSON), &filterObj); err != nil {
			return fmt.Errorf("invalid filter format. Use JSON syntax: filter={$and:[{$eq:[\"field\",\"value\"]}]}. Error: %v", err)
		}
	}
	apiClient := api.NewClient(url, token)
	apiClient.Verbose = verbose
	apiClient.HTTPClient.Timeout = getTimeoutFromConfig()

	count, err := apiClient.Count(cmd.Context(), module, &api.CountOptions{
		Filter:  filterObj,
		Include: splitCommaSeparated(include),
	})
	if err != nil {
		return output.ErrorResponse(err)
	}

	return outputCountResult(module, count, filterObj, outputFormat)
}

func outputCountResult(module string, total int, filter map[string]interface{}, outputFormat string) error {
	result := map[string]interface{}{
		"module": module,
		"total":  total,
	}
	if len(filter) > 0 {
		result["filter"] = filter
	}

	if outputFormat == "table" {
		fmt.Printf("%-20s | %v\n", "module", module)
		fmt.Printf("%-20s | %v\n", "total", total)
		return nil
	}

	return output.ItemResponse(&api.SingleResponse{Data: result}, output.Options{
		Format: outputFormat,
		Full:   false,
	})
}
