package ports

import (
	"context"
	"io/fs"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type TranscodeStartRequest struct {
	Media             domain.MediaItem
	Plan              domain.PlaybackPlan
	Preferences       domain.PlaybackPreferences
	Subtitle          *domain.SubtitleTrack
	CacheResumeFromMS int64
	CacheStartNumber  int
	CacheAppend       bool
}

type TranscodeSessionManager interface {
	Available(ctx context.Context) bool
	Start(ctx context.Context, request TranscodeStartRequest) (domain.PlaybackSession, error)
	Open(ctx context.Context, sessionID, relativePath string) (fs.File, error)
	Touch(sessionID string)
	Stats(sessionID string) (domain.PlaybackRuntimeStats, error)
	Stop(sessionID string) error
	Close() error
}
