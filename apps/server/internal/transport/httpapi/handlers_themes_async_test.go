package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/adapters/catalog/jsoncatalog"
	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/service"
)

type httpThemeProvider struct{}

func (httpThemeProvider) LookupSeriesTheme(_ context.Context, tmdbID string) (domain.ThemeSource, error) {
	return domain.ThemeSource{Provider: "themerrdb", ExternalID: tmdbID, URL: "https://www.youtube.com/watch?v=test"}, nil
}

type httpThemeDownloader struct {
	started chan struct{}
	release <-chan struct{}
}

func (d httpThemeDownloader) DownloadAudio(ctx context.Context, _ string, destination string) error {
	close(d.started)
	select {
	case <-d.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destination, []byte("theme"), 0o600)
}

func TestServeSeriesThemeReturnsAcceptedWhileBackgroundPreparationContinues(t *testing.T) {
	store, err := jsoncatalog.Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	library := domain.Library{ID: "series", Name: "Series", Kind: domain.LibraryKindSeries, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateLibrary(context.Background(), library); err != nil {
		t.Fatal(err)
	}
	item := domain.MediaItem{
		ID: "show-1", LibraryID: library.ID, Type: domain.MediaTypeSeries,
		Title: "Show", SortTitle: "Show", ExternalIDs: domain.ExternalIDs{TMDb: "1428"},
		AddedAt: now, UpdatedAt: now,
	}
	if err := store.ReplaceLibraryMedia(context.Background(), library.ID, []domain.MediaItem{item}); err != nil {
		t.Fatal(err)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	themes := service.NewThemeService(
		store,
		httpThemeProvider{},
		httpThemeDownloader{started: started, release: release},
		filepath.Join(t.TempDir(), "themes"),
		func() domain.ThemeMusicSettings {
			return domain.ThemeMusicSettings{SeriesPlayback: domain.ThemePlaybackAlways}
		},
	)
	handler := NewRouter(Dependencies{Themes: themes})

	request := httptest.NewRequest(http.MethodHead, "/api/v1/series/show-1/theme", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202 while preparing, got %d: %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Retry-After") != "2" {
		t.Fatalf("expected Retry-After 2, got %q", response.Header().Get("Retry-After"))
	}
	<-started
	close(release)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		request = httptest.NewRequest(http.MethodHead, "/api/v1/series/show-1/theme", nil)
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code == http.StatusOK {
			return
		}
		if response.Code != http.StatusAccepted {
			t.Fatalf("expected 202 or 200, got %d: %s", response.Code, response.Body.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("prepared theme was not served")
}
