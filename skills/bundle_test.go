package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallSkillBundleWritesAllFiles(t *testing.T) {
	destDir := filepath.Join(t.TempDir(), "crmservice")
	if err := InstallSkillBundle(destDir); err != nil {
		t.Fatalf("InstallSkillBundle() error: %v", err)
	}

	skillPath := filepath.Join(destDir, "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("SKILL.md missing: %v", err)
	}

	refPath := filepath.Join(destDir, "references", "filters.md")
	if _, err := os.Stat(refPath); err != nil {
		t.Fatalf("references/filters.md missing: %v", err)
	}
}

func TestCheckInstalledSkillBundleUpToDate(t *testing.T) {
	destDir := filepath.Join(t.TempDir(), "crmservice")
	if err := InstallSkillBundle(destDir); err != nil {
		t.Fatalf("InstallSkillBundle() error: %v", err)
	}

	result, err := CheckInstalledSkillBundle(filepath.Join(destDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("CheckInstalledSkillBundle() error: %v", err)
	}
	if result.Status != "up_to_date" {
		t.Errorf("status = %q, want up_to_date", result.Status)
	}
	if !result.UpToDate {
		t.Error("up_to_date = false, want true")
	}
	if result.FilesChecked < 2 {
		t.Errorf("files_checked = %d, want at least 2", result.FilesChecked)
	}
}

func TestCheckInstalledSkillBundleMissing(t *testing.T) {
	destDir := filepath.Join(t.TempDir(), "missing")
	result, err := CheckInstalledSkillBundle(filepath.Join(destDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("CheckInstalledSkillBundle() error: %v", err)
	}
	if result.Status != "missing" {
		t.Errorf("status = %q, want missing", result.Status)
	}
}

func TestCheckInstalledSkillBundleStale(t *testing.T) {
	destDir := filepath.Join(t.TempDir(), "crmservice")
	if err := InstallSkillBundle(destDir); err != nil {
		t.Fatalf("InstallSkillBundle() error: %v", err)
	}
	// #nosec G306 -- Test fixture content is non-secret documentation.
	if err := os.WriteFile(filepath.Join(destDir, "SKILL.md"), []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	result, err := CheckInstalledSkillBundle(filepath.Join(destDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("CheckInstalledSkillBundle() error: %v", err)
	}
	if result.Status != "stale" {
		t.Errorf("status = %q, want stale", result.Status)
	}
}
