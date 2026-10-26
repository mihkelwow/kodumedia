package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserFiltersFilesAndKeepsDirectories(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "Folder"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ffmpeg.exe"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	listing, err := New().Browse(context.Background(), root, true, []string{".exe"})
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Entries) != 2 || !listing.Entries[0].Directory || listing.Entries[1].Name != "ffmpeg.exe" {
		t.Fatalf("unexpected entries: %+v", listing.Entries)
	}
}

func TestBrowserCapsLargeDirectoryWithoutReturningEveryEntry(t *testing.T) {
	root := t.TempDir()
	for index := 0; index < maxEntries+1; index++ {
		name := filepath.Join(root, fmt.Sprintf("Folder-%04d", index))
		if err := os.Mkdir(name, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	listing, err := New().Browse(context.Background(), root, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Entries) != maxEntries || !listing.Truncated {
		t.Fatalf("expected %d entries and truncated=true, got %d and %v", maxEntries, len(listing.Entries), listing.Truncated)
	}
}
