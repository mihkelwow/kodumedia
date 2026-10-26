package ports

import (
	"context"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type DirectoryBrowser interface {
	Browse(ctx context.Context, path string, includeFiles bool, extensions []string) (domain.DirectoryListing, error)
}

type FilePicker interface {
	Pick(ctx context.Context, request domain.FilePickerRequest) (path string, canceled bool, err error)
}
