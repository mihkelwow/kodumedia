package updatepackage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const PlanSchemaVersion = 1

type Plan struct {
	SchemaVersion   int            `json:"schemaVersion"`
	ParentPID       int            `json:"parentPid"`
	InstallDir      string         `json:"installDir"`
	StageDir        string         `json:"stageDir"`
	BackupDir       string         `json:"backupDir"`
	StatusPath      string         `json:"statusPath"`
	EntryPoint      string         `json:"entryPoint"`
	Arguments       []string       `json:"arguments"`
	WorkingDir      string         `json:"workingDir"`
	HealthURL       string         `json:"healthUrl"`
	PreviousVersion string         `json:"previousVersion"`
	TargetVersion   string         `json:"targetVersion"`
	PackageName     string         `json:"packageName"`
	Files           []ManifestFile `json:"files"`
	CreatedAt       time.Time      `json:"createdAt"`
}

func WritePlan(path string, plan Plan) error {
	plan.SchemaVersion = PlanSchemaVersion
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = time.Now().UTC()
	}
	return WriteJSONAtomic(path, plan)
}

func ReadPlan(path string) (Plan, error) {
	file, err := os.Open(path)
	if err != nil {
		return Plan{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var plan Plan
	if err := decoder.Decode(&plan); err != nil {
		return Plan{}, fmt.Errorf("decode update plan: %w", err)
	}
	if plan.SchemaVersion != PlanSchemaVersion {
		return Plan{}, fmt.Errorf("unsupported update plan schema %d", plan.SchemaVersion)
	}
	return plan, nil
}

func WriteJSONAtomic(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".kodumedia-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	remove := true
	defer func() {
		if remove {
			_ = os.Remove(temporaryPath)
		}
	}()
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		// Windows cannot rename over an existing destination. Status and plan
		// files are small, so remove then rename while keeping the temporary
		// file on the same volume.
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			return err
		}
		if retryErr := os.Rename(temporaryPath, path); retryErr != nil {
			return retryErr
		}
	}
	remove = false
	return nil
}
