package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

// musicSidecar is an intentionally small, file-local metadata contract. It is
// designed for releases that public metadata services frequently miss, such as
// bootlegs, edits and unofficial remixes. The sidecar is treated as
// user-curated and therefore takes precedence over embedded tags.
type musicSidecar struct {
	Title       string   `json:"title"`
	Artist      string   `json:"artist"`
	Album       string   `json:"album"`
	Year        *int     `json:"year"`
	TrackNumber *int     `json:"trackNumber"`
	DiscNumber  *int     `json:"discNumber"`
	Genres      []string `json:"genres"`
	Styles      []string `json:"styles"`
	Moods       []string `json:"moods"`
}

func applyMusicSidecar(item *domain.MediaItem, mediaPath string) {
	if item == nil || strings.TrimSpace(mediaPath) == "" {
		return
	}
	path := findMusicSidecar(mediaPath)
	if path == "" {
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() > 1024*1024 {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var sidecar musicSidecar
	if json.Unmarshal(data, &sidecar) != nil {
		return
	}
	if value := strings.TrimSpace(sidecar.Title); value != "" {
		item.Title = value
	}
	if value := strings.TrimSpace(sidecar.Artist); value != "" {
		item.Artist = value
	}
	if value := strings.TrimSpace(sidecar.Album); value != "" {
		item.Album = value
	}
	if sidecar.Year != nil && *sidecar.Year >= 0 {
		item.Year = *sidecar.Year
	}
	if sidecar.TrackNumber != nil && *sidecar.TrackNumber >= 0 {
		item.TrackNumber = *sidecar.TrackNumber
	}
	if sidecar.DiscNumber != nil && *sidecar.DiscNumber >= 0 {
		item.DiscNumber = *sidecar.DiscNumber
	}

	current := domain.MusicClassification{}
	if item.MusicClassification != nil {
		current = *item.MusicClassification
	}
	classificationChanged := false
	if sidecar.Genres != nil {
		current.Genres = normalizeMusicValues(sidecar.Genres)
		current.GenreSource = "sidecar"
		classificationChanged = true
	}
	if sidecar.Styles != nil {
		current.Styles = normalizeMusicValues(sidecar.Styles)
		current.StyleSource = "sidecar"
		classificationChanged = true
	}
	if sidecar.Moods != nil {
		current.Moods = normalizeMusicValues(sidecar.Moods)
		current.MoodSource = "sidecar"
		classificationChanged = true
	}
	if classificationChanged {
		current.Source = classificationSummarySource(current)
		current.Confidence = "verified"
		item.MusicClassification = &current
		item.Metadata.Genres = append([]string(nil), current.Genres...)
	}
	item.SortTitle = normalizeSortTitle(strings.Join([]string{item.Artist, item.Album, item.Title}, " "))
}

func findMusicSidecar(mediaPath string) string {
	candidates := []string{
		mediaPath + ".kodumedia.json",
		strings.TrimSuffix(mediaPath, filepath.Ext(mediaPath)) + ".kodumedia.json",
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func normalizeMusicValues(values []string) []string {
	joined := strings.Join(values, ";")
	return splitMusicGenres(joined)
}
