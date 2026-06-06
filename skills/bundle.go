// Package skills embeds the bundled crmservice Agent Skill (SKILL.md + references/).
package skills

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const SkillBundleRoot = "crmservice"

//go:embed crmservice/**
var CRMServiceFS embed.FS

// CRMServiceSkill is the bundled skill entry point (SKILL.md).
var CRMServiceSkill string

func init() {
	data, err := CRMServiceFS.ReadFile(filepath.Join(SkillBundleRoot, "SKILL.md"))
	if err != nil {
		panic(fmt.Sprintf("read bundled SKILL.md: %v", err))
	}
	CRMServiceSkill = string(data)
}

func skillContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func skillManifestFromFS(fsys fs.FS, root string) (map[string]string, error) {
	hashes := make(map[string]string)
	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		hashes[rel] = skillContentHash(string(data))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return hashes, nil
}

func skillManifestHash(manifest map[string]string) string {
	paths := make([]string, 0, len(manifest))
	for path := range manifest {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var b strings.Builder
	for _, path := range paths {
		b.WriteString(path)
		b.WriteByte('\n')
		b.WriteString(manifest[path])
		b.WriteByte('\n')
	}
	return skillContentHash(b.String())
}

func bundledSkillManifest() (map[string]string, string, error) {
	manifest, err := skillManifestFromFS(CRMServiceFS, SkillBundleRoot)
	if err != nil {
		return nil, "", err
	}
	return manifest, skillManifestHash(manifest), nil
}

func installedSkillManifest(dir string) (map[string]string, string, error) {
	info, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, "", os.ErrNotExist
	}
	if err != nil {
		return nil, "", err
	}
	if !info.IsDir() {
		return nil, "", fmt.Errorf("installed skill path %q is not a directory", dir)
	}

	manifest := make(map[string]string)
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		manifest[rel] = skillContentHash(string(data))
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return manifest, skillManifestHash(manifest), nil
}

func InstallSkillBundle(destDir string) error {
	return fs.WalkDir(CRMServiceFS, SkillBundleRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(SkillBundleRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := CRMServiceFS.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		// #nosec G306 -- Skill content is non-secret documentation intended to be readable.
		return os.WriteFile(target, data, 0o644)
	})
}

type SkillCheckResult struct {
	Status        string
	UpToDate      bool
	Installed     bool
	Path          string
	BundledHash   string
	InstalledHash string
	FilesChecked  int
	Message       string
}

func CheckInstalledSkillBundle(skillFilePath string) (SkillCheckResult, error) {
	destDir := filepath.Dir(skillFilePath)
	bundledManifest, bundledHash, err := bundledSkillManifest()
	if err != nil {
		return SkillCheckResult{}, err
	}

	result := SkillCheckResult{
		Path:         skillFilePath,
		BundledHash:  bundledHash,
		FilesChecked: len(bundledManifest),
	}

	installedManifest, installedHash, err := installedSkillManifest(destDir)
	if errors.Is(err, os.ErrNotExist) {
		result.Status = "missing"
		result.Message = "skill not installed; run crmservice skill install"
		return result, nil
	}
	if err != nil {
		return SkillCheckResult{}, err
	}

	result.Installed = true
	result.InstalledHash = installedHash

	if skillManifestsEqual(bundledManifest, installedManifest) {
		result.Status = "up_to_date"
		result.UpToDate = true
		result.Message = "installed skill matches bundled skill"
		return result, nil
	}

	result.Status = "stale"
	result.Message = "installed skill differs from bundled skill; run crmservice skill install"
	return result, nil
}

func skillManifestsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for path, hash := range a {
		if b[path] != hash {
			return false
		}
	}
	return true
}
