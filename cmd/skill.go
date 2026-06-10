package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"crmservice/internal/output"
	"crmservice/skills"

	"github.com/spf13/cobra"
)

var skillCheckColumns = []string{
	"status", "up_to_date", "installed", "files_checked",
	"path", "bundled_hash", "installed_hash", "message",
}

func skillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Install, inspect, and print the bundled Agent Skill",
	}

	cmd.AddCommand(skillInstallCmd())

	cmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print the default Agent Skill install path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := defaultSkillInstallPath()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "print",
		Short: "Print the bundled crmservice Agent Skill entry point",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStdout(), skills.CRMServiceSkill)
		},
	})

	return cmd
}

func skillInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the bundled crmservice Agent Skill",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := defaultSkillInstallPath()
			if err != nil {
				return err
			}

			check, err := cmd.Flags().GetBool("check")
			if err != nil {
				return err
			}
			if check {
				outputFormat, err := getOutputFormatFromFlagConfig(cmd)
				if err != nil {
					return err
				}
				result, err := skills.CheckInstalledSkillBundle(path)
				if err != nil {
					return err
				}
				return outputSkillCheckResult(outputFormat, skillCheckResultToMap(result))
			}

			destDir := filepath.Dir(path)
			if err := skills.InstallSkillBundle(destDir); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Installed crmservice skill to %s\n", destDir)
			return nil
		},
	}

	cmd.Flags().Bool("check", false, "Check whether the installed skill matches the bundled skill")
	cmd.Flags().StringP("output", "o", "table", "Output format for --check: table, json, yaml, jsonl, or csv")
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	return cmd
}

var userHomeDir = os.UserHomeDir

func defaultSkillInstallPath() (string, error) {
	home, err := userHomeDir()
	if err != nil {
		return "", err
	}
	return skillInstallPath(home), nil
}

func skillInstallPath(home string) string {
	return filepath.Join(home, ".agents", "skills", "crmservice", "SKILL.md")
}

func skillCheckResultToMap(result skills.SkillCheckResult) map[string]interface{} {
	data := map[string]interface{}{
		"path":           result.Path,
		"bundled_hash":   result.BundledHash,
		"installed":      result.Installed,
		"up_to_date":     result.UpToDate,
		"installed_hash": result.InstalledHash,
		"files_checked":  result.FilesChecked,
		"status":         result.Status,
		"message":        result.Message,
	}
	return data
}

func outputSkillCheckResult(format string, data map[string]interface{}) error {
	if err := outputSkillCheck(format, data); err != nil {
		return err
	}
	if upToDate, ok := data["up_to_date"].(bool); ok && !upToDate {
		message := "installed skill is not up to date"
		if msg, ok := data["message"].(string); ok && msg != "" {
			message = msg
		}
		return &output.ReportedError{Err: fmt.Errorf("%s", message)}
	}
	return nil
}

func outputSkillCheck(format string, data map[string]interface{}) error {
	return output.WriteStructured(format, data, output.StructuredOptions{
		Columns:   skillCheckColumns,
		OmitEmpty: true,
	})
}
