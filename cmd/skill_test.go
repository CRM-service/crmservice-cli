package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"crmservice/skills"
)

func TestSkillInstallPath(t *testing.T) {
	home := filepath.Join(string(filepath.Separator), "home", "testuser")
	expected := filepath.Join(home, ".agents", "skills", "crmservice", "SKILL.md")
	if got := skillInstallPath(home); got != expected {
		t.Errorf("skillInstallPath() = %q, expected %q", got, expected)
	}
}

func TestSkillPrintReturnsNonEmptyContent(t *testing.T) {
	cmd := skillCmd()
	cmd.SetArgs([]string{"print"})
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Fatal("skill print returned empty content")
	}
	if !strings.Contains(out.String(), "# crmservice CLI Tool Skill") {
		t.Errorf("skill print did not include expected skill heading")
	}
}

func TestInstallSkillCreatesParentDirectoriesAndWritesSkill(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".agents", "skills", "crmservice", "SKILL.md")

	if err := installSkill(path, skills.CRMServiceSkill); err != nil {
		t.Fatalf("installSkill() returned error: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() returned error: %v", err)
	}
	if string(content) != skills.CRMServiceSkill {
		t.Error("installed skill content does not match embedded skill content")
	}
}

func TestSkillHelpListsSubcommands(t *testing.T) {
	cmd := skillCmd()
	cmd.SetArgs([]string{"--help"})
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	for _, name := range []string{"install", "path", "print"} {
		if !strings.Contains(out.String(), name) {
			t.Errorf("skill help missing %q; output: %s", name, out.String())
		}
	}
}

func TestRootCommandIncludesSkillCommand(t *testing.T) {
	for _, command := range rootCmd.Commands() {
		if command.Name() == "skill" {
			return
		}
	}
	t.Fatal("root command does not include skill command")
}
