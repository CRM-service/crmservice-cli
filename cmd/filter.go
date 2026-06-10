package cmd

import (
	"fmt"

	"crmservice/internal/filter"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

var filterValidateColumns = []string{"valid", "schema_checked", "module", "message"}

func filterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "filter",
		Short: "Show and validate filter language expressions",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "reference",
		Short: "Show filter language reference",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStdout(), filter.Reference)
		},
	})

	validateCmd := &cobra.Command{
		Use:           "validate [module] <filter-json>",
		Short:         "Validate filter JSON syntax and optionally check fields against module schema",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 || len(args) > 2 {
				return fmt.Errorf("requires 1 or 2 arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := getOutputFormatFromFlagConfig(cmd)
			if err != nil {
				return err
			}

			module := ""
			filterJSON := args[0]
			if len(args) == 2 {
				module = args[0]
				filterJSON = args[1]
			}

			if module == "" {
				if err := filter.ValidateFilterJSON(filterJSON); err != nil {
					return outputFilterValidateResult(outputFormat, filterValidateFailure("", filterJSON, false, err))
				}
				return outputFilterValidateResult(outputFormat, filterValidateSuccess("", filterJSON, false))
			}

			url, err := getURLFromFlagOrEnv(cmd)
			if err != nil {
				return err
			}
			token, err := getRequiredTokenFromFlagEnvConfig(cmd)
			if err != nil {
				return err
			}
			verbose, err := cmd.Flags().GetInt("verbose")
			if err != nil {
				return err
			}
			if err := validateFilterJSONAgainstModule(module, filterJSON, url, token, verbose); err != nil {
				return outputFilterValidateResult(outputFormat, filterValidateFailure(module, filterJSON, true, err))
			}
			return outputFilterValidateResult(outputFormat, filterValidateSuccess(module, filterJSON, true))
		},
	}
	addCommonFlags(validateCmd, CommonFlagSet{Output: true, Verbose: true})
	cmd.AddCommand(validateCmd)

	return cmd
}

func filterValidateFailure(module, filterJSON string, schemaChecked bool, validationErr error) map[string]interface{} {
	result := map[string]interface{}{
		"valid":          false,
		"schema_checked": schemaChecked,
		"message":        validationErr.Error(),
	}
	if module != "" {
		result["module"] = module
	}
	if parsed, err := filter.ParseFilterJSON(filterJSON); err == nil {
		result["filter"] = parsed
	}
	return result
}

func filterValidateSuccess(module, filterJSON string, schemaChecked bool) map[string]interface{} {
	result := map[string]interface{}{
		"valid":          true,
		"schema_checked": schemaChecked,
	}
	if module != "" {
		result["module"] = module
		result["message"] = fmt.Sprintf("filter OK for module %s", module)
	} else {
		result["message"] = "filter OK"
	}
	if parsed, err := filter.ParseFilterJSON(filterJSON); err == nil {
		result["filter"] = parsed
	}
	return result
}

func outputFilterValidateResult(format string, data map[string]interface{}) error {
	if err := outputFilterValidate(format, data); err != nil {
		return err
	}
	if valid, ok := data["valid"].(bool); ok && !valid {
		message := "filter validation failed"
		if msg, ok := data["message"].(string); ok && msg != "" {
			message = msg
		}
		return &output.ReportedError{Err: fmt.Errorf("%s", message)}
	}
	return nil
}

func outputFilterValidate(format string, data map[string]interface{}) error {
	return output.WriteStructured(format, data, output.StructuredOptions{
		Columns:   filterValidateColumns,
		OmitEmpty: true,
	})
}
