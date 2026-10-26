package ports

import (
	"context"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type SeriesThemeProvider interface {
	LookupSeriesTheme(ctx context.Context, tmdbID string) (domain.ThemeSource, error)
}

type ThemeAudioDownloader interface {
	DownloadAudio(ctx context.Context, sourceURL, destination string) error
}
