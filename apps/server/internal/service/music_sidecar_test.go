package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestMusicSidecarOverridesClassificationForUnofficialRelease(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mediaPath := filepath.Join(root, "Artist - Bootleg Remix.flac")
	if err := os.WriteFile(mediaPath, []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	sidecar := `{
  "title": "Bootleg Remix",
  "artist": "DJ Example",
  "album": "Unreleased Edits",
  "year": 2026,
  "trackNumber": 4,
  "genres": ["Electronic"],
  "styles": ["Progressive House", "Bootleg"],
  "moods": ["Energetic"]
}`
	if err := os.WriteFile(mediaPath+".kodumedia.json", []byte(sidecar), 0o600); err != nil {
		t.Fatal(err)
	}
	item := domain.MediaItem{
		Type:   domain.MediaTypeMusic,
		Title:  "Original",
		Artist: "Unknown",
		Album:  "Unknown",
		MusicClassification: &domain.MusicClassification{
			Genres: []string{"Dance"}, Source: "embedded", Confidence: "tagged",
		},
	}
	applyMusicSidecar(&item, mediaPath)
	if item.Title != "Bootleg Remix" || item.Artist != "DJ Example" || item.Album != "Unreleased Edits" || item.TrackNumber != 4 {
		t.Fatalf("sidecar fields not applied: %+v", item)
	}
	if item.MusicClassification == nil || item.MusicClassification.Source != "sidecar" || item.MusicClassification.Confidence != "verified" {
		t.Fatalf("sidecar provenance missing: %+v", item.MusicClassification)
	}
	if got := item.MusicClassification.Styles; len(got) != 2 || got[0] != "Progressive House" || got[1] != "Bootleg" {
		t.Fatalf("unexpected styles: %#v", got)
	}
	if len(item.Metadata.Genres) != 1 || item.Metadata.Genres[0] != "Electronic" {
		t.Fatalf("editorial genre mirror not updated: %#v", item.Metadata.Genres)
	}
}
