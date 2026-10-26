package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/adapters/catalog/jsoncatalog"
	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestGetSeriesDetailsGroupsAndSortsSeasonsAndEpisodes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := jsoncatalog.Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	library := domain.Library{ID: "library-series", Name: "Seriaalid", Kind: domain.LibraryKindSeries, Sources: []domain.LibrarySource{}, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateLibrary(ctx, library); err != nil {
		t.Fatalf("create library: %v", err)
	}
	series := domain.MediaItem{ID: "series-one", LibraryID: library.ID, Type: domain.MediaTypeSeries, Title: "Example Show", SortTitle: "Example Show", SeasonCount: 99, EpisodeCount: 99, AddedAt: now, UpdatedAt: now}
	episodes := []domain.MediaItem{
		{ID: "s2e2", LibraryID: library.ID, SeriesID: series.ID, Type: domain.MediaTypeEpisode, Title: "Second", SortTitle: "z", SeriesTitle: series.Title, Season: 2, Episode: 2, AddedAt: now, UpdatedAt: now},
		{ID: "s1e2", LibraryID: library.ID, SeriesID: series.ID, Type: domain.MediaTypeEpisode, Title: "Two", SortTitle: "y", SeriesTitle: series.Title, Season: 1, Episode: 2, AddedAt: now, UpdatedAt: now},
		{ID: "s1e1", LibraryID: library.ID, SeriesID: series.ID, Type: domain.MediaTypeEpisode, Title: "One", SortTitle: "x", SeriesTitle: series.Title, Season: 1, Episode: 1, AddedAt: now, UpdatedAt: now},
		{ID: "special", LibraryID: library.ID, SeriesID: series.ID, Type: domain.MediaTypeEpisode, Title: "Special", SortTitle: "w", SeriesTitle: series.Title, Season: 0, Episode: 1, AddedAt: now, UpdatedAt: now},
	}
	items := append([]domain.MediaItem{series}, episodes...)
	if err := store.ReplaceLibraryMedia(ctx, library.ID, items); err != nil {
		t.Fatalf("replace media: %v", err)
	}

	details, err := NewMediaService(store).GetSeriesDetails(ctx, series.ID)
	if err != nil {
		t.Fatalf("get series details: %v", err)
	}
	if details.Series.SeasonCount != 3 || details.Series.EpisodeCount != 4 {
		t.Fatalf("unexpected counts: seasons=%d episodes=%d", details.Series.SeasonCount, details.Series.EpisodeCount)
	}
	if len(details.Seasons) != 3 {
		t.Fatalf("expected three season groups, got %d", len(details.Seasons))
	}
	if details.Seasons[0].Number != 0 || details.Seasons[0].Title != "Eriepisoodid" {
		t.Fatalf("expected specials first, got %+v", details.Seasons[0])
	}
	if details.Seasons[1].Number != 1 || len(details.Seasons[1].Episodes) != 2 {
		t.Fatalf("unexpected first season: %+v", details.Seasons[1])
	}
	if details.Seasons[1].Episodes[0].Episode != 1 || details.Seasons[1].Episodes[1].Episode != 2 {
		t.Fatalf("episodes are not sorted: %+v", details.Seasons[1].Episodes)
	}
}
