package ports

import (
	"context"
	"io"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type SubtitleStore interface {
	Discover(ctx context.Context, media domain.MediaItem) ([]domain.SubtitleTrack, error)
	Resolve(ctx context.Context, media domain.MediaItem, id string) (domain.SubtitleTrack, error)
	WebVTT(ctx context.Context, media domain.MediaItem, track domain.SubtitleTrack) (string, error)
	SaveDownloaded(ctx context.Context, media domain.MediaItem, fileName string, content io.Reader) (domain.SubtitleTrack, error)
}

type OnlineSubtitleProvider interface {
	Available() bool
	Search(ctx context.Context, media domain.MediaItem, languages []string) ([]domain.SubtitleSearchResult, error)
	Download(ctx context.Context, fileID int64) (io.ReadCloser, string, error)
}
