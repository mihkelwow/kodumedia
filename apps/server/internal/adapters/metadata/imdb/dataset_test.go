package imdb

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func compressedRatings(t *testing.T, rows ...string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write([]byte("tconst\taverageRating\tnumVotes\n")); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if _, err := writer.Write([]byte(row + "\n")); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestDatasetManagerDownloadsAndConditionallyRefreshesRemoteDataset(t *testing.T) {
	payload := compressedRatings(t, "tt0000001\t8.2\t123")
	var requests atomic.Int32
	var conditional atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.Method == http.MethodHead {
			response.Header().Set("Content-Length", "1")
			response.WriteHeader(http.StatusOK)
			return
		}
		if request.Header.Get("If-None-Match") == `"ratings-v1"` {
			conditional.Add(1)
			response.WriteHeader(http.StatusNotModified)
			return
		}
		response.Header().Set("ETag", `"ratings-v1"`)
		response.Header().Set("Content-Type", "application/gzip")
		_, _ = response.Write(payload)
	}))
	defer server.Close()

	source := server.URL + "/title.ratings.tsv.gz"
	manager, err := NewDatasetManager(func() string { return source }, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	manager.client = server.Client()

	if err := manager.Refresh(context.Background(), nil); err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	if manager.LocalPath() == "" {
		t.Fatal("expected a local cache path")
	}
	if status := manager.Status(); !status.Available || status.SizeBytes != int64(len(payload)) || status.UpdatedAt == nil {
		t.Fatalf("unexpected status after download: %+v", status)
	}
	if err := manager.Refresh(context.Background(), nil); err != nil {
		t.Fatalf("conditional refresh: %v", err)
	}
	if conditional.Load() != 1 {
		t.Fatalf("expected one conditional request, got %d (all requests %d)", conditional.Load(), requests.Load())
	}
}

func TestDatasetManagerRejectsInvalidCacheOnNotModified(t *testing.T) {
	payload := compressedRatings(t, "tt0000001\t8.2\t123")
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("If-None-Match") == `"ratings-v1"` {
			response.WriteHeader(http.StatusNotModified)
			return
		}
		response.Header().Set("ETag", `"ratings-v1"`)
		_, _ = response.Write(payload)
	}))
	defer server.Close()

	source := server.URL + "/title.ratings.tsv.gz"
	manager, err := NewDatasetManager(func() string { return source }, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	manager.client = server.Client()
	if err := manager.Refresh(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manager.cachePath, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.Refresh(context.Background(), nil); err == nil {
		t.Fatal("expected refresh to reject an invalid cached dataset")
	}
	if status := manager.Status(); status.Available || status.Refreshing {
		t.Fatalf("invalid cache must not be reported as available: %+v", status)
	}
}

func TestRatingsProviderLoadsGzipIntoCompactLookupIndex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "title.ratings.tsv.gz")
	if err := os.WriteFile(path, compressedRatings(t,
		"tt0000002\t7.1\t22",
		"tt0000001\t8.4\t111",
	), 0o600); err != nil {
		t.Fatal(err)
	}
	provider := NewRatingsProvider(path)
	rating, votes, found, err := provider.Lookup(context.Background(), "tt0000001")
	if err != nil {
		t.Fatal(err)
	}
	if !found || rating < 8.39 || rating > 8.41 || votes != 111 {
		t.Fatalf("unexpected lookup: rating=%v votes=%d found=%v", rating, votes, found)
	}
	if _, _, found, err := provider.Lookup(context.Background(), "tt9999999"); err != nil || found {
		t.Fatalf("unexpected missing lookup: found=%v err=%v", found, err)
	}
}

func TestRatingsProviderDetectsCompressionByContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "title.ratings.tsv.gz")
	plain := []byte("tconst\taverageRating\tnumVotes\ntt0000007\t9.1\t77\n")
	if err := os.WriteFile(path, plain, 0o600); err != nil {
		t.Fatal(err)
	}
	provider := NewRatingsProvider(path)
	rating, votes, found, err := provider.Lookup(context.Background(), "tt0000007")
	if err != nil {
		t.Fatal(err)
	}
	if !found || rating < 9.09 || rating > 9.11 || votes != 77 {
		t.Fatalf("unexpected lookup from plain content: rating=%v votes=%d found=%v", rating, votes, found)
	}
}
