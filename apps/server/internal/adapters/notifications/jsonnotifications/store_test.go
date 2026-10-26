package jsonnotifications

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestStorePersistsNotificationHistoryAndReadState(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "notifications.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	items := []domain.Notification{{
		ID: "series:one:s01e02", Kind: domain.NotificationKindSeriesEpisodeReleased,
		SeriesID: "one", SeriesTitle: "One", Season: 1, Episode: 2, CreatedAt: created,
	}}
	if err := store.Replace(ctx, items); err != nil {
		t.Fatal(err)
	}
	readAt := created.Add(time.Minute)
	if err := store.MarkAllRead(ctx, readAt); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reopened.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ReadAt == nil || !got[0].ReadAt.Equal(readAt) {
		t.Fatalf("unexpected persisted notifications: %+v", got)
	}
}
