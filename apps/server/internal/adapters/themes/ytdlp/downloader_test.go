//go:build !windows

package ytdlp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloaderStoresM4AAudioAtomically(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "fake-yt-dlp")
	content := `#!/bin/sh
output=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output" ]; then
    output="$2"
    shift 2
    continue
  fi
  shift
done
output=$(printf '%s' "$output" | sed 's/%(ext)s/m4a/g')
printf 'theme-audio' > "$output"
`
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}
	ffmpeg := filepath.Join(root, "ffmpeg")
	if err := os.WriteFile(ffmpeg, []byte("fake"), 0o600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, "cache", "theme.m4a")
	downloader := NewDynamic(func() string { return script }, func() string { return ffmpeg })
	if err := downloader.DownloadAudio(context.Background(), "https://www.youtube.com/watch?v=test", destination); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != "theme-audio" {
		t.Fatalf("unexpected audio payload: %q", payload)
	}
}
