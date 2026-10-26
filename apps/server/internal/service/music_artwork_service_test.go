package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestMusicFileArtworkKeySeparatesFilesWithSameGenericTags(t *testing.T) {
	modified := time.Date(2026, time.July, 22, 12, 0, 0, 0, time.UTC)
	first := domain.MediaItem{
		Title: "First track", Artist: "Muusika", Album: "2026",
		File:      domain.MediaFile{Path: filepath.Join("C:", "Music", "first.flac"), SizeBytes: 100, ModifiedAt: modified},
		Technical: domain.TechnicalMetadata{DurationMS: 210000},
	}
	second := first
	second.Title = "Second track"
	second.File.Path = filepath.Join("C:", "Music", "second.flac")

	if musicFileArtworkKey(first) == musicFileArtworkKey(second) {
		t.Fatal("unrelated physical files must never share an artwork cache key")
	}
}

func TestMusicFileArtworkKeyChangesWhenFileChanges(t *testing.T) {
	item := domain.MediaItem{
		Title: "Track", Artist: "Artist", Album: "Album",
		File: domain.MediaFile{Path: filepath.Join("C:", "Music", "track.flac"), SizeBytes: 100, ModifiedAt: time.Unix(100, 0)},
	}
	changed := item
	changed.File.SizeBytes++
	if musicFileArtworkKey(item) == musicFileArtworkKey(changed) {
		t.Fatal("changed files must invalidate their cached artwork")
	}
}

func TestLocalMusicArtworkPrefersTrackSpecificSidecarInLooseFolder(t *testing.T) {
	dir := t.TempDir()
	firstAudio := filepath.Join(dir, "first.flac")
	secondAudio := filepath.Join(dir, "second.flac")
	writeTestFile(t, firstAudio)
	writeTestFile(t, secondAudio)
	writeTestFile(t, filepath.Join(dir, "cover.jpg"))
	trackCover := filepath.Join(dir, "first.jpg")
	writeTestFile(t, trackCover)

	item := domain.MediaItem{Album: "2026", File: domain.MediaFile{Path: firstAudio}}
	path, contentType := localMusicArtwork(item)
	if path != trackCover || contentType != "image/jpeg" {
		t.Fatalf("expected track sidecar %q, got %q (%s)", trackCover, path, contentType)
	}
}

func TestLocalMusicArtworkRejectsSharedCoverInGenericMultiTrackFolder(t *testing.T) {
	dir := t.TempDir()
	firstAudio := filepath.Join(dir, "first.flac")
	writeTestFile(t, firstAudio)
	writeTestFile(t, filepath.Join(dir, "second.mp3"))
	writeTestFile(t, filepath.Join(dir, "cover.jpg"))

	item := domain.MediaItem{Artist: "Muusika", Album: "2026", File: domain.MediaFile{Path: firstAudio}}
	if path, _ := localMusicArtwork(item); path != "" {
		t.Fatalf("generic loose-track folder cover must not be reused, got %q", path)
	}
}

func TestLocalMusicArtworkAcceptsCoverInMatchingAlbumFolder(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Artist - Discovery")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	audio := filepath.Join(dir, "01 One More Time.flac")
	writeTestFile(t, audio)
	writeTestFile(t, filepath.Join(dir, "02 Aerodynamic.flac"))
	cover := filepath.Join(dir, "cover.png")
	writeTestFile(t, cover)

	item := domain.MediaItem{Artist: "Daft Punk", Album: "Discovery", File: domain.MediaFile{Path: audio}}
	path, contentType := localMusicArtwork(item)
	if path != cover || contentType != "image/png" {
		t.Fatalf("expected album cover %q, got %q (%s)", cover, path, contentType)
	}
}

func TestLocalMusicArtworkAcceptsCoverForSingleTrackFolder(t *testing.T) {
	dir := t.TempDir()
	audio := filepath.Join(dir, "track.flac")
	writeTestFile(t, audio)
	cover := filepath.Join(dir, "cover.jpg")
	writeTestFile(t, cover)

	item := domain.MediaItem{Album: "2026", File: domain.MediaFile{Path: audio}}
	path, _ := localMusicArtwork(item)
	if path != cover {
		t.Fatalf("single-track folders may safely use their local cover, got %q", path)
	}
}

func TestGenerateMusicArtworkSVGCreatesSafeTrackSpecificCard(t *testing.T) {
	target := filepath.Join(t.TempDir(), "generated.svg")
	item := domain.MediaItem{
		Title:  `Night <Drive> (Bootleg)`,
		Artist: `Artist & Guest`,
		File:   domain.MediaFile{Name: "night-drive.flac"},
	}
	if err := generateMusicArtworkSVG(item, target); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	if !strings.Contains(text, "Night &lt;Drive&gt;") || !strings.Contains(text, "Artist &amp; Guest") || !strings.Contains(text, "BOOTLEG") {
		t.Fatalf("generated card omitted or failed to escape metadata: %s", text)
	}
	if strings.Contains(strings.ToLower(text), "<script") {
		t.Fatal("generated SVG must not contain scripts")
	}
}

func TestMusicCatalogArtworkKeyStaysTrackSpecificAndTracksExternalIDs(t *testing.T) {
	item := domain.MediaItem{ExternalIDs: domain.ExternalIDs{MusicBrainzRelease: "release-id", MusicBrainzReleaseGroup: "group-id"}}
	first := musicCatalogArtworkKey(item, "file-one")
	second := musicCatalogArtworkKey(item, "file-two")
	if first == second {
		t.Fatal("catalog artwork must remain track-specific even for the same release")
	}
	changed := item
	changed.ExternalIDs.MusicBrainzRelease = "corrected-release-id"
	if first == musicCatalogArtworkKey(changed, "file-one") {
		t.Fatal("corrected external IDs must invalidate cached catalog artwork")
	}
}

func writeTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPreferCatalogArtworkForGenericCollectionTags(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.flac")
	writeTestFile(t, first)
	writeTestFile(t, filepath.Join(dir, "second.mp3"))
	item := domain.MediaItem{Artist: "Muusika", Album: "2026", File: domain.MediaFile{Path: first}}
	if !preferCatalogArtwork(item) {
		t.Fatal("generic collection tags should prefer fingerprint/catalog artwork over embedded playlist art")
	}
}

func TestPreferEmbeddedArtworkForMeaningfulAlbumTags(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.flac")
	writeTestFile(t, first)
	writeTestFile(t, filepath.Join(dir, "second.mp3"))
	item := domain.MediaItem{Artist: "Daft Punk", Album: "Discovery", File: domain.MediaFile{Path: first}}
	if preferCatalogArtwork(item) {
		t.Fatal("well-tagged album tracks should keep embedded artwork priority")
	}
}

func TestPreferEmbeddedArtworkForSingleGenericTrack(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.flac")
	writeTestFile(t, first)
	item := domain.MediaItem{Artist: "Muusika", Album: "2026", File: domain.MediaFile{Path: first}}
	if preferCatalogArtwork(item) {
		t.Fatal("a single generic track should keep its embedded cover priority")
	}
}

func TestPreferEmbeddedArtworkForVariousArtistsAlbumFolder(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Compilation One")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(dir, "01.flac")
	writeTestFile(t, first)
	writeTestFile(t, filepath.Join(dir, "02.mp3"))
	item := domain.MediaItem{Artist: "Various Artists", Album: "Compilation One", File: domain.MediaFile{Path: first}}
	if preferCatalogArtwork(item) {
		t.Fatal("a matching compilation album folder should keep embedded artwork priority")
	}
}
