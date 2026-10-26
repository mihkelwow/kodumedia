package opensubtitles

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

const apiBase = "https://api.opensubtitles.com/api/v1"

type Provider struct {
	settings  func() domain.SubtitleSettings
	client    *http.Client
	userAgent string

	mu          sync.Mutex
	token       string
	tokenUser   string
	tokenExpiry time.Time
}

func NewDynamic(settings func() domain.SubtitleSettings, version string) *Provider {
	return &Provider{
		settings:  settings,
		client:    &http.Client{Timeout: 35 * time.Second},
		userAgent: "KoduMedia v" + strings.TrimSpace(version),
	}
}

func (p *Provider) current() domain.SubtitleSettings {
	if p.settings == nil {
		return domain.SubtitleSettings{}
	}
	return p.settings()
}

func (p *Provider) Available() bool { return strings.TrimSpace(p.current().OpenSubtitlesAPIKey) != "" }

type searchResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Language         string  `json:"language"`
			DownloadCount    int     `json:"download_count"`
			NewDownloadCount int     `json:"new_download_count"`
			HearingImpaired  bool    `json:"hearing_impaired"`
			Ratings          float64 `json:"ratings"`
			FromTrusted      bool    `json:"from_trusted"`
			Release          string  `json:"release"`
			Files            []struct {
				FileID   int64  `json:"file_id"`
				FileName string `json:"file_name"`
			} `json:"files"`
		} `json:"attributes"`
	} `json:"data"`
}

func (p *Provider) Search(ctx context.Context, media domain.MediaItem, languages []string) ([]domain.SubtitleSearchResult, error) {
	settings := p.current()
	apiKey := strings.TrimSpace(settings.OpenSubtitlesAPIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("%w: OpenSubtitles API-võti pole seadistatud", domain.ErrUnavailable)
	}
	if len(languages) == 0 {
		languages = settings.PreferredLanguages
	}
	normalizedLanguages := normalizeLanguages(languages)
	sort.Strings(normalizedLanguages)

	queries := buildSearchQueries(media, normalizedLanguages)
	merged := map[int64]domain.SubtitleSearchResult{}
	var lastErr error
	for index, query := range queries {
		if index > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(220 * time.Millisecond):
			}
		}
		results, err := p.searchOnce(ctx, apiKey, query.values, query.matchType)
		if err != nil {
			lastErr = err
			continue
		}
		for _, result := range results {
			existing, ok := merged[result.FileID]
			if !ok || resultPriority(result) > resultPriority(existing) {
				merged[result.FileID] = result
			}
		}
	}
	if len(merged) == 0 && lastErr != nil {
		return nil, lastErr
	}
	results := make([]domain.SubtitleSearchResult, 0, len(merged))
	for _, result := range merged {
		results = append(results, result)
	}
	sort.SliceStable(results, func(i, j int) bool {
		if resultPriority(results[i]) != resultPriority(results[j]) {
			return resultPriority(results[i]) > resultPriority(results[j])
		}
		if results[i].Trusted != results[j].Trusted {
			return results[i].Trusted
		}
		if results[i].Rating != results[j].Rating {
			return results[i].Rating > results[j].Rating
		}
		return results[i].Downloads > results[j].Downloads
	})
	if len(results) > 80 {
		results = results[:80]
	}
	return results, nil
}

type subtitleSearchQuery struct {
	values    url.Values
	matchType string
}

func buildSearchQueries(media domain.MediaItem, languages []string) []subtitleSearchQuery {
	base := func() url.Values {
		values := url.Values{}
		if len(languages) > 0 {
			values.Set("languages", strings.Join(languages, ","))
		}
		return values
	}
	queries := make([]subtitleSearchQuery, 0, 4)

	if hash, _, err := movieHash(media.File.Path); err == nil {
		values := base()
		values.Set("moviehash", hash)
		if name := cleanSearchText(strings.TrimSuffix(filepath.Base(media.File.Path), filepath.Ext(media.File.Path))); name != "" {
			values.Set("query", name)
		}
		queries = append(queries, subtitleSearchQuery{values: values, matchType: "hash"})
	}

	if imdbID := numericExternalID(media.ExternalIDs.IMDb); imdbID != "" {
		values := base()
		values.Set("imdb_id", imdbID)
		queries = append(queries, subtitleSearchQuery{values: values, matchType: "imdb"})
	} else if tmdbID := numericExternalID(media.ExternalIDs.TMDb); tmdbID != "" {
		values := base()
		values.Set("tmdb_id", tmdbID)
		if media.Type == domain.MediaTypeEpisode {
			values.Set("type", "episode")
		} else {
			values.Set("type", "movie")
		}
		queries = append(queries, subtitleSearchQuery{values: values, matchType: "tmdb"})
	}

	title := media.Title
	if media.Type == domain.MediaTypeEpisode && strings.TrimSpace(media.SeriesTitle) != "" {
		title = media.SeriesTitle
	}
	title = cleanSearchText(title)
	if title != "" {
		values := base()
		values.Set("query", title)
		if media.Type == domain.MediaTypeEpisode {
			values.Set("type", "episode")
			if media.Season > 0 {
				values.Set("season_number", strconv.Itoa(media.Season))
			}
			if media.Episode > 0 {
				values.Set("episode_number", strconv.Itoa(media.Episode))
			}
		} else {
			values.Set("type", "movie")
			if media.Year > 0 {
				values.Set("year", strconv.Itoa(media.Year))
			}
		}
		queries = append(queries, subtitleSearchQuery{values: values, matchType: "title"})

		broad := base()
		broad.Set("query", title)
		if media.Type == domain.MediaTypeEpisode {
			broad.Set("type", "episode")
		} else {
			broad.Set("type", "movie")
		}
		if broad.Encode() != values.Encode() {
			queries = append(queries, subtitleSearchQuery{values: broad, matchType: "broad-title"})
		}
	}
	return deduplicateQueries(queries)
}

func deduplicateQueries(values []subtitleSearchQuery) []subtitleSearchQuery {
	seen := map[string]struct{}{}
	result := make([]subtitleSearchQuery, 0, len(values))
	for _, value := range values {
		encoded := value.values.Encode()
		if encoded == "" {
			continue
		}
		if _, ok := seen[encoded]; ok {
			continue
		}
		seen[encoded] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (p *Provider) searchOnce(ctx context.Context, apiKey string, values url.Values, matchType string) ([]domain.SubtitleSearchResult, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBase+"/subtitles?"+values.Encode(), nil)
	if err != nil {
		return nil, err
	}
	p.headers(request, apiKey, "")
	response, err := p.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("OpenSubtitles otsing ebaõnnestus: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, apiError(response, "OpenSubtitles otsing")
	}
	var decoded searchResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("OpenSubtitles vastuse lugemine: %w", err)
	}
	results := make([]domain.SubtitleSearchResult, 0, len(decoded.Data))
	for _, row := range decoded.Data {
		for _, file := range row.Attributes.Files {
			if file.FileID <= 0 {
				continue
			}
			downloads := row.Attributes.DownloadCount
			if row.Attributes.NewDownloadCount > downloads {
				downloads = row.Attributes.NewDownloadCount
			}
			results = append(results, domain.SubtitleSearchResult{
				ID: row.ID + "-" + strconv.FormatInt(file.FileID, 10), FileID: file.FileID,
				FileName: file.FileName, Language: normalizeLanguage(row.Attributes.Language), Release: row.Attributes.Release,
				HearingImpaired: row.Attributes.HearingImpaired, Trusted: row.Attributes.FromTrusted,
				Rating: row.Attributes.Ratings, Downloads: downloads, MatchType: matchType,
			})
		}
	}
	return results, nil
}

func resultPriority(result domain.SubtitleSearchResult) int {
	switch result.MatchType {
	case "hash":
		return 5
	case "imdb", "tmdb":
		return 4
	case "title":
		return 3
	case "broad-title":
		return 2
	default:
		return 1
	}
}

func numericExternalID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "tt")
	value = strings.TrimLeft(value, "0")
	if value == "" {
		return ""
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return value
}

func cleanSearchText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer(".", " ", "_", " ", "-", " ").Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

type loginResponse struct {
	Token string `json:"token"`
}
type downloadResponse struct {
	Link     string `json:"link"`
	FileName string `json:"file_name"`
}

func (p *Provider) Download(ctx context.Context, fileID int64) (io.ReadCloser, string, error) {
	if fileID <= 0 {
		return nil, "", domain.ErrInvalidInput
	}
	settings := p.current()
	apiKey := strings.TrimSpace(settings.OpenSubtitlesAPIKey)
	if apiKey == "" {
		return nil, "", fmt.Errorf("%w: OpenSubtitles API-võti pole seadistatud", domain.ErrUnavailable)
	}
	token, err := p.login(ctx, settings)
	if err != nil {
		return nil, "", err
	}
	payload, _ := json.Marshal(map[string]any{"file_id": fileID})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+"/download", bytes.NewReader(payload))
	if err != nil {
		return nil, "", err
	}
	p.headers(request, apiKey, token)
	request.Header.Set("Content-Type", "application/json")
	response, err := p.client.Do(request)
	if err != nil {
		return nil, "", fmt.Errorf("OpenSubtitles allalaadimispäring ebaõnnestus: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, "", apiError(response, "OpenSubtitles allalaadimine")
	}
	var decoded downloadResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&decoded); err != nil {
		return nil, "", err
	}
	parsed, err := url.Parse(strings.TrimSpace(decoded.Link))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, "", fmt.Errorf("%w: OpenSubtitles tagastas vigase allalaadimislingi", domain.ErrUnavailable)
	}
	fileRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, "", err
	}
	fileRequest.Header.Set("User-Agent", p.userAgent)
	fileResponse, err := p.client.Do(fileRequest)
	if err != nil {
		return nil, "", fmt.Errorf("subtiitrifaili allalaadimine ebaõnnestus: %w", err)
	}
	if fileResponse.StatusCode != http.StatusOK {
		defer fileResponse.Body.Close()
		return nil, "", apiError(fileResponse, "subtiitrifaili allalaadimine")
	}
	name := strings.TrimSpace(decoded.FileName)
	if name == "" {
		name = fmt.Sprintf("opensubtitles-%d.srt", fileID)
	}
	return fileResponse.Body, name, nil
}

func (p *Provider) login(ctx context.Context, settings domain.SubtitleSettings) (string, error) {
	username := strings.TrimSpace(settings.OpenSubtitlesUsername)
	password := strings.TrimSpace(settings.OpenSubtitlesPassword)
	if username == "" || password == "" {
		return "", nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.token != "" && p.tokenUser == username && time.Now().Before(p.tokenExpiry) {
		return p.token, nil
	}
	payload, _ := json.Marshal(map[string]string{"username": username, "password": password})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+"/login", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	p.headers(request, settings.OpenSubtitlesAPIKey, "")
	request.Header.Set("Content-Type", "application/json")
	response, err := p.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("OpenSubtitles sisselogimine ebaõnnestus: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", apiError(response, "OpenSubtitles sisselogimine")
	}
	var decoded loginResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&decoded); err != nil {
		return "", err
	}
	if strings.TrimSpace(decoded.Token) == "" {
		return "", fmt.Errorf("%w: OpenSubtitles ei tagastanud sessioonitokenit", domain.ErrUnavailable)
	}
	p.token = decoded.Token
	p.tokenUser = username
	p.tokenExpiry = time.Now().Add(23 * time.Hour)
	return p.token, nil
}

func (p *Provider) headers(request *http.Request, apiKey, token string) {
	request.Header.Set("Api-Key", strings.TrimSpace(apiKey))
	request.Header.Set("User-Agent", p.userAgent)
	request.Header.Set("Accept", "application/json")
	if strings.TrimSpace(token) != "" {
		request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}
}

func apiError(response *http.Response, operation string) error {
	data, _ := io.ReadAll(io.LimitReader(response.Body, 32<<10))
	message := strings.TrimSpace(string(data))
	if message == "" {
		message = response.Status
	}
	if len(message) > 600 {
		message = message[:600]
	}
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return fmt.Errorf("%w: %s: kontrolli API-võtit ja kasutajakontot (%s)", domain.ErrUnavailable, operation, response.Status)
	}
	if response.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("%w: %s: päringulimiit on täis", domain.ErrUnavailable, operation)
	}
	return fmt.Errorf("%w: %s: %s", domain.ErrUnavailable, operation, message)
}

func movieHash(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", 0, err
	}
	const chunk = 64 * 1024
	if info.Size() < chunk*2 {
		return "", info.Size(), errors.New("file is too small for OpenSubtitles hash")
	}
	sum := uint64(info.Size())
	buffer := make([]byte, chunk)
	if _, err := io.ReadFull(file, buffer); err != nil {
		return "", info.Size(), err
	}
	for index := 0; index < len(buffer); index += 8 {
		sum += binary.LittleEndian.Uint64(buffer[index : index+8])
	}
	if _, err := file.Seek(-chunk, io.SeekEnd); err != nil {
		return "", info.Size(), err
	}
	if _, err := io.ReadFull(file, buffer); err != nil {
		return "", info.Size(), err
	}
	for index := 0; index < len(buffer); index += 8 {
		sum += binary.LittleEndian.Uint64(buffer[index : index+8])
	}
	return fmt.Sprintf("%016x", sum), info.Size(), nil
}

func normalizeLanguages(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = normalizeLanguage(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizeLanguage(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if dash := strings.IndexByte(value, '-'); dash > 0 {
		value = value[:dash]
	}
	aliases := map[string]string{"est": "et", "eng": "en", "fin": "fi", "rus": "ru", "deu": "de", "ger": "de", "fra": "fr", "fre": "fr", "zho": "zh", "chi": "zh"}
	if mapped := aliases[value]; mapped != "" {
		return mapped
	}
	if len(value) == 2 || len(value) == 3 {
		return value
	}
	return ""
}

var _ ports.OnlineSubtitleProvider = (*Provider)(nil)
