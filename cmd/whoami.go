package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"crmservice/internal/api"
	"crmservice/internal/config"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
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
					if err := outputWhoami(outputFormat, unauthenticatedWhoami(url, "invalid_token")); err != nil {
						return err
					}
					return &output.ReportedError{Err: fmt.Errorf("API token is invalid or unauthorized")}
				}
				return output.ErrorResponse(err)
			}

			return outputWhoami(outputFormat, user)
		},
	}

	addCommonFlags(cmd, CommonFlagSet{Output: true})

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
	return config.ResolveAPIURL(url, false)
}

func fetchCurrentUser(ctx context.Context, url, token string) (map[string]interface{}, error) {
	client := newAPIClient(url, token, 0)

	resp := &api.SingleResponse{}
	err := client.Do(ctx, http.MethodGet, "/user?fields[users]=id,name,email,is_admin,first_name,last_name", nil, resp)
	if err != nil {
		return nil, err
	}

	user := map[string]interface{}{
		"crm_url": whoamiCRMURL(url),
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
		"crm_url":       whoamiCRMURL(url),
		"authenticated": false,
		"error":         reason,
		"message":       message,
	}
}

func whoamiCRMURL(url string) interface{} {
	if url == "" {
		return nil
	}
	return config.DisplayAPIURL(url)
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
	return output.WriteStructured(format, data, output.StructuredOptions{Columns: columns})
}
