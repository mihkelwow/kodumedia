package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/config"
	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestNewBootstrapsACompletelyFreshDataDirectory(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "first-run", "data")
	t.Setenv("MEDIA_SERVER_DATA_DIR", dataDir)
	t.Setenv("MEDIA_SERVER_ADDRESS", "127.0.0.1:0")
	t.Setenv("MEDIA_SERVER_FFMPEG", "ffmpeg")
	t.Setenv("MEDIA_SERVER_FFPROBE", "ffprobe")
	t.Setenv("MEDIA_SERVER_YTDLP", "yt-dlp")

	instance, err := New(context.Background(), config.Load(), "0.8.42")
	if err != nil {
		t.Fatalf("initialize fresh application: %v", err)
	}
	t.Cleanup(func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := instance.Shutdown(shutdownContext); err != nil {
			t.Errorf("shutdown fresh application: %v", err)
		}
	})

	for _, path := range []string{
		dataDir,
		filepath.Join(dataDir, "catalog.json"),
		filepath.Join(dataDir, "settings.json"),
		filepath.Join(dataDir, "transcodes"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected first-run path %q: %v", path, err)
		}
	}

	payload, err := os.ReadFile(filepath.Join(dataDir, "settings.json"))
	if err != nil {
		t.Fatalf("read first-run settings: %v", err)
	}
	var persisted struct {
		Version  int `json:"version"`
		Settings struct {
			Quality domain.QualitySettings `json:"quality"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(payload, &persisted); err != nil {
		t.Fatalf("decode first-run settings: %v", err)
	}
	if persisted.Version != 10 {
		t.Fatalf("settings schema version = %d, want 10", persisted.Version)
	}
	if persisted.Settings.Quality.CacheMaxGB != 50 ||
		persisted.Settings.Quality.CacheFreeReserveGB != 10 ||
		persisted.Settings.Quality.CacheFreeReservePercent != 10 {
		t.Fatalf("unexpected first-run cache policy: %+v", persisted.Settings.Quality)
	}

	entries, err := os.ReadDir(filepath.Join(dataDir, "transcodes"))
	if err != nil {
		t.Fatalf("read first-run transcodes directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("first-run transcodes directory contains %d entries", len(entries))
	}
}
