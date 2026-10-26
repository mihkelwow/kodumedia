package domain

import "time"

type SeasonGroup struct {
	Number       int         `json:"number"`
	Title        string      `json:"title"`
	EpisodeCount int         `json:"episodeCount"`
	Episodes     []MediaItem `json:"episodes"`
}

type SeriesAvailability struct {
	ReleasedSeasonCount    int                 `json:"releasedSeasonCount"`
	ReleasedEpisodeCount   int                 `json:"releasedEpisodeCount"`
	AvailableSeasonCount   int                 `json:"availableSeasonCount"`
	AvailableEpisodeCount  int                 `json:"availableEpisodeCount"`
	LatestReleasedEpisode  *EpisodeReleaseInfo `json:"latestReleasedEpisode,omitempty"`
	LatestAvailableSeason  int                 `json:"latestAvailableSeason,omitempty"`
	LatestAvailableEpisode int                 `json:"latestAvailableEpisode,omitempty"`
	LatestAvailableDate    *time.Time          `json:"latestAvailableDate,omitempty"`
	NewEpisodeAvailable    bool                `json:"newEpisodeAvailable"`
}

type SeriesDetails struct {
	Series       MediaItem          `json:"series"`
	Seasons      []SeasonGroup      `json:"seasons"`
	PlayTarget   *MediaItem         `json:"playTarget,omitempty"`
	Availability SeriesAvailability `json:"availability"`
}
