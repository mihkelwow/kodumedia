package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/adapters/catalog/jsoncatalog"
	"github.com/mihkelwow/kodumedia/internal/adapters/notifications/jsonnotifications"
	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestNotificationServiceCreatesReadsAndResolvesEpisodeNotifications(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	catalog, err := jsoncatalog.Open(filepath.Join(directory, "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := jsonnotifications.Open(filepath.Join(directory, "notifications.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	released := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	library := domain.Library{ID: "series-library", Name: "Series", Kind: domain.LibraryKindSeries, CreatedAt: now, UpdatedAt: now}
	if err := catalog.CreateLibrary(ctx, library); err != nil {
		t.Fatal(err)
	}
	series := domain.MediaItem{
		ID: "series-one", LibraryID: library.ID, Type: domain.MediaTypeSeries, Title: "Example Series", SortTitle: "Example Series",
		Metadata: domain.EditorialMetadata{LatestEpisode: &domain.EpisodeReleaseInfo{Season: 1, Episode: 2, Title: "New Start", ReleaseDate: &released}},
		AddedAt:  now, UpdatedAt: now,
	}
	episodeOne := domain.MediaItem{
		ID: "episode-one", LibraryID: library.ID, SeriesID: series.ID, Type: domain.MediaTypeEpisode, Title: "Pilot", SortTitle: "01-01",
		Season: 1, Episode: 1, AddedAt: now, UpdatedAt: now,
	}
	if err := catalog.ReplaceLibraryMedia(ctx, library.ID, []domain.MediaItem{series, episodeOne}); err != nil {
		t.Fatal(err)
	}

	service := NewNotificationService(catalog, store)
	service.now = func() time.Time { return now }
	page, err := service.List(ctx, 5, 0)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Unread != 1 || len(page.Items) != 1 {
		t.Fatalf("unexpected first notification page: %+v", page)
	}
	if got := page.Items[0]; got.SeriesID != series.ID || got.Season != 1 || got.Episode != 2 || got.ResolvedAt != nil {
		t.Fatalf("unexpected notification: %+v", got)
	}

	if err := service.MarkAllRead(ctx); err != nil {
		t.Fatal(err)
	}
	page, err = service.List(ctx, 5, 0)
	if err != nil {
		t.Fatal(err)
	}
	if page.Unread != 0 || page.Items[0].ReadAt == nil {
		t.Fatalf("notification was not marked read: %+v", page)
	}

	episodeTwo := domain.MediaItem{
		ID: "episode-two", LibraryID: library.ID, SeriesID: series.ID, Type: domain.MediaTypeEpisode, Title: "New Start", SortTitle: "01-02",
		Season: 1, Episode: 2, ReleaseDate: &released, AddedAt: now, UpdatedAt: now,
	}
	if err := catalog.ReplaceLibraryMedia(ctx, library.ID, []domain.MediaItem{series, episodeOne, episodeTwo}); err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return now.Add(time.Minute) }
	page, err = service.List(ctx, 5, 0)
	if err != nil {
		t.Fatal(err)
	}
	if page.Items[0].ResolvedAt == nil {
		t.Fatalf("notification should remain in history and be resolved: %+v", page.Items[0])
	}
}
