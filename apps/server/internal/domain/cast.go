package domain

import "time"

type CastProtocol string

const (
	CastProtocolDLNA       CastProtocol = "dlna"
	CastProtocolGoogleCast CastProtocol = "google-cast"
)

type CastDevice struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Manufacturer string       `json:"manufacturer,omitempty"`
	Model        string       `json:"model,omitempty"`
	Protocol     CastProtocol `json:"protocol"`
	CanSeek      bool         `json:"canSeek"`
	CanSetVolume bool         `json:"canSetVolume"`
}

type CastPlaybackState string

const (
	CastStateConnecting CastPlaybackState = "connecting"
	CastStatePlaying    CastPlaybackState = "playing"
	CastStatePaused     CastPlaybackState = "paused"
	CastStateStopped    CastPlaybackState = "stopped"
	CastStateBuffering  CastPlaybackState = "buffering"
	CastStateUnknown    CastPlaybackState = "unknown"
)

type CastDeviceStatus struct {
	State      CastPlaybackState `json:"state"`
	PositionMS int64             `json:"positionMs,omitempty"`
	DurationMS int64             `json:"durationMs,omitempty"`
	Volume     float64           `json:"volume,omitempty"`
	Muted      bool              `json:"muted,omitempty"`
}

type CastSource struct {
	ID                string       `json:"id"`
	MediaID           string       `json:"mediaId"`
	URL               string       `json:"url"`
	ContentType       string       `json:"contentType"`
	PlaybackSessionID string       `json:"-"`
	BasePositionMS    int64        `json:"basePositionMs,omitempty"`
	StartPositionMS   int64        `json:"startPositionMs,omitempty"`
	DurationMS        int64        `json:"durationMs,omitempty"`
	Plan              PlaybackPlan `json:"plan"`
	CreatedAt         time.Time    `json:"createdAt"`
}

type CastSession struct {
	ID           string            `json:"id"`
	Device       CastDevice        `json:"device"`
	MediaID      string            `json:"mediaId"`
	SourceID     string            `json:"sourceId"`
	State        CastPlaybackState `json:"state"`
	PositionMS   int64             `json:"positionMs,omitempty"`
	DurationMS   int64             `json:"durationMs,omitempty"`
	Volume       float64           `json:"volume,omitempty"`
	Muted        bool              `json:"muted,omitempty"`
	StartedAt    time.Time         `json:"startedAt"`
	LastUpdateAt time.Time         `json:"lastUpdateAt"`
}

type CastMediaMetadata struct {
	Title       string
	Subtitle    string
	ContentType string
}

type CastCommandAction string

const (
	CastCommandPlay      CastCommandAction = "play"
	CastCommandPause     CastCommandAction = "pause"
	CastCommandStop      CastCommandAction = "stop"
	CastCommandSeek      CastCommandAction = "seek"
	CastCommandSetVolume CastCommandAction = "set-volume"
	CastCommandSetMuted  CastCommandAction = "set-muted"
)

type CastCommand struct {
	Action     CastCommandAction `json:"action"`
	PositionMS int64             `json:"positionMs,omitempty"`
	Volume     float64           `json:"volume,omitempty"`
	Muted      bool              `json:"muted,omitempty"`
}
