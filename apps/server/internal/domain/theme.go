package domain

import "time"

type ThemePlaybackMode string

const (
	ThemePlaybackAlways    ThemePlaybackMode = "always"
	ThemePlaybackFirstOpen ThemePlaybackMode = "first-open"
	ThemePlaybackOff       ThemePlaybackMode = "off"
)

const (
	SeriesThemeVolume    = 0.15
	SeriesThemeFadeInMS  = 2000
	SeriesThemeFadeOutMS = 2000
)

type ThemeMusicSettings struct {
	SeriesPlayback ThemePlaybackMode `json:"seriesPlayback"`
}

type ThemePlaybackConfiguration struct {
	SeriesPlayback ThemePlaybackMode `json:"seriesPlayback"`
	Volume         float64           `json:"volume"`
	FadeInMS       int               `json:"fadeInMs"`
	FadeOutMS      int               `json:"fadeOutMs"`
}

type ThemeSource struct {
	Provider   string `json:"provider"`
	ExternalID string `json:"externalId"`
	URL        string `json:"url"`
}

type ThemeAsset struct {
	Path       string
	Name       string
	ModifiedAt time.Time
}
