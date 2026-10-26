package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/adapters/catalog/jsoncatalog"
	"github.com/mihkelwow/kodumedia/internal/adapters/watch/jsonwatch"
	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestWatchProgressPersistsAndContinueWatchingUsesNewestIncomplete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	catalog, err := jsoncatalog.Open(filepath.Join(root, "catalog.json"))
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	watchPath := filepath.Join(root, "watch-state.json")
	watchStore, err := jsonwatch.Open(watchPath)
	if err != nil {
		t.Fatalf("open watch store: %v", err)
	}
	now := time.Date(2026, time.July, 13, 20, 0, 0, 0, time.UTC)
	library := domain.Library{ID: "movies", Name: "Filmid", Kind: domain.LibraryKindMovies, Sources: []domain.LibrarySource{}, CreatedAt: now, UpdatedAt: now}
	if err := catalog.CreateLibrary(ctx, library); err != nil {
		t.Fatalf("create library: %v", err)
	}
	items := []domain.MediaItem{
		{ID: "older", LibraryID: library.ID, Type: domain.MediaTypeMovie, Title: "Older", SortTitle: "Older", File: domain.MediaFile{Path: "older.mkv"}, Technical: domain.TechnicalMetadata{DurationMS: 100_000}, AddedAt: now, UpdatedAt: now},
		{ID: "newer", LibraryID: library.ID, Type: domain.MediaTypeMovie, Title: "Newer", SortTitle: "Newer", File: domain.MediaFile{Path: "newer.mkv"}, Technical: domain.TechnicalMetadata{DurationMS: 100_000}, AddedAt: now, UpdatedAt: now},
	}
	if err := catalog.ReplaceLibraryMedia(ctx, library.ID, items); err != nil {
		t.Fatalf("replace media: %v", err)
	}

	service := NewWatchService(catalog, watchStore)
	service.now = func() time.Time { return now }
	if _, err := service.Update(ctx, "older", domain.WatchUpdate{PositionMS: 20_000, DurationMS: 100_000}); err != nil {
		t.Fatalf("update older: %v", err)
	}
	service.now = func() time.Time { return now.Add(time.Minute) }
	if _, err := service.Update(ctx, "newer", domain.WatchUpdate{PositionMS: 40_000, DurationMS: 100_000}); err != nil {
		t.Fatalf("update newer: %v", err)
	}

	continueItems, err := service.ContinueWatching(ctx, 5)
	if err != nil {
		t.Fatalf("continue watching: %v", err)
	}
	if len(continueItems) != 2 || continueItems[0].ID != "newer" || continueItems[1].ID != "older" {
		t.Fatalf("unexpected continue order: %+v", continueItems)
	}

	reopened, err := jsonwatch.Open(watchPath)
	if err != nil {
		t.Fatalf("reopen watch store: %v", err)
	}
	state, err := reopened.Get(ctx, "newer")
	if err != nil {
		t.Fatalf("get persisted state: %v", err)
	}
	if state.Progress != 0.4 || state.PositionMS != 40_000 {
		t.Fatalf("unexpected persisted state: %+v", state)
	}
}

func TestSeriesDecorationPrefersIncompleteEpisodeAndOtherwiseAggregatesCompleted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	catalog, err := jsoncatalog.Open(filepath.Join(root, "catalog.json"))
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	watchStore, err := jsonwatch.Open(filepath.Join(root, "watch-state.json"))
	if err != nil {
		t.Fatalf("open watch store: %v", err)
	}
	now := time.Date(2026, time.July, 13, 21, 0, 0, 0, time.UTC)
	library := domain.Library{ID: "series", Name: "Seriaalid", Kind: domain.LibraryKindSeries, Sources: []domain.LibrarySource{}, CreatedAt: now, UpdatedAt: now}
	if err := catalog.CreateLibrary(ctx, library); err != nil {
		t.Fatalf("create library: %v", err)
	}
	series := domain.MediaItem{ID: "show", LibraryID: library.ID, Type: domain.MediaTypeSeries, Title: "Show", SortTitle: "Show", EpisodeCount: 3, AddedAt: now, UpdatedAt: now}
	episodes := []domain.MediaItem{
		{ID: "e1", LibraryID: library.ID, SeriesID: series.ID, Type: domain.MediaTypeEpisode, Title: "One", SortTitle: "1", Season: 1, Episode: 1, File: domain.MediaFile{Path: "e1.mkv"}, Technical: domain.TechnicalMetadata{DurationMS: 100_000}, AddedAt: now, UpdatedAt: now},
		{ID: "e2", LibraryID: library.ID, SeriesID: series.ID, Type: domain.MediaTypeEpisode, Title: "Two", SortTitle: "2", Season: 1, Episode: 2, File: domain.MediaFile{Path: "e2.mkv"}, Technical: domain.TechnicalMetadata{DurationMS: 100_000}, AddedAt: now, UpdatedAt: now},
		{ID: "e3", LibraryID: library.ID, SeriesID: series.ID, Type: domain.MediaTypeEpisode, Title: "Three", SortTitle: "3", Season: 1, Episode: 3, File: domain.MediaFile{Path: "e3.mkv"}, Technical: domain.TechnicalMetadata{DurationMS: 100_000}, AddedAt: now, UpdatedAt: now},
	}
	if err := catalog.ReplaceLibraryMedia(ctx, library.ID, append([]domain.MediaItem{series}, episodes...)); err != nil {
		t.Fatalf("replace media: %v", err)
	}
	service := NewWatchService(catalog, watchStore)
	service.now = func() time.Time { return now }
	if _, err := service.Update(ctx, "e1", domain.WatchUpdate{PositionMS: 100_000, DurationMS: 100_000, Completed: true}); err != nil {
		t.Fatalf("complete e1: %v", err)
	}
	service.now = func() time.Time { return now.Add(time.Minute) }
	if _, err := service.Update(ctx, "e2", domain.WatchUpdate{PositionMS: 35_000, DurationMS: 100_000}); err != nil {
		t.Fatalf("watch e2: %v", err)
	}

	items := []domain.MediaItem{series}
	if err := service.Decorate(ctx, items); err != nil {
		t.Fatalf("decorate: %v", err)
	}
	if items[0].Watch == nil || items[0].Watch.MediaID != "e2" || items[0].Watch.Progress != 0.35 {
		t.Fatalf("expected active episode checkpoint, got %+v", items[0].Watch)
	}

	service.now = func() time.Time { return now.Add(2 * time.Minute) }
	if _, err := service.Update(ctx, "e2", domain.WatchUpdate{PositionMS: 100_000, DurationMS: 100_000, Completed: true}); err != nil {
		t.Fatalf("complete e2: %v", err)
	}
	items = []domain.MediaItem{series}
	if err := service.Decorate(ctx, items); err != nil {
		t.Fatalf("decorate aggregate: %v", err)
	}
	if items[0].Watch == nil || items[0].Watch.Progress != 2.0/3.0 || items[0].Watch.Completed {
		t.Fatalf("expected aggregate 2/3 progress, got %+v", items[0].Watch)
	}
}

func TestContinueWatchingExcludesMusicTracks(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	catalog, err := jsoncatalog.Open(filepath.Join(root, "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	watchStore, err := jsonwatch.Open(filepath.Join(root, "watch.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	library := domain.Library{ID: "mixed", Name: "Mixed", Kind: domain.LibraryKindMixed, CreatedAt: now, UpdatedAt: now}
	if err := catalog.CreateLibrary(ctx, library); err != nil {
		t.Fatal(err)
	}
	items := []domain.MediaItem{
		{ID: "movie", LibraryID: library.ID, Type: domain.MediaTypeMovie, Title: "Movie", SortTitle: "Movie", File: domain.MediaFile{Path: "movie.mkv"}, Technical: domain.TechnicalMetadata{DurationMS: 100_000}, AddedAt: now, UpdatedAt: now},
		{ID: "song", LibraryID: library.ID, Type: domain.MediaTypeMusic, Title: "Song", SortTitle: "Song", File: domain.MediaFile{Path: "song.wav"}, Technical: domain.TechnicalMetadata{DurationMS: 100_000}, AddedAt: now, UpdatedAt: now},
	}
	if err := catalog.ReplaceLibraryMedia(ctx, library.ID, items); err != nil {
		t.Fatal(err)
	}
	service := NewWatchService(catalog, watchStore)
	service.now = func() time.Time { return now }
	if _, err := service.Update(ctx, "song", domain.WatchUpdate{PositionMS: 20_000, DurationMS: 100_000}); err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return now.Add(time.Minute) }
	if _, err := service.Update(ctx, "movie", domain.WatchUpdate{PositionMS: 20_000, DurationMS: 100_000}); err != nil {
		t.Fatal(err)
	}
	items, err = service.ContinueWatching(ctx, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "movie" {
		t.Fatalf("music leaked into Continue Watching: %+v", items)
	}
}
