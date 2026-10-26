package ports

import (
	"context"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type CastLoadRequest struct {
	MediaURL   string
	Metadata   domain.CastMediaMetadata
	PositionMS int64
}

type CastController interface {
	Discover(ctx context.Context) ([]domain.CastDevice, error)
	ReachableMediaURL(ctx context.Context, deviceID, mediaURL string) (string, error)
	Load(ctx context.Context, deviceID string, request CastLoadRequest) error
	Play(ctx context.Context, deviceID string) error
	Pause(ctx context.Context, deviceID string) error
	Stop(ctx context.Context, deviceID string) error
	Seek(ctx context.Context, deviceID string, positionMS int64) error
	SetVolume(ctx context.Context, deviceID string, volume float64) error
	SetMuted(ctx context.Context, deviceID string, muted bool) error
	Status(ctx context.Context, deviceID string) (domain.CastDeviceStatus, error)
}
