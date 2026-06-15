package cmd

import (
	"errors"
	"fmt"
	"os"

	"crmservice/internal/config"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

var (
	configFile     string
	cfg            *config.Config
	commandStarted bool
)

var rootCmd = &cobra.Command{
	Use:           "crmservice",
	Short:         "CRM-service CLI API client",
	Long:          `CRM-service CLI - A command-line tool for interacting with the CRM-service REST API.`,
	Version:       versionString(),
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		commandStarted = true
		if err := initConfig(cmd); err != nil {
			return err
		}
		setActiveOutputFormat(cmd)
		return nil
	},
}

func Execute() error {
	commandStarted = false
	executedCmd, err := rootCmd.ExecuteC()
	if err != nil {
		if !commandStarted {
			emitUsageError(executedCmd, err)
			return err
		}
		if executedCmd != nil {
			setActiveOutputFormat(executedCmd)
		}
		var reported *output.ReportedError
		if !errors.As(err, &reported) {
			output.EmitError(err)
		}
		return err
	}
	return nil
}

func emitUsageError(cmd *cobra.Command, err error) {
	if cmd == nil {
		cmd = rootCmd
	}
	fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
	if usageErr := cmd.Usage(); usageErr != nil {
		fmt.Fprintf(os.Stderr, "failed to print usage: %v\n", usageErr)
	}
}

func initConfig(cmd *cobra.Command) error {
	if err := LoadConfig(); err != nil {
		return err
	}

	flags := cmd.Root().PersistentFlags()

	if flags.Changed("url") {
		url, err := flags.GetString("url")
		if err != nil {
			return err
		}
		cfg.API.URL = url
	}

	if flags.Changed("token") {
		token, err := flags.GetString("token")
		if err != nil {
			return err
		}
		cfg.Auth.Token = token
	}

	if flags.Changed("timeout") {
		timeout, err := flags.GetInt("timeout")
		if err != nil {
			return err
		}
		cfg.API.Timeout = timeout
	}

	if flags.Changed("output") {
		outputFormat, err := flags.GetString("output")
		if err != nil {
			return err
		}
		cfg.Output.Format = outputFormat
	}

	if flags.Changed("page-size") {
		pageSize, err := flags.GetInt("page-size")
		if err != nil {
			return err
		}
		cfg.Output.PageSize = pageSize
	}

	if flags.Changed("cache-dir") {
		cacheDir, err := flags.GetString("cache-dir")
		if err != nil {
			return err
		}
		cfg.Cache.SchemaDir = cacheDir
	}

	if flags.Changed("cache-ttl-seconds") {
		ttlSeconds, err := flags.GetInt("cache-ttl-seconds")
		if err != nil {
			return err
		}
		cfg.Cache.TTLSeconds = ttlSeconds
	}

	if flags.Changed("cache-auto-refresh") {
		autoRefresh, err := flags.GetBool("cache-auto-refresh")
		if err != nil {
			return err
		}
		cfg.Cache.AutoRefresh = autoRefresh
	}

	config.ExpandPaths(cfg)

	return nil
}

func init() {
	rootCmd.SetVersionTemplate("{{.Name}} {{.Version}}\n")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path")
	rootCmd.PersistentFlags().String("url", "", "API base URL (overrides config)")
	rootCmd.PersistentFlags().String("token", "", "Bearer token (overrides config)")
	rootCmd.PersistentFlags().Int("timeout", 0, "Request timeout in seconds (overrides config)")
	rootCmd.PersistentFlags().StringP("output", "o", "", "Default output format: table, json, yaml, jsonl, or csv (overrides config)")
	rootCmd.PersistentFlags().Int("page-size", 0, "Default page size (overrides config)")
	rootCmd.PersistentFlags().String("cache-dir", "", "Schema cache directory (overrides config)")
	rootCmd.PersistentFlags().Int("cache-ttl-seconds", 0, "Schema cache TTL in seconds (overrides config)")
	rootCmd.PersistentFlags().Bool("cache-auto-refresh", true, "Refresh schema cache automatically when missing or expired")
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(getCmd())
	rootCmd.AddCommand(bulkCreateCmd())
	rootCmd.AddCommand(bulkUpdateCmd())
	rootCmd.AddCommand(createCmd())
	rootCmd.AddCommand(updateCmd())
	rootCmd.AddCommand(deleteCmd())
	rootCmd.AddCommand(uploadCmd())
	rootCmd.AddCommand(fieldsCmd())
	rootCmd.AddCommand(filterCmd())
	rootCmd.AddCommand(searchCmd())
	rootCmd.AddCommand(countCmd())
	rootCmd.AddCommand(skillCmd())
	rootCmd.AddCommand(modulesCmd())
	rootCmd.AddCommand(whoamiCmd())
	rootCmd.AddCommand(doctorCmd())
}

func LoadConfig() error {
	var err error
	cfg, err = config.LoadConfig(configFile)
	return err
}

func GetConfig() *config.Config {
	return cfg
}
