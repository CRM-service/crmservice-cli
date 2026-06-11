package cmd

import (
	"encoding/json"
	"fmt"

	"crmservice/internal/api"
	"crmservice/internal/filter"
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
	addCommonFlags(cmd, CommonFlagSet{Output: true, Verbose: true})

	return cmd
}

func runCountCommand(cmd *cobra.Command, args []string) error {
	module := args[0]

	outputFormat, err := getOutputFormatFromFlagConfig(cmd)
	if err != nil {
		return err
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

	url, err := getURLFromFlagOrEnv(cmd)
	if err != nil {
		return err
	}

	if filterJSON != "" {
		if err := validateModuleFilter(module, filterJSON, url, token, verbose); err != nil {
			return output.ErrorResponse(err)
		}
	}

	var filterObj map[string]interface{}
	if filterJSON != "" {
		var err error
		filterObj, err = filter.ParseToMap(filterJSON)
		if err != nil {
			return output.ErrorResponse(err)
		}
	}
	apiClient := newAPIClient(url, token, verbose)

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
		if len(filter) > 0 {
			filterJSON, err := json.Marshal(filter)
			if err != nil {
				return fmt.Errorf("failed to encode filter: %w", err)
			}
			fmt.Printf("%-20s | %s\n", "filter", filterJSON)
		}
		return nil
	}

	return output.ItemResponse(&api.SingleResponse{Data: result}, output.Options{
		Format: outputFormat,
		Full:   false,
	})
}
