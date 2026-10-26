//go:build windows

package selfupdate

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/updatepackage"
)

const maxUpdateUploadBytes = int64(128 << 20)

const (
	createNewProcessGroup = 0x00000200
	createNoWindow        = 0x08000000
)

type Manager struct {
	dataDir        string
	installDir     string
	currentVersion string
	address        string
	workingDir     string
	statusPath     string

	mutex      sync.Mutex
	installing bool
}

func New(dataDir, currentVersion, address string) (*Manager, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable: %w", err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}
	dataDir, err = filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve data directory: %w", err)
	}
	// Always restart from the installation directory. Startup entries do not
	// provide a stable working directory and may otherwise resolve relative
	// paths below Windows\System32.
	workingDir := filepath.Dir(executable)
	manager := &Manager{
		dataDir:        dataDir,
		installDir:     filepath.Dir(executable),
		currentVersion: strings.TrimSpace(currentVersion),
		address:        address,
		workingDir:     workingDir,
		statusPath:     filepath.Join(dataDir, "updates", "status.json"),
	}
	// A freshly started server may still be inside the updater health check.
	// Housekeeping waits for a terminal state so it never removes the staged
	// updater while that process is still running.
	go manager.cleanupAfterTerminalStatus()
	return manager, nil
}

func (m *Manager) Status() domain.UpdateStatus {
	return readStatus(m.statusPath, m.currentVersion)
}

func (m *Manager) Install(ctx context.Context, packageReader io.Reader, packageName string) (domain.UpdateStatus, error) {
	m.mutex.Lock()
	if m.installing {
		m.mutex.Unlock()
		return m.Status(), fmt.Errorf("%w: uuenduse paigaldamine on juba käimas", domain.ErrConflict)
	}
	persisted := readStatus(m.statusPath, m.currentVersion)
	if persisted.State == domain.UpdateStatePreparing || persisted.State == domain.UpdateStateInstalling || persisted.State == domain.UpdateStateRestarting {
		m.mutex.Unlock()
		return persisted, fmt.Errorf("%w: uuenduse paigaldamine on juba käimas", domain.ErrConflict)
	}
	m.installing = true
	m.mutex.Unlock()

	updatesDir := filepath.Join(m.dataDir, "updates")
	// A terminal status means no updater owns these temporary paths anymore.
	// Remove leftovers from earlier failed attempts before creating a new stage.
	cleanupTransientUpdateWork(updatesDir)

	succeeded := false
	defer func() {
		if !succeeded {
			m.mutex.Lock()
			m.installing = false
			m.mutex.Unlock()
		}
	}()

	if packageReader == nil {
		return m.failStatus(packageName, "Uuenduspakett puudub.", domain.ErrInvalidInput)
	}
	packageName = sanitizePackageName(packageName)
	if !strings.EqualFold(filepath.Ext(packageName), ".zip") {
		return m.failStatus(packageName, "Valitud fail ei ole ZIP-pakett.", domain.ErrInvalidInput)
	}

	now := time.Now().UTC()
	preparing := domain.UpdateStatus{
		State:          domain.UpdateStatePreparing,
		CurrentVersion: m.currentVersion,
		PackageName:    packageName,
		Message:        "Uuenduspaketti kontrollitakse.",
		LocalOnly:      true,
		StartedAt:      &now,
	}
	if err := writeStatus(m.statusPath, preparing); err != nil {
		return preparing, fmt.Errorf("write update status: %w", err)
	}

	identifier, err := randomID()
	if err != nil {
		return m.failStatus(packageName, "Uuenduse töökausta loomine ebaõnnestus.", err)
	}
	packagePath := filepath.Join(updatesDir, "packages", identifier+".zip")
	stageDir := filepath.Join(updatesDir, "staged", identifier)
	planPath := filepath.Join(updatesDir, "plans", identifier+".json")
	cleanupCurrentWork := true
	defer func() {
		if cleanupCurrentWork {
			cleanupUpdateWork(packagePath, stageDir, planPath)
		}
	}()
	if err := saveLimited(ctx, packagePath, packageReader, maxUpdateUploadBytes); err != nil {
		return m.failStatus(packageName, "Uuenduspaketi salvestamine ebaõnnestus.", err)
	}
	prepared, err := updatepackage.ValidateAndExtract(ctx, packagePath, stageDir, m.currentVersion)
	if err != nil {
		return m.failStatus(packageName, "Uuenduspakett ei läbinud kontrolli.", err)
	}

	backupDir := filepath.Join(updatesDir, "backups", m.currentVersion+"-to-"+prepared.Manifest.Version+"-"+identifier)
	plan := updatepackage.Plan{
		ParentPID:       os.Getpid(),
		InstallDir:      m.installDir,
		StageDir:        stageDir,
		BackupDir:       backupDir,
		StatusPath:      m.statusPath,
		EntryPoint:      prepared.Manifest.EntryPoint,
		Arguments:       append([]string(nil), os.Args[1:]...),
		WorkingDir:      m.workingDir,
		HealthURL:       localHealthURL(m.address),
		PreviousVersion: m.currentVersion,
		TargetVersion:   prepared.Manifest.Version,
		PackageName:     packageName,
		Files:           append([]updatepackage.ManifestFile(nil), prepared.Manifest.Files...),
	}
	if err := updatepackage.WritePlan(planPath, plan); err != nil {
		return m.failStatus(packageName, "Uuendusplaani loomine ebaõnnestus.", err)
	}

	installing := preparing
	installing.State = domain.UpdateStateInstalling
	installing.TargetVersion = prepared.Manifest.Version
	installing.Message = "Uuendus on kontrollitud. Server valmistub taaskäivituseks."
	if err := writeStatus(m.statusPath, installing); err != nil {
		return installing, fmt.Errorf("write update status: %w", err)
	}

	updaterPath := filepath.Join(stageDir, "kodumedia-updater.exe")
	logPath := filepath.Join(updatesDir, "last-update.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return m.failStatus(packageName, "Uuendaja logi loomine ebaõnnestus.", err)
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return m.failStatus(packageName, "Uuendaja logi avamine ebaõnnestus.", err)
	}
	command := exec.Command(updaterPath, "--plan", planPath)
	command.Dir = stageDir
	command.Stdout = logFile
	command.Stderr = logFile
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNewProcessGroup | createNoWindow}
	if err := command.Start(); err != nil {
		_ = logFile.Close()
		return m.failStatus(packageName, "Uuendaja käivitamine ebaõnnestus.", err)
	}
	_ = command.Process.Release()
	_ = logFile.Close()
	cleanupCurrentWork = false
	succeeded = true
	return installing, nil
}

func (m *Manager) cleanupAfterTerminalStatus() {
	deadline := time.Now().Add(3 * time.Minute)
	for {
		status := readStatus(m.statusPath, m.currentVersion)
		switch status.State {
		case domain.UpdateStateSucceeded, domain.UpdateStateFailed, domain.UpdateStateIdle:
			// Give the updater time to close its executable and log handles on
			// Windows before removing transient work.
			time.Sleep(2 * time.Second)
			cleanupTerminalUpdateWork(filepath.Join(m.dataDir, "updates"), m.statusPath, m.currentVersion)
			return
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (m *Manager) failStatus(packageName, message string, cause error) (domain.UpdateStatus, error) {
	now := time.Now().UTC()
	status := domain.UpdateStatus{
		State:          domain.UpdateStateFailed,
		CurrentVersion: m.currentVersion,
		PackageName:    packageName,
		Message:        message,
		Error:          cause.Error(),
		LocalOnly:      true,
		UpdatedAt:      &now,
		CompletedAt:    &now,
	}
	_ = writeStatus(m.statusPath, status)
	return status, cause
}

func saveLimited(ctx context.Context, destination string, reader io.Reader, limit int64) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".upload-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	written, copyErr := io.Copy(temporary, io.LimitReader(&contextReader{ctx: ctx, reader: reader}, limit+1))
	closeErr := temporary.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written > limit {
		return fmt.Errorf("%w: uuenduspakett on suurem kui 128 MiB", domain.ErrInvalidInput)
	}
	if err := os.Rename(temporaryPath, destination); err != nil {
		return err
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}

func sanitizePackageName(value string) string {
	value = filepath.Base(strings.TrimSpace(value))
	if value == "." || value == "" {
		return "kodumedia-update.zip"
	}
	if len(value) > 160 {
		value = value[:160]
	}
	return value
}

func randomID() (string, error) {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func localHealthURL(address string) string {
	address = strings.TrimSpace(address)
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		if strings.HasPrefix(address, ":") {
			port = strings.TrimPrefix(address, ":")
		} else {
			port = "8096"
		}
	}
	host = strings.Trim(host, "[]")
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "8096"
	}
	return (&url.URL{Scheme: "http", Host: net.JoinHostPort(host, port), Path: "/api/v1/health"}).String()
}
