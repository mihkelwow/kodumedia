package service

import (
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

var (
	leadingBracketGroupPattern = regexp.MustCompile(`^\s*[\[(]([^\])]{1,64})[\])]\s*[-_. ]*`)
	leadingLooseGroupPattern   = regexp.MustCompile(`^\s*([A-Za-z0-9][A-Za-z0-9._ -]{1,31})\]\s*[-_. ]*`)
	bracketGroupPattern        = regexp.MustCompile(`[\[(]([^\])]{1,96})[\])]`)
	websitePrefixPattern       = regexp.MustCompile(`(?i)^\s*((?:www\.)?[a-z0-9-]+(?:\.[a-z0-9-]+)*\.(?:com|net|org|tv|to|me|io|co))([-_. ]+)(.+)$`)
	releaseSuffixPattern       = regexp.MustCompile(`(?i)[ ._-]+(?:proper|repack|internal|limited|readnfo|rerip|extended|uncut|remastered)(?:[ ._-].*)?$`)
	shortReleaseCodePattern    = regexp.MustCompile(`(?i)^[a-z0-9]{2,6}[ ._-]+[a-z0-9]{2,10}$`)
)

var genericMediaFolders = map[string]struct{}{
	"film": {}, "films": {}, "filmid": {}, "movie": {}, "movies": {},
	"series": {}, "seriaal": {}, "seriaalid": {}, "shows": {}, "tv": {},
	"media": {}, "video": {}, "videos": {}, "downloads": {}, "download": {},
	"incoming": {}, "complete": {}, "completed": {}, "torrent": {}, "torrents": {},
	"1080p": {}, "2160p": {}, "4k": {}, "uhd": {}, "bluray": {}, "web-dl": {},
}

type rankedMetadataCandidate struct {
	candidate domain.MetadataTitleCandidate
	score     int
}

// metadataTitleCandidates produces a small, ranked list of clean search hints.
// It deliberately stays independent from TMDb so another metadata provider can
// reuse the same release-name parsing later.
func metadataTitleCandidates(item domain.MediaItem, parent *domain.MediaItem) []domain.MetadataTitleCandidate {
	ranked := make([]rankedMetadataCandidate, 0, 8)
	seen := make(map[string]struct{}, 8)

	add := func(raw string, fallbackYear, baseScore int) {
		candidate, quality, ok := parseMetadataCandidate(raw, fallbackYear)
		if !ok {
			return
		}
		key := normalizeCandidateKey(candidate.Title, candidate.Year)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		ranked = append(ranked, rankedMetadataCandidate{candidate: candidate, score: baseScore + quality})
	}

	if item.Type == domain.MediaTypeEpisode {
		if parent != nil {
			add(parent.Title, firstPositive(parent.Year, item.Year), 140)
		}
		add(item.SeriesTitle, item.Year, 120)
	} else {
		add(item.Title, item.Year, 105)
	}

	if item.File.Name != "" {
		add(item.File.Name, item.Year, 125)
	}

	filePath := strings.TrimSpace(item.File.Path)
	if filePath == "" {
		filePath = strings.TrimSpace(item.File.RelativePath)
	}
	if filePath != "" {
		segments := splitPathSegments(filePath)
		if len(segments) > 0 {
			add(segments[len(segments)-1], item.Year, 125)
		}
		folderRank := 0
		for index := len(segments) - 2; index >= 0 && folderRank < 1; index-- {
			segment := strings.TrimSpace(segments[index])
			if segment == "" || isGenericMediaFolder(segment) || seasonFolderPattern.MatchString(segment) {
				continue
			}
			add(segment, item.Year, 135-(folderRank*14))
			folderRank++
		}
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return len(ranked[i].candidate.Title) > len(ranked[j].candidate.Title)
		}
		return ranked[i].score > ranked[j].score
	})
	if len(ranked) > 6 {
		ranked = ranked[:6]
	}
	result := make([]domain.MetadataTitleCandidate, 0, len(ranked))
	for _, value := range ranked {
		result = append(result, value.candidate)
	}
	return result
}

func bestMovieCandidate(filePath, fallbackTitle string, fallbackYear int) domain.MetadataTitleCandidate {
	item := domain.MediaItem{
		Type:  domain.MediaTypeMovie,
		Title: fallbackTitle,
		Year:  fallbackYear,
		File: domain.MediaFile{
			Path: filePath,
			Name: lastPathSegment(filePath),
		},
	}
	candidates := metadataTitleCandidates(item, nil)
	if len(candidates) > 0 {
		return candidates[0]
	}
	return domain.MetadataTitleCandidate{Title: cleanText(fallbackTitle), Year: fallbackYear}
}

func parseMetadataCandidate(raw string, fallbackYear int) (domain.MetadataTitleCandidate, int, bool) {
	value := lastPathSegment(strings.TrimSpace(raw))
	if value == "" {
		return domain.MetadataTitleCandidate{}, 0, false
	}
	if extension := strings.ToLower(path.Ext(strings.ReplaceAll(value, "\\", "/"))); extension != "" {
		if _, knownMediaExtension := supportedExtensions[extension]; knownMediaExtension {
			value = strings.TrimSuffix(value, extension)
		}
	}
	value = stripReleaseDecorations(value)

	year := fallbackYear
	if foundYear, index, found := releaseYear(value); found {
		year = foundYear
		if index > 0 {
			value = value[:index]
		}
	}
	value = noisePattern.ReplaceAllString(value, "")
	value = releaseSuffixPattern.ReplaceAllString(value, "")
	value = cleanText(value)
	if value == "" || isGenericMediaFolder(value) {
		return domain.MetadataTitleCandidate{}, 0, false
	}

	words := strings.Fields(value)
	letters := 0
	for _, r := range value {
		if unicode.IsLetter(r) {
			letters++
		}
	}
	if letters < 2 {
		return domain.MetadataTitleCandidate{}, 0, false
	}

	quality := minInt(len(words), 7) * 5
	quality += minInt(len([]rune(value))/4, 18)
	if year > 0 {
		quality += 18
	}
	if len(words) == 1 && len([]rune(value)) < 9 {
		quality -= 28
	}
	if shortReleaseCodePattern.MatchString(value) {
		quality -= 18
	}
	if looksMostlyLikeReleaseCode(value) {
		quality -= 24
	}
	return domain.MetadataTitleCandidate{Title: value, Year: year}, quality, true
}

func stripReleaseDecorations(value string) string {
	value = strings.TrimSpace(value)
	if match := websitePrefixPattern.FindStringSubmatch(value); len(match) > 0 {
		domainName := strings.ToLower(strings.TrimSpace(match[1]))
		separator := match[2]
		if strings.HasPrefix(domainName, "www.") || strings.ContainsAny(separator, " -") || knownReleaseWebsite(domainName) {
			value = strings.TrimSpace(match[3])
		}
	}

	for {
		match := leadingBracketGroupPattern.FindStringSubmatchIndex(value)
		if len(match) == 0 {
			break
		}
		group := value[match[2]:match[3]]
		remainder := strings.TrimSpace(value[match[1]:])
		if !looksLikeReleaseGroup(group, remainder) {
			break
		}
		value = remainder
	}

	if match := leadingLooseGroupPattern.FindStringSubmatchIndex(value); len(match) > 0 {
		group := value[match[2]:match[3]]
		remainder := strings.TrimSpace(value[match[1]:])
		if looksLikeReleaseGroup(group, remainder) {
			value = remainder
		}
	}

	value = bracketGroupPattern.ReplaceAllStringFunc(value, func(group string) string {
		inner := strings.Trim(group, "[]() ")
		if isTechnicalGroup(inner) {
			return " "
		}
		return group
	})
	return strings.TrimSpace(value)
}

func knownReleaseWebsite(domainName string) bool {
	for _, marker := range []string{
		"eztv", "ettv", "rarbg", "yts", "yify", "usabit", "torrent", "1337x", "thepiratebay", "limetorrents",
	} {
		if strings.Contains(domainName, marker) {
			return true
		}
	}
	return false
}

func looksLikeReleaseGroup(group, remainder string) bool {
	group = strings.TrimSpace(group)
	remainder = strings.TrimSpace(remainder)
	if group == "" || remainder == "" {
		return false
	}
	lower := strings.ToLower(group)
	for _, marker := range []string{
		"anime", "subs", "sub", "dub", "encode", "encoder", "release", "group", "team",
		"torrent", "rip", "web", "www", ".com", ".net", ".org", ".tv", ".to",
		"rarbg", "yts", "yify", "ettv", "eztv", "usabit", "bonkai", "scampler",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	if _, _, startsWithYear := releaseYear(remainder); startsWithYear && yearAtStart(remainder) {
		return false
	}
	wordsAfter := strings.Fields(cleanText(remainder))
	if len(wordsAfter) < 2 || len([]rune(group)) > 28 {
		return false
	}
	hasDigit := strings.IndexFunc(group, unicode.IsDigit) >= 0
	return hasDigit || strings.Contains(group, " ") || strings.ContainsAny(group, "._-")
}

func isTechnicalGroup(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return false
	}
	for _, marker := range []string{
		"4320p", "2160p", "1080p", "720p", "576p", "480p", "4k", "uhd",
		"x264", "x265", "h264", "h265", "hevc", "av1", "10bit", "8bit", "hdr",
		"bluray", "blu-ray", "web-dl", "webrip", "bdrip", "brrip", "dvdrip", "hdtv",
		"aac", "dts", "truehd", "atmos", "ddp", "ac3", "flac", "multi sub", "multisub",
		"multi audio", "dual audio", "subbed", "dubbed", "proper", "repack",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func looksMostlyLikeReleaseCode(value string) bool {
	letters := 0
	digits := 0
	separators := 0
	upper := 0
	for _, r := range value {
		switch {
		case unicode.IsLetter(r):
			letters++
			if unicode.IsUpper(r) {
				upper++
			}
		case unicode.IsDigit(r):
			digits++
		case r == '-' || r == '_' || r == '.':
			separators++
		}
	}
	if letters == 0 {
		return true
	}
	return len(strings.Fields(value)) <= 2 && digits > 0 && (separators > 0 || upper == 0)
}

func splitPathSegments(value string) []string {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	parts := strings.Split(normalized, "/")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" && !strings.HasSuffix(trimmed, ":") {
			result = append(result, trimmed)
		}
	}
	return result
}

func lastPathSegment(value string) string {
	parts := splitPathSegments(value)
	if len(parts) == 0 {
		return strings.TrimSpace(value)
	}
	return parts[len(parts)-1]
}

func isGenericMediaFolder(value string) bool {
	cleaned := strings.ToLower(cleanText(value))
	_, found := genericMediaFolders[cleaned]
	return found
}

func normalizeCandidateKey(title string, year int) string {
	return strings.ToLower(strings.Join(strings.FieldsFunc(title, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), " ")) + "|" + strconv.Itoa(year)
}

func yearAtStart(value string) bool {
	trimmed := strings.TrimLeft(value, " ._-([")
	_, index, found := releaseYear(trimmed)
	return found && index == 0
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
