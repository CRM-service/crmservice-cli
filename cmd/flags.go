package cmd

import (
	"fmt"

	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

const (
	outputFlagHelp  = "Output format: table, json, yaml, jsonl, or csv"
	verboseFlagHelp = "Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)"
	fullFlagHelp    = "Include full response (not just attributes)"
)

// CommonFlagSet describes CLI flags shared across multiple commands.
type CommonFlagSet struct {
	Output  bool
	Verbose bool
	Full    bool
}

// CommonFlagValues holds parsed values for shared command flags.
type CommonFlagValues struct {
	OutputFormat string
	Verbose      int
	Full         bool
}

func addCommonFlags(cmd *cobra.Command, flags CommonFlagSet) {
	if flags.Output {
		cmd.Flags().StringP("output", "o", "table", outputFlagHelp)
	}
	if flags.Verbose {
		cmd.Flags().Int("verbose", 0, verboseFlagHelp)
	}
	if flags.Full {
		cmd.Flags().Bool("full", false, fullFlagHelp)
	}
}

func readCommonFlags(cmd *cobra.Command) (CommonFlagValues, error) {
	var values CommonFlagValues
	var err error

	if cmd.Flags().Lookup("output") != nil {
		values.OutputFormat, err = getOutputFormatFromFlagConfig(cmd)
		if err != nil {
			return CommonFlagValues{}, err
		}
	}

	if cmd.Flags().Lookup("verbose") != nil {
		values.Verbose, err = cmd.Flags().GetInt("verbose")
		if err != nil {
			return CommonFlagValues{}, err
		}
	}

	if cmd.Flags().Lookup("full") != nil {
		values.Full, err = cmd.Flags().GetBool("full")
		if err != nil {
			return CommonFlagValues{}, err
		}
	}

	return values, nil
}

func readVerboseFlag(cmd *cobra.Command) (int, error) {
	if cmd.Flags().Lookup("verbose") == nil {
		return 0, nil
	}
	return cmd.Flags().GetInt("verbose")
}

func validateOutputFormatFlag(format string) error {
	if format == "" {
		return fmt.Errorf("output format is required")
	}
	return output.ValidateOutputFormat(format)
}