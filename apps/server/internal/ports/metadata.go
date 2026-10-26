package ports

import (
	"context"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type MetadataProvider interface {
	Name() string
	Available() bool
	Match(ctx context.Context, query domain.MetadataMatchQuery) (domain.MetadataResult, error)
}

type RatingsProvider interface {
	Name() string
	Available() bool
	Lookup(ctx context.Context, externalID string) (rating float64, votes int64, found bool, err error)
}

type RatingsDataset interface {
	LocalPath() string
	Status() domain.ComponentStatus
	Inspect(ctx context.Context, source string) domain.ComponentStatus
	Refresh(ctx context.Context, progress JobProgress) error
}
