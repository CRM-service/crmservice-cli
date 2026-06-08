package cmd

import (
	"net/http"

	"crmservice/internal/api"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

func modulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "modules",
		Short: "List API modules",
		RunE: func(cmd *cobra.Command, args []string) error {
			url, err := getURLFromFlagOrEnv(cmd)
			if err != nil {
				return err
			}
			token, err := getRequiredTokenFromFlagEnvConfig(cmd)
			if err != nil {
				return err
			}
			full, err := cmd.Flags().GetBool("full")
			if err != nil {
				return err
			}
			verbose, err := cmd.Flags().GetInt("verbose")
			if err != nil {
				return err
			}
			outputFormat, err := getOutputFormatFromFlagConfig(cmd)
			if err != nil {
				return err
			}

			client := api.NewClient(url, token)
			client.Verbose = verbose
			client.HTTPClient.Timeout = getTimeoutFromConfig()

			type rawResponse struct {
				Meta  interface{}            `json:"meta"`
				Links map[string]interface{} `json:"links"`
			}

			var raw rawResponse
			if err := client.Do(cmd.Context(), http.MethodGet, "/", nil, &raw); err != nil {
				return output.ErrorResponse(err)
			}

			data := modulesDataFromLinks(raw.Links, full)

			respObj := &api.Response{
				Data:  data,
				Meta:  nil,
				Links: nil,
			}

			return output.ListResponse(respObj, output.Options{
				Format: outputFormat,
				Full:   full,
			})
		},
	}

	cmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, jsonl, or csv")
	cmd.Flags().Bool("full", false, "Include full response (not just attributes)")
	cmd.Flags().Int("verbose", 0, "Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)")

	return cmd
}
