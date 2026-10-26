package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mihkelwow/kodumedia/internal/adapters/atomicfile"
	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/updatepackage"
)

const (
	parentExitTimeout = 60 * time.Second
	healthTimeout     = 60 * time.Second
)

type backupState struct {
	SchemaVersion   int       `json:"schemaVersion,omitempty"`
	PreviousVersion string    `json:"previousVersion,omitempty"`
	TargetVersion   string    `json:"targetVersion,omitempty"`
	CreatedAt       time.Time `json:"createdAt,omitempty"`
	Existing        []string  `json:"existing"`
	Installed       []string  `json:"installed,omitempty"`
}

func Run(planPath string) error {
	plan, err := updatepackage.ReadPlan(planPath)
	if err != nil {
		return err
	}
	if err := validatePlan(plan); err != nil {
		return finishFailed(plan, err, false)
	}
	started := time.Now().UTC()
	_ = writeStatus(plan, domain.UpdateStateRestarting, "Vana server suletakse ja failid paigaldatakse.", "", false, &started, nil)
	if err := waitForParentExit(plan.ParentPID, parentExitTimeout); err != nil {
		return finishFailed(plan, fmt.Errorf("wait for server exit: %w", err), false)
	}

	if err := applyFiles(plan); err != nil {
		rollbackErr := rollbackFiles(plan)
		if rollbackErr != nil {
			err = fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
		}
		_ = restartPrevious(plan)
		return finishFailed(plan, err, rollbackErr == nil)
	}

	command, err := startServer(plan)
	if err != nil {
		rollbackErr := rollbackFiles(plan)
		_ = restartPrevious(plan)
		if rollbackErr != nil {
			err = fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
		}
		return finishFailed(plan, err, rollbackErr == nil)
	}
	if err := waitForHealthyVersion(plan.HealthURL, plan.TargetVersion, healthTimeout); err != nil {
		if command.Process != nil {
			_ = command.Process.Kill()
			_, _ = command.Process.Wait()
		}
		rollbackErr := rollbackFiles(plan)
		_ = restartPrevious(plan)
		if rollbackErr != nil {
			err = fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
		}
		return finishFailed(plan, err, rollbackErr == nil)
	}
	completed := time.Now().UTC()
	if err := writeStatus(plan, domain.UpdateStateSucceeded, "KoduMedia uuendus paigaldati edukalt.", "", false, &started, &completed); err != nil {
		return err
	}
	return nil
}

func validatePlan(plan updatepackage.Plan) error {
	if plan.SchemaVersion != updatepackage.PlanSchemaVersion {
		return fmt.Errorf("unsupported plan schema")
	}
	for name, value := range map[string]string{
		"installDir": plan.InstallDir,
		"stageDir":   plan.StageDir,
		"backupDir":  plan.BackupDir,
		"statusPath": plan.StatusPath,
		"workingDir": plan.WorkingDir,
	} {
		if strings.TrimSpace(value) == "" || !filepath.IsAbs(value) {
			return fmt.Errorf("%s must be an absolute path", name)
		}
	}
	if samePath(plan.InstallDir, plan.StageDir) || samePath(plan.InstallDir, plan.BackupDir) {
		return fmt.Errorf("update directories overlap")
	}
	if plan.EntryPoint != "kodumedia.exe" || strings.TrimSpace(plan.TargetVersion) == "" || len(plan.Files) == 0 {
		return fmt.Errorf("invalid update plan")
	}
	for _, file := range plan.Files {
		if _, err := safeRelativePath(file.Path); err != nil {
			return err
		}
	}
	return nil
}

func applyFiles(plan updatepackage.Plan) error {
	files := installOrder(plan.Files)
	for _, file := range files {
		if err := verifyFile(filepath.Join(plan.StageDir, filepath.FromSlash(file.Path)), file); err != nil {
			return fmt.Errorf("verify staged %s: %w", file.Path, err)
		}
	}
	changed := make([]updatepackage.ManifestFile, 0, len(files))
	for _, file := range files {
		destination := filepath.Join(plan.InstallDir, filepath.FromSlash(file.Path))
		if err := verifyFile(destination, file); err != nil {
			changed = append(changed, file)
		}
	}
	if err := os.RemoveAll(plan.BackupDir); err != nil {
		return fmt.Errorf("clear backup directory: %w", err)
	}
	if err := os.MkdirAll(plan.BackupDir, 0o755); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}

	state := backupState{
		SchemaVersion:   2,
		PreviousVersion: plan.PreviousVersion,
		TargetVersion:   plan.TargetVersion,
		CreatedAt:       time.Now().UTC(),
		Existing:        make([]string, 0, len(changed)),
		Installed:       make([]string, 0, len(changed)),
	}
	for _, file := range changed {
		destination := filepath.Join(plan.InstallDir, filepath.FromSlash(file.Path))
		if info, err := os.Stat(destination); err == nil && !info.IsDir() {
			backup := filepath.Join(plan.BackupDir, filepath.FromSlash(file.Path))
			if err := copyFile(destination, backup); err != nil {
				return fmt.Errorf("backup %s: %w", file.Path, err)
			}
			state.Existing = append(state.Existing, file.Path)
		} else if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("inspect existing %s: %w", file.Path, err)
		}
		state.Installed = append(state.Installed, file.Path)
	}
	if err := updatepackage.WriteJSONAtomic(filepath.Join(plan.BackupDir, "backup-state.json"), state); err != nil {
		return fmt.Errorf("write backup state: %w", err)
	}
	for _, file := range changed {
		source := filepath.Join(plan.StageDir, filepath.FromSlash(file.Path))
		destination := filepath.Join(plan.InstallDir, filepath.FromSlash(file.Path))
		if err := replaceFrom(source, destination); err != nil {
			return fmt.Errorf("install %s: %w", file.Path, err)
		}
	}
	return nil
}

func rollbackFiles(plan updatepackage.Plan) error {
	stateBytes, err := os.ReadFile(filepath.Join(plan.BackupDir, "backup-state.json"))
	if err != nil {
		return fmt.Errorf("read backup state: %w", err)
	}
	var state backupState
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		return fmt.Errorf("decode backup state: %w", err)
	}
	existing := make(map[string]struct{}, len(state.Existing))
	for _, name := range state.Existing {
		existing[strings.ToLower(name)] = struct{}{}
	}
	installed := state.Installed
	if len(installed) == 0 && state.SchemaVersion < 2 {
		installed = make([]string, 0, len(plan.Files))
		for _, file := range plan.Files {
			installed = append(installed, file.Path)
		}
	}
	byPath := make(map[string]updatepackage.ManifestFile, len(plan.Files))
	for _, file := range plan.Files {
		byPath[strings.ToLower(file.Path)] = file
	}
	files := make([]updatepackage.ManifestFile, 0, len(installed))
	for _, path := range installed {
		if file, found := byPath[strings.ToLower(path)]; found {
			files = append(files, file)
		}
	}
	files = installOrder(files)
	for index := len(files) - 1; index >= 0; index-- {
		file := files[index]
		destination := filepath.Join(plan.InstallDir, filepath.FromSlash(file.Path))
		if _, wasExisting := existing[strings.ToLower(file.Path)]; wasExisting {
			backup := filepath.Join(plan.BackupDir, filepath.FromSlash(file.Path))
			if err := replaceFrom(backup, destination); err != nil {
				return fmt.Errorf("restore %s: %w", file.Path, err)
			}
			continue
		}
		if err := os.Remove(destination); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove newly installed %s: %w", file.Path, err)
		}
	}
	return nil
}

func replaceFrom(source, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".kodumedia-update-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	input, err := os.Open(source)
	if err != nil {
		_ = temporary.Close()
		return err
	}
	_, copyErr := io.Copy(temporary, input)
	inputCloseErr := input.Close()
	if copyErr == nil {
		copyErr = temporary.Sync()
	}
	closeErr := temporary.Close()
	if copyErr != nil {
		return copyErr
	}
	if inputCloseErr != nil {
		return inputCloseErr
	}
	if closeErr != nil {
		return closeErr
	}
	return atomicfile.Replace(temporaryPath, destination)
}

func copyFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	if copyErr == nil {
		copyErr = output.Sync()
	}
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func verifyFile(path string, expected updatepackage.ManifestFile) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() != expected.Size {
		return fmt.Errorf("size mismatch")
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}
	if hex.EncodeToString(hasher.Sum(nil)) != expected.SHA256 {
		return fmt.Errorf("SHA-256 mismatch")
	}
	return nil
}

func startServer(plan updatepackage.Plan) (*exec.Cmd, error) {
	executable := filepath.Join(plan.InstallDir, plan.EntryPoint)
	command := exec.Command(executable, plan.Arguments...)
	command.Dir = plan.WorkingDir
	configureChildProcess(command)
	logPath := filepath.Join(filepath.Dir(filepath.Dir(plan.StatusPath)), "kodumedia.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	command.Stdout = logFile
	command.Stderr = logFile
	if err := command.Start(); err != nil {
		_ = logFile.Close()
		return nil, err
	}
	_ = logFile.Close()
	return command, nil
}

func restartPrevious(plan updatepackage.Plan) error {
	command, err := startServer(updatepackage.Plan{
		InstallDir: plan.InstallDir,
		EntryPoint: plan.EntryPoint,
		Arguments:  plan.Arguments,
		WorkingDir: plan.WorkingDir,
		StatusPath: plan.StatusPath,
	})
	if err != nil {
		return err
	}
	if command.Process != nil {
		return command.Process.Release()
	}
	return nil
}

func waitForHealthyVersion(endpoint, expected string, timeout time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
		if err == nil {
			response, requestErr := client.Do(request)
			if requestErr == nil {
				var payload struct {
					Status  string `json:"status"`
					Version string `json:"version"`
				}
				decodeErr := json.NewDecoder(response.Body).Decode(&payload)
				_ = response.Body.Close()
				if response.StatusCode == http.StatusOK && decodeErr == nil && payload.Status == "ok" && payload.Version == expected {
					return nil
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("server did not become healthy as version %s", expected)
}

func finishFailed(plan updatepackage.Plan, cause error, rollback bool) error {
	completed := time.Now().UTC()
	_ = writeStatus(plan, domain.UpdateStateFailed, "Uuenduse paigaldamine ebaõnnestus.", cause.Error(), rollback, nil, &completed)
	return cause
}

func writeStatus(plan updatepackage.Plan, state domain.UpdateState, message, errorMessage string, rollback bool, started, completed *time.Time) error {
	now := time.Now().UTC()
	status := domain.UpdateStatus{
		State:           state,
		CurrentVersion:  plan.PreviousVersion,
		TargetVersion:   plan.TargetVersion,
		PackageName:     plan.PackageName,
		Message:         message,
		Error:           errorMessage,
		LocalOnly:       true,
		StartedAt:       started,
		CompletedAt:     completed,
		UpdatedAt:       &now,
		RollbackApplied: rollback,
	}
	if state == domain.UpdateStateSucceeded {
		status.CurrentVersion = plan.TargetVersion
	}
	return updatepackage.WriteJSONAtomic(plan.StatusPath, status)
}

func safeRelativePath(value string) (string, error) {
	if value == "" || strings.Contains(value, "\\") || strings.Contains(value, ":") || filepath.IsAbs(value) {
		return "", fmt.Errorf("unsafe update path %q", value)
	}
	cleaned := filepath.ToSlash(filepath.Clean(value))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("unsafe update path %q", value)
	}
	lower := strings.ToLower(cleaned)
	if lower == ".env" || lower == "settings.json" || lower == "catalog.json" || lower == "data" || strings.HasPrefix(lower, "data/") {
		return "", fmt.Errorf("protected update path %q", value)
	}
	return cleaned, nil
}

func installOrder(files []updatepackage.ManifestFile) []updatepackage.ManifestFile {
	result := append([]updatepackage.ManifestFile(nil), files...)
	priority := func(name string) int {
		switch strings.ToLower(name) {
		case "kodumedia-updater.exe":
			return 1
		case "kodumedia.exe":
			return 2
		default:
			return 0
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		left, right := priority(result[i].Path), priority(result[j].Path)
		if left != right {
			return left < right
		}
		return strings.ToLower(result[i].Path) < strings.ToLower(result[j].Path)
	})
	return result
}

func samePath(left, right string) bool {
	left, _ = filepath.Abs(left)
	right, _ = filepath.Abs(right)
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}
