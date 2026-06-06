package cmd

import (
	"github.com/spf13/cobra"

	"crmservice/internal/output"
)

func resolveOutputFormat(cmd *cobra.Command) string {
	for current := cmd; current != nil; current = current.Parent() {
		if current.Flags().Lookup("output") == nil {
			continue
		}
		format, err := getOutputFormatFromFlagConfig(current)
		if err == nil && format != "" {
			return format
		}
	}
	if cfg != nil && cfg.Output.Format != "" {
		return cfg.Output.Format
	}
	return "table"
}

func setActiveOutputFormat(cmd *cobra.Command) {
	output.SetActiveFormat(resolveOutputFormat(cmd))
}
