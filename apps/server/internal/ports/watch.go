package ports

import (
	"context"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type WatchStore interface {
	Get(ctx context.Context, mediaID string) (domain.WatchState, error)
	List(ctx context.Context) ([]domain.WatchState, error)
	Upsert(ctx context.Context, state domain.WatchState) error
	Delete(ctx context.Context, mediaID string) error
}
