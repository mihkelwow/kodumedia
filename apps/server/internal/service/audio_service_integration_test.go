package service

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/adapters/catalog/jsoncatalog"
	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestLosslessAudioFallbackPreservesDecodedSamples(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not available")
	}
	root := t.TempDir()
	source := filepath.Join(root, "source.wav")
	generate := exec.Command(ffmpeg,
		"-v", "error",
		"-f", "lavfi", "-i", "sine=frequency=1000:sample_rate=96000:duration=0.25",
		"-ac", "2",
		"-c:a", "pcm_s24le",
		"-y", source,
	)
	if output, generateErr := generate.CombinedOutput(); generateErr != nil {
		t.Fatalf("generate source: %v: %s", generateErr, output)
	}
	catalog, err := jsoncatalog.Open(filepath.Join(root, "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	library := domain.Library{ID: "music", Name: "Music", Kind: domain.LibraryKindMusic, CreatedAt: now, UpdatedAt: now}
	if err := catalog.CreateLibrary(ctx, library); err != nil {
		t.Fatal(err)
	}
	item := domain.MediaItem{
		ID: "track", LibraryID: library.ID, Type: domain.MediaTypeMusic, Title: "Track", SortTitle: "Track",
		File:      domain.MediaFile{Path: source, Name: filepath.Base(source), Extension: ".wav"},
		Technical: domain.TechnicalMetadata{AudioCodec: "pcm_s24le", AudioBitDepth: 24, AudioSampleRate: 96000},
		AddedAt:   now, UpdatedAt: now,
	}
	if err := catalog.ReplaceLibraryMedia(ctx, library.ID, []domain.MediaItem{item}); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	service := NewAudioService(catalog, func() string { return ffmpeg })
	if err := service.StreamLossless(ctx, item.ID, &output); err != nil {
		t.Fatal(err)
	}
	converted := filepath.Join(root, "converted.wav")
	if err := os.WriteFile(converted, output.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if sourceHash := decodedAudioHash(t, ffmpeg, source); sourceHash != decodedAudioHash(t, ffmpeg, converted) {
		t.Fatalf("lossless fallback changed decoded PCM samples")
	}
}

func decodedAudioHash(t *testing.T, ffmpeg, path string) string {
	t.Helper()
	command := exec.Command(ffmpeg, "-v", "error", "-i", path, "-map", "0:a:0", "-f", "framemd5", "-")
	output, err := command.Output()
	if err != nil {
		t.Fatalf("hash %s: %v", path, err)
	}
	return string(output)
}
