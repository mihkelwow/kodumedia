package ytdlp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/platform/childprocess"
)

type Downloader struct {
	ytdlpPath  func() string
	ffmpegPath func() string
	managed    *managedExecutable
}

func NewDynamic(ytdlpPath, ffmpegPath func() string) *Downloader {
	return &Downloader{ytdlpPath: ytdlpPath, ffmpegPath: ffmpegPath}
}

// NewManaged uses an explicitly configured or PATH-resolved yt-dlp first. On
// Windows, when the generic command is unavailable, it downloads the pinned
// official executable once into the server data directory and verifies its
// SHA-256 checksum before use.
func NewManaged(ytdlpPath, ffmpegPath func() string, cacheDir, version string) *Downloader {
	return &Downloader{
		ytdlpPath:  ytdlpPath,
		ffmpegPath: ffmpegPath,
		managed:    newManagedExecutable(cacheDir, version),
	}
}

func (d *Downloader) resolveYTDLP(ctx context.Context) (string, error) {
	configured := ""
	if d.ytdlpPath != nil {
		configured = strings.TrimSpace(d.ytdlpPath())
	}
	if d.managed != nil {
		return d.managed.Resolve(ctx, configured)
	}
	if configured == "" {
		return "", fmt.Errorf("%w: yt-dlp is not configured", domain.ErrUnavailable)
	}
	return configured, nil
}

func (d *Downloader) DownloadAudio(ctx context.Context, sourceURL, destination string) error {
	sourceURL = strings.TrimSpace(sourceURL)
	destination = strings.TrimSpace(destination)
	if sourceURL == "" || destination == "" {
		return fmt.Errorf("%w: theme source and destination are required", domain.ErrInvalidInput)
	}
	ytdlp, err := d.resolveYTDLP(ctx)
	if err != nil {
		return err
	}
	ffmpeg := strings.TrimSpace(d.ffmpegPath())
	if ffmpeg == "" {
		return fmt.Errorf("%w: FFmpeg is not configured", domain.ErrUnavailable)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create theme cache directory: %w", err)
	}
	temporaryDirectory, err := os.MkdirTemp(filepath.Dir(destination), ".theme-download-*")
	if err != nil {
		return fmt.Errorf("create theme download directory: %w", err)
	}
	defer os.RemoveAll(temporaryDirectory)

	outputTemplate := filepath.Join(temporaryDirectory, "theme.%(ext)s")
	arguments := []string{
		"--ignore-config",
		"--no-playlist",
		"--no-progress",
		"--no-warnings",
		"--no-part",
		"--socket-timeout", "20",
		"--retries", "3",
		"--fragment-retries", "3",
		"--match-filter", "duration <= 600",
		"--format", "bestaudio[ext=m4a]/bestaudio",
		"--extract-audio",
		"--audio-format", "m4a",
		"--audio-quality", "5",
		"--ffmpeg-location", filepath.Dir(ffmpeg),
		"--output", outputTemplate,
		"--", sourceURL,
	}
	command := exec.CommandContext(ctx, ytdlp, arguments...)
	childprocess.Hide(command)
	output, err := command.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return ctx.Err()
		}
		message := strings.TrimSpace(string(output))
		if len(message) > 1200 {
			message = message[len(message)-1200:]
		}
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("%w: yt-dlp failed: %s", domain.ErrUnavailable, message)
	}

	downloaded := filepath.Join(temporaryDirectory, "theme.m4a")
	info, err := os.Stat(downloaded)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return fmt.Errorf("%w: yt-dlp did not create a usable m4a file", domain.ErrUnavailable)
	}
	if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("replace cached theme: %w", err)
	}
	if err := os.Rename(downloaded, destination); err != nil {
		return fmt.Errorf("store cached theme: %w", err)
	}
	return nil
}
