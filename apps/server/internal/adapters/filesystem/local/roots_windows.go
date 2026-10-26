//go:build windows

package local

import (
	"fmt"
	"syscall"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

var getLogicalDrives = syscall.NewLazyDLL("kernel32.dll").NewProc("GetLogicalDrives")

func availableRoots() []domain.FileSystemEntry {
	mask, _, _ := getLogicalDrives.Call()
	roots := make([]domain.FileSystemEntry, 0, 8)
	for index := 0; index < 26; index++ {
		if mask&(1<<index) == 0 {
			continue
		}
		letter := rune('A' + index)
		path := fmt.Sprintf("%c:\\", letter)
		roots = append(roots, domain.FileSystemEntry{Name: fmt.Sprintf("%c:", letter), Path: path, Directory: true})
	}
	return roots
}
