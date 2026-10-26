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

func TestMusicLandingBuildsArtistAlbumTrackHierarchy(t *testing.T) {
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
	library := domain.Library{ID: "music", Name: "Muusika", Kind: domain.LibraryKindMusic, CreatedAt: now, UpdatedAt: now}
	if err := catalog.CreateLibrary(ctx, library); err != nil {
		t.Fatal(err)
	}
	tracks := []domain.MediaItem{
		{ID: "track-2", LibraryID: library.ID, Type: domain.MediaTypeMusic, Title: "Second", SortTitle: "Artist Album 0002 Second", Artist: "Artist", Album: "Album", TrackNumber: 2, Year: 2025, File: domain.MediaFile{Path: "2.flac"}, AddedAt: now.Add(time.Minute), UpdatedAt: now},
		{ID: "track-1", LibraryID: library.ID, Type: domain.MediaTypeMusic, Title: "First", SortTitle: "Artist Album 0001 First", Artist: "Artist", Album: "Album", TrackNumber: 1, Year: 2025, File: domain.MediaFile{Path: "1.wav"}, AddedAt: now, UpdatedAt: now},
		{ID: "other", LibraryID: library.ID, Type: domain.MediaTypeMusic, Title: "Other", SortTitle: "Other EP 0001 Other", Artist: "Other", Album: "EP", TrackNumber: 1, File: domain.MediaFile{Path: "other.flac"}, AddedAt: now.Add(2 * time.Minute), UpdatedAt: now},
	}
	if err := catalog.ReplaceLibraryMedia(ctx, library.ID, tracks); err != nil {
		t.Fatal(err)
	}
	watch := NewWatchService(catalog, watchStore)
	watch.now = func() time.Time { return now.Add(3 * time.Minute) }
	if _, err := watch.Update(ctx, "track-1", domain.WatchUpdate{PositionMS: 10_000, DurationMS: 100_000}); err != nil {
		t.Fatal(err)
	}

	landing, err := NewMusicService(catalog, watch).Landing(ctx, library.ID)
	if err != nil {
		t.Fatal(err)
	}
	if landing.TotalTracks != 3 {
		t.Fatalf("unexpected total track count: %d", landing.TotalTracks)
	}
	if len(landing.Artists) != 2 || len(landing.Albums) != 2 {
		t.Fatalf("unexpected hierarchy: %+v", landing)
	}
	if len(landing.RecentlyPlayed) != 1 || landing.RecentlyPlayed[0].ID != "track-1" {
		t.Fatalf("unexpected recently played: %+v", landing.RecentlyPlayed)
	}
	if landing.RecentlyAdded[0].ID != "other" {
		t.Fatalf("recently added not sorted: %+v", landing.RecentlyAdded)
	}
	album, err := NewMusicService(catalog, watch).Album(ctx, library.ID, landing.Albums[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if album.Album.Artist == "Artist" && (len(album.Album.Tracks) != 2 || album.Album.Tracks[0].TrackNumber != 1) {
		t.Fatalf("album tracks are not ordered: %+v", album.Album.Tracks)
	}
	for _, item := range landing.RecentlyAdded {
		if item.Metadata.PosterURL == "" {
			t.Fatalf("track %s lacks artwork URL", item.ID)
		}
	}
}

func TestLosslessPCMCodecPreservesSourceDepth(t *testing.T) {
	t.Parallel()
	cases := []struct {
		depth        int
		source, want string
	}{
		{16, "flac", "pcm_s16le"}, {24, "flac", "pcm_s24le"}, {32, "wav", "pcm_s32le"}, {32, "pcm_f32le", "pcm_f32le"},
	}
	for _, test := range cases {
		item := domain.MediaItem{Technical: domain.TechnicalMetadata{AudioCodec: test.source, AudioBitDepth: test.depth}}
		if got := losslessPCMCodec(item); got != test.want {
			t.Fatalf("depth %d source %s: got %s want %s", test.depth, test.source, got, test.want)
		}
	}
}

func TestMusicTracksFiltersSortsAndBuildsFacets(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	catalog, err := jsoncatalog.Open(filepath.Join(root, "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.July, 21, 10, 0, 0, 0, time.UTC)
	library := domain.Library{ID: "music-filters", Name: "Music", Kind: domain.LibraryKindMusic, CreatedAt: now, UpdatedAt: now}
	if err := catalog.CreateLibrary(ctx, library); err != nil {
		t.Fatal(err)
	}
	tracks := []domain.MediaItem{
		{ID: "one", LibraryID: library.ID, Type: domain.MediaTypeMusic, Title: "Night Drive", SortTitle: "Night Drive", Artist: "Artist A", Album: "Album A", Year: 2025, File: domain.MediaFile{Extension: ".flac"}, Technical: domain.TechnicalMetadata{AudioCodec: "flac", AudioSampleRate: 96000, AudioBitDepth: 24, DurationMS: 210000}, MusicClassification: &domain.MusicClassification{Genres: []string{"Electronic"}, Styles: []string{"Progressive House"}, Source: "sidecar", Confidence: "verified"}, AddedAt: now, UpdatedAt: now},
		{ID: "two", LibraryID: library.ID, Type: domain.MediaTypeMusic, Title: "Club Edit", SortTitle: "Club Edit", Artist: "Artist B", Album: "Bootlegs", Year: 2024, File: domain.MediaFile{Extension: ".mp3"}, Technical: domain.TechnicalMetadata{AudioCodec: "mp3", AudioSampleRate: 44100, AudioBitDepth: 16, DurationMS: 190000}, MusicClassification: &domain.MusicClassification{Genres: []string{"Electronic"}, Styles: []string{"Bootleg"}, Source: "embedded", Confidence: "tagged"}, AddedAt: now.Add(time.Minute), UpdatedAt: now},
		{ID: "three", LibraryID: library.ID, Type: domain.MediaTypeMusic, Title: "Acoustic", SortTitle: "Acoustic", Artist: "Artist C", Album: "Sessions", Year: 2023, File: domain.MediaFile{Extension: ".wav"}, Technical: domain.TechnicalMetadata{AudioCodec: "pcm_s24le", AudioSampleRate: 48000, AudioBitDepth: 24, DurationMS: 180000}, AddedAt: now.Add(2 * time.Minute), UpdatedAt: now},
	}
	if err := catalog.ReplaceLibraryMedia(ctx, library.ID, tracks); err != nil {
		t.Fatal(err)
	}
	page, err := NewMusicService(catalog, nil).Tracks(ctx, domain.MusicTrackQuery{LibraryID: library.ID, Genre: "Electronic", Sort: "quality", Order: domain.SortDescending, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || page.Items[0].ID != "one" {
		t.Fatalf("unexpected filtered/sorted page: %+v", page.Items)
	}
	if page.Facets.HiRes != 2 || page.Facets.Lossless != 2 || page.Facets.Unclassified != 1 {
		t.Fatalf("unexpected quality facets: %+v", page.Facets)
	}
	if len(page.Facets.Styles) != 2 {
		t.Fatalf("expected two style facets, got %+v", page.Facets.Styles)
	}
}

func TestMusicArtworkURLIncludesCacheVersion(t *testing.T) {
	got := musicArtworkURL("track id")
	if got != "/api/v1/media/track id/artwork?v=2" {
		t.Fatalf("unexpected artwork URL: %q", got)
	}
}
