package selfupdate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/updatepackage"
)

func defaultStatus(version string) domain.UpdateStatus {
	return domain.UpdateStatus{
		State:          domain.UpdateStateIdle,
		CurrentVersion: version,
		LocalOnly:      true,
	}
}

func readStatus(path, version string) domain.UpdateStatus {
	file, err := os.Open(path)
	if err != nil {
		return defaultStatus(version)
	}
	defer file.Close()
	var status domain.UpdateStatus
	if err := json.NewDecoder(file).Decode(&status); err != nil {
		return defaultStatus(version)
	}
	status.CurrentVersion = version
	status.LocalOnly = true
	if status.State == "" {
		status.State = domain.UpdateStateIdle
	}
	return status
}

func writeStatus(path string, status domain.UpdateStatus) error {
	now := time.Now().UTC()
	status.UpdatedAt = &now
	status.LocalOnly = true
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return updatepackage.WriteJSONAtomic(path, status)
}
