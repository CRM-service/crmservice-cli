package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"crmservice/internal/api"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

func createCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <module>",
		Short: "Create a new record",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWriteCommand(cmd, args[0], "", "create")
		},
	}

	cmd.Flags().StringArray("field", []string{}, "Field values to set")
	cmd.Flags().Bool("dry-run", false, "Build the request body without sending it to the API")
	addCommonFlags(cmd, CommonFlagSet{Output: true, Verbose: true, Full: true})

	return cmd
}

func updateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <module> <id>",
		Short: "Update a record",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWriteCommand(cmd, args[0], args[1], "update")
		},
	}

	cmd.Flags().StringArray("field", []string{}, "Field values to update")
	cmd.Flags().Bool("dry-run", false, "Build the request body without sending it to the API")
	addCommonFlags(cmd, CommonFlagSet{Output: true, Verbose: true, Full: true})

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
			common, err := readCommonFlags(cmd)
			if err != nil {
				return err
			}

			url, err := getURLFromFlagOrEnv(cmd)
			if err != nil {
				return err
			}

			apiClient := newAPIClient(url, token, common.Verbose)

			if err := apiClient.Delete(cmd.Context(), module, id); err != nil {
				return output.ErrorResponse(err)
			}

			return outputDeleteResult(module, id, common.OutputFormat)
		},
	}

	addCommonFlags(cmd, CommonFlagSet{Output: true, Verbose: true})

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
			common, err := readCommonFlags(cmd)
			if err != nil {
				return err
			}
			force, err := cmd.Flags().GetBool("force")
			if err != nil {
				return err
			}

			url, err := getURLFromFlagOrEnv(cmd)
			if err != nil {
				return err
			}

			body, err := getSchemaBody(module, url, token, common.Verbose, force)
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

			if common.OutputFormat == "table" && len(primaryKey) > 0 {
				fmt.Printf("Primary Key: %v\n", primaryKey)
			}

			data := interface{}(fieldList)
			if common.Full && common.OutputFormat != "table" {
				data = schemaResponseWithSortedAttributes(rawResp)
			}

			return output.ListResponse(&api.Response{
				Data: data,
			}, output.Options{
				Format:  common.OutputFormat,
				Columns: []string{"name", "type", "size", "scale", "nullable", "defaultValue", "label"},
				Full:    common.Full,
			})
		},
	}

	addCommonFlags(cmd, CommonFlagSet{Output: true, Verbose: true, Full: true})
	cmd.Flags().Bool("force", false, "Force refresh schema cache")

	return cmd
}

func runWriteCommand(cmd *cobra.Command, module, id, operation string) error {
	common, err := readCommonFlags(cmd)
	if err != nil {
		return err
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}

	input, err := getBodyInput(cmd, id, operation)
	if err != nil {
		return err
	}
	if bodyInputHasEmptyAttributes(input) {
		return fmt.Errorf("no fields provided; use --field or pass JSON via stdin")
	}

	url, err := getURLFromFlagOrEnv(cmd)
	if err != nil {
		return err
	}
	token, err := getTokenFromFlagEnvConfig(cmd)
	if err != nil {
		return err
	}
	if token != "" {
		if err := validateBodyInputAgainstModule(module, input, url, token, common.Verbose); err != nil {
			return output.ErrorResponse(err)
		}
	}

	if dryRun {
		body, err := singleRecordRequestBody(module, id, operation, input)
		if err != nil {
			return err
		}
		return outputDryRunRequest(operation, module, id, body, common.OutputFormat)
	}

	token, err = getRequiredTokenFromFlagEnvConfig(cmd)
	if err != nil {
		return err
	}

	apiClient := newAPIClient(url, token, common.Verbose)

	var resp *api.SingleResponse
	if input.raw {
		resp = &api.SingleResponse{}
		method := http.MethodPost
		path := "/" + module
		if operation == "update" {
			method = http.MethodPatch
			path = fmt.Sprintf("/%s/%s", module, id)
		}
		err = apiClient.Do(cmd.Context(), method, path, input.body, resp)
	} else if operation == "create" {
		resp, err = apiClient.Create(cmd.Context(), module, input.body)
	} else {
		resp, err = apiClient.Update(cmd.Context(), module, id, input.body)
	}
	if err != nil {
		return output.ErrorResponse(err)
	}

	return output.ItemResponse(resp, output.Options{
		Format: common.OutputFormat,
		Full:   common.Full,
	})
}