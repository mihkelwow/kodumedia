package domain

import "time"

// WatchState is a lightweight, single-profile playback checkpoint. It is
// stored separately from the media catalog so rescans and application updates
// cannot erase viewing history.
type WatchState struct {
	MediaID       string    `json:"mediaId"`
	SeriesID      string    `json:"seriesId,omitempty"`
	MediaType     MediaType `json:"mediaType"`
	Season        int       `json:"season,omitempty"`
	Episode       int       `json:"episode,omitempty"`
	PositionMS    int64     `json:"positionMs"`
	DurationMS    int64     `json:"durationMs"`
	Progress      float64   `json:"progress"`
	Completed     bool      `json:"completed"`
	LastWatchedAt time.Time `json:"lastWatchedAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type WatchUpdate struct {
	PositionMS int64 `json:"positionMs"`
	DurationMS int64 `json:"durationMs"`
	Completed  bool  `json:"completed,omitempty"`
}
