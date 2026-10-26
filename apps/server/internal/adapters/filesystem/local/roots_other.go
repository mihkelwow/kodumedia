//go:build !windows

package local

import "github.com/mihkelwow/kodumedia/internal/domain"

func availableRoots() []domain.FileSystemEntry {
	return []domain.FileSystemEntry{{Name: "/", Path: "/", Directory: true}}
}
