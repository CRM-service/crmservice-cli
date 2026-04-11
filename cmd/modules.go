package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"crmservice/internal/api"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

func modulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "modules",
		Short: "List API modules",
		RunE: func(cmd *cobra.Command, args []string) error {
			url := getURLFromFlagOrEnv(cmd)
			full, err := cmd.Flags().GetBool("full")
			if err != nil {
				return err
			}
			verbose, err := cmd.Flags().GetInt("verbose")
			if err != nil {
				return err
			}
			outputFormat, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}

			client := &api.Client{
				BaseURL:    strings.TrimSuffix(url, "/"),
				HTTPClient: &http.Client{Timeout: 30 * time.Second},
				DefaultHeaders: map[string]string{
					"Content-Type": "application/vnd.api+json",
					"Accept":       "application/vnd.api+json",
				},
				Verbose: verbose,
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return err
			}

			req.Header.Set("Accept", "application/vnd.api+json")
			req.Header.Set("Content-Type", "application/vnd.api+json")

			resp, err := client.HTTPClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			type RawResponse struct {
				Meta  interface{}            `json:"meta"`
				Links map[string]interface{} `json:"links"`
			}

			var raw RawResponse
			if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
				return output.ErrorResponse(err)
			}

			var data interface{}
			if full && raw.Links != nil {
				// Keep original response - extract links into a slice for proper output
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
				// Extract module names from links
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

	cmd.Flags().String("url", "", "API base URL")
	cmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, or csv")
	cmd.Flags().Bool("full", false, "Include full response (not just attributes)")
	cmd.Flags().Int("verbose", 0, "Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)")

	return cmd
}
