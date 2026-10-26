package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/adapters/catalog/jsoncatalog"
	jsonwatchadapter "github.com/mihkelwow/kodumedia/internal/adapters/watch/jsonwatch"
	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestPostPlayReturnsNextEpisodeInChronologicalOrder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	catalog, watch := postPlayTestStores(t, ctx)
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	items := []domain.MediaItem{
		postPlaySeries("series-current", "Current Show", []string{"Drama"}, now),
		postPlayEpisode("episode-2", "series-current", 1, 2, now),
		postPlayEpisode("episode-1", "series-current", 1, 1, now),
		postPlayEpisode("episode-3", "series-current", 2, 1, now),
	}
	if err := catalog.ReplaceLibraryMedia(ctx, "library", items); err != nil {
		t.Fatalf("replace media: %v", err)
	}

	result, err := NewPostPlayService(catalog, watch).Get(ctx, "episode-1")
	if err != nil {
		t.Fatalf("get post play: %v", err)
	}
	if result.NextEpisode == nil || result.NextEpisode.ID != "episode-2" {
		t.Fatalf("expected episode-2, got %+v", result.NextEpisode)
	}
	if len(result.Recommendations) != 0 {
		t.Fatalf("did not expect recommendations while next episode exists")
	}
}

func TestPostPlayRecommendsUnwatchedRelatedTitlesAndOldestUnwatchedEpisode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	catalog, watch := postPlayTestStores(t, ctx)
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	items := []domain.MediaItem{
		postPlaySeries("series-current", "Current Crime", []string{"Drama", "Crime"}, now),
		postPlayEpisode("current-last", "series-current", 1, 8, now),
		postPlaySeries("series-related", "Related Show", []string{"Crime", "Thriller"}, now),
		postPlayEpisode("related-1", "series-related", 1, 1, now),
		postPlayEpisode("related-2", "series-related", 1, 2, now),
		postPlayMovie("movie-related", "Related Movie", []string{"Drama"}, 8.2, now),
		postPlayMovie("movie-fallback", "Fallback Movie", []string{"Comedy"}, 7.5, now),
		postPlayMovie("movie-watched", "Watched Crime", []string{"Crime"}, 9.0, now),
	}
	if err := catalog.ReplaceLibraryMedia(ctx, "library", items); err != nil {
		t.Fatalf("replace media: %v", err)
	}
	states := []domain.WatchState{
		{MediaID: "related-1", SeriesID: "series-related", MediaType: domain.MediaTypeEpisode, Season: 1, Episode: 1, PositionMS: 3_600_000, DurationMS: 3_600_000, Progress: 1, Completed: true, LastWatchedAt: now, UpdatedAt: now},
		{MediaID: "movie-watched", MediaType: domain.MediaTypeMovie, PositionMS: 60_000, DurationMS: 3_600_000, Progress: 0.02, LastWatchedAt: now, UpdatedAt: now},
	}
	for _, state := range states {
		if err := watch.store.Upsert(ctx, state); err != nil {
			t.Fatalf("write watch state: %v", err)
		}
	}

	result, err := NewPostPlayService(catalog, watch).Get(ctx, "current-last")
	if err != nil {
		t.Fatalf("get post play: %v", err)
	}
	if result.NextEpisode != nil {
		t.Fatalf("did not expect a next episode")
	}
	if len(result.Recommendations) != 3 {
		t.Fatalf("expected three recommendations, got %d", len(result.Recommendations))
	}
	var relatedSeries *domain.PostPlayRecommendation
	for index := range result.Recommendations {
		recommendation := &result.Recommendations[index]
		if recommendation.Item.ID == "movie-watched" {
			t.Fatalf("watched movie must not be recommended")
		}
		if recommendation.Item.ID == "series-related" {
			relatedSeries = recommendation
		}
	}
	if relatedSeries == nil {
		t.Fatalf("expected related series among recommendations")
	}
	if relatedSeries.PlayTarget == nil || relatedSeries.PlayTarget.ID != "related-2" {
		t.Fatalf("expected oldest unwatched episode related-2, got %+v", relatedSeries.PlayTarget)
	}
}

func postPlayTestStores(t *testing.T, ctx context.Context) (*jsoncatalog.Store, *WatchService) {
	t.Helper()
	catalog, err := jsoncatalog.Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	library := domain.Library{ID: "library", Name: "Library", Kind: domain.LibraryKindMixed, CreatedAt: now, UpdatedAt: now}
	if err := catalog.CreateLibrary(ctx, library); err != nil {
		t.Fatalf("create library: %v", err)
	}
	store, err := jsonwatchadapter.Open(filepath.Join(t.TempDir(), "watch-state.json"))
	if err != nil {
		t.Fatalf("open watch store: %v", err)
	}
	return catalog, NewWatchService(catalog, store)
}

func postPlaySeries(id, title string, genres []string, now time.Time) domain.MediaItem {
	return domain.MediaItem{ID: id, LibraryID: "library", Type: domain.MediaTypeSeries, Title: title, SortTitle: title, Metadata: domain.EditorialMetadata{Genres: genres, IMDbRating: 8}, AddedAt: now, UpdatedAt: now}
}

func postPlayEpisode(id, seriesID string, season, episode int, now time.Time) domain.MediaItem {
	return domain.MediaItem{ID: id, LibraryID: "library", SeriesID: seriesID, Type: domain.MediaTypeEpisode, Title: id, SortTitle: id, SeriesTitle: seriesID, Season: season, Episode: episode, File: domain.MediaFile{Path: id + ".mkv", Name: id + ".mkv"}, Technical: domain.TechnicalMetadata{DurationMS: 3_600_000}, AddedAt: now, UpdatedAt: now}
}

func postPlayMovie(id, title string, genres []string, rating float64, now time.Time) domain.MediaItem {
	return domain.MediaItem{ID: id, LibraryID: "library", Type: domain.MediaTypeMovie, Title: title, SortTitle: title, File: domain.MediaFile{Path: id + ".mkv", Name: id + ".mkv"}, Metadata: domain.EditorialMetadata{Genres: genres, IMDbRating: rating}, Technical: domain.TechnicalMetadata{DurationMS: 7_200_000}, AddedAt: now, UpdatedAt: now}
}
