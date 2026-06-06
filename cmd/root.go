package cmd

import (
	"fmt"
	"os"

	"crmservice/internal/config"

	"github.com/spf13/cobra"
)

var (
	configFile string
	cfg        *config.Config
)

var rootCmd = &cobra.Command{
	Use:               "crmservice",
	Short:             "CRM-service CLI API client",
	Long:              `CRM-service CLI - A command-line tool for interacting with the CRM-service REST API.`,
	Version:           versionString(),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return initConfig(cmd) },
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return nil
}

func initConfig(cmd *cobra.Command) error {
	if err := LoadConfig(); err != nil {
		return err
	}

	url, err := cmd.Flags().GetString("url")
	if err != nil {
		return err
	}
	if url != "" {
		cfg.API.URL = url
	}

	token, err := cmd.Flags().GetString("token")
	if err != nil {
		return err
	}
	if token != "" {
		cfg.Auth.Token = token
	}

	return nil
}

func init() {
	rootCmd.SetVersionTemplate("{{.Name}} {{.Version}}\n")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path")
	rootCmd.PersistentFlags().String("url", "", "API base URL (overrides config)")
	rootCmd.PersistentFlags().String("token", "", "Bearer token (overrides config)")
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(getCmd())
	rootCmd.AddCommand(bulkCreateCmd())
	rootCmd.AddCommand(bulkUpdateCmd())
	rootCmd.AddCommand(createCmd())
	rootCmd.AddCommand(updateCmd())
	rootCmd.AddCommand(deleteCmd())
	rootCmd.AddCommand(fieldsCmd())
	rootCmd.AddCommand(filterCmd())
	rootCmd.AddCommand(searchCmd())
	rootCmd.AddCommand(skillCmd())
	rootCmd.AddCommand(modulesCmd())
	rootCmd.AddCommand(whoamiCmd())
}

func LoadConfig() error {
	var err error
	cfg, err = config.LoadConfig(configFile)
	return err
}

func GetConfig() *config.Config {
	return cfg
}
