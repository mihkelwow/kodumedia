package service

import (
	"context"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type memorySettings struct{ value domain.ServerSettings }

func (m *memorySettings) Current() domain.ServerSettings { return m.value }
func (m *memorySettings) Update(_ context.Context, value domain.ServerSettings) error {
	m.value = value
	return nil
}

type staticInspector struct{}

func (staticInspector) Inspect(_ context.Context, _ domain.ServerSettings) domain.SystemStatus {
	return domain.SystemStatus{}
}

func TestSettingsUpdateKeepsMaskedTokenWhenOmitted(t *testing.T) {
	repo := &memorySettings{value: domain.ServerSettings{FFmpegPath: "ffmpeg", FFprobePath: "ffprobe", Metadata: domain.MetadataSettings{TMDbToken: "secret-token", Language: "et-EE"}}}
	service := NewSettingsService(repo, staticInspector{}, "test")
	_, err := service.Update(context.Background(), domain.ServerSettingsUpdate{FFmpegPath: "ffmpeg.exe", FFprobePath: "ffprobe.exe", Metadata: domain.MetadataSettingsUpdate{Language: "en_us", IMDbRatingsSource: domain.DefaultIMDbRatingsSource}})
	if err != nil {
		t.Fatal(err)
	}
	if repo.value.Metadata.TMDbToken != "secret-token" || repo.value.Metadata.Language != "en-US" || repo.value.Metadata.IMDbRatingsSource != domain.DefaultIMDbRatingsSource {
		t.Fatalf("unexpected update: %+v", repo.value)
	}
}

func TestSettingsUpdateStoresAndMasksOptionalAcoustIDKey(t *testing.T) {
	repo := &memorySettings{value: domain.ServerSettings{FFmpegPath: "ffmpeg", FFprobePath: "ffprobe", Metadata: domain.MetadataSettings{Language: "et-EE", IMDbRatingsSource: domain.DefaultIMDbRatingsSource}}}
	service := NewSettingsService(repo, staticInspector{}, "test")
	apiKey := "acoustid-key-value"
	configuration, err := service.Update(context.Background(), domain.ServerSettingsUpdate{
		FFmpegPath: "ffmpeg", FFprobePath: "ffprobe",
		Metadata: domain.MetadataSettingsUpdate{Language: "et-EE", IMDbRatingsSource: domain.DefaultIMDbRatingsSource},
		Music:    domain.MusicSettingsUpdate{MetadataEnabled: true, ShazamEnabled: true, FingerprintEnabled: true, AcoustIDAPIKey: &apiKey},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !repo.value.Music.MetadataEnabled || !repo.value.Music.ShazamEnabled || !repo.value.Music.FingerprintEnabled || repo.value.Music.AcoustIDAPIKey != apiKey {
		t.Fatalf("music metadata settings were not stored: %+v", repo.value.Music)
	}
	if !configuration.Settings.Music.AcoustIDAPIKeyConfigured || configuration.Settings.Music.AcoustIDAPIKeyHint == apiKey {
		t.Fatalf("public settings leaked or omitted the API key: %+v", configuration.Settings.Music)
	}
}

func TestSettingsUpdateAllowsMetadataWithoutAcoustIDKey(t *testing.T) {
	repo := &memorySettings{value: domain.ServerSettings{FFmpegPath: "ffmpeg", FFprobePath: "ffprobe", Metadata: domain.MetadataSettings{Language: "et-EE", IMDbRatingsSource: domain.DefaultIMDbRatingsSource}}}
	service := NewSettingsService(repo, staticInspector{}, "test")
	_, err := service.Update(context.Background(), domain.ServerSettingsUpdate{
		FFmpegPath: "ffmpeg", FFprobePath: "ffprobe",
		Metadata: domain.MetadataSettingsUpdate{Language: "et-EE", IMDbRatingsSource: domain.DefaultIMDbRatingsSource},
		Music:    domain.MusicSettingsUpdate{MetadataEnabled: true, ShazamEnabled: true, FingerprintEnabled: true},
	})
	if err != nil {
		t.Fatalf("AcoustID key must remain optional: %v", err)
	}
}

func TestSettingsUpdateStoresThemeMusicMode(t *testing.T) {
	repo := &memorySettings{value: domain.ServerSettings{
		FFmpegPath: "ffmpeg", FFprobePath: "ffprobe",
		Metadata:   domain.MetadataSettings{Language: "et-EE", IMDbRatingsSource: domain.DefaultIMDbRatingsSource},
		ThemeMusic: domain.ThemeMusicSettings{SeriesPlayback: domain.ThemePlaybackAlways},
	}}
	settings := NewSettingsService(repo, staticInspector{}, "test")
	configuration, err := settings.Update(context.Background(), domain.ServerSettingsUpdate{
		FFmpegPath: "ffmpeg", FFprobePath: "ffprobe",
		Metadata:   domain.MetadataSettingsUpdate{Language: "et-EE", IMDbRatingsSource: domain.DefaultIMDbRatingsSource},
		ThemeMusic: domain.ThemeMusicSettings{SeriesPlayback: domain.ThemePlaybackFirstOpen},
	})
	if err != nil {
		t.Fatal(err)
	}
	if repo.value.ThemeMusic.SeriesPlayback != domain.ThemePlaybackFirstOpen || configuration.Settings.ThemeMusic.SeriesPlayback != domain.ThemePlaybackFirstOpen {
		t.Fatalf("theme setting was not stored: repo=%+v public=%+v", repo.value.ThemeMusic, configuration.Settings.ThemeMusic)
	}
}

func TestSettingsUpdateRejectsUnknownThemeMusicMode(t *testing.T) {
	repo := &memorySettings{value: domain.ServerSettings{
		FFmpegPath: "ffmpeg", FFprobePath: "ffprobe",
		Metadata: domain.MetadataSettings{Language: "et-EE", IMDbRatingsSource: domain.DefaultIMDbRatingsSource},
	}}
	settings := NewSettingsService(repo, staticInspector{}, "test")
	_, err := settings.Update(context.Background(), domain.ServerSettingsUpdate{
		FFmpegPath: "ffmpeg", FFprobePath: "ffprobe",
		Metadata:   domain.MetadataSettingsUpdate{Language: "et-EE", IMDbRatingsSource: domain.DefaultIMDbRatingsSource},
		ThemeMusic: domain.ThemeMusicSettings{SeriesPlayback: "sometimes"},
	})
	if err == nil {
		t.Fatal("unknown theme playback mode must fail")
	}
}

func TestUpdateThemeMusicPreservesOtherSettings(t *testing.T) {
	repository := &memorySettings{value: domain.ServerSettings{
		FFmpegPath:  "custom-ffmpeg.exe",
		FFprobePath: "custom-ffprobe.exe",
		Metadata: domain.MetadataSettings{
			TMDbToken:         "secret",
			Language:          "en-US",
			IMDbRatingsSource: domain.DefaultIMDbRatingsSource,
		},
		ThemeMusic: domain.ThemeMusicSettings{SeriesPlayback: domain.ThemePlaybackAlways},
	}}
	settings := NewSettingsService(repository, staticInspector{}, "test")

	updated, err := settings.UpdateThemeMusic(context.Background(), domain.ThemeMusicSettings{SeriesPlayback: domain.ThemePlaybackOff})
	if err != nil {
		t.Fatal(err)
	}
	if updated.SeriesPlayback != domain.ThemePlaybackOff || repository.value.ThemeMusic.SeriesPlayback != domain.ThemePlaybackOff {
		t.Fatalf("theme mode was not updated: returned=%+v stored=%+v", updated, repository.value.ThemeMusic)
	}
	if repository.value.FFmpegPath != "custom-ffmpeg.exe" || repository.value.FFprobePath != "custom-ffprobe.exe" || repository.value.Metadata.TMDbToken != "secret" {
		t.Fatalf("unrelated settings changed: %+v", repository.value)
	}
}

func TestUpdateThemeMusicRejectsUnknownMode(t *testing.T) {
	repository := &memorySettings{value: domain.ServerSettings{ThemeMusic: domain.ThemeMusicSettings{SeriesPlayback: domain.ThemePlaybackAlways}}}
	settings := NewSettingsService(repository, staticInspector{}, "test")

	if _, err := settings.UpdateThemeMusic(context.Background(), domain.ThemeMusicSettings{SeriesPlayback: "sometimes"}); err == nil {
		t.Fatal("unknown theme mode must be rejected")
	}
	if repository.value.ThemeMusic.SeriesPlayback != domain.ThemePlaybackAlways {
		t.Fatalf("invalid update changed the stored setting: %+v", repository.value.ThemeMusic)
	}
}
