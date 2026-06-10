package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"crmservice/internal/config"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

func getURLFromFlagOrEnv(cmd *cobra.Command) (string, error) {
	url := ""
	if cmd.Flags().Lookup("url") != nil {
		var err error
		url, err = cmd.Flags().GetString("url")
		if err != nil {
			return "", err
		}
	}
	if url == "" {
		root := cmd.Root()
		if root != nil && root.PersistentFlags().Lookup("url") != nil {
			var err error
			url, err = root.PersistentFlags().GetString("url")
			if err != nil {
				return "", err
			}
		}
	}
	if url == "" {
		url = os.Getenv("CRMSERVICE_API_URL")
	}
	if url == "" && cfg != nil {
		url = cfg.API.URL
	}
	return config.ResolveAPIURL(url, true)
}

func getTokenFromFlagEnvConfig(cmd *cobra.Command) (string, error) {
	token := ""
	if cmd.Flags().Lookup("token") != nil {
		var err error
		token, err = cmd.Flags().GetString("token")
		if err != nil {
			return "", err
		}
	}
	if token == "" {
		token = os.Getenv("CRMSERVICE_AUTH_TOKEN")
	}
	if token == "" && cfg != nil {
		token = cfg.Auth.Token
	}
	return token, nil
}

func getRequiredTokenFromFlagEnvConfig(cmd *cobra.Command) (string, error) {
	token, err := getTokenFromFlagEnvConfig(cmd)
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", fmt.Errorf("API token not provided. Set CRMSERVICE_AUTH_TOKEN environment variable, config auth.token, or use --token flag")
	}
	return token, nil
}

func getOutputFormatFromFlagConfig(cmd *cobra.Command) (string, error) {
	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return "", err
	}
	if !cmd.Flags().Changed("output") && cfg != nil && cfg.Output.Format != "" {
		outputFormat = cfg.Output.Format
	}
	if err := output.ValidateOutputFormat(outputFormat); err != nil {
		return "", err
	}
	output.SetActiveFormat(outputFormat)
	return outputFormat, nil
}

func getPageSizeFromFlagConfig(cmd *cobra.Command) (int, error) {
	pageSize, err := cmd.Flags().GetInt("page-size")
	if err != nil {
		return 0, err
	}
	if !cmd.Flags().Changed("page-size") && cfg != nil && cfg.Output.PageSize > 0 {
		pageSize = cfg.Output.PageSize
	}
	return pageSize, nil
}

func getTimeoutFromConfig() time.Duration {
	if cfg != nil && cfg.API.Timeout > 0 {
		return time.Duration(cfg.API.Timeout) * time.Second
	}
	return 30 * time.Second
}

func splitCommaSeparated(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return items
}