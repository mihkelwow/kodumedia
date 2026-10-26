package local

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestDiscoverAdjacentSubtitleLabelsWithoutMistakingTitleForLanguage(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mediaPath := filepath.Join(root, "It.2017.1080p.mkv")
	mustWrite(t, mediaPath, nil)
	mustWrite(t, filepath.Join(root, "It.2017.1080p.en.forced.srt"), []byte("1\n00:00:00,000 --> 00:00:01,000\nHello\n"))
	mustWrite(t, filepath.Join(root, "unrelated.et.srt"), []byte("ignore"))

	store, err := New(func() string { return "ffmpeg" }, filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	tracks, err := store.Discover(context.Background(), domain.MediaItem{
		ID: "movie-it", File: domain.MediaFile{Path: mediaPath},
	})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(tracks) != 1 {
		t.Fatalf("expected 1 matching subtitle, got %d: %#v", len(tracks), tracks)
	}
	if tracks[0].Language != "en" || !tracks[0].Forced {
		t.Fatalf("unexpected labels: %#v", tracks[0])
	}
	if tracks[0].URL == "" {
		t.Fatal("text subtitle URL was not generated")
	}
}

func TestDiscoverEmbeddedAndVobSubPair(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mediaPath := filepath.Join(root, "Film.2024.mkv")
	mustWrite(t, mediaPath, nil)
	mustWrite(t, filepath.Join(root, "Film.2024.et.idx"), []byte("VobSub index file"))
	mustWrite(t, filepath.Join(root, "Film.2024.et.sub"), []byte("binary payload"))

	store, err := New(nil, filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	tracks, err := store.Discover(context.Background(), domain.MediaItem{
		ID: "movie", File: domain.MediaFile{Path: mediaPath},
		Technical: domain.TechnicalMetadata{Tracks: []domain.MediaTrack{
			{Index: 2, StreamOrdinal: 0, Kind: "subtitle", Codec: "hdmv_pgs_subtitle", Language: "est", Forced: true},
		}},
	})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(tracks) != 2 {
		t.Fatalf("expected embedded PGS and one VobSub track, got %d: %#v", len(tracks), tracks)
	}
	for _, track := range tracks {
		if track.Kind != domain.SubtitleKindImage || !track.RequiresBurnIn {
			t.Fatalf("expected image subtitle requiring burn-in: %#v", track)
		}
	}
}

func TestSaveDownloadedSanitizesAndDiscoversTrack(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mediaPath := filepath.Join(root, "Movie.2020.mkv")
	mustWrite(t, mediaPath, nil)
	store, err := New(nil, filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	track, err := store.SaveDownloaded(context.Background(), domain.MediaItem{
		ID: "movie/unsafe", File: domain.MediaFile{Path: mediaPath},
	}, "../../Movie.2020.et.srt", strings.NewReader("1\n00:00:00,000 --> 00:00:01,000\nTere\n"))
	if err != nil {
		t.Fatalf("SaveDownloaded: %v", err)
	}
	if track.Source != domain.SubtitleSourceDownloaded || track.Language != "et" {
		t.Fatalf("unexpected downloaded track: %#v", track)
	}
	if filepath.Base(track.Path) != "Movie.2020.et.srt" {
		t.Fatalf("unsafe filename was not sanitized: %q", track.Path)
	}
	if _, err := os.Stat(track.Path); err != nil {
		t.Fatalf("saved subtitle missing: %v", err)
	}
}

func mustWrite(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestWebVTTConvertsExternalSubtitleThroughTemporaryFile(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	root := t.TempDir()
	mediaPath := filepath.Join(root, "Movie.2024.mkv")
	subtitlePath := filepath.Join(root, "Movie.2024.et.srt")
	mustWrite(t, mediaPath, nil)
	mustWrite(t, subtitlePath, []byte("1\n00:00:00,000 --> 00:00:01,500\nTere maailm!\n"))
	store, err := New(func() string { return "ffmpeg" }, filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	media := domain.MediaItem{ID: "movie-vtt", File: domain.MediaFile{Path: mediaPath}}
	tracks, err := store.Discover(context.Background(), media)
	if err != nil || len(tracks) != 1 {
		t.Fatalf("Discover: tracks=%#v err=%v", tracks, err)
	}
	path, err := store.WebVTT(context.Background(), media, tracks[0])
	if err != nil {
		t.Fatalf("WebVTT: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.HasPrefix(strings.TrimPrefix(text, "\ufeff"), "WEBVTT") || !strings.Contains(text, "Tere maailm!") {
		t.Fatalf("unexpected WebVTT output: %q", text)
	}
}
