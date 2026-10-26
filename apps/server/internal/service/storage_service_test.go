package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/adapters/catalog/jsoncatalog"
	"github.com/mihkelwow/kodumedia/internal/domain"
)

type storageTestTranscoder struct {
	usage          int64
	policy         string
	maximumBytes   int64
	reserveBytes   int64
	reservePercent int
}

func (s *storageTestTranscoder) ApplyStoragePolicy(policy string, maximumBytes, reserveBytes int64, reservePercent int) {
	s.policy = policy
	s.maximumBytes = maximumBytes
	s.reserveBytes = reserveBytes
	s.reservePercent = reservePercent
}

func (s *storageTestTranscoder) OptimizedCacheUsage() int64 {
	return s.usage
}

func (s *storageTestTranscoder) ClearOptimizedCache() int64 {
	freed := s.usage
	s.usage = 0
	return freed
}

func TestStorageServiceSupportsEmptyFirstRunAndImmediatePolicy(t *testing.T) {
	dataDir := t.TempDir()
	catalog, err := jsoncatalog.Open(filepath.Join(dataDir, "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	userFile := filepath.Join(dataDir, "user-owned-data.txt")
	if err := os.WriteFile(userFile, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	transcoder := &storageTestTranscoder{usage: 256 << 20}
	settings := domain.ServerSettings{Quality: domain.QualitySettings{
		CachePolicy: "watched", CacheMaxGB: 12, CacheFreeReserveGB: 8, CacheFreeReservePercent: 15,
	}}
	service := NewStorageService(dataDir, catalog, func() domain.ServerSettings { return settings }, transcoder, nil, nil, nil, nil, nil)

	result, err := service.Cleanup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if transcoder.policy != "watched" || transcoder.maximumBytes != 12<<30 || transcoder.reserveBytes != 8<<30 || transcoder.reservePercent != 15 {
		t.Fatalf("saved storage policy was not applied: %+v", transcoder)
	}
	if result.Report.DataDirectory != dataDir || len(result.Report.Categories) != 9 {
		t.Fatalf("unexpected first-run storage report: %+v", result.Report)
	}
	if result.Report.RequiredReserveBytes < 8<<30 {
		t.Fatalf("storage report ignored the absolute free-space reserve: %+v", result.Report)
	}
	if payload, err := os.ReadFile(userFile); err != nil || string(payload) != "keep" {
		t.Fatalf("unrelated first-run data was removed or changed: payload=%q error=%v", payload, err)
	}

	cleared, err := service.Clear(context.Background(), "clear-optimized")
	if err != nil {
		t.Fatal(err)
	}
	if cleared.FreedBytes != 256<<20 || transcoder.usage != 0 {
		t.Fatalf("optimized cache clear did not report reclaimed bytes: %+v", cleared)
	}
}
