package ports

import (
	"context"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type Catalog interface {
	ListLibraries(ctx context.Context) ([]domain.Library, error)
	GetLibrary(ctx context.Context, id string) (domain.Library, error)
	CreateLibrary(ctx context.Context, library domain.Library) error
	UpdateLibrary(ctx context.Context, library domain.Library) error
	DeleteLibrary(ctx context.Context, id string) error

	ListMedia(ctx context.Context, query domain.MediaQuery) (domain.MediaPage, error)
	GetMedia(ctx context.Context, id string) (domain.MediaItem, error)
	ReplaceLibraryMedia(ctx context.Context, libraryID string, items []domain.MediaItem) error
	UpdateMedia(ctx context.Context, item domain.MediaItem) error
	UpdateMediaBatch(ctx context.Context, items []domain.MediaItem) error
}
