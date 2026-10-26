package ports

import (
	"context"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type NotificationRepository interface {
	List(ctx context.Context) ([]domain.Notification, error)
	Replace(ctx context.Context, items []domain.Notification) error
	MarkAllRead(ctx context.Context, readAt time.Time) error
}
