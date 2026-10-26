package service

import (
	"testing"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestApplyMetadataResultReplacesVersionTwoTrailerAssets(t *testing.T) {
	item := &domain.MediaItem{Metadata: domain.EditorialMetadata{
		TrailerYouTubeKey: "old-restricted",
		TrailerName:       "Old trailer",
		TrailerCandidates: []domain.TrailerCandidate{{YouTubeKey: "old-restricted"}},
		HeroAssetsVersion: 1,
	}}

	applyMetadataResult(item, domain.MetadataResult{Metadata: domain.EditorialMetadata{
		HeroAssetsVersion: 2,
	}})

	if item.Metadata.TrailerYouTubeKey != "" || item.Metadata.TrailerName != "" || len(item.Metadata.TrailerCandidates) != 0 {
		t.Fatalf("stale trailer assets survived authoritative refresh: %+v", item.Metadata)
	}
	if item.Metadata.HeroAssetsVersion != 2 {
		t.Fatalf("hero asset version was not updated: %+v", item.Metadata)
	}
}
