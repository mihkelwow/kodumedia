package ports

import (
	"context"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type ProbeResult struct {
	Technical domain.TechnicalMetadata
	Tags      map[string]string
}

type MediaProbe interface {
	Available(ctx context.Context) bool
	Probe(ctx context.Context, path string) (ProbeResult, error)
}
