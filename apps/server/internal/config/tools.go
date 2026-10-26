package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

// BundledToolPaths returns the release-local FFmpeg toolchain. Development
// builds keep working with PATH when the bundled binaries are absent.
func BundledToolPaths() (ffmpegPath, ffprobePath string) {
	executable, err := os.Executable()
	if err != nil {
		return "", ""
	}
	root := filepath.Dir(executable)
	ffmpeg := filepath.Join(root, "tools", "ffmpeg", "bin", "ffmpeg.exe")
	ffprobe := filepath.Join(root, "tools", "ffmpeg", "bin", "ffprobe.exe")
	if regularFile(ffmpeg) {
		ffmpegPath = ffmpeg
	}
	if regularFile(ffprobe) {
		ffprobePath = ffprobe
	}
	return ffmpegPath, ffprobePath
}

// BundledYTDLPPath returns the release-local yt-dlp executable used only for
// resolving ThemerrDB YouTube references into locally cached audio.
func BundledYTDLPPath() string {
	executable, err := os.Executable()
	if err != nil {
		return ""
	}
	path := filepath.Join(filepath.Dir(executable), "tools", "yt-dlp", "yt-dlp.exe")
	if regularFile(path) {
		return path
	}
	return ""
}

// PreferBundledTools migrates empty, PATH-only, or now-missing tool paths to
// the release-local binaries. A valid custom path is always preserved.
func PreferBundledTools(value domain.ServerSettings) domain.ServerSettings {
	bundledFFmpeg, bundledFFprobe := BundledToolPaths()
	if shouldUseBundled(value.FFmpegPath, "ffmpeg", "ffmpeg.exe") && bundledFFmpeg != "" {
		value.FFmpegPath = bundledFFmpeg
	}
	if shouldUseBundled(value.FFprobePath, "ffprobe", "ffprobe.exe") && bundledFFprobe != "" {
		value.FFprobePath = bundledFFprobe
	}
	return value
}

func shouldUseBundled(configured string, generic ...string) bool {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return true
	}
	for _, value := range generic {
		if strings.EqualFold(configured, value) {
			return true
		}
	}
	if filepath.IsAbs(configured) || strings.ContainsAny(configured, `\\/`) {
		return !regularFile(configured)
	}
	return false
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
