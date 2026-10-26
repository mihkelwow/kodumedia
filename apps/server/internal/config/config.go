package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type Config struct {
	Address          string
	DataDir          string
	InitialSettings  domain.ServerSettings
	AllowedOrigins   []string
	TranscodeIdleTTL time.Duration
	YTDLPPath        string
}

func Load() Config {
	loadDotEnvFromExecutableDir()
	dataDir := env("MEDIA_SERVER_DATA_DIR", defaultDataDir())
	if absolute, err := filepath.Abs(dataDir); err == nil {
		dataDir = absolute
	}

	initial := domain.ServerSettings{
		FFmpegPath:  env("MEDIA_SERVER_FFMPEG", "ffmpeg"),
		FFprobePath: env("MEDIA_SERVER_FFPROBE", "ffprobe"),
		Metadata: domain.MetadataSettings{
			TMDbToken:         strings.TrimSpace(os.Getenv("TMDB_TOKEN")),
			Language:          env("MEDIA_SERVER_METADATA_LANGUAGE", domain.DefaultMetadataLanguage),
			IMDbRatingsSource: env("IMDB_RATINGS_SOURCE", firstNonEmpty(strings.TrimSpace(os.Getenv("IMDB_RATINGS_PATH")), domain.DefaultIMDbRatingsSource)),
		},
		ThemeMusic: domain.ThemeMusicSettings{SeriesPlayback: domain.ThemePlaybackAlways},
		Music: domain.MusicSettings{
			MetadataEnabled:         strings.EqualFold(env("MUSIC_METADATA_ENABLED", "true"), "true"),
			ShazamEnabled:           strings.EqualFold(env("MUSIC_SHAZAM_ENABLED", "true"), "true"),
			FingerprintEnabled:      strings.EqualFold(env("MUSIC_FINGERPRINT_ENABLED", "true"), "true"),
			CatalogFallbackEnabled:  strings.EqualFold(env("MUSIC_CATALOG_FALLBACK_ENABLED", "true"), "true"),
			AudioAnalysisEnabled:    strings.EqualFold(env("MUSIC_AUDIO_ANALYSIS_ENABLED", "true"), "true"),
			GeneratedArtworkEnabled: strings.EqualFold(env("MUSIC_GENERATED_ARTWORK_ENABLED", "true"), "true"),
			AcoustIDAPIKey:          strings.TrimSpace(os.Getenv("ACOUSTID_API_KEY")),
		},
		Subtitles: domain.SubtitleSettings{
			PreferredLanguages:    splitCSV(env("MEDIA_SERVER_SUBTITLE_LANGUAGES", domain.DefaultSubtitleLanguage)),
			OpenSubtitlesAPIKey:   strings.TrimSpace(os.Getenv("OPENSUBTITLES_API_KEY")),
			OpenSubtitlesUsername: strings.TrimSpace(os.Getenv("OPENSUBTITLES_USERNAME")),
			OpenSubtitlesPassword: strings.TrimSpace(os.Getenv("OPENSUBTITLES_PASSWORD")),
		},
		Quality: domain.QualitySettings{
			Profile: "original", PreferredCodec: "auto", HardwarePreference: "auto",
			CachePolicy: "watched", CacheMaxGB: 50, CacheFreeReserveGB: 10, CacheFreeReservePercent: 10, GracePeriodSeconds: 15,
			VMAFEnabled: true, VMAFTarget: 95, VMAFSampleSeconds: 12,
		},
	}
	initial = PreferBundledTools(initial)

	return Config{
		Address:          env("MEDIA_SERVER_ADDRESS", "0.0.0.0:8096"),
		DataDir:          dataDir,
		InitialSettings:  initial,
		AllowedOrigins:   splitCSV(os.Getenv("MEDIA_SERVER_ALLOWED_ORIGINS")),
		TranscodeIdleTTL: durationEnv("MEDIA_SERVER_TRANSCODE_IDLE_TTL", 2*time.Minute),
		YTDLPPath:        env("MEDIA_SERVER_YTDLP", firstNonEmpty(BundledYTDLPPath(), "yt-dlp")),
	}
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	if parsed, err := time.ParseDuration(value); err == nil {
		return parsed
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func defaultDataDir() string {
	executable, err := os.Executable()
	if err == nil {
		if absolute, absErr := filepath.Abs(executable); absErr == nil {
			return filepath.Join(filepath.Dir(absolute), "data")
		}
	}
	return "./data"
}

func loadDotEnvFromExecutableDir() {
	executable, err := os.Executable()
	if err != nil {
		return
	}
	file, err := os.Open(filepath.Join(filepath.Dir(executable), ".env"))
	if err != nil {
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		_ = os.Setenv(key, value)
	}
}
