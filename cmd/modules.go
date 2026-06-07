package cmd

import (
	"fmt"
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
			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, jsonl, csv", outputFormat)
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

			var data interface{}
			if full && raw.Links != nil {
				linksAsSlice := make([]map[string]interface{}, 0)
				for key, value := range raw.Links {
					if key != "self" && key != "meta" {
						if link, ok := value.(map[string]interface{}); ok {
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
					}
				}
				data = linksAsSlice
			} else if raw.Links != nil {
				moduleNames := make([]map[string]interface{}, 0)
				for key := range raw.Links {
					if key != "self" {
						moduleNames = append(moduleNames, map[string]interface{}{
							"id":   key,
							"type": "modules",
							"attributes": map[string]interface{}{
								"name": key,
							},
						})
					}
				}
				data = moduleNames
			}

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
