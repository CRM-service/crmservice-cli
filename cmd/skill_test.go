package cmd

import (
	"bytes"
	"encoding/json"
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
	if !strings.Contains(out.String(), "# crmservice CLI") {
		t.Errorf("skill print did not include expected skill heading")
	}
}

func TestInstallSkillBundleCreatesTree(t *testing.T) {
	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, ".agents", "skills", "crmservice")

	if err := skills.InstallSkillBundle(destDir); err != nil {
		t.Fatalf("InstallSkillBundle() returned error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(destDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("os.ReadFile() returned error: %v", err)
	}
	if string(content) != skills.CRMServiceSkill {
		t.Error("installed SKILL.md does not match embedded entry point")
	}

	if _, err := os.Stat(filepath.Join(destDir, "references", "filters.md")); err != nil {
		t.Fatalf("references/filters.md missing: %v", err)
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

func TestSkillInstallCheckUpToDate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := defaultSkillInstallPath()
	if err != nil {
		t.Fatalf("defaultSkillInstallPath() returned error: %v", err)
	}
	if err := skills.InstallSkillBundle(filepath.Dir(path)); err != nil {
		t.Fatalf("InstallSkillBundle() returned error: %v", err)
	}

	cmd := skillCmd()
	cmd.SetArgs([]string{"install", "--check", "-o", "json"})
	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() returned error: %v", err)
		}
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("json.Unmarshal() returned error: %v\noutput: %s", err, out)
	}
	if result["status"] != "up_to_date" {
		t.Errorf("status = %v, expected up_to_date", result["status"])
	}
	if result["up_to_date"] != true {
		t.Errorf("up_to_date = %v, expected true", result["up_to_date"])
	}
	if filesChecked, ok := result["files_checked"].(float64); !ok || filesChecked < 2 {
		t.Errorf("files_checked = %v, expected >= 2", result["files_checked"])
	}
}

func TestSkillInstallCheckStale(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := defaultSkillInstallPath()
	if err != nil {
		t.Fatalf("defaultSkillInstallPath() returned error: %v", err)
	}
	if err := skills.InstallSkillBundle(filepath.Dir(path)); err != nil {
		t.Fatalf("InstallSkillBundle() returned error: %v", err)
	}
	// #nosec G306 -- Test fixture content is non-secret documentation.
	if err := os.WriteFile(path, []byte("stale skill content"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() returned error: %v", err)
	}

	cmd := skillCmd()
	cmd.SetArgs([]string{"install", "--check", "-o", "json"})
	stderr := captureStderr(t, func() {
		stdout := captureStdout(t, func() {
			if err := cmd.Execute(); err == nil {
				t.Fatal("Execute() error = nil, expected stale skill failure")
			}
		})

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("json.Unmarshal() returned error: %v\nstdout: %s", err, stdout)
		}
		if result["status"] != "stale" {
			t.Errorf("status = %v, expected stale", result["status"])
		}
		if result["up_to_date"] != false {
			t.Errorf("up_to_date = %v, expected false", result["up_to_date"])
		}
	})
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, expected empty", stderr)
	}
}

func TestSkillInstallCheckMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cmd := skillCmd()
	cmd.SetArgs([]string{"install", "--check", "-o", "json"})
	stderr := captureStderr(t, func() {
		stdout := captureStdout(t, func() {
			if err := cmd.Execute(); err == nil {
				t.Fatal("Execute() error = nil, expected missing skill failure")
			}
		})

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("json.Unmarshal() returned error: %v\nstdout: %s", err, stdout)
		}
		if result["status"] != "missing" {
			t.Errorf("status = %v, expected missing", result["status"])
		}
		if result["installed"] != false {
			t.Errorf("installed = %v, expected false", result["installed"])
		}
	})
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, expected empty", stderr)
	}
}
