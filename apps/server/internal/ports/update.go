package ports

import (
	"context"
	"io"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type UpdateInstaller interface {
	Status() domain.UpdateStatus
	Install(ctx context.Context, packageReader io.Reader, packageName string) (domain.UpdateStatus, error)
}
