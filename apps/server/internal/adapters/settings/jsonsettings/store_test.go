package jsonsettings

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestStoreUsesEnglishFirstRunLanguageDefaults(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "settings.json"), domain.ServerSettings{})
	if err != nil {
		t.Fatal(err)
	}
	settings := store.Current()
	if settings.Metadata.Language != domain.DefaultMetadataLanguage {
		t.Fatalf("metadata language = %q, want %q", settings.Metadata.Language, domain.DefaultMetadataLanguage)
	}
	if len(settings.Subtitles.PreferredLanguages) != 1 || settings.Subtitles.PreferredLanguages[0] != domain.DefaultSubtitleLanguage {
		t.Fatalf("subtitle languages = %#v, want [%q]", settings.Subtitles.PreferredLanguages, domain.DefaultSubtitleLanguage)
	}
}

func TestStorePersistsAndReloadsSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	defaults := domain.ServerSettings{
		FFmpegPath:  "ffmpeg",
		FFprobePath: "ffprobe",
		Metadata: domain.MetadataSettings{
			Language:          "et-EE",
			IMDbRatingsSource: domain.DefaultIMDbRatingsSource,
		},
	}
	store, err := Open(path, defaults)
	if err != nil {
		t.Fatal(err)
	}
	updated := store.Current()
	updated.FFmpegPath = `C:\Tools\ffmpeg.exe`
	updated.Metadata.TMDbToken = "secret"
	updated.Metadata.IMDbRatingsSource = `C:\KoduMedia\metadata\title.ratings.tsv.gz`
	updated.Music = domain.MusicSettings{MetadataEnabled: true, ShazamEnabled: true, FingerprintEnabled: true, AcoustIDAPIKey: "acoustid-key"}
	if err := store.Update(context.Background(), updated); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Open(path, defaults)
	if err != nil {
		t.Fatal(err)
	}
	got := reloaded.Current()
	if got.FFmpegPath != updated.FFmpegPath || got.Metadata.TMDbToken != "secret" || got.Metadata.IMDbRatingsSource != updated.Metadata.IMDbRatingsSource || !got.Music.MetadataEnabled || !got.Music.ShazamEnabled || !got.Music.FingerprintEnabled || got.Music.AcoustIDAPIKey != "acoustid-key" {
		t.Fatalf("unexpected settings: %+v", got)
	}
}

func TestStoreMigratesLegacyIMDbPathToSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	legacyPath := `C:\KoduMedia\metadata\title.ratings.tsv.gz`
	legacy := map[string]any{
		"version": 1,
		"settings": map[string]any{
			"ffmpegPath":  "ffmpeg",
			"ffprobePath": "ffprobe",
			"metadata": map[string]any{
				"language":          "et-EE",
				"imdbRatingsPath":   legacyPath,
				"autoRefreshOnScan": true,
			},
		},
	}
	payload, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(path, domain.ServerSettings{})
	if err != nil {
		t.Fatal(err)
	}
	if got := store.Current().Metadata.IMDbRatingsSource; got != legacyPath {
		t.Fatalf("expected legacy path %q, got %q", legacyPath, got)
	}

	migratedPayload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var migrated persistedSettings
	if err := json.Unmarshal(migratedPayload, &migrated); err != nil {
		t.Fatal(err)
	}
	if migrated.Version != schemaVersion || migrated.Settings.Metadata.IMDbRatingsSource != legacyPath {
		t.Fatalf("unexpected migrated settings: %+v", migrated)
	}
}

func TestStoreMigratesMissingQualitySettingsToDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	legacy := map[string]any{
		"version": 3,
		"settings": map[string]any{
			"ffmpegPath":  "ffmpeg",
			"ffprobePath": "ffprobe",
			"metadata": map[string]any{
				"language":          "et-EE",
				"imdbRatingsSource": domain.DefaultIMDbRatingsSource,
			},
			"subtitles": map[string]any{"preferredLanguages": []string{"et", "en"}},
		},
	}
	payload, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	defaults := domain.ServerSettings{Quality: domain.QualitySettings{
		Profile: "original", PreferredCodec: "auto", HardwarePreference: "auto",
		CachePolicy: "watched", CacheMaxGB: 50, GracePeriodSeconds: 15,
		CacheFreeReserveGB: 10, CacheFreeReservePercent: 10,
		VMAFEnabled: true, VMAFTarget: 95, VMAFSampleSeconds: 12,
	}}
	store, err := Open(path, defaults)
	if err != nil {
		t.Fatal(err)
	}
	quality := store.Current().Quality
	if quality.Profile != "original" || quality.CachePolicy != "watched" || quality.CacheMaxGB != 50 ||
		quality.CacheFreeReserveGB != 10 || quality.CacheFreeReservePercent != 10 || !quality.VMAFEnabled {
		t.Fatalf("unexpected migrated quality settings: %+v", quality)
	}
}

func TestStorePreservesExplicitDisabledVMAFAndZeroGracePeriod(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	defaults := domain.ServerSettings{Quality: domain.QualitySettings{
		Profile: "original", PreferredCodec: "auto", HardwarePreference: "auto",
		CachePolicy: "watched", CacheMaxGB: 50, GracePeriodSeconds: 15,
		VMAFEnabled: true, VMAFTarget: 95, VMAFSampleSeconds: 12,
	}}
	store, err := Open(path, defaults)
	if err != nil {
		t.Fatal(err)
	}
	settings := store.Current()
	settings.Quality.GracePeriodSeconds = 0
	settings.Quality.VMAFEnabled = false
	if err := store.Update(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Open(path, defaults)
	if err != nil {
		t.Fatal(err)
	}
	quality := reloaded.Current().Quality
	if quality.GracePeriodSeconds != 0 || quality.VMAFEnabled {
		t.Fatalf("explicit quality preferences were not preserved: %+v", quality)
	}
}
func TestStoreMigratesMissingShazamSettingToEnabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	legacy := map[string]any{
		"version": 7,
		"settings": map[string]any{
			"ffmpegPath":  "ffmpeg",
			"ffprobePath": "ffprobe",
			"metadata": map[string]any{
				"language":          "et-EE",
				"imdbRatingsSource": domain.DefaultIMDbRatingsSource,
			},
			"music": map[string]any{
				"metadataEnabled":    true,
				"fingerprintEnabled": true,
			},
			"subtitles": map[string]any{"preferredLanguages": []string{"et", "en"}},
		},
	}
	payload, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(path, domain.ServerSettings{})
	if err != nil {
		t.Fatal(err)
	}
	if !store.Current().Music.ShazamEnabled {
		t.Fatal("legacy settings must enable Shazam by default")
	}
}

func TestStoreMigratesMissingThemeMusicSettingToAlways(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	legacy := map[string]any{
		"version": 8,
		"settings": map[string]any{
			"ffmpegPath":  "ffmpeg",
			"ffprobePath": "ffprobe",
			"metadata": map[string]any{
				"language":          "et-EE",
				"imdbRatingsSource": domain.DefaultIMDbRatingsSource,
			},
			"subtitles": map[string]any{"preferredLanguages": []string{"et", "en"}},
		},
	}
	payload, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(path, domain.ServerSettings{})
	if err != nil {
		t.Fatal(err)
	}
	if got := store.Current().ThemeMusic.SeriesPlayback; got != domain.ThemePlaybackAlways {
		t.Fatalf("expected always, got %q", got)
	}
}
