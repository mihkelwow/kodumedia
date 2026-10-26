package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/updatepackage"
)

func TestApplyAndRollbackPreservesUserData(t *testing.T) {
	root := t.TempDir()
	installDir := filepath.Join(root, "install")
	stageDir := filepath.Join(root, "stage")
	backupDir := filepath.Join(root, "backup")
	for _, directory := range []string{installDir, stageDir, filepath.Join(installDir, "data")} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(installDir, "kodumedia.exe"), []byte("old server"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "data", "catalog.json"), []byte("user catalog"), 0o644); err != nil {
		t.Fatal(err)
	}
	newContent := []byte("new server")
	if err := os.WriteFile(filepath.Join(stageDir, "kodumedia.exe"), newContent, 0o644); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(newContent)
	plan := updatepackage.Plan{
		SchemaVersion: updatepackage.PlanSchemaVersion,
		InstallDir:    installDir,
		StageDir:      stageDir,
		BackupDir:     backupDir,
		Files: []updatepackage.ManifestFile{{
			Path: "kodumedia.exe", SHA256: hex.EncodeToString(hash[:]), Size: int64(len(newContent)),
		}},
	}
	if err := applyFiles(plan); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, filepath.Join(installDir, "kodumedia.exe"), "new server")
	assertFileContent(t, filepath.Join(installDir, "data", "catalog.json"), "user catalog")
	if err := rollbackFiles(plan); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, filepath.Join(installDir, "kodumedia.exe"), "old server")
	assertFileContent(t, filepath.Join(installDir, "data", "catalog.json"), "user catalog")
}

func TestApplyFilesSkipsIdenticalInstalledPayloads(t *testing.T) {
	root := t.TempDir()
	installDir := filepath.Join(root, "install")
	stageDir := filepath.Join(root, "stage")
	backupDir := filepath.Join(root, "backup")
	for _, directory := range []string{installDir, stageDir} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	unchanged := []byte("large bundled tool that did not change")
	changed := []byte("new server")
	if err := os.WriteFile(filepath.Join(installDir, "ffmpeg.exe"), unchanged, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stageDir, "ffmpeg.exe"), unchanged, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "kodumedia.exe"), []byte("old server"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stageDir, "kodumedia.exe"), changed, 0o644); err != nil {
		t.Fatal(err)
	}
	originalTime := time.Date(2020, time.January, 2, 3, 4, 5, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(installDir, "ffmpeg.exe"), originalTime, originalTime); err != nil {
		t.Fatal(err)
	}

	manifestFile := func(path string, content []byte) updatepackage.ManifestFile {
		sum := sha256.Sum256(content)
		return updatepackage.ManifestFile{Path: path, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(content))}
	}
	plan := updatepackage.Plan{
		SchemaVersion:   updatepackage.PlanSchemaVersion,
		PreviousVersion: "0.8.35",
		TargetVersion:   "0.8.36",
		InstallDir:      installDir,
		StageDir:        stageDir,
		BackupDir:       backupDir,
		Files: []updatepackage.ManifestFile{
			manifestFile("ffmpeg.exe", unchanged),
			manifestFile("kodumedia.exe", changed),
		},
	}
	if err := applyFiles(plan); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(installDir, "ffmpeg.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(originalTime) {
		t.Fatalf("identical bundled tool was replaced: modified=%s want=%s", info.ModTime(), originalTime)
	}
	if _, err := os.Stat(filepath.Join(backupDir, "ffmpeg.exe")); !os.IsNotExist(err) {
		t.Fatalf("identical bundled tool was unnecessarily backed up: %v", err)
	}
	payload, err := os.ReadFile(filepath.Join(backupDir, "backup-state.json"))
	if err != nil {
		t.Fatal(err)
	}
	var state backupState
	if err := json.Unmarshal(payload, &state); err != nil {
		t.Fatal(err)
	}
	if state.SchemaVersion != 2 || len(state.Installed) != 1 || state.Installed[0] != "kodumedia.exe" {
		t.Fatalf("unexpected differential backup state: %+v", state)
	}

	if err := rollbackFiles(plan); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, filepath.Join(installDir, "kodumedia.exe"), "old server")
	assertFileContent(t, filepath.Join(installDir, "ffmpeg.exe"), string(unchanged))
}

func TestSafeRelativePathRejectsProtectedAndEscapingPaths(t *testing.T) {
	for _, value := range []string{"../outside", "C:/absolute", "data/catalog.json", ".env", `folder\\file.txt`} {
		if _, err := safeRelativePath(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
	if got, err := safeRelativePath("docs/SECURITY.md"); err != nil || got != "docs/SECURITY.md" {
		t.Fatalf("safe path rejected: %q, %v", got, err)
	}
}

func assertFileContent(t *testing.T, path, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != expected {
		t.Fatalf("%s = %q, want %q", path, content, expected)
	}
}
