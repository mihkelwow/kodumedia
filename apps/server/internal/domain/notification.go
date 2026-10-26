package domain

import "time"

type NotificationKind string

const NotificationKindSeriesEpisodeReleased NotificationKind = "series_episode_released"

type Notification struct {
	ID           string           `json:"id"`
	Kind         NotificationKind `json:"kind"`
	SeriesID     string           `json:"seriesId"`
	SeriesTitle  string           `json:"seriesTitle"`
	Season       int              `json:"season"`
	Episode      int              `json:"episode"`
	EpisodeTitle string           `json:"episodeTitle,omitempty"`
	ReleaseDate  *time.Time       `json:"releaseDate,omitempty"`
	CreatedAt    time.Time        `json:"createdAt"`
	ReadAt       *time.Time       `json:"readAt,omitempty"`
	ResolvedAt   *time.Time       `json:"resolvedAt,omitempty"`
}

type NotificationPage struct {
	Items  []Notification `json:"items"`
	Total  int            `json:"total"`
	Unread int            `json:"unread"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}
