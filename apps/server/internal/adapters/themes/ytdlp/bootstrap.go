package ytdlp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

const (
	managedReleaseVersion = "2026.07.04"
	managedWindowsURL     = "https://github.com/yt-dlp/yt-dlp/releases/download/" + managedReleaseVersion + "/yt-dlp.exe"
	managedWindowsSHA256  = "52fe3c26dcf71fbdc85b528589020bb0b8e383155cfa81b64dd447bbe35e24b8"
	maxManagedBinaryBytes = int64(32 << 20)
)

type managedExecutable struct {
	cacheDir string
	version  string
	client   *http.Client
	mu       sync.Mutex
}

func newManagedExecutable(cacheDir, version string) *managedExecutable {
	return &managedExecutable{
		cacheDir: strings.TrimSpace(cacheDir),
		version:  strings.TrimSpace(version),
		client:   &http.Client{Timeout: 4 * time.Minute},
	}
}

func (m *managedExecutable) Resolve(ctx context.Context, configured string) (string, error) {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		configured = "yt-dlp"
	}
	if path, err := exec.LookPath(configured); err == nil {
		return path, nil
	}
	if !isGenericYTDLPPath(configured) {
		return "", fmt.Errorf("%w: configured yt-dlp executable was not found: %s", domain.ErrUnavailable, configured)
	}
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("%w: yt-dlp is not installed; set MEDIA_SERVER_YTDLP to an executable path", domain.ErrUnavailable)
	}
	if m.cacheDir == "" {
		return "", fmt.Errorf("%w: yt-dlp tool cache directory is not configured", domain.ErrUnavailable)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	destination := filepath.Join(m.cacheDir, "yt-dlp.exe")
	if validManagedBinary(destination) {
		return destination, nil
	}
	if err := os.MkdirAll(m.cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("create yt-dlp tool cache: %w", err)
	}
	if err := m.download(ctx, destination); err != nil {
		return "", err
	}
	return destination, nil
}

func (m *managedExecutable) download(ctx context.Context, destination string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, managedWindowsURL, nil)
	if err != nil {
		return fmt.Errorf("create yt-dlp download request: %w", err)
	}
	userAgent := "KoduMedia"
	if m.version != "" {
		userAgent += "/" + m.version
	}
	request.Header.Set("User-Agent", userAgent)
	response, err := m.client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: download yt-dlp: %v", domain.ErrUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: yt-dlp download returned HTTP %d", domain.ErrUnavailable, response.StatusCode)
	}

	temporary, err := os.CreateTemp(m.cacheDir, ".yt-dlp-*.exe")
	if err != nil {
		return fmt.Errorf("create temporary yt-dlp executable: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(temporary, hash), io.LimitReader(response.Body, maxManagedBinaryBytes+1))
	closeErr := temporary.Close()
	if copyErr != nil {
		return fmt.Errorf("download yt-dlp: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("store yt-dlp: %w", closeErr)
	}
	if written <= 0 || written > maxManagedBinaryBytes {
		return fmt.Errorf("%w: downloaded yt-dlp has an invalid size", domain.ErrUnavailable)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, managedWindowsSHA256) {
		return fmt.Errorf("%w: downloaded yt-dlp checksum mismatch", domain.ErrUnavailable)
	}
	if err := os.Chmod(temporaryPath, 0o755); err != nil {
		return fmt.Errorf("make yt-dlp executable: %w", err)
	}
	if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("replace yt-dlp executable: %w", err)
	}
	if err := os.Rename(temporaryPath, destination); err != nil {
		return fmt.Errorf("install yt-dlp executable: %w", err)
	}
	return nil
}

func validManagedBinary(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, io.LimitReader(file, maxManagedBinaryBytes+1)); err != nil {
		return false
	}
	return strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), managedWindowsSHA256)
}

func isGenericYTDLPPath(value string) bool {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(value)))
	return base == "yt-dlp" || base == "yt-dlp.exe"
}
