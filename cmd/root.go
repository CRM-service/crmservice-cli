package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	configFile string
)

var rootCmd = &cobra.Command{
	Use:              "crmservice",
	Short:            "CRM-service CLI API client",
	Long:             `CRM-service CLI - A command-line tool for interacting with the CRM-service REST API.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {},
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path")
	rootCmd.PersistentFlags().String("url", "", "API base URL (overrides config)")
	rootCmd.PersistentFlags().String("token", "", "Bearer token (overrides config)")
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(getCmd())
	rootCmd.AddCommand(createCmd())
	rootCmd.AddCommand(updateCmd())
	rootCmd.AddCommand(deleteCmd())
	rootCmd.AddCommand(fieldsCmd())
	rootCmd.AddCommand(searchCmd())
	rootCmd.AddCommand(modulesCmd())
}
