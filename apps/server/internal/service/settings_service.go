package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

var languagePattern = regexp.MustCompile(`^[A-Za-z]{2,3}(?:[-_][A-Za-z]{2})?$`)
var shortLanguagePattern = regexp.MustCompile(`^[A-Za-z]{2,3}$`)

type SettingsChangeHandler func(previous, current domain.ServerSettings)

type SettingsService struct {
	repository ports.SettingsRepository
	inspector  ports.SettingsInspector
	version    string
	onChange   SettingsChangeHandler
}

func NewSettingsService(repository ports.SettingsRepository, inspector ports.SettingsInspector, version string) *SettingsService {
	return &SettingsService{repository: repository, inspector: inspector, version: version}
}

func (s *SettingsService) SetChangeHandler(handler SettingsChangeHandler) { s.onChange = handler }
func (s *SettingsService) Current() domain.ServerSettings                 { return s.repository.Current() }

func (s *SettingsService) Configuration(ctx context.Context) domain.SystemConfiguration {
	return s.configurationFor(ctx, s.repository.Current())
}

func (s *SettingsService) Test(ctx context.Context, input domain.ServerSettingsUpdate) (domain.SystemConfiguration, error) {
	candidate, err := mergeSettings(s.repository.Current(), input)
	if err != nil {
		return domain.SystemConfiguration{}, err
	}
	return s.configurationFor(ctx, candidate), nil
}

func (s *SettingsService) Update(ctx context.Context, input domain.ServerSettingsUpdate) (domain.SystemConfiguration, error) {
	previous := s.repository.Current()
	candidate, err := mergeSettings(previous, input)
	if err != nil {
		return domain.SystemConfiguration{}, err
	}
	if err := s.repository.Update(ctx, candidate); err != nil {
		return domain.SystemConfiguration{}, err
	}
	if s.onChange != nil {
		s.onChange(previous, candidate)
	}
	return s.configurationFor(ctx, candidate), nil
}

func (s *SettingsService) UpdateThemeMusic(ctx context.Context, input domain.ThemeMusicSettings) (domain.ThemeMusicSettings, error) {
	value, err := normalizeThemeMusicSettings(input)
	if err != nil {
		return domain.ThemeMusicSettings{}, err
	}
	previous := s.repository.Current()
	candidate := previous
	candidate.ThemeMusic = value
	if err := s.repository.Update(ctx, candidate); err != nil {
		return domain.ThemeMusicSettings{}, err
	}
	if s.onChange != nil {
		s.onChange(previous, candidate)
	}
	return value, nil
}

func (s *SettingsService) configurationFor(ctx context.Context, settings domain.ServerSettings) domain.SystemConfiguration {
	status := domain.SystemStatus{}
	if s.inspector != nil {
		status = s.inspector.Inspect(ctx, settings)
	}
	return domain.SystemConfiguration{Version: s.version, Settings: publicSettings(settings), Status: status}
}

func mergeSettings(current domain.ServerSettings, input domain.ServerSettingsUpdate) (domain.ServerSettings, error) {
	ffmpegPath := strings.TrimSpace(input.FFmpegPath)
	ffprobePath := strings.TrimSpace(input.FFprobePath)
	if ffmpegPath == "" {
		return domain.ServerSettings{}, fmt.Errorf("%w: FFmpeg tee on kohustuslik", domain.ErrInvalidInput)
	}
	if ffprobePath == "" {
		return domain.ServerSettings{}, fmt.Errorf("%w: FFprobe tee on kohustuslik", domain.ErrInvalidInput)
	}
	language, err := normalizeLanguage(input.Metadata.Language)
	if err != nil {
		return domain.ServerSettings{}, err
	}
	source := strings.TrimSpace(input.Metadata.IMDbRatingsSource)
	if source == "" {
		source = domain.DefaultIMDbRatingsSource
	}
	preferred, err := normalizeSubtitleLanguages(input.Subtitles.PreferredLanguages)
	if err != nil {
		return domain.ServerSettings{}, err
	}
	quality, err := normalizeQualitySettings(input.Quality)
	if err != nil {
		return domain.ServerSettings{}, err
	}

	candidate := current
	candidate.FFmpegPath = ffmpegPath
	candidate.FFprobePath = ffprobePath
	candidate.Metadata.Language = language
	candidate.Metadata.IMDbRatingsSource = source
	candidate.Music.MetadataEnabled = input.Music.MetadataEnabled
	candidate.Music.ShazamEnabled = input.Music.ShazamEnabled
	candidate.Music.FingerprintEnabled = input.Music.FingerprintEnabled
	candidate.Music.CatalogFallbackEnabled = input.Music.CatalogFallbackEnabled
	candidate.Music.AudioAnalysisEnabled = input.Music.AudioAnalysisEnabled
	candidate.Music.GeneratedArtworkEnabled = input.Music.GeneratedArtworkEnabled
	themeMusic, err := normalizeThemeMusicSettings(input.ThemeMusic)
	if err != nil {
		return domain.ServerSettings{}, err
	}
	candidate.ThemeMusic = themeMusic
	candidate.Subtitles.PreferredLanguages = preferred
	candidate.Quality = quality
	if input.Metadata.ClearTMDbToken {
		candidate.Metadata.TMDbToken = ""
	} else {
		applySecret(&candidate.Metadata.TMDbToken, input.Metadata.TMDbToken, false)
	}
	applySecret(&candidate.Music.AcoustIDAPIKey, input.Music.AcoustIDAPIKey, input.Music.ClearAcoustIDAPIKey)
	applySecret(&candidate.Subtitles.OpenSubtitlesAPIKey, input.Subtitles.OpenSubtitlesAPIKey, input.Subtitles.ClearOpenSubtitlesAPIKey)
	applySecret(&candidate.Subtitles.OpenSubtitlesUsername, input.Subtitles.OpenSubtitlesUsername, input.Subtitles.ClearOpenSubtitlesUsername)
	applySecret(&candidate.Subtitles.OpenSubtitlesPassword, input.Subtitles.OpenSubtitlesPassword, input.Subtitles.ClearOpenSubtitlesPassword)
	return candidate, nil
}

func normalizeThemeMusicSettings(value domain.ThemeMusicSettings) (domain.ThemeMusicSettings, error) {
	value.SeriesPlayback = domain.ThemePlaybackMode(strings.ToLower(strings.TrimSpace(string(value.SeriesPlayback))))
	if value.SeriesPlayback == "" {
		value.SeriesPlayback = domain.ThemePlaybackAlways
	}
	switch value.SeriesPlayback {
	case domain.ThemePlaybackAlways, domain.ThemePlaybackFirstOpen, domain.ThemePlaybackOff:
		return value, nil
	default:
		return value, fmt.Errorf("%w: tundmatu seriaali tunnusmuusika esitamise režiim", domain.ErrInvalidInput)
	}
}

func normalizeQualitySettings(value domain.QualitySettings) (domain.QualitySettings, error) {
	value.Profile = strings.ToLower(strings.TrimSpace(value.Profile))
	if value.Profile == "" {
		value.Profile = "original"
	}
	switch value.Profile {
	case "original", "maximum", "balanced", "data-saver":
	default:
		return value, fmt.Errorf("%w: tundmatu pildikvaliteedi profiil", domain.ErrInvalidInput)
	}
	value.PreferredCodec = strings.ToLower(strings.TrimSpace(value.PreferredCodec))
	if value.PreferredCodec == "h265" {
		value.PreferredCodec = "hevc"
	}
	if value.PreferredCodec == "" {
		value.PreferredCodec = "auto"
	}
	switch value.PreferredCodec {
	case "auto", "h264", "hevc", "av1":
	default:
		return value, fmt.Errorf("%w: tundmatu väljundcodec", domain.ErrInvalidInput)
	}
	value.HardwarePreference = strings.ToLower(strings.TrimSpace(value.HardwarePreference))
	if value.HardwarePreference == "" {
		value.HardwarePreference = "auto"
	}
	switch value.HardwarePreference {
	case "auto", "software", "qsv", "nvenc", "amf":
	default:
		return value, fmt.Errorf("%w: tundmatu hardware encoder", domain.ErrInvalidInput)
	}
	value.CachePolicy = strings.ToLower(strings.TrimSpace(value.CachePolicy))
	if value.CachePolicy == "" {
		value.CachePolicy = "watched"
	}
	switch value.CachePolicy {
	case "off", "watched", "complete", "finish":
	default:
		return value, fmt.Errorf("%w: tundmatu optimeeritud cache'i poliitika", domain.ErrInvalidInput)
	}
	if value.CacheMaxGB == 0 {
		value.CacheMaxGB = 50
	}
	if value.CacheMaxGB < 1 || value.CacheMaxGB > 4096 {
		return value, fmt.Errorf("%w: cache'i maht peab olema 1–4096 GB", domain.ErrInvalidInput)
	}
	if value.CacheFreeReserveGB == 0 {
		value.CacheFreeReserveGB = 10
	}
	if value.CacheFreeReserveGB < 1 || value.CacheFreeReserveGB > 1024 {
		return value, fmt.Errorf("%w: vaba kettaruumi reserv peab olema 1–1024 GB", domain.ErrInvalidInput)
	}
	if value.CacheFreeReservePercent == 0 {
		value.CacheFreeReservePercent = 10
	}
	if value.CacheFreeReservePercent < 1 || value.CacheFreeReservePercent > 50 {
		return value, fmt.Errorf("%w: vaba kettaruumi reserv peab olema 1–50%%", domain.ErrInvalidInput)
	}
	if value.GracePeriodSeconds < 0 || value.GracePeriodSeconds > 300 {
		return value, fmt.Errorf("%w: grace-periood peab olema 0–300 sekundit", domain.ErrInvalidInput)
	}
	if value.VMAFTarget == 0 {
		value.VMAFTarget = 95
	}
	if value.VMAFTarget < 70 || value.VMAFTarget > 100 {
		return value, fmt.Errorf("%w: VMAF siht peab olema 70–100", domain.ErrInvalidInput)
	}
	if value.VMAFSampleSeconds == 0 {
		value.VMAFSampleSeconds = 12
	}
	if value.VMAFSampleSeconds < 4 || value.VMAFSampleSeconds > 60 {
		return value, fmt.Errorf("%w: VMAF proov peab olema 4–60 sekundit", domain.ErrInvalidInput)
	}
	return value, nil
}

func applySecret(target *string, incoming *string, clear bool) {
	if clear {
		*target = ""
		return
	}
	if incoming != nil {
		if value := strings.TrimSpace(*incoming); value != "" {
			*target = value
		}
	}
}

func normalizeLanguage(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%w: metaandmete keel on kohustuslik", domain.ErrInvalidInput)
	}
	if !languagePattern.MatchString(value) {
		return "", fmt.Errorf("%w: kasuta keelekuju et või et-EE", domain.ErrInvalidInput)
	}
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == '-' || r == '_' })
	parts[0] = strings.ToLower(parts[0])
	if len(parts) == 2 {
		parts[1] = strings.ToUpper(parts[1])
	}
	return strings.Join(parts, "-"), nil
}

func normalizeSubtitleLanguages(values []string) ([]string, error) {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, raw := range values {
		for _, value := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
			value = strings.ToLower(strings.TrimSpace(value))
			if !shortLanguagePattern.MatchString(value) {
				return nil, fmt.Errorf("%w: subtiitrite keel %q peab olema 2–3 täheline kood", domain.ErrInvalidInput, value)
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	if len(result) == 0 {
		result = []string{domain.DefaultSubtitleLanguage}
	}
	if len(result) > 12 {
		return nil, fmt.Errorf("%w: eeliskeeli saab olla kuni 12", domain.ErrInvalidInput)
	}
	return result, nil
}

func publicSettings(settings domain.ServerSettings) domain.PublicServerSettings {
	token := strings.TrimSpace(settings.Metadata.TMDbToken)
	source := strings.TrimSpace(settings.Metadata.IMDbRatingsSource)
	if source == "" {
		source = domain.DefaultIMDbRatingsSource
	}
	return domain.PublicServerSettings{
		FFmpegPath:  settings.FFmpegPath,
		FFprobePath: settings.FFprobePath,
		Metadata: domain.PublicMetadataSettings{
			TMDbTokenConfigured: token != "",
			TMDbTokenHint:       secretHint(token),
			Language:            settings.Metadata.Language,
			IMDbRatingsSource:   source,
		},
		ThemeMusic: settings.ThemeMusic,
		Music: domain.PublicMusicSettings{
			MetadataEnabled:          settings.Music.MetadataEnabled,
			ShazamEnabled:            settings.Music.ShazamEnabled,
			FingerprintEnabled:       settings.Music.FingerprintEnabled,
			CatalogFallbackEnabled:   settings.Music.CatalogFallbackEnabled,
			AudioAnalysisEnabled:     settings.Music.AudioAnalysisEnabled,
			GeneratedArtworkEnabled:  settings.Music.GeneratedArtworkEnabled,
			AcoustIDAPIKeyConfigured: strings.TrimSpace(settings.Music.AcoustIDAPIKey) != "",
			AcoustIDAPIKeyHint:       secretHint(settings.Music.AcoustIDAPIKey),
		},
		Subtitles: domain.PublicSubtitleSettings{
			PreferredLanguages:              append([]string(nil), settings.Subtitles.PreferredLanguages...),
			OpenSubtitlesAPIKeyConfigured:   strings.TrimSpace(settings.Subtitles.OpenSubtitlesAPIKey) != "",
			OpenSubtitlesAPIKeyHint:         secretHint(settings.Subtitles.OpenSubtitlesAPIKey),
			OpenSubtitlesUsernameConfigured: strings.TrimSpace(settings.Subtitles.OpenSubtitlesUsername) != "",
			OpenSubtitlesUsernameHint:       usernameHint(settings.Subtitles.OpenSubtitlesUsername),
			OpenSubtitlesPasswordConfigured: strings.TrimSpace(settings.Subtitles.OpenSubtitlesPassword) != "",
		},
		Quality: settings.Quality,
	}
}

func secretHint(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	visible := value
	if len(visible) > 4 {
		visible = visible[len(visible)-4:]
	}
	return "••••" + visible
}

func usernameHint(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 2 {
		return value[:1] + "…"
	}
	return value[:1] + "…" + value[len(value)-1:]
}
