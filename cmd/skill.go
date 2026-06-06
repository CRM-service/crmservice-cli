package cmd

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"crmservice/internal/output"
	"crmservice/skills"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

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
		Short: "Print the bundled crmservice Agent Skill",
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
				if !ValidOutputFormat(outputFormat) {
					return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, jsonl, csv", outputFormat)
				}
				result, err := checkInstalledSkill(path, skills.CRMServiceSkill)
				if err != nil {
					return err
				}
				return outputSkillCheckResult(outputFormat, result)
			}

			if err := installSkill(path, skills.CRMServiceSkill); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Installed crmservice skill to %s\n", path)
			return nil
		},
	}

	cmd.Flags().Bool("check", false, "Check whether the installed skill matches the bundled skill")
	cmd.Flags().StringP("output", "o", "table", "Output format for --check: table, json, yaml, jsonl, or csv")
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	return cmd
}

func defaultSkillInstallPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return skillInstallPath(home), nil
}

func skillInstallPath(home string) string {
	return filepath.Join(home, ".agents", "skills", "crmservice", "SKILL.md")
}

func installSkill(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// #nosec G306 -- Skill content is non-secret documentation intended to be readable.
	return os.WriteFile(path, []byte(content), 0o644)
}

func skillContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func checkInstalledSkill(path, bundled string) (map[string]interface{}, error) {
	bundledHash := skillContentHash(bundled)
	result := map[string]interface{}{
		"path":           path,
		"bundled_hash":   bundledHash,
		"installed":      false,
		"up_to_date":     false,
		"installed_hash": "",
	}

	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		result["status"] = "missing"
		result["message"] = "skill not installed; run crmservice skill install"
		return result, nil
	}
	if err != nil {
		return nil, err
	}

	installedHash := skillContentHash(string(content))
	result["installed"] = true
	result["installed_hash"] = installedHash

	if string(content) == bundled {
		result["status"] = "up_to_date"
		result["up_to_date"] = true
		result["message"] = "installed skill matches bundled skill"
		return result, nil
	}

	result["status"] = "stale"
	result["message"] = "installed skill differs from bundled skill; run crmservice skill install"
	return result, nil
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
	switch format {
	case "table":
		keys := []string{"status", "up_to_date", "installed", "path", "bundled_hash", "installed_hash", "message"}
		for _, key := range keys {
			if value, ok := data[key]; ok && value != nil && value != "" {
				fmt.Printf("%-20s | %v\n", key, value)
			}
		}
		return nil
	case "csv":
		columns := []string{"status", "up_to_date", "installed", "path", "bundled_hash", "installed_hash", "message"}
		writer := csv.NewWriter(os.Stdout)
		if err := writer.Write(columns); err != nil {
			return err
		}
		row := make([]string, len(columns))
		for i, column := range columns {
			if value, ok := data[column]; ok && value != nil {
				row[i] = fmt.Sprintf("%v", value)
			}
		}
		if err := writer.Write(row); err != nil {
			return err
		}
		writer.Flush()
		return writer.Error()
	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		encoder.SetIndent(2)
		return encoder.Encode(data)
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(data)
	case "jsonl":
		return json.NewEncoder(os.Stdout).Encode(data)
	default:
		return fmt.Errorf("invalid output format: %s. Valid formats: table, json, yaml, jsonl, csv", format)
	}
}
