package opensubtitles

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestMovieHashUsesFileSizeAndFirstAndLastChunks(t *testing.T) {
	t.Parallel()
	const chunk = 64 * 1024
	content := make([]byte, chunk*2+16)
	for index := range content {
		content[index] = byte(index % 251)
	}
	path := filepath.Join(t.TempDir(), "movie.mkv")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	got, size, err := movieHash(path)
	if err != nil {
		t.Fatalf("movieHash: %v", err)
	}
	wantSum := uint64(len(content))
	for index := 0; index < chunk; index += 8 {
		wantSum += binary.LittleEndian.Uint64(content[index : index+8])
	}
	last := len(content) - chunk
	for index := last; index < len(content); index += 8 {
		wantSum += binary.LittleEndian.Uint64(content[index : index+8])
	}
	if got != fmt.Sprintf("%016x", wantSum) || size != int64(len(content)) {
		t.Fatalf("movieHash = %q, %d; want %016x, %d", got, size, wantSum, len(content))
	}
}

func TestNormalizeLanguagesDeduplicatesAliases(t *testing.T) {
	t.Parallel()
	got := normalizeLanguages([]string{"est", "et-EE", "ENG", "en", "unknown-language"})
	if len(got) != 2 || got[0] != "et" || got[1] != "en" {
		t.Fatalf("normalizeLanguages() = %#v", got)
	}
}

func TestBuildSearchQueriesUsesHashIDsAndBroadTitleFallback(t *testing.T) {
	t.Parallel()
	const chunk = 64 * 1024
	path := filepath.Join(t.TempDir(), "Send.Help.2025.1080p.WEB-DL.mkv")
	if err := os.WriteFile(path, make([]byte, chunk*2), 0o644); err != nil {
		t.Fatal(err)
	}
	queries := buildSearchQueries(domain.MediaItem{
		Type:        domain.MediaTypeMovie,
		Title:       "Send Help",
		Year:        2025,
		File:        domain.MediaFile{Path: path},
		ExternalIDs: domain.ExternalIDs{IMDb: "tt0012345", TMDb: "999"},
	}, []string{"et", "en"})
	if len(queries) < 4 {
		t.Fatalf("expected hash, id, title and broad title searches, got %d", len(queries))
	}
	if queries[0].values.Get("moviehash") == "" || queries[0].values.Get("query") == "" {
		t.Fatalf("hash query is incomplete: %v", queries[0].values)
	}
	if got := queries[1].values.Get("imdb_id"); got != "12345" {
		t.Fatalf("imdb_id = %q; want 12345", got)
	}
	if got := queries[2].values.Get("query"); got != "send help" {
		t.Fatalf("title query = %q; want send help", got)
	}
	if queries[2].values.Get("year") != "2025" || queries[3].values.Get("year") != "" {
		t.Fatalf("expected year-filtered then broad fallback: %v / %v", queries[2].values, queries[3].values)
	}
}

func TestBuildSearchQueriesUsesSeriesTitleForEpisode(t *testing.T) {
	t.Parallel()
	queries := buildSearchQueries(domain.MediaItem{
		Type:        domain.MediaTypeEpisode,
		Title:       "The File Release Name",
		SeriesTitle: "Slow Horses",
		Season:      3,
		Episode:     4,
	}, []string{"en"})
	if len(queries) == 0 {
		t.Fatal("expected episode search queries")
	}
	values := queries[0].values
	if values.Get("query") != "slow horses" || values.Get("season_number") != "3" || values.Get("episode_number") != "4" {
		t.Fatalf("unexpected episode query: %v", values)
	}
}
