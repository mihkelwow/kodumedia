package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type deezerArtist struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Link string `json:"link"`
}

type deezerAlbumSummary struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Cover     string `json:"cover"`
	CoverBig  string `json:"cover_big"`
	CoverXL   string `json:"cover_xl"`
	Tracklist string `json:"tracklist"`
}

type deezerTrack struct {
	ID           int64              `json:"id"`
	Title        string             `json:"title"`
	TitleShort   string             `json:"title_short"`
	TitleVersion string             `json:"title_version"`
	Link         string             `json:"link"`
	Duration     int64              `json:"duration"`
	Rank         int                `json:"rank"`
	BPM          float64            `json:"bpm"`
	ISRC         string             `json:"isrc"`
	Artist       deezerArtist       `json:"artist"`
	Album        deezerAlbumSummary `json:"album"`
}

type deezerGenre struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type deezerAlbum struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Link        string `json:"link"`
	Cover       string `json:"cover"`
	CoverBig    string `json:"cover_big"`
	CoverXL     string `json:"cover_xl"`
	ReleaseDate string `json:"release_date"`
	Genres      struct {
		Data []deezerGenre `json:"data"`
	} `json:"genres"`
}

func (p *MusicMetadataProvider) lookupDeezer(ctx context.Context, item domain.MediaItem) (MusicMetadataMatch, bool, error) {
	title := strings.TrimSpace(item.Title)
	if title == "" {
		return MusicMetadataMatch{}, false, nil
	}
	queryParts := make([]string, 0, 2)
	if artist := strings.TrimSpace(item.Artist); artist != "" {
		queryParts = append(queryParts, `artist:"`+strings.ReplaceAll(artist, `"`, ``)+`"`)
	}
	queryParts = append(queryParts, `track:"`+strings.ReplaceAll(title, `"`, ``)+`"`)
	endpoint := strings.TrimRight(p.deezerURL, "/") + "/search/track?limit=15&q=" + url.QueryEscape(strings.Join(queryParts, " "))
	var payload struct {
		Data  []deezerTrack `json:"data"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"error,omitempty"`
	}
	if err := p.deezerJSON(ctx, endpoint, &payload); err != nil {
		return MusicMetadataMatch{}, false, err
	}
	if payload.Error != nil {
		return MusicMetadataMatch{}, false, fmt.Errorf("Deezer search failed: %s", strings.TrimSpace(payload.Error.Message))
	}
	type scored struct {
		track deezerTrack
		score int
	}
	candidates := make([]scored, 0, len(payload.Data))
	for _, candidate := range payload.Data {
		score := scoreDeezerTrack(item, candidate)
		if score > 0 {
			candidates = append(candidates, scored{track: candidate, score: score})
		}
	}
	if len(candidates) == 0 {
		return MusicMetadataMatch{}, false, nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].track.Rank > candidates[j].track.Rank
	})
	if candidates[0].score < 72 || (len(candidates) > 1 && candidates[0].score-candidates[1].score < 5 && candidates[0].track.ID != candidates[1].track.ID) {
		return MusicMetadataMatch{}, false, nil
	}

	best := candidates[0].track
	if best.ID != 0 {
		var details deezerTrack
		if err := p.deezerJSON(ctx, strings.TrimRight(p.deezerURL, "/")+"/track/"+strconv.FormatInt(best.ID, 10), &details); err == nil && details.ID != 0 {
			if details.Album.ID == 0 {
				details.Album = best.Album
			}
			if details.Artist.ID == 0 {
				details.Artist = best.Artist
			}
			best = details
		}
	}

	album := deezerAlbum{}
	if best.Album.ID != 0 {
		_ = p.deezerJSON(ctx, strings.TrimRight(p.deezerURL, "/")+"/album/"+strconv.FormatInt(best.Album.ID, 10), &album)
	}
	genres := make([]string, 0, len(album.Genres.Data))
	for _, genre := range album.Genres.Data {
		if name := strings.TrimSpace(genre.Name); name != "" {
			genres = append(genres, name)
		}
	}
	genres = normalizeCatalogGenres(genres)
	artworkURL := firstNonEmptyMusicURL(album.CoverXL, best.Album.CoverXL, album.CoverBig, best.Album.CoverBig, album.Cover, best.Album.Cover)
	sourceURL := strings.TrimSpace(best.Link)
	if sourceURL == "" {
		sourceURL = strings.TrimSpace(album.Link)
	}
	return MusicMetadataMatch{
		Title:  strings.TrimSpace(best.Title),
		Artist: strings.TrimSpace(best.Artist.Name),
		Album:  firstNonEmptyMusicText(strings.TrimSpace(best.Album.Title), strings.TrimSpace(album.Title)),
		// Deezer exposes broad album genres but not a dependable style field.
		// Keep styles empty here so the enrichment layer can mark any title/BPM
		// inference as local rather than incorrectly attributing it to Deezer.
		Genres:        genres,
		ArtworkURL:    artworkURL,
		ArtworkSource: "deezer",
		Source:        "deezer",
		SourceURL:     sourceURL,
		Method:        "catalog",
		Score:         candidates[0].score,
		BPM:           best.BPM,
	}, true, nil
}

func (p *MusicMetadataProvider) deezerJSON(ctx context.Context, endpoint string, target any) error {
	p.deezerMu.Lock()
	defer p.deezerMu.Unlock()
	if wait := time.Until(p.lastDeezer.Add(140 * time.Millisecond)); wait > 0 {
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", p.userAgent())
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	p.lastDeezer = p.now()
	if resp.StatusCode == http.StatusNotFound {
		return domain.ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Deezer returned %s", resp.Status)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(target)
}

func scoreDeezerTrack(item domain.MediaItem, candidate deezerTrack) int {
	localTitle := normalizeMusicMatchText(item.Title)
	remoteTitle := normalizeMusicMatchText(firstNonEmptyMusicText(candidate.Title, candidate.TitleShort+" "+candidate.TitleVersion))
	if localTitle == "" || remoteTitle == "" {
		return 0
	}
	localBase := normalizeMusicBaseTitle(item.Title)
	remoteBase := normalizeMusicBaseTitle(firstNonEmptyMusicText(candidate.Title, candidate.TitleShort))
	localVersions := musicVersionTokens(item.Title)
	remoteVersions := musicVersionTokens(candidate.Title + " " + candidate.TitleVersion)
	if hasCriticalMusicVersion(localVersions) && !musicVersionOverlap(localVersions, remoteVersions) {
		return 0
	}

	score := 0
	similarity := musicTokenSimilarity(localTitle, remoteTitle)
	baseSimilarity := musicTokenSimilarity(localBase, remoteBase)
	switch {
	case localTitle == remoteTitle:
		score += 58
	case similarity >= 0.88:
		score += 50
	case localBase != "" && localBase == remoteBase:
		score += 42
	case baseSimilarity >= 0.86:
		score += 36
	default:
		return 0
	}
	if len(localVersions) > 0 && musicVersionOverlap(localVersions, remoteVersions) {
		score += 12
	}
	localArtist := normalizeMusicMatchText(item.Artist)
	remoteArtist := normalizeMusicMatchText(candidate.Artist.Name)
	if localArtist != "" {
		artistSimilarity := musicTokenSimilarity(localArtist, remoteArtist)
		switch {
		case localArtist == remoteArtist:
			score += 26
		case artistSimilarity >= 0.82:
			score += 21
		case strings.Contains(remoteArtist, localArtist) || strings.Contains(localArtist, remoteArtist):
			score += 14
		default:
			return 0
		}
	}
	if item.Technical.DurationMS > 0 && candidate.Duration > 0 {
		remoteDuration := candidate.Duration * 1000
		diff := item.Technical.DurationMS - remoteDuration
		if diff < 0 {
			diff = -diff
		}
		ratio := float64(diff) / float64(item.Technical.DurationMS)
		switch {
		case diff <= 3000:
			score += 20
		case diff <= 7000:
			score += 13
		case ratio <= 0.06:
			score += 7
		case ratio > 0.15:
			return 0
		}
	}
	if album := normalizeMusicMatchText(item.Album); album != "" {
		remoteAlbum := normalizeMusicMatchText(candidate.Album.Title)
		if album == remoteAlbum {
			score += 8
		} else if musicTokenSimilarity(album, remoteAlbum) >= 0.8 {
			score += 5
		}
	}
	return score
}

func normalizeMusicBaseTitle(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, separator := range []string{"(", "[", "{", " - "} {
		if index := strings.Index(value, separator); index > 0 {
			value = value[:index]
			break
		}
	}
	return normalizeMusicMatchText(value)
}

func musicVersionTokens(value string) map[string]struct{} {
	normalized := normalizedClassificationText(value)
	result := map[string]struct{}{}
	aliases := map[string][]string{
		"remix":        {"remix", "rmx"},
		"bootleg":      {"bootleg"},
		"mashup":       {"mashup", "mash-up"},
		"edit":         {"edit", "re-edit", "reedit"},
		"rework":       {"rework"},
		"vip":          {"vip"},
		"live":         {"live"},
		"acoustic":     {"acoustic", "unplugged"},
		"instrumental": {"instrumental"},
		"extended":     {"extended", "club mix"},
		"radio":        {"radio edit", "radio mix"},
		"dub":          {"dub", "dub mix"},
	}
	for key, values := range aliases {
		if containsAnyPhrase(normalized, values) {
			result[key] = struct{}{}
		}
	}
	return result
}

func hasCriticalMusicVersion(values map[string]struct{}) bool {
	for _, value := range []string{"remix", "bootleg", "mashup", "edit", "rework", "vip"} {
		if _, ok := values[value]; ok {
			return true
		}
	}
	return false
}

func musicVersionOverlap(left, right map[string]struct{}) bool {
	for value := range left {
		if _, ok := right[value]; ok {
			return true
		}
	}
	return false
}

func firstNonEmptyMusicText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonEmptyMusicURL(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://") {
			return value
		}
	}
	return ""
}
