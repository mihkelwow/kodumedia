package jsonsettings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mihkelwow/kodumedia/internal/adapters/atomicfile"
	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

const schemaVersion = 10

type persistedSettings struct {
	Version  int                     `json:"version"`
	Settings persistedServerSettings `json:"settings"`
}

type persistedServerSettings struct {
	FFmpegPath  string                    `json:"ffmpegPath"`
	FFprobePath string                    `json:"ffprobePath"`
	Metadata    persistedMetadataSettings `json:"metadata"`
	Music       persistedMusicSettings    `json:"music,omitempty"`
	ThemeMusic  domain.ThemeMusicSettings `json:"themeMusic,omitempty"`
	Subtitles   persistedSubtitleSettings `json:"subtitles,omitempty"`
	Quality     domain.QualitySettings    `json:"quality,omitempty"`
}

type persistedMetadataSettings struct {
	TMDbToken         string `json:"tmdbToken"`
	Language          string `json:"language"`
	IMDbRatingsSource string `json:"imdbRatingsSource,omitempty"`
	IMDbRatingsPath   string `json:"imdbRatingsPath,omitempty"`
	AutoRefreshOnScan *bool  `json:"autoRefreshOnScan,omitempty"`
}

type persistedMusicSettings struct {
	MetadataEnabled         *bool  `json:"metadataEnabled,omitempty"`
	ShazamEnabled           *bool  `json:"shazamEnabled,omitempty"`
	FingerprintEnabled      *bool  `json:"fingerprintEnabled,omitempty"`
	CatalogFallbackEnabled  *bool  `json:"catalogFallbackEnabled,omitempty"`
	AudioAnalysisEnabled    *bool  `json:"audioAnalysisEnabled,omitempty"`
	GeneratedArtworkEnabled *bool  `json:"generatedArtworkEnabled,omitempty"`
	AcoustIDAPIKey          string `json:"acoustIdApiKey,omitempty"`

	// Legacy fields are accepted while migrating schema versions before 6.
	SoundCloudEnabled      bool   `json:"soundCloudEnabled,omitempty"`
	SoundCloudClientID     string `json:"soundCloudClientId,omitempty"`
	SoundCloudClientSecret string `json:"soundCloudClientSecret,omitempty"`
}

type persistedSubtitleSettings struct {
	PreferredLanguages    []string `json:"preferredLanguages,omitempty"`
	OpenSubtitlesAPIKey   string   `json:"openSubtitlesApiKey,omitempty"`
	OpenSubtitlesUsername string   `json:"openSubtitlesUsername,omitempty"`
	OpenSubtitlesPassword string   `json:"openSubtitlesPassword,omitempty"`
}

type Store struct {
	path string
	mu   sync.RWMutex
	data domain.ServerSettings
}

func Open(path string, defaults domain.ServerSettings) (*Store, error) {
	defaults = normalize(defaults, domain.ServerSettings{})
	store := &Store{path: path, data: defaults}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create settings directory: %w", err)
	}
	payload, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) || (err == nil && len(payload) == 0) {
		if err := store.persistLocked(defaults); err != nil {
			return nil, err
		}
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read settings: %w", err)
	}
	var persisted persistedSettings
	if err := json.Unmarshal(payload, &persisted); err != nil {
		return nil, fmt.Errorf("decode settings: %w", err)
	}
	if persisted.Version < 1 || persisted.Version > schemaVersion {
		return nil, fmt.Errorf("unsupported settings schema version %d", persisted.Version)
	}
	loaded := fromPersisted(persisted.Settings)
	store.data = normalize(loaded, defaults)
	if persisted.Version != schemaVersion || persisted.Settings.Metadata.IMDbRatingsSource == "" || len(persisted.Settings.Subtitles.PreferredLanguages) == 0 {
		if err := store.persistLocked(store.data); err != nil {
			return nil, fmt.Errorf("migrate settings: %w", err)
		}
	}
	return store, nil
}

func (s *Store) Current() domain.ServerSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSettings(s.data)
}

func (s *Store) Update(ctx context.Context, settings domain.ServerSettings) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	settings = normalize(settings, s.Current())
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.persistLocked(settings); err != nil {
		return err
	}
	s.data = cloneSettings(settings)
	return nil
}

func (s *Store) persistLocked(settings domain.ServerSettings) error {
	payload, err := json.MarshalIndent(persistedSettings{Version: schemaVersion, Settings: toPersisted(settings)}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	payload = append(payload, '\n')
	temporary := s.path + ".tmp"
	if err := os.WriteFile(temporary, payload, 0o600); err != nil {
		return fmt.Errorf("write temporary settings: %w", err)
	}
	if err := atomicfile.Replace(temporary, s.path); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("replace settings: %w", err)
	}
	return nil
}

func fromPersisted(value persistedServerSettings) domain.ServerSettings {
	source := firstNonEmpty(value.Metadata.IMDbRatingsSource, value.Metadata.IMDbRatingsPath)
	return domain.ServerSettings{
		FFmpegPath:  value.FFmpegPath,
		FFprobePath: value.FFprobePath,
		Metadata: domain.MetadataSettings{
			TMDbToken:         value.Metadata.TMDbToken,
			Language:          value.Metadata.Language,
			IMDbRatingsSource: source,
		},
		ThemeMusic: value.ThemeMusic,
		Music: domain.MusicSettings{
			MetadataEnabled:         boolValue(value.Music.MetadataEnabled, true),
			ShazamEnabled:           boolValue(value.Music.ShazamEnabled, true),
			FingerprintEnabled:      boolValue(value.Music.FingerprintEnabled, true),
			CatalogFallbackEnabled:  boolValue(value.Music.CatalogFallbackEnabled, true),
			AudioAnalysisEnabled:    boolValue(value.Music.AudioAnalysisEnabled, true),
			GeneratedArtworkEnabled: boolValue(value.Music.GeneratedArtworkEnabled, true),
			AcoustIDAPIKey:          value.Music.AcoustIDAPIKey,
		},
		Subtitles: domain.SubtitleSettings{
			PreferredLanguages:    append([]string(nil), value.Subtitles.PreferredLanguages...),
			OpenSubtitlesAPIKey:   value.Subtitles.OpenSubtitlesAPIKey,
			OpenSubtitlesUsername: value.Subtitles.OpenSubtitlesUsername,
			OpenSubtitlesPassword: value.Subtitles.OpenSubtitlesPassword,
		},
		Quality: value.Quality,
	}
}

func toPersisted(value domain.ServerSettings) persistedServerSettings {
	return persistedServerSettings{
		FFmpegPath:  value.FFmpegPath,
		FFprobePath: value.FFprobePath,
		Metadata: persistedMetadataSettings{
			TMDbToken:         value.Metadata.TMDbToken,
			Language:          value.Metadata.Language,
			IMDbRatingsSource: value.Metadata.IMDbRatingsSource,
		},
		ThemeMusic: value.ThemeMusic,
		Music: persistedMusicSettings{
			MetadataEnabled:         boolPointer(value.Music.MetadataEnabled),
			ShazamEnabled:           boolPointer(value.Music.ShazamEnabled),
			FingerprintEnabled:      boolPointer(value.Music.FingerprintEnabled),
			CatalogFallbackEnabled:  boolPointer(value.Music.CatalogFallbackEnabled),
			AudioAnalysisEnabled:    boolPointer(value.Music.AudioAnalysisEnabled),
			GeneratedArtworkEnabled: boolPointer(value.Music.GeneratedArtworkEnabled),
			AcoustIDAPIKey:          value.Music.AcoustIDAPIKey,
		},
		Subtitles: persistedSubtitleSettings{
			PreferredLanguages:    append([]string(nil), value.Subtitles.PreferredLanguages...),
			OpenSubtitlesAPIKey:   value.Subtitles.OpenSubtitlesAPIKey,
			OpenSubtitlesUsername: value.Subtitles.OpenSubtitlesUsername,
			OpenSubtitlesPassword: value.Subtitles.OpenSubtitlesPassword,
		},
		Quality: value.Quality,
	}
}

func normalize(value, fallback domain.ServerSettings) domain.ServerSettings {
	value.FFmpegPath = strings.TrimSpace(value.FFmpegPath)
	if value.FFmpegPath == "" {
		value.FFmpegPath = firstNonEmpty(fallback.FFmpegPath, "ffmpeg")
	}
	value.FFprobePath = strings.TrimSpace(value.FFprobePath)
	if value.FFprobePath == "" {
		value.FFprobePath = firstNonEmpty(fallback.FFprobePath, "ffprobe")
	}
	value.Metadata.TMDbToken = strings.TrimSpace(value.Metadata.TMDbToken)
	value.Metadata.Language = strings.TrimSpace(value.Metadata.Language)
	if value.Metadata.Language == "" {
		value.Metadata.Language = firstNonEmpty(fallback.Metadata.Language, domain.DefaultMetadataLanguage)
	}
	value.Metadata.IMDbRatingsSource = strings.TrimSpace(value.Metadata.IMDbRatingsSource)
	if value.Metadata.IMDbRatingsSource == "" {
		value.Metadata.IMDbRatingsSource = firstNonEmpty(fallback.Metadata.IMDbRatingsSource, domain.DefaultIMDbRatingsSource)
	}
	value.Music.AcoustIDAPIKey = strings.TrimSpace(value.Music.AcoustIDAPIKey)
	value.ThemeMusic = normalizeThemeMusic(value.ThemeMusic, fallback.ThemeMusic)
	value.Subtitles.OpenSubtitlesAPIKey = strings.TrimSpace(value.Subtitles.OpenSubtitlesAPIKey)
	value.Subtitles.OpenSubtitlesUsername = strings.TrimSpace(value.Subtitles.OpenSubtitlesUsername)
	value.Subtitles.OpenSubtitlesPassword = strings.TrimSpace(value.Subtitles.OpenSubtitlesPassword)
	value.Subtitles.PreferredLanguages = normalizeLanguages(value.Subtitles.PreferredLanguages)
	if len(value.Subtitles.PreferredLanguages) == 0 {
		value.Subtitles.PreferredLanguages = normalizeLanguages(fallback.Subtitles.PreferredLanguages)
	}
	if len(value.Subtitles.PreferredLanguages) == 0 {
		value.Subtitles.PreferredLanguages = []string{domain.DefaultSubtitleLanguage}
	}
	value.Quality = normalizeQuality(value.Quality, fallback.Quality)
	return value
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func boolPointer(value bool) *bool {
	result := value
	return &result
}

func normalizeQuality(value, fallback domain.QualitySettings) domain.QualitySettings {
	if value == (domain.QualitySettings{}) {
		value = fallback
	}
	value.Profile = strings.ToLower(strings.TrimSpace(value.Profile))
	if value.Profile == "" {
		value.Profile = firstNonEmpty(fallback.Profile, "original")
	}
	switch value.Profile {
	case "original", "maximum", "balanced", "data-saver":
	default:
		value.Profile = "original"
	}
	value.PreferredCodec = strings.ToLower(strings.TrimSpace(value.PreferredCodec))
	if value.PreferredCodec == "h265" {
		value.PreferredCodec = "hevc"
	}
	if value.PreferredCodec == "" {
		value.PreferredCodec = firstNonEmpty(fallback.PreferredCodec, "auto")
	}
	switch value.PreferredCodec {
	case "auto", "h264", "hevc", "av1":
	default:
		value.PreferredCodec = "auto"
	}
	value.HardwarePreference = strings.ToLower(strings.TrimSpace(value.HardwarePreference))
	if value.HardwarePreference == "" {
		value.HardwarePreference = firstNonEmpty(fallback.HardwarePreference, "auto")
	}
	switch value.HardwarePreference {
	case "auto", "software", "qsv", "nvenc", "amf":
	default:
		value.HardwarePreference = "auto"
	}
	value.CachePolicy = strings.ToLower(strings.TrimSpace(value.CachePolicy))
	if value.CachePolicy == "" {
		value.CachePolicy = firstNonEmpty(fallback.CachePolicy, "watched")
	}
	switch value.CachePolicy {
	case "off", "watched", "complete", "finish":
	default:
		value.CachePolicy = "watched"
	}
	if value.CacheMaxGB <= 0 {
		value.CacheMaxGB = fallback.CacheMaxGB
	}
	if value.CacheMaxGB <= 0 {
		value.CacheMaxGB = 50
	}
	if value.CacheMaxGB > 4096 {
		value.CacheMaxGB = 4096
	}
	if value.CacheFreeReserveGB <= 0 {
		value.CacheFreeReserveGB = fallback.CacheFreeReserveGB
	}
	if value.CacheFreeReserveGB <= 0 {
		value.CacheFreeReserveGB = 10
	}
	if value.CacheFreeReserveGB > 1024 {
		value.CacheFreeReserveGB = 1024
	}
	if value.CacheFreeReservePercent <= 0 {
		value.CacheFreeReservePercent = fallback.CacheFreeReservePercent
	}
	if value.CacheFreeReservePercent <= 0 {
		value.CacheFreeReservePercent = 10
	}
	if value.CacheFreeReservePercent > 50 {
		value.CacheFreeReservePercent = 50
	}
	if value.GracePeriodSeconds < 0 {
		value.GracePeriodSeconds = fallback.GracePeriodSeconds
	}
	if value.GracePeriodSeconds < 0 {
		value.GracePeriodSeconds = 15
	}
	if value.GracePeriodSeconds > 300 {
		value.GracePeriodSeconds = 300
	}
	if value.VMAFTarget <= 0 {
		value.VMAFTarget = fallback.VMAFTarget
	}
	if value.VMAFTarget <= 0 {
		value.VMAFTarget = 95
	}
	if value.VMAFTarget > 100 {
		value.VMAFTarget = 100
	}
	if value.VMAFSampleSeconds <= 0 {
		value.VMAFSampleSeconds = fallback.VMAFSampleSeconds
	}
	if value.VMAFSampleSeconds <= 0 {
		value.VMAFSampleSeconds = 12
	}
	if value.VMAFSampleSeconds > 60 {
		value.VMAFSampleSeconds = 60
	}
	return value
}

func normalizeThemeMusic(value, fallback domain.ThemeMusicSettings) domain.ThemeMusicSettings {
	mode := domain.ThemePlaybackMode(strings.ToLower(strings.TrimSpace(string(value.SeriesPlayback))))
	if mode == "" {
		mode = domain.ThemePlaybackMode(strings.ToLower(strings.TrimSpace(string(fallback.SeriesPlayback))))
	}
	switch mode {
	case domain.ThemePlaybackAlways, domain.ThemePlaybackFirstOpen, domain.ThemePlaybackOff:
		value.SeriesPlayback = mode
	default:
		value.SeriesPlayback = domain.ThemePlaybackAlways
	}
	return value
}

func normalizeLanguages(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
			part = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(part, "_", "-")))
			if dash := strings.IndexByte(part, '-'); dash > 0 {
				part = part[:dash]
			}
			if len(part) < 2 || len(part) > 3 {
				continue
			}
			if _, ok := seen[part]; ok {
				continue
			}
			seen[part] = struct{}{}
			result = append(result, part)
		}
	}
	return result
}

func cloneSettings(value domain.ServerSettings) domain.ServerSettings {
	value.Subtitles.PreferredLanguages = append([]string(nil), value.Subtitles.PreferredLanguages...)
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

var _ ports.SettingsRepository = (*Store)(nil)
