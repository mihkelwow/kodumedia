package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/adapters/catalog/jsoncatalog"
	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

func TestParseFilenameUsesLastYearAsReleaseYear(t *testing.T) {
	item := domain.MediaItem{
		Type: domain.MediaTypeMovie,
		File: domain.MediaFile{
			Name: "Blade.Runner.2049.2017.2160p.BluRay.x265.mkv",
		},
	}

	parseFilename(&item)

	if item.Title != "Blade Runner 2049" {
		t.Fatalf("expected title Blade Runner 2049, got %q", item.Title)
	}
	if item.Year != 2017 {
		t.Fatalf("expected year 2017, got %d", item.Year)
	}
}

func TestParseFilenameExtractsEpisodeAndRemovesReleaseNoise(t *testing.T) {
	item := domain.MediaItem{
		Type: domain.MediaTypeEpisode,
		File: domain.MediaFile{
			Name: "Severance.S02E03.Who.Is.Alive.1080p.WEB-DL.mkv",
		},
	}

	parseFilename(&item)

	if item.SeriesTitle != "Severance" {
		t.Fatalf("expected series title Severance, got %q", item.SeriesTitle)
	}
	if item.Season != 2 || item.Episode != 3 {
		t.Fatalf("expected S02E03, got S%02dE%02d", item.Season, item.Episode)
	}
	if item.Title != "Who Is Alive" {
		t.Fatalf("expected clean episode title, got %q", item.Title)
	}
}

func TestParseFilenameBuildsMusicFallbackFromFolders(t *testing.T) {
	item := domain.MediaItem{
		Type: domain.MediaTypeMusic,
		File: domain.MediaFile{
			Name: "01 - Opening Track.flac",
			Path: filepath.Join("media", "Artist Name", "Album Name", "01 - Opening Track.flac"),
		},
	}

	parseFilename(&item)

	if item.Artist != "Artist Name" || item.Album != "Album Name" {
		t.Fatalf("expected artist and album from folders, got artist=%q album=%q", item.Artist, item.Album)
	}
}

type scanTestProbe struct{}

func (scanTestProbe) Available(context.Context) bool { return true }
func (scanTestProbe) Probe(context.Context, string) (ports.ProbeResult, error) {
	return ports.ProbeResult{Technical: domain.TechnicalMetadata{Container: "mkv", VideoCodec: "h264", AudioCodec: "aac"}}, nil
}

func TestScanAutomaticallyClassifiesMoviesAndGroupsSeries(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	files := []string{
		filepath.Join(root, "Dune.2021.1080p.mkv"),
		filepath.Join(root, "Severance", "Season 01", "01 - Good News.mkv"),
		filepath.Join(root, "Severance", "Season 02", "Severance.S02E03.Who.Is.Alive.1080p.WEB-DL.mkv"),
	}
	for _, path := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create media directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
			t.Fatalf("write media file: %v", err)
		}
	}

	store, err := jsoncatalog.Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	library := domain.Library{
		ID: "mixed", Name: "Kõik", Kind: domain.LibraryKindMixed,
		Sources:   []domain.LibrarySource{{ID: "source", Path: root, CreatedAt: now}},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := store.CreateLibrary(ctx, library); err != nil {
		t.Fatalf("create library: %v", err)
	}

	scanner := NewScanService(store, scanTestProbe{}, NewJobManager())
	scanner.workers = 2
	scanner.now = func() time.Time { return now }
	if err := scanner.scan(ctx, library.ID, func(int, string) {}); err != nil {
		t.Fatalf("scan library: %v", err)
	}

	movies, err := store.ListMedia(ctx, domain.MediaQuery{LibraryID: library.ID, Types: []domain.MediaType{domain.MediaTypeMovie}, Limit: 20})
	if err != nil {
		t.Fatalf("list movies: %v", err)
	}
	series, err := store.ListMedia(ctx, domain.MediaQuery{LibraryID: library.ID, Types: []domain.MediaType{domain.MediaTypeSeries}, Limit: 20})
	if err != nil {
		t.Fatalf("list series: %v", err)
	}
	episodes, err := store.ListMedia(ctx, domain.MediaQuery{LibraryID: library.ID, Types: []domain.MediaType{domain.MediaTypeEpisode}, Limit: 20})
	if err != nil {
		t.Fatalf("list episodes: %v", err)
	}

	if len(movies.Items) != 1 || movies.Items[0].Title != "Dune" || movies.Items[0].Year != 2021 {
		t.Fatalf("movie classification failed: %+v", movies.Items)
	}
	if len(series.Items) != 1 || series.Items[0].Title != "Severance" || series.Items[0].SeasonCount != 2 || series.Items[0].EpisodeCount != 2 {
		t.Fatalf("series grouping failed: %+v", series.Items)
	}
	if len(episodes.Items) != 2 {
		t.Fatalf("expected two episodes, got %+v", episodes.Items)
	}
	for _, episode := range episodes.Items {
		if episode.SeriesID != series.Items[0].ID || episode.SeriesTitle != "Severance" {
			t.Fatalf("episode is not linked to series: %+v", episode)
		}
	}
}

type countingCompleteProbe struct {
	calls int
}

func (probe *countingCompleteProbe) Available(context.Context) bool { return true }
func (probe *countingCompleteProbe) Probe(context.Context, string) (ports.ProbeResult, error) {
	probe.calls++
	return ports.ProbeResult{Technical: domain.TechnicalMetadata{
		ProbeVersion: 3,
		Container:    "mkv",
		DurationMS:   7_200_000,
		VideoCodec:   "hevc",
		AudioCodec:   "eac3",
		Tracks: []domain.MediaTrack{
			{Index: 0, Kind: "video", Codec: "hevc", Width: 3840, Height: 2160},
			{Index: 1, Kind: "audio", Codec: "eac3", Channels: 6},
		},
	}}, nil
}

func TestScanStoresCompleteTechnicalMetadataAndDoesNotReprobeUnchangedFile(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	mediaPath := filepath.Join(root, "Example.Movie.2026.mkv")
	if err := os.WriteFile(mediaPath, []byte("test"), 0o600); err != nil {
		t.Fatalf("write media: %v", err)
	}
	modifiedAt := time.Date(2026, time.July, 12, 15, 30, 0, 0, time.UTC)
	if err := os.Chtimes(mediaPath, modifiedAt, modifiedAt); err != nil {
		t.Fatalf("set modified time: %v", err)
	}
	store, err := jsoncatalog.Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	library := domain.Library{
		ID: "movies", Name: "Filmid", Kind: domain.LibraryKindMovies,
		Sources:   []domain.LibrarySource{{ID: "source", Path: root, CreatedAt: modifiedAt}},
		CreatedAt: modifiedAt, UpdatedAt: modifiedAt,
	}
	if err := store.CreateLibrary(ctx, library); err != nil {
		t.Fatalf("create library: %v", err)
	}
	probe := &countingCompleteProbe{}
	scanner := NewScanService(store, probe, NewJobManager())
	scanner.workers = 1
	if err := scanner.scan(ctx, library.ID, func(int, string) {}); err != nil {
		t.Fatalf("first scan: %v", err)
	}
	if err := scanner.scan(ctx, library.ID, func(int, string) {}); err != nil {
		t.Fatalf("second scan: %v", err)
	}
	if probe.calls != 1 {
		t.Fatalf("expected one FFprobe call for an unchanged complete file, got %d", probe.calls)
	}
	page, err := store.ListMedia(ctx, domain.MediaQuery{LibraryID: library.ID, Types: []domain.MediaType{domain.MediaTypeMovie}, Sort: domain.MediaSortModifiedAt, Order: domain.SortDescending})
	if err != nil {
		t.Fatalf("list media: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected one item, got %d", len(page.Items))
	}
	item := page.Items[0]
	if item.Technical.DurationMS != 7_200_000 || item.Technical.VideoCodec != "hevc" || len(item.Technical.Tracks) != 2 {
		t.Fatalf("technical metadata was not persisted: %+v", item.Technical)
	}
	if !item.File.ModifiedAt.Equal(modifiedAt) {
		t.Fatalf("modified time was not preserved: got %s want %s", item.File.ModifiedAt, modifiedAt)
	}
}

func TestSeriesGroupingUsesSourceIdentityInsteadOfPersistedEpisodeYear(t *testing.T) {
	t.Parallel()
	first := domain.MediaItem{Type: domain.MediaTypeEpisode, SeriesTitle: "IT: Welcome to Derry", Year: 2025, File: domain.MediaFile{Name: "IT.Welcome.to.Derry.S01E01.1080p.WEB.mkv"}}
	second := domain.MediaItem{Type: domain.MediaTypeEpisode, SeriesTitle: "IT Welcome to Derry", Year: 2026, File: domain.MediaFile{Name: "IT.Welcome.to.Derry.S01E08.1080p.WEB.mkv"}}
	firstTitle, firstYear := classifiedSeriesIdentity(domain.LibraryKindSeries, first)
	secondTitle, secondYear := classifiedSeriesIdentity(domain.LibraryKindSeries, second)
	firstID := stableSeriesID("series-library", firstTitle, firstYear)
	secondID := stableSeriesID("series-library", secondTitle, secondYear)
	if firstID != secondID {
		t.Fatalf("expected one stable series id, got %q and %q", firstID, secondID)
	}
}

func TestRepairSeriesGroupingMergesLegacySplitParents(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := jsoncatalog.Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	now := time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)
	library := domain.Library{ID: "series", Name: "Seriaalid", Kind: domain.LibraryKindSeries, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateLibrary(ctx, library); err != nil {
		t.Fatalf("create library: %v", err)
	}
	legacyA := "series-legacy-2019"
	legacyB := "series-legacy-2026"
	episode8 := domain.MediaItem{
		ID: "episode-8", LibraryID: library.ID, Type: domain.MediaTypeEpisode,
		Title: "Episode 8", SortTitle: "For All Mankind 5 8", SeriesID: legacyA, SeriesTitle: "For All Mankind",
		Year: 2019, Season: 5, Episode: 8,
		File:    domain.MediaFile{Path: filepath.Join(t.TempDir(), "For.All.Mankind.S05E08.1080p.WEB.mkv"), Name: "For.All.Mankind.S05E08.1080p.WEB.mkv", ModifiedAt: now.Add(-time.Hour)},
		AddedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	}
	episode10 := domain.MediaItem{
		ID: "episode-10", LibraryID: library.ID, Type: domain.MediaTypeEpisode,
		Title: "Episode 10", SortTitle: "For All Mankind 5 10", SeriesID: legacyB, SeriesTitle: "For All Mankind",
		Year: 2026, Season: 5, Episode: 10,
		File:    domain.MediaFile{Path: filepath.Join(t.TempDir(), "For.All.Mankind.S05E10.1080p.WEB.mkv"), Name: "For.All.Mankind.S05E10.1080p.WEB.mkv", ModifiedAt: now},
		AddedAt: now, UpdatedAt: now,
	}
	parentA := domain.MediaItem{
		ID: legacyA, LibraryID: library.ID, Type: domain.MediaTypeSeries, Title: "For All Mankind", SortTitle: "For All Mankind",
		EpisodeCount: 1, SeasonCount: 1, ExternalIDs: domain.ExternalIDs{TMDb: "95396"},
		Metadata: domain.EditorialMetadata{MetadataSource: "tmdb", Summary: "Official series metadata", PosterURL: "https://example.test/poster.jpg"},
		AddedAt:  now.Add(-time.Hour), UpdatedAt: now,
	}
	parentB := domain.MediaItem{ID: legacyB, LibraryID: library.ID, Type: domain.MediaTypeSeries, Title: "For All Mankind", SortTitle: "For All Mankind", EpisodeCount: 1, SeasonCount: 1, AddedAt: now, UpdatedAt: now}
	if err := store.ReplaceLibraryMedia(ctx, library.ID, []domain.MediaItem{episode8, episode10, parentA, parentB}); err != nil {
		t.Fatalf("seed catalog: %v", err)
	}

	scanner := NewScanService(store, nil, NewJobManager())
	repaired, err := scanner.RepairSeriesGrouping(ctx)
	if err != nil {
		t.Fatalf("repair grouping: %v", err)
	}
	if repaired != 1 {
		t.Fatalf("expected one repaired library, got %d", repaired)
	}
	seriesPage, err := store.ListMedia(ctx, domain.MediaQuery{LibraryID: library.ID, Types: []domain.MediaType{domain.MediaTypeSeries}, Limit: 20})
	if err != nil {
		t.Fatalf("list series: %v", err)
	}
	if len(seriesPage.Items) != 1 {
		t.Fatalf("expected one grouped series, got %+v", seriesPage.Items)
	}
	series := seriesPage.Items[0]
	if series.EpisodeCount != 2 || series.SeasonCount != 1 {
		t.Fatalf("unexpected grouped counts: %+v", series)
	}
	if series.ExternalIDs.TMDb != "95396" || series.Metadata.PosterURL == "" {
		t.Fatalf("expected the strongest legacy metadata to survive: %+v", series)
	}
	episodesPage, err := store.ListMedia(ctx, domain.MediaQuery{LibraryID: library.ID, Types: []domain.MediaType{domain.MediaTypeEpisode}, Limit: 20})
	if err != nil {
		t.Fatalf("list episodes: %v", err)
	}
	if len(episodesPage.Items) != 2 {
		t.Fatalf("expected two episodes, got %+v", episodesPage.Items)
	}
	for _, episode := range episodesPage.Items {
		if episode.SeriesID != series.ID {
			t.Fatalf("episode still points to a split parent: %+v", episode)
		}
	}
	details, err := NewMediaService(store).GetSeriesDetails(ctx, series.ID)
	if err != nil {
		t.Fatalf("get series details: %v", err)
	}
	if len(details.Seasons) != 1 || len(details.Seasons[0].Episodes) != 2 || details.Seasons[0].Episodes[0].Episode != 8 || details.Seasons[0].Episodes[1].Episode != 10 {
		t.Fatalf("episodes are not grouped chronologically: %+v", details.Seasons)
	}
}

func TestSeriesGroupingNormalizesMixedReleaseNamesAndWeakFolderYears(t *testing.T) {
	t.Parallel()
	libraryID := "series-library"
	paths := []string{
		filepath.Join("D:\\Media\\IT Welcome to Derry (2025)", "IT.Welcome.to.Derry.S01E01.1080p.WEB.h264-ETHEL[EZTVx.to].mkv"),
		filepath.Join("D:\\Downloads\\IT Welcome to Derry 2026", "IT.Welcome.to.Derry.S01E02.1080p.x265-ELiTE.mkv"),
		filepath.Join("D:\\Downloads\\TV 2026", "IT Welcome to Derry S01E03 Now You See It 2160p HMAX WEB-DL DDP5 1 Atmos DV HDR H 265-FLUX.mkv"),
		filepath.Join("D:\\Downloads\\Completed", "IT - Welcome to Derry - S01E08 - Winter Fire.mkv"),
	}
	var expectedID string
	for _, path := range paths {
		item := domain.MediaItem{Type: domain.MediaTypeEpisode, File: domain.MediaFile{Path: path, Name: filepath.Base(path)}}
		title, year := classifiedSeriesIdentity(domain.LibraryKindSeries, item)
		if canonicalSeriesTitleKey(title) != "it welcome to derry" {
			t.Fatalf("unexpected canonical title for %q: title=%q key=%q", path, title, canonicalSeriesTitleKey(title))
		}
		if year != 0 {
			t.Fatalf("weak release-folder year must not split %q: got %d", path, year)
		}
		id := stableSeriesID(libraryID, title, year)
		if expectedID == "" {
			expectedID = id
		} else if id != expectedID {
			t.Fatalf("mixed release names split one series: %q != %q for %q", id, expectedID, path)
		}
	}
}

func TestSeriesGroupingKeepsExplicitRemakeYearsSeparate(t *testing.T) {
	t.Parallel()
	first := domain.MediaItem{Type: domain.MediaTypeEpisode, File: domain.MediaFile{Name: "Doctor.Who.1963.S01E01.mkv"}}
	second := domain.MediaItem{Type: domain.MediaTypeEpisode, File: domain.MediaFile{Name: "Doctor.Who.2005.S01E01.mkv"}}
	firstTitle, firstYear := classifiedSeriesIdentity(domain.LibraryKindSeries, first)
	secondTitle, secondYear := classifiedSeriesIdentity(domain.LibraryKindSeries, second)
	if firstYear != 1963 || secondYear != 2005 {
		t.Fatalf("explicit remake years were lost: %q %d / %q %d", firstTitle, firstYear, secondTitle, secondYear)
	}
	if stableSeriesID("series", firstTitle, firstYear) == stableSeriesID("series", secondTitle, secondYear) {
		t.Fatal("explicit remake years must remain separate")
	}
}

func TestRepairSeriesGroupingMergesScreenshotStyleSplit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := jsoncatalog.Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	now := time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)
	library := domain.Library{ID: "series", Name: "Seriaalid", Kind: domain.LibraryKindSeries, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateLibrary(ctx, library); err != nil {
		t.Fatalf("create library: %v", err)
	}
	legacyMain := "series-main"
	legacyAlt := "series-alt"
	files := []struct {
		id, name, parent string
		season, episode  int
	}{
		{"e1", "IT.Welcome.to.Derry.S01E01.1080p.WEB.h264-ETHEL[EZTVx.to].mkv", legacyMain, 1, 1},
		{"e2", "IT.Welcome.to.Derry.S01E02.1080p.x265-ELiTE.mkv", legacyMain, 1, 2},
		{"e3", "IT Welcome to Derry S01E03 Now You See It 2160p HMAX WEB-DL DDP5 1 Atmos DV HDR H 265-FLUX.mkv", legacyAlt, 1, 3},
		{"e4", "IT.Welcome.to.Derry.S01E04.1080p.WEB.mkv", legacyMain, 1, 4},
		{"e8", "IT - Welcome to Derry - S01E08 - Winter Fire.mkv", legacyAlt, 1, 8},
	}
	items := make([]domain.MediaItem, 0, len(files)+2)
	for index, file := range files {
		folder := "IT Welcome to Derry (2025)"
		if file.parent == legacyAlt {
			folder = "IT Welcome to Derry 2026"
		}
		items = append(items, domain.MediaItem{
			ID: file.id, LibraryID: library.ID, Type: domain.MediaTypeEpisode,
			Title: fmt.Sprintf("Episode %d", file.episode), SortTitle: fmt.Sprintf("IT %02d", file.episode),
			SeriesID: file.parent, SeriesTitle: "IT: Welcome to Derry", Year: 2025,
			Season: file.season, Episode: file.episode,
			File:    domain.MediaFile{Path: filepath.Join("D:\\TV", folder, file.name), Name: file.name, ModifiedAt: now.Add(time.Duration(index) * time.Minute)},
			AddedAt: now, UpdatedAt: now,
		})
	}
	items = append(items,
		domain.MediaItem{ID: legacyMain, LibraryID: library.ID, Type: domain.MediaTypeSeries, Title: "IT: Welcome to Derry", EpisodeCount: 3, SeasonCount: 1, ExternalIDs: domain.ExternalIDs{TMDb: "200875"}, Metadata: domain.EditorialMetadata{MetadataSource: "tmdb", PosterURL: "https://example.test/it.jpg"}, AddedAt: now, UpdatedAt: now},
		domain.MediaItem{ID: legacyAlt, LibraryID: library.ID, Type: domain.MediaTypeSeries, Title: "IT: Welcome to Derry", EpisodeCount: 2, SeasonCount: 1, AddedAt: now, UpdatedAt: now},
	)
	if err := store.ReplaceLibraryMedia(ctx, library.ID, items); err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	scanner := NewScanService(store, nil, NewJobManager())
	if repaired, err := scanner.RepairSeriesGrouping(ctx); err != nil || repaired != 1 {
		t.Fatalf("repair grouping: repaired=%d err=%v", repaired, err)
	}
	seriesPage, err := store.ListMedia(ctx, domain.MediaQuery{LibraryID: library.ID, Types: []domain.MediaType{domain.MediaTypeSeries}, Limit: 20})
	if err != nil {
		t.Fatalf("list series: %v", err)
	}
	if len(seriesPage.Items) != 1 || seriesPage.Items[0].EpisodeCount != len(files) {
		t.Fatalf("expected one five-episode parent, got %+v", seriesPage.Items)
	}
	details, err := NewMediaService(store).GetSeriesDetails(ctx, seriesPage.Items[0].ID)
	if err != nil {
		t.Fatalf("series details: %v", err)
	}
	got := make([]int, 0, len(details.Seasons[0].Episodes))
	for _, episode := range details.Seasons[0].Episodes {
		got = append(got, episode.Episode)
	}
	want := []int{1, 2, 3, 4, 8}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("episode order mismatch: got %v want %v", got, want)
	}
}

func TestMusicLibraryCandidateScanIgnoresVideoFiles(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"track.wav", "album.flac", "music-video.mkv", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	items, err := collectCandidates(domain.LibraryKindMusic, []domain.LibrarySource{{Path: root}})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected only two audio candidates, got %d: %v", len(items), items)
	}
	for _, item := range items {
		if filepath.Ext(item) != ".wav" && filepath.Ext(item) != ".flac" {
			t.Fatalf("video leaked into music scan: %s", item)
		}
	}
}
