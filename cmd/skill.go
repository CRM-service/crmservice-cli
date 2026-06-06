package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"crmservice/skills"

	"github.com/spf13/cobra"
)

func skillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Install, inspect, and print the bundled Agent Skill",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Install the bundled crmservice Agent Skill",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := defaultSkillInstallPath()
			if err != nil {
				return err
			}
			if err := installSkill(path, skills.CRMServiceSkill); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Installed crmservice skill to %s\n", path)
			return nil
		},
	})

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
