package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

const apiBaseURL = "https://api.themoviedb.org/3"
const imageBaseURL = "https://image.tmdb.org/t/p"

type Provider struct {
	config      func() (token string, language string)
	client      *http.Client
	cacheMu     sync.RWMutex
	searchCache map[string]rankedSearchResult
}

func New(token, language string) *Provider {
	return NewDynamic(func() (string, string) { return token, language })
}

func NewDynamic(config func() (token string, language string)) *Provider {
	return &Provider{
		config:      config,
		client:      &http.Client{Timeout: 15 * time.Second},
		searchCache: map[string]rankedSearchResult{},
	}
}

func (p *Provider) Name() string { return "tmdb" }

func (p *Provider) currentConfig() (string, string) {
	if p.config == nil {
		return "", domain.DefaultMetadataLanguage
	}
	token, language := p.config()
	return strings.TrimSpace(token), firstNonEmpty(strings.TrimSpace(language), domain.DefaultMetadataLanguage)
}

func (p *Provider) Available() bool {
	token, _ := p.currentConfig()
	return token != ""
}

func (p *Provider) Match(ctx context.Context, query domain.MetadataMatchQuery) (domain.MetadataResult, error) {
	token, configuredLanguage := p.currentConfig()
	if token == "" {
		return domain.MetadataResult{}, domain.ErrUnavailable
	}
	if query.Type == domain.MediaTypeMusic {
		return domain.MetadataResult{}, domain.ErrUnsupported
	}
	language := firstNonEmpty(query.Language, configuredLanguage, domain.DefaultMetadataLanguage)
	candidates := query.Candidates
	if len(candidates) == 0 {
		title := query.Title
		if query.Type == domain.MediaTypeEpisode {
			title = firstNonEmpty(query.SeriesTitle, query.Title)
		}
		candidates = []domain.MetadataTitleCandidate{{Title: title, Year: query.Year}}
	}

	if query.Type == domain.MediaTypeEpisode {
		showID, _ := strconv.Atoi(query.ParentExternalIDs.TMDb)
		var show searchResult
		if showID == 0 {
			matched, err := p.searchCandidates(ctx, token, language, "tv", candidates)
			if err != nil {
				return domain.MetadataResult{}, err
			}
			show = matched
			showID = matched.ID
		}
		if query.Season <= 0 || query.Episode <= 0 {
			return domain.MetadataResult{}, domain.ErrUnsupported
		}
		return p.episode(ctx, token, language, showID, query.Season, query.Episode, show)
	}

	mediaType := "movie"
	if query.Type == domain.MediaTypeSeries {
		mediaType = "tv"
	}
	if existingID, _ := strconv.Atoi(strings.TrimSpace(query.ExternalIDs.TMDb)); existingID > 0 {
		return p.details(ctx, token, language, mediaType, existingID)
	}
	match, err := p.searchCandidates(ctx, token, language, mediaType, candidates)
	if err != nil {
		return domain.MetadataResult{}, err
	}
	return p.details(ctx, token, language, mediaType, match.ID)
}

type searchResult struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	Name          string  `json:"name"`
	OriginalName  string  `json:"original_name"`
	ReleaseDate   string  `json:"release_date"`
	FirstAirDate  string  `json:"first_air_date"`
	Overview      string  `json:"overview"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	Popularity    float64 `json:"popularity"`
}

type searchResponse struct {
	Results []searchResult `json:"results"`
}

type rankedSearchResult struct {
	Result searchResult
	Score  float64
}

func (p *Provider) searchCandidates(ctx context.Context, token, language, mediaType string, candidates []domain.MetadataTitleCandidate) (searchResult, error) {
	seen := make(map[string]struct{}, len(candidates))
	best := rankedSearchResult{Score: math.Inf(-1)}
	found := false
	var lastErr error
	for index, candidate := range candidates {
		title := strings.TrimSpace(candidate.Title)
		if title == "" {
			continue
		}
		key := normalizeTitle(title) + "|" + strconv.Itoa(candidate.Year)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		matched, score, err := p.search(ctx, token, language, mediaType, title, candidate.Year)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				lastErr = err
				continue
			}
			return searchResult{}, err
		}
		// Candidate order is already quality-ranked by the filename parser. A
		// tiny bonus breaks ties without allowing a noisy first candidate to
		// beat an exact later title.
		score += float64(maxInt(0, len(candidates)-index)) * 0.15
		if !found || score > best.Score {
			best = rankedSearchResult{Result: matched, Score: score}
			found = true
		}
		if score >= 129 { // exact title and exact year
			break
		}
	}
	if found {
		return best.Result, nil
	}
	if lastErr != nil {
		return searchResult{}, lastErr
	}
	return searchResult{}, domain.ErrNotFound
}

func (p *Provider) search(ctx context.Context, token, language, mediaType, title string, year int) (searchResult, float64, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return searchResult{}, 0, domain.ErrInvalidInput
	}
	cacheKey := strings.Join([]string{language, mediaType, normalizeTitle(title), strconv.Itoa(year)}, "|")
	p.cacheMu.RLock()
	cached, found := p.searchCache[cacheKey]
	p.cacheMu.RUnlock()
	if found {
		return cached.Result, cached.Score, nil
	}

	results, err := p.searchRequest(ctx, token, language, mediaType, title, year)
	if err != nil {
		return searchResult{}, 0, err
	}
	if len(results) == 0 && year > 0 {
		results, err = p.searchRequest(ctx, token, language, mediaType, title, 0)
		if err != nil {
			return searchResult{}, 0, err
		}
	}
	if len(results) == 0 && !strings.EqualFold(language, "en-US") {
		results, err = p.searchRequest(ctx, token, "en-US", mediaType, title, year)
		if err != nil {
			return searchResult{}, 0, err
		}
		if len(results) == 0 && year > 0 {
			results, err = p.searchRequest(ctx, token, "en-US", mediaType, title, 0)
			if err != nil {
				return searchResult{}, 0, err
			}
		}
	}
	if len(results) == 0 {
		return searchResult{}, 0, domain.ErrNotFound
	}

	best, score := chooseBestResult(results, title, year)
	minimum := 20.0
	if year > 0 {
		minimum = 35.0
	}
	if score < minimum {
		return searchResult{}, 0, domain.ErrNotFound
	}
	p.cacheMu.Lock()
	p.searchCache[cacheKey] = rankedSearchResult{Result: best, Score: score}
	p.cacheMu.Unlock()
	return best, score, nil
}

func (p *Provider) searchRequest(ctx context.Context, token, language, mediaType, title string, year int) ([]searchResult, error) {
	values := url.Values{}
	values.Set("query", title)
	values.Set("language", language)
	values.Set("include_adult", "false")
	if year > 0 {
		if mediaType == "movie" {
			values.Set("year", strconv.Itoa(year))
		} else {
			values.Set("first_air_date_year", strconv.Itoa(year))
		}
	}
	var response searchResponse
	if err := p.get(ctx, token, "/search/"+mediaType, values, &response); err != nil {
		return nil, err
	}
	return response.Results, nil
}

func chooseBestResult(results []searchResult, title string, year int) (searchResult, float64) {
	target := normalizeTitle(title)
	best := results[0]
	bestScore := math.Inf(-1)
	for _, result := range results {
		names := []string{result.Title, result.Name, result.OriginalTitle, result.OriginalName}
		score := 0.0
		for _, name := range names {
			normalized := normalizeTitle(name)
			if normalized == "" {
				continue
			}
			switch {
			case normalized == target:
				score = math.Max(score, 100)
			case containsTokenSequence(normalized, target) || containsTokenSequence(target, normalized):
				// Partial-title matches are allowed only on complete word boundaries.
				// Raw substring matching made short titles such as "Silo" match
				// unrelated names such as "Love of Silom".
				score = math.Max(score, 65)
			default:
				score = math.Max(score, similarity(normalized, target)*50)
			}
		}
		resultYear := yearFromDate(firstNonEmpty(result.ReleaseDate, result.FirstAirDate))
		if year > 0 && resultYear > 0 {
			difference := abs(year - resultYear)
			if difference == 0 {
				score += 30
			} else if difference == 1 {
				score += 8
			} else {
				score -= float64(difference * 4)
			}
		}
		if result.PosterPath != "" || result.BackdropPath != "" {
			score += 2
		}
		score += math.Log1p(math.Max(result.Popularity, 0)) * 0.6
		if score > bestScore {
			bestScore = score
			best = result
		}
	}
	return best, bestScore
}

func normalizeTitle(value string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), " ")
}

func containsTokenSequence(haystack, needle string) bool {
	haystackTokens := strings.Fields(haystack)
	needleTokens := strings.Fields(needle)
	if len(needleTokens) == 0 || len(needleTokens) > len(haystackTokens) {
		return false
	}
	for start := 0; start <= len(haystackTokens)-len(needleTokens); start++ {
		matched := true
		for offset := range needleTokens {
			if haystackTokens[start+offset] != needleTokens[offset] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func similarity(left, right string) float64 {
	if left == right {
		return 1
	}
	l := strings.Fields(left)
	r := strings.Fields(right)
	if len(l) == 0 || len(r) == 0 {
		return 0
	}
	set := map[string]struct{}{}
	for _, value := range l {
		set[value] = struct{}{}
	}
	matches := 0
	for _, value := range r {
		if _, ok := set[value]; ok {
			matches++
		}
	}
	return float64(2*matches) / float64(len(l)+len(r))
}

func yearFromDate(value string) int {
	if len(value) < 4 {
		return 0
	}
	year, _ := strconv.Atoi(value[:4])
	return year
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

type genre struct {
	Name string `json:"name"`
}

type credit struct {
	Name        string `json:"name"`
	Character   string `json:"character"`
	Job         string `json:"job"`
	ProfilePath string `json:"profile_path"`
}

type credits struct {
	Cast []credit `json:"cast"`
	Crew []credit `json:"crew"`
}

type externalIDs struct {
	IMDbID string `json:"imdb_id"`
	TVDbID int    `json:"tvdb_id"`
}

type seasonSummary struct {
	AirDate      string `json:"air_date"`
	EpisodeCount int    `json:"episode_count"`
	SeasonNumber int    `json:"season_number"`
}

type latestEpisodeResponse struct {
	Name          string `json:"name"`
	AirDate       string `json:"air_date"`
	EpisodeNumber int    `json:"episode_number"`
	SeasonNumber  int    `json:"season_number"`
}

type imageAsset struct {
	AspectRatio float64 `json:"aspect_ratio"`
	FilePath    string  `json:"file_path"`
	Height      int     `json:"height"`
	Language    string  `json:"iso_639_1"`
	VoteAverage float64 `json:"vote_average"`
	VoteCount   int     `json:"vote_count"`
	Width       int     `json:"width"`
}

type imageCollection struct {
	Logos []imageAsset `json:"logos"`
}

type videoAsset struct {
	Language    string `json:"iso_639_1"`
	Country     string `json:"iso_3166_1"`
	Name        string `json:"name"`
	Key         string `json:"key"`
	Site        string `json:"site"`
	Size        int    `json:"size"`
	Type        string `json:"type"`
	Official    bool   `json:"official"`
	PublishedAt string `json:"published_at"`
}

type videoCollection struct {
	Results []videoAsset `json:"results"`
}

type detailsResponse struct {
	ID               int                    `json:"id"`
	IMDbID           string                 `json:"imdb_id"`
	Title            string                 `json:"title"`
	Name             string                 `json:"name"`
	Overview         string                 `json:"overview"`
	Tagline          string                 `json:"tagline"`
	ReleaseDate      string                 `json:"release_date"`
	FirstAirDate     string                 `json:"first_air_date"`
	PosterPath       string                 `json:"poster_path"`
	BackdropPath     string                 `json:"backdrop_path"`
	Genres           []genre                `json:"genres"`
	Credits          credits                `json:"credits"`
	ExternalIDs      externalIDs            `json:"external_ids"`
	NumberOfSeasons  int                    `json:"number_of_seasons"`
	NumberOfEpisodes int                    `json:"number_of_episodes"`
	Seasons          []seasonSummary        `json:"seasons"`
	LastEpisodeToAir *latestEpisodeResponse `json:"last_episode_to_air"`
	Images           imageCollection        `json:"images"`
	Videos           videoCollection        `json:"videos"`
}

func (p *Provider) details(ctx context.Context, token, language, mediaType string, id int) (domain.MetadataResult, error) {
	response, err := p.fetchDetails(ctx, token, language, mediaType, id)
	if err != nil {
		return domain.MetadataResult{}, err
	}
	if !strings.EqualFold(language, "en-US") && detailsNeedFallback(response) {
		if fallback, fallbackErr := p.fetchDetails(ctx, token, "en-US", mediaType, id); fallbackErr == nil {
			response = mergeDetails(response, fallback)
		}
	}
	logo := chooseLogo(response.Images.Logos, language)
	trailers := chooseTrailers(response.Videos.Results, language, 8)
	trailer := videoAsset{}
	if len(trailers) > 0 {
		trailer = trailers[0]
	}
	result := domain.MetadataResult{
		Title: firstNonEmpty(response.Title, response.Name),
		ExternalIDs: domain.ExternalIDs{
			IMDb: firstNonEmpty(response.IMDbID, response.ExternalIDs.IMDbID),
			TMDb: strconv.Itoa(response.ID),
		},
		Metadata: domain.EditorialMetadata{
			Summary:           response.Overview,
			Tagline:           response.Tagline,
			Genres:            genreNames(response.Genres),
			Cast:              castCredits(response.Credits.Cast, 16),
			Directors:         directorCredits(response.Credits.Crew),
			PosterURL:         imageURL("w500", response.PosterPath),
			BackdropURL:       imageURL("w1280", response.BackdropPath),
			LogoURL:           imageURL("w500", logo.FilePath),
			TrailerYouTubeKey: trailer.Key,
			TrailerName:       trailer.Name,
			TrailerCandidates: trailerCandidates(trailers),
			HeroAssetsVersion: 2,
			MetadataSource:    "tmdb",
		},
	}
	if mediaType == "tv" {
		releasedSeasons, releasedEpisodes := releasedCounts(response)
		result.Metadata.OfficialSeasonCount = releasedSeasons
		result.Metadata.OfficialEpisodeCount = releasedEpisodes
		if response.LastEpisodeToAir != nil {
			result.Metadata.LatestEpisode = &domain.EpisodeReleaseInfo{
				Season:      response.LastEpisodeToAir.SeasonNumber,
				Episode:     response.LastEpisodeToAir.EpisodeNumber,
				Title:       response.LastEpisodeToAir.Name,
				ReleaseDate: parseDate(response.LastEpisodeToAir.AirDate),
			}
		}
	}
	if response.ExternalIDs.TVDbID > 0 {
		result.ExternalIDs.TVDb = strconv.Itoa(response.ExternalIDs.TVDbID)
	}
	result.ReleaseDate = parseDate(firstNonEmpty(response.ReleaseDate, response.FirstAirDate))
	if result.ReleaseDate != nil {
		result.Year = result.ReleaseDate.Year()
	}
	return result, nil
}

func (p *Provider) fetchDetails(ctx context.Context, token, language, mediaType string, id int) (detailsResponse, error) {
	values := url.Values{}
	values.Set("language", language)
	values.Set("append_to_response", "credits,external_ids,images,videos")
	values.Set("include_image_language", imageLanguageFilter(language))
	var response detailsResponse
	if err := p.get(ctx, token, fmt.Sprintf("/%s/%d", mediaType, id), values, &response); err != nil {
		return detailsResponse{}, err
	}
	return response, nil
}

func detailsNeedFallback(value detailsResponse) bool {
	return firstNonEmpty(value.Title, value.Name) == "" ||
		strings.TrimSpace(value.Overview) == "" ||
		strings.TrimSpace(value.PosterPath) == "" ||
		strings.TrimSpace(value.BackdropPath) == "" ||
		len(value.Images.Logos) == 0 ||
		!containsYouTubeTrailer(value.Videos.Results)
}

func mergeDetails(primary, fallback detailsResponse) detailsResponse {
	primary.ID = firstNonZero(primary.ID, fallback.ID)
	primary.IMDbID = firstNonEmpty(primary.IMDbID, fallback.IMDbID)
	primary.Title = firstNonEmpty(primary.Title, fallback.Title)
	primary.Name = firstNonEmpty(primary.Name, fallback.Name)
	primary.Overview = firstNonEmpty(primary.Overview, fallback.Overview)
	primary.Tagline = firstNonEmpty(primary.Tagline, fallback.Tagline)
	primary.ReleaseDate = firstNonEmpty(primary.ReleaseDate, fallback.ReleaseDate)
	primary.FirstAirDate = firstNonEmpty(primary.FirstAirDate, fallback.FirstAirDate)
	primary.PosterPath = firstNonEmpty(primary.PosterPath, fallback.PosterPath)
	primary.BackdropPath = firstNonEmpty(primary.BackdropPath, fallback.BackdropPath)
	if len(primary.Genres) == 0 {
		primary.Genres = fallback.Genres
	}
	if len(primary.Credits.Cast) == 0 {
		primary.Credits.Cast = fallback.Credits.Cast
	}
	if len(primary.Credits.Crew) == 0 {
		primary.Credits.Crew = fallback.Credits.Crew
	}
	primary.ExternalIDs.IMDbID = firstNonEmpty(primary.ExternalIDs.IMDbID, fallback.ExternalIDs.IMDbID)
	if primary.ExternalIDs.TVDbID == 0 {
		primary.ExternalIDs.TVDbID = fallback.ExternalIDs.TVDbID
	}
	if primary.NumberOfSeasons == 0 {
		primary.NumberOfSeasons = fallback.NumberOfSeasons
	}
	if primary.NumberOfEpisodes == 0 {
		primary.NumberOfEpisodes = fallback.NumberOfEpisodes
	}
	if len(primary.Seasons) == 0 {
		primary.Seasons = fallback.Seasons
	}
	if primary.LastEpisodeToAir == nil {
		primary.LastEpisodeToAir = fallback.LastEpisodeToAir
	}
	if len(primary.Images.Logos) == 0 {
		primary.Images.Logos = fallback.Images.Logos
	} else if len(fallback.Images.Logos) > 0 {
		primary.Images.Logos = append(primary.Images.Logos, fallback.Images.Logos...)
	}
	if len(primary.Videos.Results) == 0 {
		primary.Videos.Results = fallback.Videos.Results
	} else if len(fallback.Videos.Results) > 0 {
		primary.Videos.Results = append(primary.Videos.Results, fallback.Videos.Results...)
	}
	return primary
}

func releasedCounts(value detailsResponse) (int, int) {
	latest := value.LastEpisodeToAir
	if latest == nil {
		return value.NumberOfSeasons, value.NumberOfEpisodes
	}
	seasonCount := 0
	episodeCount := 0
	for _, season := range value.Seasons {
		if season.SeasonNumber <= 0 || season.SeasonNumber > latest.SeasonNumber {
			continue
		}
		seasonCount++
		if season.SeasonNumber < latest.SeasonNumber {
			episodeCount += season.EpisodeCount
		} else {
			episodeCount += latest.EpisodeNumber
		}
	}
	if seasonCount == 0 && latest.SeasonNumber > 0 {
		seasonCount = latest.SeasonNumber
	}
	if episodeCount == 0 && latest.EpisodeNumber > 0 {
		episodeCount = latest.EpisodeNumber
	}
	return seasonCount, episodeCount
}

type episodeResponse struct {
	ID          int         `json:"id"`
	Name        string      `json:"name"`
	Overview    string      `json:"overview"`
	AirDate     string      `json:"air_date"`
	StillPath   string      `json:"still_path"`
	Credits     credits     `json:"credits"`
	ExternalIDs externalIDs `json:"external_ids"`
}

func (p *Provider) episode(ctx context.Context, token, language string, showID, season, episode int, show searchResult) (domain.MetadataResult, error) {
	response, err := p.fetchEpisode(ctx, token, language, showID, season, episode)
	if err != nil {
		return domain.MetadataResult{}, err
	}
	if !strings.EqualFold(language, "en-US") && episodeNeedsFallback(response) {
		if fallback, fallbackErr := p.fetchEpisode(ctx, token, "en-US", showID, season, episode); fallbackErr == nil {
			response = mergeEpisode(response, fallback)
		}
	}
	result := domain.MetadataResult{
		Title: response.Name,
		ExternalIDs: domain.ExternalIDs{
			IMDb: response.ExternalIDs.IMDbID,
			TMDb: strconv.Itoa(response.ID),
		},
		Metadata: domain.EditorialMetadata{
			Summary:        response.Overview,
			Cast:           castCredits(response.Credits.Cast, 16),
			Directors:      directorCredits(response.Credits.Crew),
			PosterURL:      imageURL("w780", response.StillPath),
			BackdropURL:    imageURL("w1280", firstNonEmpty(response.StillPath, show.BackdropPath)),
			MetadataSource: "tmdb",
		},
	}
	result.ReleaseDate = parseDate(response.AirDate)
	if result.ReleaseDate != nil {
		result.Year = result.ReleaseDate.Year()
	}
	return result, nil
}

func (p *Provider) fetchEpisode(ctx context.Context, token, language string, showID, season, episode int) (episodeResponse, error) {
	values := url.Values{}
	values.Set("language", language)
	values.Set("append_to_response", "credits,external_ids")
	var response episodeResponse
	if err := p.get(ctx, token, fmt.Sprintf("/tv/%d/season/%d/episode/%d", showID, season, episode), values, &response); err != nil {
		return episodeResponse{}, err
	}
	return response, nil
}

func episodeNeedsFallback(value episodeResponse) bool {
	return strings.TrimSpace(value.Name) == "" || strings.TrimSpace(value.Overview) == "" || strings.TrimSpace(value.StillPath) == ""
}

func mergeEpisode(primary, fallback episodeResponse) episodeResponse {
	primary.ID = firstNonZero(primary.ID, fallback.ID)
	primary.Name = firstNonEmpty(primary.Name, fallback.Name)
	primary.Overview = firstNonEmpty(primary.Overview, fallback.Overview)
	primary.AirDate = firstNonEmpty(primary.AirDate, fallback.AirDate)
	primary.StillPath = firstNonEmpty(primary.StillPath, fallback.StillPath)
	if len(primary.Credits.Cast) == 0 {
		primary.Credits.Cast = fallback.Credits.Cast
	}
	if len(primary.Credits.Crew) == 0 {
		primary.Credits.Crew = fallback.Credits.Crew
	}
	primary.ExternalIDs.IMDbID = firstNonEmpty(primary.ExternalIDs.IMDbID, fallback.ExternalIDs.IMDbID)
	if primary.ExternalIDs.TVDbID == 0 {
		primary.ExternalIDs.TVDbID = fallback.ExternalIDs.TVDbID
	}
	return primary
}

func (p *Provider) get(ctx context.Context, token, path string, query url.Values, target any) error {
	endpoint := apiBaseURL + path
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("tmdb request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return domain.ErrNotFound
	}
	if response.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("%w: tmdb rate limit", domain.ErrUnavailable)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("tmdb returned HTTP %d", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode tmdb response: %w", err)
	}
	return nil
}

func genreNames(genres []genre) []string {
	result := make([]string, 0, len(genres))
	for _, genre := range genres {
		if genre.Name != "" {
			result = append(result, genre.Name)
		}
	}
	return result
}

func castCredits(values []credit, limit int) []domain.PersonCredit {
	if len(values) > limit {
		values = values[:limit]
	}
	result := make([]domain.PersonCredit, 0, len(values))
	for _, value := range values {
		result = append(result, domain.PersonCredit{
			Name:      value.Name,
			Character: value.Character,
			Role:      "Näitleja",
			ImageURL:  imageURL("w185", value.ProfilePath),
		})
	}
	return result
}

func directorCredits(values []credit) []domain.PersonCredit {
	result := make([]domain.PersonCredit, 0)
	for _, value := range values {
		if strings.EqualFold(value.Job, "Director") {
			result = append(result, domain.PersonCredit{
				Name:     value.Name,
				Role:     "Režissöör",
				ImageURL: imageURL("w185", value.ProfilePath),
			})
		}
	}
	return result
}

func imageLanguageFilter(language string) string {
	preferred := baseLanguage(language)
	values := make([]string, 0, 3)
	seen := map[string]struct{}{}
	for _, value := range []string{preferred, "en", "null"} {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return strings.Join(values, ",")
}

func baseLanguage(language string) string {
	value := strings.ToLower(strings.TrimSpace(language))
	if separator := strings.IndexAny(value, "-_"); separator >= 0 {
		value = value[:separator]
	}
	return value
}

func chooseLogo(values []imageAsset, language string) imageAsset {
	preferred := baseLanguage(language)
	best := imageAsset{}
	bestScore := math.Inf(-1)
	for _, value := range values {
		if strings.TrimSpace(value.FilePath) == "" {
			continue
		}
		score := 0.0
		assetLanguage := baseLanguage(value.Language)
		switch {
		case preferred != "" && assetLanguage == preferred:
			score += 80
		case assetLanguage == "en":
			score += 55
		case assetLanguage == "":
			score += 35
		default:
			score += 5
		}
		if value.AspectRatio >= 1.4 {
			score += 12
		}
		score += math.Min(float64(value.Width)/250, 12)
		score += math.Min(value.VoteAverage, 10)
		score += math.Min(float64(value.VoteCount)/5, 8)
		if score > bestScore {
			best = value
			bestScore = score
		}
	}
	return best
}

func chooseTrailers(values []videoAsset, language string, limit int) []videoAsset {
	preferred := baseLanguage(language)
	type rankedVideo struct {
		asset videoAsset
		score float64
	}
	ranked := make([]rankedVideo, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		key := strings.TrimSpace(value.Key)
		if !strings.EqualFold(strings.TrimSpace(value.Site), "YouTube") || key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		value.Key = key
		score := 0.0
		switch strings.ToLower(strings.TrimSpace(value.Type)) {
		case "trailer":
			score += 120
		case "teaser":
			score += 85
		case "clip":
			score += 35
		case "featurette":
			score += 25
		default:
			score += 5
		}
		if value.Official {
			score += 90
		}
		assetLanguage := baseLanguage(value.Language)
		switch {
		case preferred != "" && assetLanguage == preferred:
			score += 35
		case assetLanguage == "en":
			score += 25
		case assetLanguage == "":
			score += 8
		}
		score += math.Min(float64(value.Size)/90, 18)
		name := strings.ToLower(value.Name)
		if strings.Contains(name, "official") {
			score += 12
		}
		if strings.Contains(name, "trailer") {
			score += 7
		}
		// Red-band and uncensored cuts are commonly age-restricted. Keep them only
		// as a last-resort candidate so the player can prefer a normal trailer.
		for _, warning := range []string{"red band", "red-band", "uncensored", "restricted", "18+", "nsfw"} {
			if strings.Contains(name, warning) {
				score -= 500
				break
			}
		}
		ranked = append(ranked, rankedVideo{asset: value, score: score})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].asset.PublishedAt > ranked[j].asset.PublishedAt
		}
		return ranked[i].score > ranked[j].score
	})
	if limit <= 0 || limit > len(ranked) {
		limit = len(ranked)
	}
	result := make([]videoAsset, 0, limit)
	for _, value := range ranked[:limit] {
		result = append(result, value.asset)
	}
	return result
}

func chooseTrailer(values []videoAsset, language string) videoAsset {
	values = chooseTrailers(values, language, 1)
	if len(values) == 0 {
		return videoAsset{}
	}
	return values[0]
}

func trailerCandidates(values []videoAsset) []domain.TrailerCandidate {
	result := make([]domain.TrailerCandidate, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value.Key) == "" {
			continue
		}
		result = append(result, domain.TrailerCandidate{YouTubeKey: value.Key, Name: value.Name})
	}
	return result
}

func containsYouTubeTrailer(values []videoAsset) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value.Site), "YouTube") &&
			strings.EqualFold(strings.TrimSpace(value.Type), "Trailer") &&
			strings.TrimSpace(value.Key) != "" {
			return true
		}
	}
	return false
}

func imageURL(size, value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return imageBaseURL + "/" + size + value
}

func parseDate(value string) *time.Time {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil
	}
	return &parsed
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

var _ ports.MetadataProvider = (*Provider)(nil)
