package selfupdate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

const retainedUpdateBackups = 2

func cleanupUpdateWork(packagePath, stageDir, planPath string) {
	_ = os.RemoveAll(stageDir)
	_ = os.Remove(packagePath)
	_ = os.Remove(planPath)
}

func cleanupTransientUpdateWork(updatesDir string) {
	for _, name := range []string{"staged", "packages", "plans"} {
		directory := filepath.Join(updatesDir, name)
		entries, err := os.ReadDir(directory)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			_ = os.RemoveAll(filepath.Join(directory, entry.Name()))
		}
	}
}

type retainedBackup struct {
	path      string
	createdAt time.Time
}

func cleanupTerminalUpdateWork(updatesDir, statusPath string, currentVersion string) {
	status := readStatus(statusPath, currentVersion)
	switch status.State {
	case domain.UpdateStateSucceeded, domain.UpdateStateFailed, domain.UpdateStateIdle:
	default:
		return
	}
	cleanupTransientUpdateWork(updatesDir)
	pruneUpdateBackups(filepath.Join(updatesDir, "backups"), retainedUpdateBackups)
	rotateUpdateLog(filepath.Join(updatesDir, "last-update.log"), 2<<20)
}

func pruneUpdateBackups(root string, keep int) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	backups := make([]retainedBackup, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		createdAt := time.Time{}
		payload, readErr := os.ReadFile(filepath.Join(path, "backup-state.json"))
		if readErr == nil {
			var metadata struct {
				CreatedAt time.Time `json:"createdAt"`
			}
			if json.Unmarshal(payload, &metadata) == nil {
				createdAt = metadata.CreatedAt
			}
		}
		if createdAt.IsZero() {
			if info, statErr := entry.Info(); statErr == nil {
				createdAt = info.ModTime()
			}
		}
		backups = append(backups, retainedBackup{path: path, createdAt: createdAt})
	}
	sort.SliceStable(backups, func(i, j int) bool {
		return backups[i].createdAt.After(backups[j].createdAt)
	})
	if keep < 0 {
		keep = 0
	}
	for _, backup := range backups[keep:] {
		_ = os.RemoveAll(backup.path)
	}
}

func rotateUpdateLog(path string, maximumBytes int64) {
	info, err := os.Stat(path)
	if err != nil || info.Size() <= maximumBytes {
		return
	}
	_ = os.Remove(path + ".1")
	_ = os.Rename(path, path+".1")
}
