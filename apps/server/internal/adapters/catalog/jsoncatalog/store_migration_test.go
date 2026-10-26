package jsoncatalog

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestOpenRepairsSeriesModifiedTimeFromNewestEpisode(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "catalog.json")
	oldTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2026, time.July, 12, 18, 30, 0, 0, time.UTC)
	state := persistedState{
		Version: schemaVersion,
		Libraries: map[string]domain.Library{
			"series": {ID: "series", Name: "Seriaalid", Kind: domain.LibraryKindSeries, Sources: []domain.LibrarySource{}},
		},
		Media: map[string]domain.MediaItem{
			"show":    {ID: "show", LibraryID: "series", Type: domain.MediaTypeSeries, Title: "Show", SortTitle: "Show", File: domain.MediaFile{ModifiedAt: oldTime}},
			"episode": {ID: "episode", LibraryID: "series", SeriesID: "show", Type: domain.MediaTypeEpisode, Title: "Episode", SortTitle: "Episode", File: domain.MediaFile{ModifiedAt: newTime}},
		},
		MediaPaths: map[string]string{"episode": `D:\\Shows\\Show.S01E01.mkv`},
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	store, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	page, err := store.ListMedia(context.Background(), domain.MediaQuery{Types: []domain.MediaType{domain.MediaTypeSeries}, Sort: domain.MediaSortModifiedAt, Order: domain.SortDescending})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 1 || !page.Items[0].File.ModifiedAt.Equal(newTime) {
		t.Fatalf("series modified time was not repaired: %+v", page.Items)
	}
}
