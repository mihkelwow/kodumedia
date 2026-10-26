package service

import (
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type inferredMusicClassification struct {
	Genres     []string
	Styles     []string
	Moods      []string
	Source     string
	Confidence string
}

type musicStyleRule struct {
	Style    string
	Genre    string
	Keywords []string
}

// The rules are deliberately ordered from specific subgenres to broad parent
// styles. They are used only for missing fields and never overwrite tags or a
// strong catalog result.
var musicStyleRules = []musicStyleRule{
	{Style: "Afro House", Genre: "Electronic", Keywords: []string{"afro house", "afrohouse"}},
	{Style: "Deep House", Genre: "Electronic", Keywords: []string{"deep house", "deephouse"}},
	{Style: "Tech House", Genre: "Electronic", Keywords: []string{"tech house", "techhouse"}},
	{Style: "Progressive House", Genre: "Electronic", Keywords: []string{"progressive house", "prog house"}},
	{Style: "Future House", Genre: "Electronic", Keywords: []string{"future house"}},
	{Style: "Tropical House", Genre: "Electronic", Keywords: []string{"tropical house"}},
	{Style: "Bass House", Genre: "Electronic", Keywords: []string{"bass house"}},
	{Style: "Electro House", Genre: "Electronic", Keywords: []string{"electro house"}},
	{Style: "Slap House", Genre: "Electronic", Keywords: []string{"slap house"}},
	{Style: "Acid House", Genre: "Electronic", Keywords: []string{"acid house"}},
	{Style: "Jackin House", Genre: "Electronic", Keywords: []string{"jackin house", "jackin' house"}},
	{Style: "Melodic Techno", Genre: "Electronic", Keywords: []string{"melodic techno"}},
	{Style: "Industrial Techno", Genre: "Electronic", Keywords: []string{"industrial techno"}},
	{Style: "Minimal Techno", Genre: "Electronic", Keywords: []string{"minimal techno"}},
	{Style: "Hard Techno", Genre: "Electronic", Keywords: []string{"hard techno"}},
	{Style: "Peak Time Techno", Genre: "Electronic", Keywords: []string{"peak time techno", "peak-time techno"}},
	{Style: "Progressive Trance", Genre: "Electronic", Keywords: []string{"progressive trance", "prog trance"}},
	{Style: "Uplifting Trance", Genre: "Electronic", Keywords: []string{"uplifting trance"}},
	{Style: "Vocal Trance", Genre: "Electronic", Keywords: []string{"vocal trance"}},
	{Style: "Psytrance", Genre: "Electronic", Keywords: []string{"psytrance", "psy trance", "psy-trance"}},
	{Style: "Goa Trance", Genre: "Electronic", Keywords: []string{"goa trance"}},
	{Style: "Liquid Drum & Bass", Genre: "Electronic", Keywords: []string{"liquid drum and bass", "liquid drum & bass", "liquid dnb"}},
	{Style: "Neurofunk", Genre: "Electronic", Keywords: []string{"neurofunk"}},
	{Style: "Jump Up", Genre: "Electronic", Keywords: []string{"jump up dnb", "jump-up dnb"}},
	{Style: "Drum & Bass", Genre: "Electronic", Keywords: []string{"drum and bass", "drum & bass", "dnb", "d&b"}},
	{Style: "Dubstep", Genre: "Electronic", Keywords: []string{"dubstep"}},
	{Style: "UK Garage", Genre: "Electronic", Keywords: []string{"uk garage", "2-step", "2 step garage"}},
	{Style: "Future Garage", Genre: "Electronic", Keywords: []string{"future garage"}},
	{Style: "Hardstyle", Genre: "Electronic", Keywords: []string{"hardstyle"}},
	{Style: "Hardcore", Genre: "Electronic", Keywords: []string{"hardcore techno", "gabber", "happy hardcore"}},
	{Style: "Breakbeat", Genre: "Electronic", Keywords: []string{"breakbeat", "breaks"}},
	{Style: "Synthwave", Genre: "Electronic", Keywords: []string{"synthwave", "retrowave", "outrun"}},
	{Style: "Chillwave", Genre: "Electronic", Keywords: []string{"chillwave"}},
	{Style: "Downtempo", Genre: "Electronic", Keywords: []string{"downtempo", "down tempo"}},
	{Style: "Trip-Hop", Genre: "Electronic", Keywords: []string{"trip hop", "trip-hop"}},
	{Style: "Ambient", Genre: "Electronic", Keywords: []string{"ambient", "soundscape", "drone"}},
	{Style: "Lo-Fi Hip-Hop", Genre: "Hip-Hop", Keywords: []string{"lofi hip hop", "lo-fi hip-hop", "lo fi hip hop", "chillhop"}},
	{Style: "Boom Bap", Genre: "Hip-Hop", Keywords: []string{"boom bap", "boombap"}},
	{Style: "Trap", Genre: "Hip-Hop", Keywords: []string{"trap", "trap beat"}},
	{Style: "Drill", Genre: "Hip-Hop", Keywords: []string{"drill", "uk drill"}},
	{Style: "Cloud Rap", Genre: "Hip-Hop", Keywords: []string{"cloud rap"}},
	{Style: "Alternative Hip-Hop", Genre: "Hip-Hop", Keywords: []string{"alternative hip hop", "alternative hip-hop"}},
	{Style: "Neo Soul", Genre: "R&B", Keywords: []string{"neo soul", "neo-soul"}},
	{Style: "Contemporary R&B", Genre: "R&B", Keywords: []string{"contemporary r&b", "contemporary rnb"}},
	{Style: "Indie Pop", Genre: "Pop", Keywords: []string{"indie pop"}},
	{Style: "Synthpop", Genre: "Pop", Keywords: []string{"synthpop", "synth pop"}},
	{Style: "Electropop", Genre: "Pop", Keywords: []string{"electropop", "electro pop"}},
	{Style: "Dream Pop", Genre: "Pop", Keywords: []string{"dream pop"}},
	{Style: "K-Pop", Genre: "Pop", Keywords: []string{"k-pop", "kpop"}},
	{Style: "Alternative Rock", Genre: "Rock", Keywords: []string{"alternative rock", "alt rock"}},
	{Style: "Indie Rock", Genre: "Rock", Keywords: []string{"indie rock"}},
	{Style: "Hard Rock", Genre: "Rock", Keywords: []string{"hard rock"}},
	{Style: "Punk Rock", Genre: "Rock", Keywords: []string{"punk rock", "pop punk"}},
	{Style: "Post-Rock", Genre: "Rock", Keywords: []string{"post rock", "post-rock"}},
	{Style: "Shoegaze", Genre: "Rock", Keywords: []string{"shoegaze"}},
	{Style: "Progressive Rock", Genre: "Rock", Keywords: []string{"progressive rock", "prog rock"}},
	{Style: "Metalcore", Genre: "Metal", Keywords: []string{"metalcore"}},
	{Style: "Death Metal", Genre: "Metal", Keywords: []string{"death metal"}},
	{Style: "Black Metal", Genre: "Metal", Keywords: []string{"black metal"}},
	{Style: "Heavy Metal", Genre: "Metal", Keywords: []string{"heavy metal"}},
	{Style: "Progressive Metal", Genre: "Metal", Keywords: []string{"progressive metal", "prog metal"}},
	{Style: "Smooth Jazz", Genre: "Jazz", Keywords: []string{"smooth jazz"}},
	{Style: "Bebop", Genre: "Jazz", Keywords: []string{"bebop", "be-bop"}},
	{Style: "Jazz Fusion", Genre: "Jazz", Keywords: []string{"jazz fusion", "fusion jazz"}},
	{Style: "Modern Classical", Genre: "Classical", Keywords: []string{"modern classical", "neo classical", "neoclassical"}},
	{Style: "Film Score", Genre: "Soundtrack", Keywords: []string{"film score", "movie score", "original score", "ost"}},
	{Style: "Reggae", Genre: "Reggae", Keywords: []string{"reggae"}},
	{Style: "Dancehall", Genre: "Reggae", Keywords: []string{"dancehall"}},
	{Style: "Afrobeats", Genre: "World", Keywords: []string{"afrobeats", "afrobeat"}},
	{Style: "Latin Pop", Genre: "Latin", Keywords: []string{"latin pop"}},
	{Style: "Bossa Nova", Genre: "Latin", Keywords: []string{"bossa nova"}},
	{Style: "Funk", Genre: "Funk", Keywords: []string{"funk"}},
	{Style: "Disco", Genre: "Electronic", Keywords: []string{"disco", "nu disco", "nu-disco"}},
	{Style: "House", Genre: "Electronic", Keywords: []string{"house mix", "house remix", "house music"}},
	{Style: "Techno", Genre: "Electronic", Keywords: []string{"techno"}},
	{Style: "Trance", Genre: "Electronic", Keywords: []string{"trance"}},
	{Style: "Hip-Hop", Genre: "Hip-Hop", Keywords: []string{"hip hop", "hip-hop", "rap"}},
}

var genreAliases = map[string]string{
	"alternative":        "Alternative",
	"alternative & punk": "Alternative",
	"ambient":            "Electronic",
	"blues":              "Blues",
	"children's music":   "Children",
	"classical":          "Classical",
	"country":            "Country",
	"dance":              "Electronic",
	"dance & electronic": "Electronic",
	"electronic":         "Electronic",
	"electronica":        "Electronic",
	"folk":               "Folk",
	"funk":               "Funk",
	"hip hop":            "Hip-Hop",
	"hip-hop":            "Hip-Hop",
	"hip-hop/rap":        "Hip-Hop",
	"jazz":               "Jazz",
	"latin":              "Latin",
	"metal":              "Metal",
	"new age":            "New Age",
	"pop":                "Pop",
	"r&b":                "R&B",
	"r&b/soul":           "R&B",
	"reggae":             "Reggae",
	"rock":               "Rock",
	"singer/songwriter":  "Singer-Songwriter",
	"soul":               "Soul",
	"soundtrack":         "Soundtrack",
	"world":              "World",
	"worldwide":          "World",
}

var defaultStyleByGenre = map[string]string{
	"Alternative":       "Alternative",
	"Blues":             "Blues",
	"Children":          "Children",
	"Classical":         "Classical",
	"Country":           "Country",
	"Electronic":        "Electronic",
	"Folk":              "Folk",
	"Funk":              "Funk",
	"Hip-Hop":           "Hip-Hop",
	"Jazz":              "Jazz",
	"Latin":             "Latin",
	"Metal":             "Metal",
	"New Age":           "New Age",
	"Pop":               "Pop",
	"R&B":               "R&B",
	"Reggae":            "Reggae",
	"Rock":              "Rock",
	"Singer-Songwriter": "Singer-Songwriter",
	"Soul":              "Soul",
	"Soundtrack":        "Soundtrack",
	"World":             "World",
}

func inferMusicClassification(item domain.MediaItem, seedGenres, seedStyles []string, bpm float64) inferredMusicClassification {
	genres := normalizeCatalogGenres(seedGenres)
	styles := normalizeMusicValues(seedStyles)
	text := normalizedClassificationText(strings.Join([]string{
		item.Artist,
		item.Title,
		item.Album,
		item.File.RelativePath,
		filepath.Base(item.File.Path),
		strings.Join(genres, " "),
		strings.Join(styles, " "),
	}, " "))

	for _, rule := range musicStyleRules {
		if containsAnyPhrase(text, rule.Keywords) {
			styles = appendUniqueFold(styles, rule.Style)
			if len(genres) == 0 {
				genres = append(genres, rule.Genre)
			}
			break
		}
	}

	if len(styles) == 0 && bpm > 0 {
		genre, style := classificationFromBPM(genres, bpm)
		if genre != "" && len(genres) == 0 {
			genres = append(genres, genre)
		}
		if style != "" {
			styles = append(styles, style)
		}
	}

	if len(genres) == 0 && len(styles) > 0 {
		if genre := genreForStyle(styles[0]); genre != "" {
			genres = append(genres, genre)
		}
	}
	if len(styles) == 0 && len(genres) > 0 {
		if style := defaultStyleByGenre[genres[0]]; style != "" {
			styles = append(styles, style)
		}
	}
	genres = removeOtherWhenSpecific(genres)
	styles = removeOtherWhenSpecific(styles)

	source := "local-inference"
	confidence := "estimated"
	if len(genres) == 0 && len(styles) == 0 {
		// Do not leave the UI empty. This fallback is deliberately explicit and
		// low-confidence so it cannot be confused with a verified catalog style.
		genres = []string{"Other"}
		styles = []string{"Other"}
		source = "fallback"
		confidence = "low"
	}
	return inferredMusicClassification{
		Genres: normalizeMusicValues(genres), Styles: normalizeMusicValues(styles),
		Source: source, Confidence: confidence,
	}
}

func normalizeCatalogGenres(values []string) []string {
	result := make([]string, 0, len(values))
	for _, raw := range values {
		raw = strings.TrimSpace(raw)
		if alias := genreAliases[strings.ToLower(raw)]; alias != "" {
			result = appendUniqueFold(result, alias)
			continue
		}
		for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == '/' || r == '|' }) {
			value := strings.TrimSpace(part)
			if value == "" {
				continue
			}
			key := strings.ToLower(value)
			if alias := genreAliases[key]; alias != "" {
				value = alias
			} else {
				value = titleMusicValue(value)
			}
			result = appendUniqueFold(result, value)
		}
	}
	if len(result) > 3 {
		result = result[:3]
	}
	return result
}

func titleMusicValue(value string) string {
	words := strings.Fields(strings.TrimSpace(value))
	for index, word := range words {
		lower := strings.ToLower(word)
		switch lower {
		case "r&b", "rnb", "dj", "uk", "edm":
			words[index] = strings.ToUpper(word)
		default:
			runes := []rune(lower)
			if len(runes) > 0 {
				runes[0] = unicode.ToUpper(runes[0])
			}
			words[index] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

func normalizedClassificationText(value string) string {
	value = strings.ToLower(value)
	var builder strings.Builder
	previousSpace := true
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '&' || r == '-' || r == '\'' {
			builder.WriteRune(r)
			previousSpace = false
			continue
		}
		if !previousSpace {
			builder.WriteByte(' ')
			previousSpace = true
		}
	}
	return strings.TrimSpace(builder.String())
}

func containsAnyPhrase(text string, phrases []string) bool {
	padded := " " + text + " "
	for _, phrase := range phrases {
		value := " " + normalizedClassificationText(phrase) + " "
		if strings.Contains(padded, value) {
			return true
		}
	}
	return false
}

func appendUniqueFold(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || containsFold(values, value) {
		return values
	}
	return append(values, value)
}

func genreForStyle(style string) string {
	key := strings.ToLower(strings.TrimSpace(style))
	for _, rule := range musicStyleRules {
		if strings.EqualFold(rule.Style, style) {
			return rule.Genre
		}
	}
	switch {
	case strings.Contains(key, "house"), strings.Contains(key, "techno"), strings.Contains(key, "trance"), strings.Contains(key, "ambient"), strings.Contains(key, "electro"), strings.Contains(key, "drum"), strings.Contains(key, "dubstep"):
		return "Electronic"
	case strings.Contains(key, "rock"):
		return "Rock"
	case strings.Contains(key, "metal"):
		return "Metal"
	case strings.Contains(key, "hip-hop"), strings.Contains(key, "rap"):
		return "Hip-Hop"
	}
	return ""
}

func classificationFromBPM(genres []string, bpm float64) (string, string) {
	primary := ""
	if len(genres) > 0 {
		primary = genres[0]
	}
	if primary != "" && primary != "Electronic" && primary != "Hip-Hop" && primary != "R&B" {
		return "", ""
	}
	switch {
	case bpm >= 158 && bpm <= 190:
		return "Electronic", "Drum & Bass"
	case bpm >= 138 && bpm < 158:
		return "Electronic", "Trance"
	case bpm >= 118 && bpm < 138:
		return "Electronic", "House"
	case bpm >= 82 && bpm < 108:
		if primary == "R&B" {
			return "R&B", "Contemporary R&B"
		}
		if primary == "Hip-Hop" {
			return "Hip-Hop", "Hip-Hop"
		}
	}
	return "", ""
}

func needsAudioClassification(current domain.MusicClassification, match MusicMetadataMatch, inferred inferredMusicClassification) bool {
	currentGenreProtected := len(current.Genres) > 0 && musicClassificationSourceRank(current.GenreSource, true) >= musicClassificationSourceRank("deezer", true)
	currentStyleProtected := len(current.Styles) > 0 && musicClassificationSourceRank(current.StyleSource, true) >= musicClassificationSourceRank("deezer", true)
	if currentGenreProtected && currentStyleProtected {
		return false
	}
	if len(match.Genres) > 0 && len(match.Styles) > 0 {
		return false
	}
	return inferred.Source == "fallback" || containsFold(inferred.Genres, "Other") || containsFold(inferred.Styles, "Other") || isBroadInferredStyle(inferred)
}

func removeOtherWhenSpecific(values []string) []string {
	values = normalizeMusicValues(values)
	if len(values) <= 1 || !containsFold(values, "Other") {
		return values
	}
	result := make([]string, 0, len(values)-1)
	for _, value := range values {
		if !strings.EqualFold(strings.TrimSpace(value), "Other") {
			result = append(result, value)
		}
	}
	return result
}

func mergeInferredClassification(base, audio inferredMusicClassification) inferredMusicClassification {
	if audio.Source == "" {
		return base
	}
	// A simple local tempo/spectrum heuristic is not reliable enough to
	// contradict a fingerprint or catalog genre. In particular, many house and
	// downtempo tracks fall into the same BPM range as hip-hop.
	if len(base.Genres) > 0 && !containsFold(base.Genres, "Other") && len(audio.Genres) > 0 {
		compatible := false
		for _, genre := range audio.Genres {
			if containsFold(base.Genres, genre) {
				compatible = true
				break
			}
		}
		if !compatible {
			return base
		}
	}
	if base.Source == "fallback" || containsFold(base.Genres, "Other") {
		base.Genres = append([]string(nil), audio.Genres...)
	}
	if base.Source == "fallback" || containsFold(base.Styles, "Other") || isBroadInferredStyle(base) {
		base.Styles = append([]string(nil), audio.Styles...)
	}
	if len(base.Moods) == 0 {
		base.Moods = append([]string(nil), audio.Moods...)
	}
	base.Source = audio.Source
	base.Confidence = audio.Confidence
	return base
}

func isBroadInferredStyle(value inferredMusicClassification) bool {
	if value.Source != "local-inference" || len(value.Genres) == 0 || len(value.Styles) != 1 {
		return false
	}
	defaultStyle := defaultStyleByGenre[value.Genres[0]]
	return defaultStyle != "" && strings.EqualFold(defaultStyle, value.Styles[0])
}

func sortClassification(values []string) []string {
	result := normalizeMusicValues(values)
	sort.SliceStable(result, func(i, j int) bool { return strings.ToLower(result[i]) < strings.ToLower(result[j]) })
	return result
}
