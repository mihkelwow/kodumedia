package service

import (
	"testing"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestLikelyBoundaryFuzzyMismatchDetectsSiloSilom(t *testing.T) {
	if !likelyBoundaryFuzzyMismatch("Love of Silom", "Silo") {
		t.Fatal("expected Silo/Silom boundary mismatch to be detected")
	}
	if likelyBoundaryFuzzyMismatch("The Silo", "Silo") {
		t.Fatal("whole-word Silo title must not be treated as a mismatch")
	}
	if likelyBoundaryFuzzyMismatch("La casa de papel", "Money Heist") {
		t.Fatal("localized titles must not be invalidated merely because they differ")
	}
}

func TestRepairSuspiciousSeriesMatchesUsesEpisodeFilename(t *testing.T) {
	items := []domain.MediaItem{
		{
			ID: "series-silo", Type: domain.MediaTypeSeries, Title: "Love of Silom", SortTitle: "love of silom", Year: 2026,
			ExternalIDs: domain.ExternalIDs{TMDb: "999"},
			Metadata:    domain.EditorialMetadata{MetadataSource: "tmdb", Summary: "Wrong show"},
		},
		{
			ID: "episode-1", SeriesID: "series-silo", Type: domain.MediaTypeEpisode,
			Title: "Episode 1", SeriesTitle: "Love of Silom", Season: 3, Episode: 1,
			File:        domain.MediaFile{Path: `D:\Series\Silo.S03E01.1080p.WEB.h264-ETHEL[EZTVx.to].mkv`, Name: "Silo.S03E01.1080p.WEB.h264-ETHEL[EZTVx.to].mkv"},
			ExternalIDs: domain.ExternalIDs{TMDb: "123"},
			Metadata:    domain.EditorialMetadata{MetadataSource: "tmdb", Summary: "Wrong episode"},
		},
	}
	repaired := repairSuspiciousSeriesMatches(items)
	if !repaired["series-silo"] {
		t.Fatal("series should have been scheduled for a fresh metadata match")
	}
	if items[0].Title != "Silo" || items[0].ExternalIDs.TMDb != "" || items[0].Metadata.MetadataSource != "" {
		t.Fatalf("series was not reset to filename identity: %+v", items[0])
	}
	if items[1].SeriesTitle != "Silo" || items[1].ExternalIDs.TMDb != "" || items[1].Metadata.MetadataSource != "" {
		t.Fatalf("episode was not reset with its corrected parent identity: %+v", items[1])
	}
}
