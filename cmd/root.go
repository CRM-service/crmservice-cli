package cmd

import (
	"fmt"
	"os"
	"strings"

	"crmservice/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configFile string
	cfg        *config.Config
)

var rootCmd = &cobra.Command{
	Use:              "crmservice",
	Short:            "CRM-service CLI API client",
	Long:             `CRM-service CLI - A command-line tool for interacting with the CRM-service REST API.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) { initConfig(cmd) },
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

	v := viper.New()
	v.SetEnvPrefix("CRMSERVICE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if url, _ := cmd.Flags().GetString("url"); url != "" {
		cfg.API.URL = url
	}
	if token, _ := cmd.Flags().GetString("token"); token != "" {
		cfg.Auth.Token = token
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

func LoadConfig() error {
	var err error
	cfg, err = config.LoadConfig(configFile)
	return err
}

func GetConfig() *config.Config {
	return cfg
}
