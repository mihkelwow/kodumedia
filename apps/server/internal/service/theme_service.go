package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type themeFailure struct {
	err   error
	until time.Time
}

type themeCacheMetadata struct {
	SchemaVersion int       `json:"schemaVersion"`
	Provider      string    `json:"provider"`
	TMDbID        string    `json:"tmdbId"`
	SourceURL     string    `json:"sourceUrl"`
	CachedAt      time.Time `json:"cachedAt"`
}

type ThemeServiceOption func(*ThemeService)

// WithThemeBackgroundContext ties detached theme downloads to the server
// lifecycle instead of to the browser request that triggered them.
func WithThemeBackgroundContext(ctx context.Context) ThemeServiceOption {
	return func(service *ThemeService) {
		if ctx != nil {
			service.background = ctx
		}
	}
}

type ThemeService struct {
	catalog    ports.Catalog
	provider   ports.SeriesThemeProvider
	downloader ports.ThemeAudioDownloader
	cacheRoot  string
	settings   func() domain.ThemeMusicSettings
	background context.Context

	stateMu   sync.Mutex
	preparing map[string]struct{}
	failures  map[string]themeFailure
}

func NewThemeService(
	catalog ports.Catalog,
	provider ports.SeriesThemeProvider,
	downloader ports.ThemeAudioDownloader,
	cacheRoot string,
	settings func() domain.ThemeMusicSettings,
	options ...ThemeServiceOption,
) *ThemeService {
	service := &ThemeService{
		catalog:    catalog,
		provider:   provider,
		downloader: downloader,
		cacheRoot:  cacheRoot,
		settings:   settings,
		background: context.Background(),
		preparing:  map[string]struct{}{},
		failures:   map[string]themeFailure{},
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service
}

func (s *ThemeService) PlaybackConfiguration() domain.ThemePlaybackConfiguration {
	mode := domain.ThemePlaybackAlways
	if s.settings != nil {
		mode = s.settings().SeriesPlayback
	}
	switch mode {
	case domain.ThemePlaybackAlways, domain.ThemePlaybackFirstOpen, domain.ThemePlaybackOff:
	default:
		mode = domain.ThemePlaybackAlways
	}
	return domain.ThemePlaybackConfiguration{
		SeriesPlayback: mode,
		Volume:         domain.SeriesThemeVolume,
		FadeInMS:       domain.SeriesThemeFadeInMS,
		FadeOutMS:      domain.SeriesThemeFadeOutMS,
	}
}

// ResolveSeriesTheme returns a cached asset immediately. If the asset has not
// been cached yet, it starts one detached background preparation and returns
// ErrPending. Repeated requests share the same preparation job.
func (s *ThemeService) ResolveSeriesTheme(ctx context.Context, seriesID string) (domain.ThemeAsset, error) {
	if s == nil || s.catalog == nil || s.provider == nil || s.downloader == nil {
		return domain.ThemeAsset{}, fmt.Errorf("%w: theme music service is not configured", domain.ErrUnavailable)
	}
	if s.PlaybackConfiguration().SeriesPlayback == domain.ThemePlaybackOff {
		return domain.ThemeAsset{}, domain.ErrNotFound
	}
	series, err := s.catalog.GetMedia(ctx, strings.TrimSpace(seriesID))
	if err != nil {
		return domain.ThemeAsset{}, err
	}
	if series.Type != domain.MediaTypeSeries {
		return domain.ThemeAsset{}, fmt.Errorf("%w: media item is not a series", domain.ErrInvalidInput)
	}
	tmdbID, err := normalizeTMDbID(series.ExternalIDs.TMDb)
	if err != nil {
		return domain.ThemeAsset{}, err
	}
	path := filepath.Join(s.cacheRoot, tmdbID, "theme.m4a")
	if asset, ok := cachedThemeAsset(path); ok {
		now := time.Now()
		_ = os.Chtimes(path, now, now)
		return asset, nil
	}
	if failure, ok := s.activeFailure(tmdbID); ok {
		return domain.ThemeAsset{}, failure
	}
	s.startPreparation(tmdbID, path)
	return domain.ThemeAsset{}, domain.ErrPending
}

// Maintain removes themes for series that are no longer in the catalog and
// applies an LRU cap to the remaining cache.
func (s *ThemeService) Maintain(ctx context.Context, maximumBytes int64) int64 {
	if s == nil || s.catalog == nil {
		return 0
	}
	before := directorySizeForMaintenance(s.cacheRoot)
	valid := map[string]struct{}{}
	for offset := 0; ; offset += 500 {
		page, err := s.catalog.ListMedia(ctx, domain.MediaQuery{Types: []domain.MediaType{domain.MediaTypeSeries}, Limit: 500, Offset: offset})
		if err != nil {
			return 0
		}
		for _, item := range page.Items {
			if id, normalizeErr := normalizeTMDbID(item.ExternalIDs.TMDb); normalizeErr == nil {
				valid[id] = struct{}{}
			}
		}
		if offset+len(page.Items) >= page.Total {
			break
		}
	}
	entries, _ := os.ReadDir(s.cacheRoot)
	type candidate struct {
		path string
		size int64
		last time.Time
	}
	candidates := make([]candidate, 0, len(entries))
	var total int64
	for _, entry := range entries {
		path := filepath.Join(s.cacheRoot, entry.Name())
		if !entry.IsDir() {
			_ = os.Remove(path)
			continue
		}
		if _, keep := valid[entry.Name()]; !keep {
			_ = os.RemoveAll(path)
			continue
		}
		themePath := filepath.Join(path, "theme.m4a")
		info, err := os.Stat(themePath)
		if err != nil {
			_ = os.RemoveAll(path)
			continue
		}
		size := directorySizeForMaintenance(path)
		total += size
		candidates = append(candidates, candidate{path: path, size: size, last: info.ModTime()})
	}
	if maximumBytes > 0 && total > maximumBytes {
		sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].last.Before(candidates[j].last) })
		for _, item := range candidates {
			if total <= maximumBytes {
				break
			}
			if os.RemoveAll(item.path) == nil {
				total -= item.size
			}
		}
	}
	after := directorySizeForMaintenance(s.cacheRoot)
	if after >= before {
		return 0
	}
	return before - after
}

func (s *ThemeService) startPreparation(tmdbID, path string) {
	s.stateMu.Lock()
	if _, exists := s.preparing[tmdbID]; exists {
		s.stateMu.Unlock()
		return
	}
	if failure, exists := s.failures[tmdbID]; exists && time.Now().Before(failure.until) {
		s.stateMu.Unlock()
		return
	}
	s.preparing[tmdbID] = struct{}{}
	s.stateMu.Unlock()

	go func() {
		defer func() {
			s.stateMu.Lock()
			delete(s.preparing, tmdbID)
			s.stateMu.Unlock()
		}()

		downloadContext, cancel := context.WithTimeout(s.background, 3*time.Minute)
		defer cancel()

		source, err := s.provider.LookupSeriesTheme(downloadContext, tmdbID)
		if err != nil {
			s.rememberFailure(tmdbID, err)
			return
		}
		pendingPath := path + ".pending"
		_ = os.Remove(pendingPath)
		defer os.Remove(pendingPath)
		if err := s.downloader.DownloadAudio(downloadContext, source.URL, pendingPath); err != nil {
			s.rememberFailure(tmdbID, err)
			return
		}
		if err := writeThemeMetadata(filepath.Join(filepath.Dir(path), "metadata.json"), themeCacheMetadata{
			SchemaVersion: 1,
			Provider:      source.Provider,
			TMDbID:        tmdbID,
			SourceURL:     source.URL,
			CachedAt:      time.Now().UTC(),
		}); err != nil {
			s.rememberFailure(tmdbID, err)
			return
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			s.rememberFailure(tmdbID, fmt.Errorf("replace prepared theme: %w", err))
			return
		}
		if err := os.Rename(pendingPath, path); err != nil {
			s.rememberFailure(tmdbID, fmt.Errorf("store prepared theme: %w", err))
			return
		}
		s.clearFailure(tmdbID)
	}()
}

func normalizeTMDbID(value string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return "", fmt.Errorf("%w: series has no usable TMDb id", domain.ErrNotFound)
	}
	return strconv.FormatInt(parsed, 10), nil
}

func cachedThemeAsset(path string) (domain.ThemeAsset, bool) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 {
		return domain.ThemeAsset{}, false
	}
	return domain.ThemeAsset{Path: path, Name: "theme.m4a", ModifiedAt: info.ModTime()}, true
}

func writeThemeMetadata(path string, metadata themeCacheMetadata) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create theme metadata directory: %w", err)
	}
	payload, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("encode theme metadata: %w", err)
	}
	payload = append(payload, '\n')
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, payload, 0o600); err != nil {
		return fmt.Errorf("write theme metadata: %w", err)
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("store theme metadata: %w", err)
	}
	return nil
}

func (s *ThemeService) activeFailure(key string) (error, bool) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	failure, ok := s.failures[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(failure.until) {
		delete(s.failures, key)
		return nil, false
	}
	return failure.err, true
}

func (s *ThemeService) rememberFailure(key string, err error) {
	if errors.Is(err, context.Canceled) && s.background.Err() != nil {
		return
	}
	ttl := 30 * time.Second
	if errors.Is(err, domain.ErrNotFound) {
		ttl = 6 * time.Hour
	}
	s.stateMu.Lock()
	s.failures[key] = themeFailure{err: err, until: time.Now().Add(ttl)}
	s.stateMu.Unlock()
}

func (s *ThemeService) clearFailure(key string) {
	s.stateMu.Lock()
	delete(s.failures, key)
	s.stateMu.Unlock()
}
