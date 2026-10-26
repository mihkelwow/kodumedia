package selfupdate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupUpdateWorkRemovesCurrentAttemptOnly(t *testing.T) {
	root := t.TempDir()
	packagePath := filepath.Join(root, "packages", "attempt.zip")
	stageDir := filepath.Join(root, "staged", "attempt")
	planPath := filepath.Join(root, "plans", "attempt.json")
	for _, path := range []string{packagePath, filepath.Join(stageDir, "kodumedia-updater.exe"), planPath} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cleanupUpdateWork(packagePath, stageDir, planPath)

	for _, path := range []string{packagePath, stageDir, planPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed, got %v", path, err)
		}
	}
}

func TestCleanupTransientUpdateWorkKeepsStatusBackupsAndLogs(t *testing.T) {
	root := t.TempDir()
	transient := []string{
		filepath.Join(root, "staged", "old", "kodumedia-updater.exe"),
		filepath.Join(root, "packages", "old.zip"),
		filepath.Join(root, "plans", "old.json"),
	}
	preserved := []string{
		filepath.Join(root, "status.json"),
		filepath.Join(root, "last-update.log"),
		filepath.Join(root, "backups", "0.5.9-to-0.5.10", "backup-state.json"),
	}
	for _, path := range append(transient, preserved...) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cleanupTransientUpdateWork(root)

	for _, path := range transient {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected transient path %s to be removed, got %v", path, err)
		}
	}
	for _, path := range preserved {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to remain: %v", path, err)
		}
	}
}

func TestTerminalCleanupKeepsOnlyTwoNewestBackups(t *testing.T) {
	root := t.TempDir()
	statusPath := filepath.Join(root, "status.json")
	if err := os.WriteFile(statusPath, []byte(`{"state":"succeeded"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 4; index++ {
		directory := filepath.Join(root, "backups", string(rune('a'+index)))
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		state := map[string]any{"createdAt": time.Date(2026, time.July, 20+index, 12, 0, 0, 0, time.UTC)}
		payload, err := json.Marshal(state)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "backup-state.json"), payload, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{
		filepath.Join(root, "packages", "old.zip"),
		filepath.Join(root, "staged", "old", "kodumedia-updater.exe"),
		filepath.Join(root, "plans", "old.json"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	cleanupTerminalUpdateWork(root, statusPath, "0.8.36")

	for _, name := range []string{"a", "b"} {
		if _, err := os.Stat(filepath.Join(root, "backups", name)); !os.IsNotExist(err) {
			t.Fatalf("old backup %s was not pruned: %v", name, err)
		}
	}
	for _, name := range []string{"c", "d"} {
		if _, err := os.Stat(filepath.Join(root, "backups", name)); err != nil {
			t.Fatalf("new backup %s was removed: %v", name, err)
		}
	}
	for _, name := range []string{"packages", "staged", "plans"} {
		entries, err := os.ReadDir(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Fatalf("transient directory %s was not emptied", name)
		}
	}
}
