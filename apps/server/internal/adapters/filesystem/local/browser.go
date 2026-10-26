package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

const (
	listingCacheTTL = 2 * time.Second
	maxEntries      = 1000
)

type cachedListing struct {
	expires time.Time
	value   domain.DirectoryListing
}

type Browser struct {
	mu    sync.Mutex
	cache map[string]cachedListing
	now   func() time.Time
}

func New() *Browser {
	return &Browser{cache: make(map[string]cachedListing), now: time.Now}
}

func (b *Browser) Browse(ctx context.Context, path string, includeFiles bool, extensions []string) (domain.DirectoryListing, error) {
	path = strings.TrimSpace(path)
	extensions = normalizeExtensions(extensions)
	key := cacheKey(path, includeFiles, extensions)
	if listing, found := b.cached(key); found {
		return listing, nil
	}

	roots := availableRoots()
	if path == "" {
		listing := domain.DirectoryListing{Roots: roots, Entries: []domain.FileSystemEntry{}}
		b.store(key, listing)
		return listing, nil
	}
	if !filepath.IsAbs(path) {
		return domain.DirectoryListing{}, fmt.Errorf("%w: kaustatee peab olema absoluutne", domain.ErrInvalidInput)
	}
	clean := filepath.Clean(path)
	info, err := os.Stat(clean)
	if err != nil {
		return domain.DirectoryListing{}, fmt.Errorf("%w: kausta ei saanud avada: %v", domain.ErrInvalidInput, err)
	}
	if !info.IsDir() {
		return domain.DirectoryListing{}, fmt.Errorf("%w: valitud tee ei ole kaust", domain.ErrInvalidInput)
	}
	directory, err := os.Open(clean)
	if err != nil {
		return domain.DirectoryListing{}, fmt.Errorf("%w: kausta sisu ei saanud lugeda: %v", domain.ErrInvalidInput, err)
	}
	defer directory.Close()

	// Read in small native batches and stop after the first extra matching
	// entry. os.ReadDir(path) reads and sorts the whole directory up front,
	// which is needlessly expensive for large media or network folders.
	result := make([]domain.FileSystemEntry, 0, min(128, maxEntries))
	truncated := false
readLoop:
	for {
		entries, readErr := directory.ReadDir(256)
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return domain.DirectoryListing{}, err
			}
			name := entry.Name()
			if shouldHideSystemEntry(name) {
				continue
			}
			isDirectory := entry.IsDir()
			if !isDirectory && !includeFiles {
				continue
			}
			if !isDirectory && len(extensions) > 0 && !extensionAllowed(name, extensions) {
				continue
			}
			if len(result) >= maxEntries {
				truncated = true
				break readLoop
			}
			item := domain.FileSystemEntry{Name: name, Path: filepath.Join(clean, name), Directory: isDirectory}
			if !isDirectory {
				if itemInfo, infoErr := entry.Info(); infoErr == nil {
					item.SizeBytes = itemInfo.Size()
				}
			}
			result = append(result, item)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return domain.DirectoryListing{}, fmt.Errorf("%w: kausta sisu ei saanud lugeda: %v", domain.ErrInvalidInput, readErr)
		}
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].Directory != result[right].Directory {
			return result[left].Directory
		}
		return strings.ToLower(result[left].Name) < strings.ToLower(result[right].Name)
	})
	listing := domain.DirectoryListing{CurrentPath: clean, ParentPath: parentPath(clean), Roots: roots, Entries: result, Truncated: truncated}
	b.store(key, listing)
	return listing, nil
}

func (b *Browser) cached(key string) (domain.DirectoryListing, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	entry, found := b.cache[key]
	if !found || b.now().After(entry.expires) {
		if found {
			delete(b.cache, key)
		}
		return domain.DirectoryListing{}, false
	}
	return cloneListing(entry.value), true
}

func (b *Browser) store(key string, listing domain.DirectoryListing) {
	b.mu.Lock()
	if len(b.cache) > 64 {
		b.cache = make(map[string]cachedListing)
	}
	b.cache[key] = cachedListing{expires: b.now().Add(listingCacheTTL), value: cloneListing(listing)}
	b.mu.Unlock()
}

func cloneListing(value domain.DirectoryListing) domain.DirectoryListing {
	value.Roots = append([]domain.FileSystemEntry(nil), value.Roots...)
	value.Entries = append([]domain.FileSystemEntry(nil), value.Entries...)
	return value
}

func cacheKey(path string, includeFiles bool, extensions []string) string {
	return strings.ToLower(filepath.Clean(path)) + "|" + fmt.Sprint(includeFiles) + "|" + strings.Join(extensions, ",")
}

func parentPath(path string) string {
	volume := filepath.VolumeName(path)
	root := volume + string(filepath.Separator)
	if volume == "" {
		root = string(filepath.Separator)
	}
	if samePath(path, root) {
		return ""
	}
	parent := filepath.Dir(path)
	if samePath(parent, path) {
		return ""
	}
	return parent
}

func normalizeExtensions(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if !strings.HasPrefix(value, ".") {
			value = "." + value
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func extensionAllowed(name string, extensions []string) bool {
	extension := strings.ToLower(filepath.Ext(name))
	for _, allowed := range extensions {
		if extension == allowed {
			return true
		}
	}
	return false
}

func shouldHideSystemEntry(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "$recycle.bin", "system volume information":
		return true
	default:
		return false
	}
}

func samePath(left, right string) bool {
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}

var _ ports.DirectoryBrowser = (*Browser)(nil)
