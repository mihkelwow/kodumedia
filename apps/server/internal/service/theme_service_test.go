package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/adapters/catalog/jsoncatalog"
	"github.com/mihkelwow/kodumedia/internal/domain"
)

type fakeThemeProvider struct {
	calls  atomic.Int32
	source domain.ThemeSource
	err    error
}

func (f *fakeThemeProvider) LookupSeriesTheme(_ context.Context, _ string) (domain.ThemeSource, error) {
	f.calls.Add(1)
	return f.source, f.err
}

type fakeThemeDownloader struct {
	calls   atomic.Int32
	started chan context.Context
	release <-chan struct{}
}

func (f *fakeThemeDownloader) DownloadAudio(ctx context.Context, _ string, destination string) error {
	f.calls.Add(1)
	if f.started != nil {
		f.started <- ctx
	}
	if f.release != nil {
		select {
		case <-f.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destination, []byte("cached-theme"), 0o600)
}

func themeTestCatalog(t *testing.T, tmdbID string) *jsoncatalog.Store {
	t.Helper()
	store, err := jsoncatalog.Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	library := domain.Library{ID: "series", Name: "Series", Kind: domain.LibraryKindSeries, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateLibrary(context.Background(), library); err != nil {
		t.Fatal(err)
	}
	series := domain.MediaItem{
		ID: "show-1", LibraryID: library.ID, Type: domain.MediaTypeSeries,
		Title: "Show", SortTitle: "Show", ExternalIDs: domain.ExternalIDs{TMDb: tmdbID},
		AddedAt: now, UpdatedAt: now,
	}
	if err := store.ReplaceLibraryMedia(context.Background(), library.ID, []domain.MediaItem{series}); err != nil {
		t.Fatal(err)
	}
	return store
}

func waitForTheme(t *testing.T, service *ThemeService, seriesID string) domain.ThemeAsset {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		asset, err := service.ResolveSeriesTheme(context.Background(), seriesID)
		if err == nil {
			return asset
		}
		if !errors.Is(err, domain.ErrPending) {
			t.Fatalf("resolve prepared theme: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("theme preparation did not complete")
	return domain.ThemeAsset{}
}

func TestThemeServicePreparesOnceInBackgroundAndUsesPersistentCache(t *testing.T) {
	catalog := themeTestCatalog(t, "1428")
	provider := &fakeThemeProvider{source: domain.ThemeSource{Provider: "themerrdb", ExternalID: "1428", URL: "https://www.youtube.com/watch?v=test"}}
	downloader := &fakeThemeDownloader{}
	cacheRoot := filepath.Join(t.TempDir(), "themes")
	settings := func() domain.ThemeMusicSettings {
		return domain.ThemeMusicSettings{SeriesPlayback: domain.ThemePlaybackAlways}
	}
	service := NewThemeService(catalog, provider, downloader, cacheRoot, settings)

	if _, err := service.ResolveSeriesTheme(context.Background(), "show-1"); !errors.Is(err, domain.ErrPending) {
		t.Fatalf("first resolve should start a background preparation, got %v", err)
	}
	if _, err := service.ResolveSeriesTheme(context.Background(), "show-1"); !errors.Is(err, domain.ErrPending) {
		t.Fatalf("second resolve should share the pending preparation, got %v", err)
	}
	first := waitForTheme(t, service, "show-1")
	second, err := service.ResolveSeriesTheme(context.Background(), "show-1")
	if err != nil {
		t.Fatal(err)
	}
	if first.Path != second.Path || provider.calls.Load() != 1 || downloader.calls.Load() != 1 {
		t.Fatalf("cache was not reused: first=%+v second=%+v provider=%d downloader=%d", first, second, provider.calls.Load(), downloader.calls.Load())
	}
	if _, err := os.Stat(filepath.Join(cacheRoot, "1428", "metadata.json")); err != nil {
		t.Fatalf("theme metadata was not cached: %v", err)
	}
}

func TestThemeServicePreparationOutlivesTriggeringRequest(t *testing.T) {
	catalog := themeTestCatalog(t, "1428")
	provider := &fakeThemeProvider{source: domain.ThemeSource{Provider: "themerrdb", ExternalID: "1428", URL: "https://www.youtube.com/watch?v=test"}}
	started := make(chan context.Context, 1)
	release := make(chan struct{})
	downloader := &fakeThemeDownloader{started: started, release: release}
	service := NewThemeService(catalog, provider, downloader, t.TempDir(), func() domain.ThemeMusicSettings {
		return domain.ThemeMusicSettings{SeriesPlayback: domain.ThemePlaybackAlways}
	})

	requestContext, cancelRequest := context.WithCancel(context.Background())
	if _, err := service.ResolveSeriesTheme(requestContext, "show-1"); !errors.Is(err, domain.ErrPending) {
		t.Fatalf("expected pending preparation, got %v", err)
	}
	backgroundContext := <-started
	cancelRequest()
	select {
	case <-backgroundContext.Done():
		t.Fatal("background preparation was canceled with the browser request")
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	_ = waitForTheme(t, service, "show-1")
}

func TestThemeServiceDoesNotResolveWhenDisabled(t *testing.T) {
	catalog := themeTestCatalog(t, "1428")
	provider := &fakeThemeProvider{}
	downloader := &fakeThemeDownloader{}
	service := NewThemeService(catalog, provider, downloader, t.TempDir(), func() domain.ThemeMusicSettings {
		return domain.ThemeMusicSettings{SeriesPlayback: domain.ThemePlaybackOff}
	})
	if _, err := service.ResolveSeriesTheme(context.Background(), "show-1"); err == nil {
		t.Fatal("disabled theme playback must not resolve a theme")
	}
	if provider.calls.Load() != 0 || downloader.calls.Load() != 0 {
		t.Fatalf("disabled service called external dependencies: provider=%d downloader=%d", provider.calls.Load(), downloader.calls.Load())
	}
}

func TestThemePlaybackConfigurationUsesRequiredDefaults(t *testing.T) {
	service := NewThemeService(nil, nil, nil, "", func() domain.ThemeMusicSettings {
		return domain.ThemeMusicSettings{SeriesPlayback: domain.ThemePlaybackFirstOpen}
	})
	configuration := service.PlaybackConfiguration()
	if configuration.SeriesPlayback != domain.ThemePlaybackFirstOpen || configuration.Volume != 0.15 || configuration.FadeInMS != 2000 || configuration.FadeOutMS != 2000 {
		t.Fatalf("unexpected theme playback configuration: %+v", configuration)
	}
}
