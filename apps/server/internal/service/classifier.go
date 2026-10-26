package service

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

var (
	seasonEpisodePattern = regexp.MustCompile(`(?i)^(.*?)[ ._-]+s([0-9]{1,2})[ ._-]*e([0-9]{1,3})(?:[ ._-]+(.*))?$`)
	xEpisodePattern      = regexp.MustCompile(`(?i)^(.*?)[ ._-]+([0-9]{1,2})x([0-9]{1,3})(?:[ ._-]+(.*))?$`)
	wordEpisodePattern   = regexp.MustCompile(`(?i)^(.*?)[ ._-]+season[ ._-]*([0-9]{1,2})[ ._-]+episode[ ._-]*([0-9]{1,3})(?:[ ._-]+(.*))?$`)
	seasonFolderPattern  = regexp.MustCompile(`(?i)^(?:season|series|hooaeg|s)[ ._-]*([0-9]{1,2})$`)
	leadingEpisode       = regexp.MustCompile(`(?i)^(?:e(?:pisode)?[ ._-]*)?([0-9]{1,3})[ ._-]+(.*)$`)
	musicTrackPattern    = regexp.MustCompile(`(?i)^(?:disc[ ._-]*[0-9]+[ ._-]*)?([0-9]{1,3})[ ._-]+(.*)$`)
	yearPattern          = regexp.MustCompile(`19[0-9]{2}|20[0-9]{2}`)
	noisePattern         = regexp.MustCompile(`(?i)\b(4320p|2160p|1080p|720p|576p|480p|uhd|bluray|blu-ray|web[ ._-]?dl|webrip|hdrip|brrip|dvdrip|remux|xvid|x264|x265|h264|h265|hevc|av1|hdr10\+?|hdr|dolby[ ._-]?vision|proper|repack|aac|dts(?:-hd)?|truehd|atmos)\b.*$`)
)

type mediaIdentity struct {
	Type         domain.MediaType
	Title        string
	SortTitle    string
	Year         int
	SeriesTitle  string
	Season       int
	Episode      int
	EpisodeTitle string
	Artist       string
	Album        string
	TrackNumber  int
	// SeriesYearExplicit is true only when the year belongs to the series
	// identity itself (for example "Doctor.Who.2005.S01E01" or a
	// "Doctor Who (2005)\Season 01" folder). A release folder year may still
	// be useful for metadata search, but it must not split otherwise identical
	// episode groups.
	SeriesYearExplicit bool
}

func classifyMediaPath(kind domain.LibraryKind, path string) mediaIdentity {
	fileName := lastPathSegment(path)
	extension := strings.ToLower(filepath.Ext(fileName))
	base := strings.TrimSuffix(fileName, extension)
	if _, audio := audioExtensions[extension]; audio {
		title := cleanText(base)
		trackNumber := 0
		if matches := musicTrackPattern.FindStringSubmatch(base); len(matches) > 0 {
			trackNumber, _ = strconv.Atoi(matches[1])
			title = cleanText(matches[2])
		}
		album := cleanText(pathSegmentFromEnd(path, 1))
		artist := cleanText(pathSegmentFromEnd(path, 2))
		return mediaIdentity{Type: domain.MediaTypeMusic, Title: title, Album: album, Artist: artist, TrackNumber: trackNumber, SortTitle: normalizeSortTitle(fmt.Sprintf("%s %s %04d %s", artist, album, trackNumber, title))}
	}
	if episode, found := parseExplicitEpisode(base); found {
		return enrichEpisodeFromFolders(episode, path)
	}
	if episode, found := parseSeasonFolderEpisode(path, base); found {
		return episode
	}
	if kind == domain.LibraryKindSeries {
		return parseSeriesFallback(path, base)
	}
	return parseMovie(path, base)
}

func enrichEpisodeFromFolders(identity mediaIdentity, path string) mediaIdentity {
	parent := pathSegmentFromEnd(path, 1)
	if seasonFolderPattern.MatchString(parent) {
		parent = pathSegmentFromEnd(path, 2)
	}
	folderTitle, folderYear := cleanSeriesTitle(parent)
	missingSeriesTitle := identity.SeriesTitle == "" || strings.EqualFold(identity.SeriesTitle, "Tundmatu seriaal")
	if missingSeriesTitle {
		identity.SeriesTitle = folderTitle
		if identity.Year == 0 && folderYear > 0 {
			identity.Year = folderYear
			identity.SeriesYearExplicit = true
		}
	} else if identity.Year == 0 && folderYear > 0 {
		// Keep the folder year as a weak metadata hint, but do not use it in
		// the grouping key when the file name already contains a reliable
		// series title. Different release folders often contain different
		// years for episodes of the same show.
		identity.Year = folderYear
	}
	identity.SortTitle = episodeIdentitySortTitle(identity.SeriesTitle, identity.Season, identity.Episode, identity.EpisodeTitle)
	return identity
}
func parseExplicitEpisode(base string) (mediaIdentity, bool) {
	for _, pattern := range []*regexp.Regexp{seasonEpisodePattern, xEpisodePattern, wordEpisodePattern} {
		matches := pattern.FindStringSubmatch(base)
		if len(matches) == 0 {
			continue
		}
		seriesTitle, year := cleanSeriesTitle(matches[1])
		season, _ := strconv.Atoi(matches[2])
		episode, _ := strconv.Atoi(matches[3])
		episodeTitle := cleanText(noisePattern.ReplaceAllString(matches[4], ""))
		if episodeTitle == "" {
			episodeTitle = fmt.Sprintf("Osa %d", episode)
		}
		return mediaIdentity{Type: domain.MediaTypeEpisode, Title: episodeTitle, SortTitle: episodeIdentitySortTitle(seriesTitle, season, episode, episodeTitle), Year: year, SeriesTitle: seriesTitle, Season: season, Episode: episode, EpisodeTitle: episodeTitle, SeriesYearExplicit: year > 0}, true
	}
	return mediaIdentity{}, false
}
func parseSeasonFolderEpisode(path, base string) (mediaIdentity, bool) {
	parent := pathSegmentFromEnd(path, 1)
	matches := seasonFolderPattern.FindStringSubmatch(parent)
	if len(matches) == 0 {
		return mediaIdentity{}, false
	}
	season, _ := strconv.Atoi(matches[1])
	seriesTitle, year := cleanSeriesTitle(pathSegmentFromEnd(path, 2))
	episode := 0
	episodeTitle := cleanText(noisePattern.ReplaceAllString(base, ""))
	if parts := leadingEpisode.FindStringSubmatch(base); len(parts) > 0 {
		episode, _ = strconv.Atoi(parts[1])
		episodeTitle = cleanText(noisePattern.ReplaceAllString(parts[2], ""))
	}
	if episodeTitle == "" {
		if episode > 0 {
			episodeTitle = fmt.Sprintf("Osa %d", episode)
		} else {
			episodeTitle = cleanText(base)
		}
	}
	return mediaIdentity{Type: domain.MediaTypeEpisode, Title: episodeTitle, SortTitle: episodeIdentitySortTitle(seriesTitle, season, episode, episodeTitle), Year: year, SeriesTitle: seriesTitle, Season: season, Episode: episode, EpisodeTitle: episodeTitle, SeriesYearExplicit: year > 0}, true
}
func parseSeriesFallback(path, base string) mediaIdentity {
	seriesTitle, year := cleanSeriesTitle(pathSegmentFromEnd(path, 1))
	episode := 0
	title := cleanText(noisePattern.ReplaceAllString(base, ""))
	if matches := leadingEpisode.FindStringSubmatch(base); len(matches) > 0 {
		episode, _ = strconv.Atoi(matches[1])
		title = cleanText(noisePattern.ReplaceAllString(matches[2], ""))
	}
	if title == "" {
		title = cleanText(base)
	}
	return mediaIdentity{Type: domain.MediaTypeEpisode, Title: title, SortTitle: episodeIdentitySortTitle(seriesTitle, 0, episode, title), Year: year, SeriesTitle: seriesTitle, Episode: episode, EpisodeTitle: title, SeriesYearExplicit: year > 0}
}
func parseMovie(filePath, base string) mediaIdentity {
	candidate := bestMovieCandidate(filePath, base, 0)
	title := candidate.Title
	if title == "" {
		title = cleanText(base)
	}
	return mediaIdentity{Type: domain.MediaTypeMovie, Title: title, SortTitle: normalizeSortTitle(title), Year: candidate.Year}
}

func pathSegmentFromEnd(value string, offset int) string {
	segments := splitPathSegments(value)
	index := len(segments) - 1 - offset
	if index < 0 || index >= len(segments) {
		return ""
	}
	return segments[index]
}

func cleanSeriesTitle(value string) (string, int) {
	if candidate, _, ok := parseMetadataCandidate(value, 0); ok {
		return candidate.Title, candidate.Year
	}
	value = cleanText(noisePattern.ReplaceAllString(value, ""))
	if value == "" {
		value = "Tundmatu seriaal"
	}
	return value, 0
}
func episodeIdentitySortTitle(seriesTitle string, season, episode int, title string) string {
	return normalizeSortTitle(fmt.Sprintf("%s %04d %04d %s", seriesTitle, season, episode, title))
}
func releaseYear(value string) (int, int, bool) {
	matches := yearPattern.FindAllStringIndex(value, -1)
	for index := len(matches) - 1; index >= 0; index-- {
		match := matches[index]
		if !hasYearBoundaries(value, match[0], match[1]) {
			continue
		}
		year, err := strconv.Atoi(value[match[0]:match[1]])
		if err == nil {
			return year, match[0], true
		}
	}
	return 0, 0, false
}
func hasYearBoundaries(value string, start, end int) bool {
	const before = " ._-(["
	const after = " ._-)]"
	leftOK := start == 0 || strings.ContainsRune(before, rune(value[start-1]))
	rightOK := end == len(value) || strings.ContainsRune(after, rune(value[end]))
	return leftOK && rightOK
}
func cleanText(value string) string {
	value = strings.NewReplacer(".", " ", "_", " ").Replace(value)
	value = strings.Trim(value, " -_[]()")
	return strings.Join(strings.Fields(value), " ")
}
func normalizeSortTitle(value string) string {
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)
	for _, article := range []string{"the ", "an ", "a "} {
		if strings.HasPrefix(lower, article) {
			return strings.TrimSpace(value[len(article):])
		}
	}
	return value
}
