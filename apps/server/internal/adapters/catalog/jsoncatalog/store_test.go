package jsoncatalog

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestCreateLibrarySerializesEmptySourcesAsArray(t *testing.T) {
	t.Parallel()

	catalogPath := filepath.Join(t.TempDir(), "catalog.json")
	store, err := Open(catalogPath)
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}

	now := time.Now().UTC()
	library := domain.Library{
		ID:        "library-movies",
		Name:      "Filmid",
		Kind:      domain.LibraryKindMovies,
		System:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.CreateLibrary(context.Background(), library); err != nil {
		t.Fatalf("create library: %v", err)
	}

	libraries, err := store.ListLibraries(context.Background())
	if err != nil {
		t.Fatalf("list libraries: %v", err)
	}
	if len(libraries) != 1 {
		t.Fatalf("expected one library, got %d", len(libraries))
	}
	if libraries[0].Sources == nil {
		t.Fatal("expected sources to be a non-nil empty slice")
	}

	data, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}
	if !strings.Contains(string(data), `"sources": []`) {
		t.Fatalf("expected persisted sources array, got:\n%s", data)
	}
}

func TestOpenRepairsLegacyNullSources(t *testing.T) {
	t.Parallel()

	catalogPath := filepath.Join(t.TempDir(), "catalog.json")
	legacy := `{
  "version": 1,
  "libraries": {
    "library-movies": {
      "id": "library-movies",
      "name": "Filmid",
      "kind": "movies",
      "system": true,
      "sources": null,
      "createdAt": "2026-01-01T00:00:00Z",
      "updatedAt": "2026-01-01T00:00:00Z"
    }
  },
  "media": {},
  "mediaPaths": {}
}`
	if err := os.WriteFile(catalogPath, []byte(legacy), 0o600); err != nil {
		t.Fatalf("write legacy catalog: %v", err)
	}

	store, err := Open(catalogPath)
	if err != nil {
		t.Fatalf("open legacy catalog: %v", err)
	}
	library, err := store.GetLibrary(context.Background(), "library-movies")
	if err != nil {
		t.Fatalf("get repaired library: %v", err)
	}
	if library.Sources == nil {
		t.Fatal("expected legacy null sources to be repaired to an empty slice")
	}

	data, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatalf("read repaired catalog: %v", err)
	}
	if strings.Contains(string(data), `"sources": null`) {
		t.Fatalf("legacy null sources were not repaired:\n%s", data)
	}
	if !strings.Contains(string(data), `"sources": []`) {
		t.Fatalf("expected repaired sources array, got:\n%s", data)
	}
}

func TestListMediaSearchMatchesIndependentTokensAcrossMetadata(t *testing.T) {
	t.Parallel()

	store, err := Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	now := time.Now().UTC()
	library := domain.Library{ID: "library-movies", Name: "Filmid", Kind: domain.LibraryKindMovies, Sources: []domain.LibrarySource{}, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateLibrary(context.Background(), library); err != nil {
		t.Fatalf("create library: %v", err)
	}
	items := []domain.MediaItem{
		{
			ID: "movie-1", LibraryID: library.ID, Type: domain.MediaTypeMovie,
			Title: "John Wick: Ballerina", SortTitle: "John Wick Ballerina",
			File: domain.MediaFile{Name: "Ballerina.2025.mkv", RelativePath: "Movies/Ballerina.2025.mkv"},
			Metadata: domain.EditorialMetadata{
				Genres: []string{"Action", "Thriller"},
				Cast:   []domain.PersonCredit{{Name: "Ana de Armas", Character: "Eve Macarro"}},
			},
			AddedAt: now, UpdatedAt: now,
		},
	}
	if err := store.ReplaceLibraryMedia(context.Background(), library.ID, items); err != nil {
		t.Fatalf("replace media: %v", err)
	}

	page, err := store.ListMedia(context.Background(), domain.MediaQuery{Search: "Ana Ballerina", Limit: 20})
	if err != nil {
		t.Fatalf("search media: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != "movie-1" {
		t.Fatalf("expected tokenized metadata match, got %+v", page)
	}

	page, err = store.ListMedia(context.Background(), domain.MediaQuery{Search: "thriller eve", Limit: 20})
	if err != nil {
		t.Fatalf("search media by genre and character: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("expected genre/character match, got %d", page.Total)
	}
}

func TestListMediaMatchesAnyRequestedGenre(t *testing.T) {
	t.Parallel()

	store, err := Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	now := time.Now().UTC()
	library := domain.Library{ID: "library-movies", Name: "Filmid", Kind: domain.LibraryKindMovies, Sources: []domain.LibrarySource{}, CreatedAt: now, UpdatedAt: now}
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

	page, err := store.ListMedia(context.Background(), domain.MediaQuery{Types: []domain.MediaType{domain.MediaTypeMovie}, Genres: []string{"Action", "Adventure"}, Limit: 20})
	if err != nil {
		t.Fatalf("list media: %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("expected two action or adventure movies, got %+v", page.Items)
	}
	for _, item := range page.Items {
		if item.ID == "comedy" {
			t.Fatalf("unexpected comedy result: %+v", page.Items)
		}
	}
}

func TestListMediaReusesSortedQueryAndInvalidatesOnMutation(t *testing.T) {
	t.Parallel()

	store, err := Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	library := domain.Library{ID: "library-music", Name: "Muusika", Kind: domain.LibraryKindMusic, Sources: []domain.LibrarySource{}, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateLibrary(context.Background(), library); err != nil {
		t.Fatal(err)
	}
	items := []domain.MediaItem{
		{ID: "track-b", LibraryID: library.ID, Type: domain.MediaTypeMusic, Title: "B", SortTitle: "B", AddedAt: now, UpdatedAt: now},
		{ID: "track-a", LibraryID: library.ID, Type: domain.MediaTypeMusic, Title: "A", SortTitle: "A", AddedAt: now, UpdatedAt: now},
	}
	if err := store.ReplaceLibraryMedia(context.Background(), library.ID, items); err != nil {
		t.Fatal(err)
	}
	query := domain.MediaQuery{Types: []domain.MediaType{domain.MediaTypeMusic}, Sort: "title", Order: "asc", Limit: 1}
	first, err := store.ListMedia(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if first.Total != 2 || len(first.Items) != 1 || first.Items[0].ID != "track-a" {
		t.Fatalf("unexpected first page: %+v", first)
	}
	if len(store.queryCache) != 1 {
		t.Fatalf("expected one cached sorted result, got %d", len(store.queryCache))
	}
	query.Offset = 1
	second, err := store.ListMedia(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].ID != "track-b" || len(store.queryCache) != 1 {
		t.Fatalf("pagination did not reuse cached query: page=%+v cache=%d", second, len(store.queryCache))
	}

	items = append(items, domain.MediaItem{
		ID: "track-c", LibraryID: library.ID, Type: domain.MediaTypeMusic, Title: "0", SortTitle: "0", AddedAt: now, UpdatedAt: now,
	})
	if err := store.ReplaceLibraryMedia(context.Background(), library.ID, items); err != nil {
		t.Fatal(err)
	}
	if len(store.queryCache) != 0 {
		t.Fatal("catalog mutation did not invalidate the query cache")
	}
	query.Offset = 0
	updated, err := store.ListMedia(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Total != 3 || updated.Items[0].ID != "track-c" {
		t.Fatalf("query cache returned stale media after mutation: %+v", updated)
	}
}
