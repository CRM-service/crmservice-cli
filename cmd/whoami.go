package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"crmservice/internal/api"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var whoamiFields = []string{"crm_url", "id", "name", "email", "is_admin", "first_name", "last_name"}

func whoamiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show the authenticated CRM user",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := getOutputFormatFromFlagConfig(cmd)
			if err != nil {
				return err
			}
			if !ValidOutputFormat(outputFormat) {
				return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, jsonl, csv", outputFormat)
			}

			url, err := getOptionalURLFromFlagEnvConfig(cmd)
			if err != nil {
				return err
			}
			token, err := getTokenFromFlagEnvConfig(cmd)
			if err != nil {
				return err
			}

			if url == "" {
				if err := outputWhoami(outputFormat, unauthenticatedWhoami(url, "missing_url")); err != nil {
					return err
				}
				return &output.ReportedError{Err: fmt.Errorf("CRM URL not provided")}
			}
			if token == "" {
				if err := outputWhoami(outputFormat, unauthenticatedWhoami(url, "missing_token")); err != nil {
					return err
				}
				return &output.ReportedError{Err: fmt.Errorf("API token not provided")}
			}

			user, err := fetchCurrentUser(cmd.Context(), url, token)
			if err != nil {
				if isUnauthorizedError(err) {
					return outputWhoami(outputFormat, unauthenticatedWhoami(url, "invalid_token"))
				}
				return output.ErrorResponse(err)
			}

			return outputWhoami(outputFormat, user)
		},
	}

	cmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, jsonl, or csv")
	cmd.Flags().Int("verbose", 0, "Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)")

	return cmd
}

func getOptionalURLFromFlagEnvConfig(cmd *cobra.Command) (string, error) {
	url := ""
	if cmd.Flags().Lookup("url") != nil {
		var err error
		url, err = cmd.Flags().GetString("url")
		if err != nil {
			return "", err
		}
	}
	if url == "" {
		url = os.Getenv("CRMSERVICE_API_URL")
	}
	if url == "" && cfg != nil {
		url = cfg.API.URL
	}
	if url == "" {
		return "", nil
	}
	return normalizeAPIURL(url), nil
}

func fetchCurrentUser(ctx context.Context, url, token string) (map[string]interface{}, error) {
	client := api.NewClient(url, token)
	client.HTTPClient.Timeout = getTimeoutFromConfig()

	resp := &api.SingleResponse{}
	err := client.Do(ctx, http.MethodGet, "/user?fields[users]=id,name,email,is_admin,first_name,last_name", nil, resp)
	if err != nil {
		return nil, err
	}

	user := map[string]interface{}{
		"crm_url": strings.TrimSuffix(url, "/"),
	}

	if data, ok := resp.Data.(map[string]interface{}); ok {
		if id, ok := data["id"]; ok {
			user["id"] = id
		}
		if attrs, ok := data["attributes"].(map[string]interface{}); ok {
			for _, field := range whoamiFields[2:] {
				if value, ok := attrs[field]; ok {
					user[field] = value
				}
			}
		}
	}

	return user, nil
}

func unauthenticatedWhoami(url, reason string) map[string]interface{} {
	message := "Unauthenticated"
	switch reason {
	case "missing_url":
		message = "CRM URL not provided"
	case "missing_token":
		message = "API token not provided"
	case "invalid_token":
		message = "API token is invalid or unauthorized"
	}

	return map[string]interface{}{
		"crm_url":       strings.TrimSuffix(url, "/"),
		"authenticated": false,
		"error":         reason,
		"message":       message,
	}
}

func isUnauthorizedError(err error) bool {
	apiErr, ok := err.(*api.Error)
	return ok && (apiErr.Status == http.StatusUnauthorized || apiErr.Status == http.StatusForbidden)
}

func outputWhoami(format string, data map[string]interface{}) error {
	columns := whoamiFields
	if _, ok := data["error"]; ok {
		columns = []string{"crm_url", "authenticated", "error", "message"}
	}

	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(data)
	case "jsonl":
		return json.NewEncoder(os.Stdout).Encode(data)
	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		encoder.SetIndent(2)
		return encoder.Encode(data)
	case "csv":
		writer := csv.NewWriter(os.Stdout)
		if err := writer.Write(columns); err != nil {
			return err
		}
		if err := writer.Write(whoamiRow(data, columns)); err != nil {
			return err
		}
		writer.Flush()
		return writer.Error()
	case "table":
		for _, column := range columns {
			fmt.Printf("%-20s | %v\n", column, data[column])
		}
		return nil
	default:
		return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, jsonl, csv", format)
	}
}

func whoamiRow(data map[string]interface{}, columns []string) []string {
	row := make([]string, len(columns))
	for i, column := range columns {
		if value, ok := data[column]; ok && value != nil {
			row[i] = fmt.Sprintf("%v", value)
		}
	}
	return row
}
