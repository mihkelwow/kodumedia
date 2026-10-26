package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/adapters/catalog/jsoncatalog"
	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/service"
)

func TestListMediaAcceptsCommaSeparatedGenreAlternatives(t *testing.T) {
	t.Parallel()

	store, err := jsoncatalog.Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	now := time.Now().UTC()
	library := domain.Library{ID: "movies", Name: "Filmid", Kind: domain.LibraryKindMovies, Sources: []domain.LibrarySource{}, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateLibrary(context.Background(), library); err != nil {
		t.Fatalf("create library: %v", err)
	}
	items := []domain.MediaItem{
		{ID: "action", LibraryID: library.ID, Type: domain.MediaTypeMovie, Title: "Action", SortTitle: "Action", Metadata: domain.EditorialMetadata{Genres: []string{"Action"}}, AddedAt: now, UpdatedAt: now},
		{ID: "adventure", LibraryID: library.ID, Type: domain.MediaTypeMovie, Title: "Adventure", SortTitle: "Adventure", Metadata: domain.EditorialMetadata{Genres: []string{"Adventure"}}, AddedAt: now, UpdatedAt: now},
		{ID: "comedy", LibraryID: library.ID, Type: domain.MediaTypeMovie, Title: "Comedy", SortTitle: "Comedy", Metadata: domain.EditorialMetadata{Genres: []string{"Comedy"}}, AddedAt: now, UpdatedAt: now},
	}
	if err := store.ReplaceLibraryMedia(context.Background(), library.ID, items); err != nil {
		t.Fatalf("replace media: %v", err)
	}

	handler := NewRouter(Dependencies{Media: service.NewMediaService(store)})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/media?type=movie&genre="+url.QueryEscape("Action,Adventure"), nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}

	var page domain.MediaPage
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("expected two matching films, got %+v", page.Items)
	}
}

func TestGetMediaIncludesAbsoluteFilePath(t *testing.T) {
	t.Parallel()

	store, err := jsoncatalog.Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	now := time.Now().UTC()
	library := domain.Library{ID: "movies-path", Name: "Filmid", Kind: domain.LibraryKindMovies, Sources: []domain.LibrarySource{}, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateLibrary(context.Background(), library); err != nil {
		t.Fatalf("create library: %v", err)
	}
	absolutePath := filepath.Join(t.TempDir(), "Movies", "Example.mkv")
	item := domain.MediaItem{
		ID: "movie-path", LibraryID: library.ID, Type: domain.MediaTypeMovie,
		Title: "Example", SortTitle: "Example",
		File:    domain.MediaFile{Path: absolutePath, RelativePath: filepath.Join("Movies", "Example.mkv"), Name: "Example.mkv"},
		AddedAt: now, UpdatedAt: now,
	}
	if err := store.ReplaceLibraryMedia(context.Background(), library.ID, []domain.MediaItem{item}); err != nil {
		t.Fatalf("replace media: %v", err)
	}

	handler := NewRouter(Dependencies{Media: service.NewMediaService(store)})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/media/movie-path", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload domain.MediaItem
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.File.Path != absolutePath {
		t.Fatalf("file path = %q, want %q", payload.File.Path, absolutePath)
	}
}
