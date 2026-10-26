package service

import (
	"path/filepath"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestClassifierRecognizesCommonEpisodeNames(t *testing.T) {
	tests := []struct {
		name            string
		season, episode int
		series          string
	}{{"Severance.S02E03.Who.Is.Alive.1080p.WEB-DL.mkv", 2, 3, "Severance"}, {"Slow Horses - 3x04 - Uninvited Guests.mkv", 3, 4, "Slow Horses"}, {"Foundation Season 2 Episode 5 The Sighted and the Seen.mkv", 2, 5, "Foundation"}}
	for _, test := range tests {
		identity := classifyMediaPath(domain.LibraryKindMixed, filepath.Join("C:\\Media", test.name))
		if identity.Type != domain.MediaTypeEpisode || identity.Season != test.season || identity.Episode != test.episode || identity.SeriesTitle != test.series {
			t.Fatalf("unexpected classification for %s: %+v", test.name, identity)
		}
	}
}
func TestClassifierUsesSeasonFolderStructure(t *testing.T) {
	path := filepath.Join("Media", "The Bear", "Season 02", "03 - Sundae.mkv")
	identity := classifyMediaPath(domain.LibraryKindMixed, path)
	if identity.Type != domain.MediaTypeEpisode || identity.SeriesTitle != "The Bear" || identity.Season != 2 || identity.Episode != 3 || identity.Title != "Sundae" {
		t.Fatalf("unexpected classification: %+v", identity)
	}
}
func TestClassifierKeepsMovieYearAndCleansReleaseNoise(t *testing.T) {
	identity := classifyMediaPath(domain.LibraryKindMixed, filepath.Join("Movies", "Dune.Part.Two.2024.2160p.BluRay.x265.mkv"))
	if identity.Type != domain.MediaTypeMovie || identity.Title != "Dune Part Two" || identity.Year != 2024 {
		t.Fatalf("unexpected classification: %+v", identity)
	}
}
func TestClassifierDetectsEpisodeInsideMovieLibrary(t *testing.T) {
	identity := classifyMediaPath(domain.LibraryKindMovies, filepath.Join("Shows", "Severance.S01E01.Good.News.mkv"))
	if identity.Type != domain.MediaTypeEpisode {
		t.Fatalf("expected episode, got %+v", identity)
	}
}

func TestClassifierStripsAnimeReleaseGroupAndTechnicalTags(t *testing.T) {
	identity := classifyMediaPath(domain.LibraryKindMovies, filepath.Join("Movies", "[Anime Time] Suzume No Tojimari [1080p][HEVC 10bit x265][AAC][Multi Sub].mkv"))
	if identity.Type != domain.MediaTypeMovie || identity.Title != "Suzume No Tojimari" {
		t.Fatalf("unexpected classification: %+v", identity)
	}
}

func TestClassifierUsesInformativeParentFolderForShortReleaseFilename(t *testing.T) {
	identity := classifyMediaPath(domain.LibraryKindMovies, filepath.Join("Movies", "[ UsaBit.com ] - Lotte.And.The.Moonstone.Secret.2011.DVDRip.XviD-Ltu", "ltu-lotte.avi"))
	if identity.Type != domain.MediaTypeMovie || identity.Title != "Lotte And The Moonstone Secret" || identity.Year != 2011 {
		t.Fatalf("unexpected classification: %+v", identity)
	}
}

func TestClassifierStripsLooseReleaseGroupPrefix(t *testing.T) {
	identity := classifyMediaPath(domain.LibraryKindMovies, filepath.Join("Movies", "bonkai77] Wolf Children.2012.1080p.BluRay.x265.mkv"))
	if identity.Type != domain.MediaTypeMovie || identity.Title != "Wolf Children" || identity.Year != 2012 {
		t.Fatalf("unexpected classification: %+v", identity)
	}
}

func TestMusicFilenameProvidesTrackNumberAndCleanTitle(t *testing.T) {
	identity := classifyMediaPath(domain.LibraryKindMusic, filepath.Join("Artist", "Album", "01 - Opening Track.flac"))
	if identity.Type != domain.MediaTypeMusic || identity.TrackNumber != 1 || identity.Title != "Opening Track" || identity.Artist != "Artist" || identity.Album != "Album" {
		t.Fatalf("unexpected music identity: %+v", identity)
	}
}
