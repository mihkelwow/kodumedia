package ports

import (
	"context"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type SettingsRepository interface {
	Current() domain.ServerSettings
	Update(ctx context.Context, settings domain.ServerSettings) error
}

type SettingsInspector interface {
	Inspect(ctx context.Context, settings domain.ServerSettings) domain.SystemStatus
}
