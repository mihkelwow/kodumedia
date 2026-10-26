package domain

import "time"

const (
	DefaultIMDbRatingsSource = "https://datasets.imdbws.com/title.ratings.tsv.gz"
	DefaultMetadataLanguage  = "en-US"
	DefaultSubtitleLanguage  = "en"
)

// ServerSettings contains settings that can be changed while the server is
// running. Environment variables are used only as first-run defaults.
type ServerSettings struct {
	FFmpegPath  string             `json:"ffmpegPath"`
	FFprobePath string             `json:"ffprobePath"`
	Metadata    MetadataSettings   `json:"metadata"`
	Music       MusicSettings      `json:"music"`
	ThemeMusic  ThemeMusicSettings `json:"themeMusic"`
	Subtitles   SubtitleSettings   `json:"subtitles"`
	Quality     QualitySettings    `json:"quality"`
}

type QualitySettings struct {
	Profile                 string `json:"profile"`
	PreferredCodec          string `json:"preferredCodec"`
	HardwarePreference      string `json:"hardwarePreference"`
	CachePolicy             string `json:"cachePolicy"`
	CacheMaxGB              int    `json:"cacheMaxGb"`
	CacheFreeReserveGB      int    `json:"cacheFreeReserveGb"`
	CacheFreeReservePercent int    `json:"cacheFreeReservePercent"`
	GracePeriodSeconds      int    `json:"gracePeriodSeconds"`
	VMAFEnabled             bool   `json:"vmafEnabled"`
	VMAFTarget              int    `json:"vmafTarget"`
	VMAFSampleSeconds       int    `json:"vmafSampleSeconds"`
}

type MetadataSettings struct {
	TMDbToken         string `json:"tmdbToken"`
	Language          string `json:"language"`
	IMDbRatingsSource string `json:"imdbRatingsSource"`
}

type MusicSettings struct {
	MetadataEnabled         bool   `json:"metadataEnabled"`
	ShazamEnabled           bool   `json:"shazamEnabled"`
	FingerprintEnabled      bool   `json:"fingerprintEnabled"`
	CatalogFallbackEnabled  bool   `json:"catalogFallbackEnabled"`
	AudioAnalysisEnabled    bool   `json:"audioAnalysisEnabled"`
	GeneratedArtworkEnabled bool   `json:"generatedArtworkEnabled"`
	AcoustIDAPIKey          string `json:"acoustIdApiKey"`
}

type SubtitleSettings struct {
	PreferredLanguages    []string `json:"preferredLanguages"`
	OpenSubtitlesAPIKey   string   `json:"openSubtitlesApiKey"`
	OpenSubtitlesUsername string   `json:"openSubtitlesUsername"`
	OpenSubtitlesPassword string   `json:"openSubtitlesPassword"`
}

type ServerSettingsUpdate struct {
	FFmpegPath  string                 `json:"ffmpegPath"`
	FFprobePath string                 `json:"ffprobePath"`
	Metadata    MetadataSettingsUpdate `json:"metadata"`
	Music       MusicSettingsUpdate    `json:"music"`
	ThemeMusic  ThemeMusicSettings     `json:"themeMusic"`
	Subtitles   SubtitleSettingsUpdate `json:"subtitles"`
	Quality     QualitySettings        `json:"quality"`
}

type MetadataSettingsUpdate struct {
	TMDbToken         *string `json:"tmdbToken,omitempty"`
	ClearTMDbToken    bool    `json:"clearTmdbToken,omitempty"`
	Language          string  `json:"language"`
	IMDbRatingsSource string  `json:"imdbRatingsSource"`
}

type MusicSettingsUpdate struct {
	MetadataEnabled         bool    `json:"metadataEnabled"`
	ShazamEnabled           bool    `json:"shazamEnabled"`
	FingerprintEnabled      bool    `json:"fingerprintEnabled"`
	CatalogFallbackEnabled  bool    `json:"catalogFallbackEnabled"`
	AudioAnalysisEnabled    bool    `json:"audioAnalysisEnabled"`
	GeneratedArtworkEnabled bool    `json:"generatedArtworkEnabled"`
	AcoustIDAPIKey          *string `json:"acoustIdApiKey,omitempty"`
	ClearAcoustIDAPIKey     bool    `json:"clearAcoustIdApiKey,omitempty"`
}

type SubtitleSettingsUpdate struct {
	PreferredLanguages         []string `json:"preferredLanguages"`
	OpenSubtitlesAPIKey        *string  `json:"openSubtitlesApiKey,omitempty"`
	ClearOpenSubtitlesAPIKey   bool     `json:"clearOpenSubtitlesApiKey,omitempty"`
	OpenSubtitlesUsername      *string  `json:"openSubtitlesUsername,omitempty"`
	ClearOpenSubtitlesUsername bool     `json:"clearOpenSubtitlesUsername,omitempty"`
	OpenSubtitlesPassword      *string  `json:"openSubtitlesPassword,omitempty"`
	ClearOpenSubtitlesPassword bool     `json:"clearOpenSubtitlesPassword,omitempty"`
}

type PublicServerSettings struct {
	FFmpegPath  string                 `json:"ffmpegPath"`
	FFprobePath string                 `json:"ffprobePath"`
	Metadata    PublicMetadataSettings `json:"metadata"`
	Music       PublicMusicSettings    `json:"music"`
	ThemeMusic  ThemeMusicSettings     `json:"themeMusic"`
	Subtitles   PublicSubtitleSettings `json:"subtitles"`
	Quality     QualitySettings        `json:"quality"`
}

type PublicMetadataSettings struct {
	TMDbTokenConfigured bool   `json:"tmdbTokenConfigured"`
	TMDbTokenHint       string `json:"tmdbTokenHint,omitempty"`
	Language            string `json:"language"`
	IMDbRatingsSource   string `json:"imdbRatingsSource"`
}

type PublicMusicSettings struct {
	MetadataEnabled          bool   `json:"metadataEnabled"`
	ShazamEnabled            bool   `json:"shazamEnabled"`
	FingerprintEnabled       bool   `json:"fingerprintEnabled"`
	CatalogFallbackEnabled   bool   `json:"catalogFallbackEnabled"`
	AudioAnalysisEnabled     bool   `json:"audioAnalysisEnabled"`
	GeneratedArtworkEnabled  bool   `json:"generatedArtworkEnabled"`
	AcoustIDAPIKeyConfigured bool   `json:"acoustIdApiKeyConfigured"`
	AcoustIDAPIKeyHint       string `json:"acoustIdApiKeyHint,omitempty"`
}

type PublicSubtitleSettings struct {
	PreferredLanguages              []string `json:"preferredLanguages"`
	OpenSubtitlesAPIKeyConfigured   bool     `json:"openSubtitlesApiKeyConfigured"`
	OpenSubtitlesAPIKeyHint         string   `json:"openSubtitlesApiKeyHint,omitempty"`
	OpenSubtitlesUsernameConfigured bool     `json:"openSubtitlesUsernameConfigured"`
	OpenSubtitlesUsernameHint       string   `json:"openSubtitlesUsernameHint,omitempty"`
	OpenSubtitlesPasswordConfigured bool     `json:"openSubtitlesPasswordConfigured"`
}

type ComponentStatus struct {
	Available     bool       `json:"available"`
	Refreshing    bool       `json:"refreshing,omitempty"`
	Version       string     `json:"version,omitempty"`
	Message       string     `json:"message,omitempty"`
	LastCheckedAt *time.Time `json:"lastCheckedAt,omitempty"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
	SizeBytes     int64      `json:"sizeBytes,omitempty"`
}

type MetadataStatus struct {
	TMDb ComponentStatus `json:"tmdb"`
	IMDb ComponentStatus `json:"imdb"`
}

type SystemStatus struct {
	FFmpeg   ComponentStatus `json:"ffmpeg"`
	FFprobe  ComponentStatus `json:"ffprobe"`
	Metadata MetadataStatus  `json:"metadata"`
}

type SystemConfiguration struct {
	Version  string               `json:"version"`
	Settings PublicServerSettings `json:"settings"`
	Status   SystemStatus         `json:"status"`
}
