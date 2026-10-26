package themerrdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

const defaultBaseURL = "https://app.lizardbyte.dev/ThemerrDB"

type Provider struct {
	client    *http.Client
	baseURL   string
	userAgent string
}

type response struct {
	YouTubeThemeURL string `json:"youtube_theme_url"`
}

func New(version string) *Provider {
	return NewWithClient(defaultBaseURL, &http.Client{Timeout: 20 * time.Second}, "KoduMedia/"+strings.TrimSpace(version))
}

func NewWithClient(baseURL string, client *http.Client, userAgent string) *Provider {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	return &Provider{
		client:    client,
		baseURL:   strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		userAgent: strings.TrimSpace(userAgent),
	}
}

func (p *Provider) LookupSeriesTheme(ctx context.Context, tmdbID string) (domain.ThemeSource, error) {
	tmdbID = strings.TrimSpace(tmdbID)
	parsedID, err := strconv.ParseInt(tmdbID, 10, 64)
	if err != nil || parsedID <= 0 {
		return domain.ThemeSource{}, fmt.Errorf("%w: invalid TMDb series id", domain.ErrInvalidInput)
	}
	requestURL := fmt.Sprintf("%s/tv_shows/themoviedb/%s.json", p.baseURL, tmdbID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return domain.ThemeSource{}, fmt.Errorf("create ThemerrDB request: %w", err)
	}
	if p.userAgent != "" {
		req.Header.Set("User-Agent", p.userAgent)
	}
	req.Header.Set("Accept", "application/json")
	res, err := p.client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return domain.ThemeSource{}, err
		}
		return domain.ThemeSource{}, fmt.Errorf("%w: ThemerrDB request failed: %v", domain.ErrUnavailable, err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return domain.ThemeSource{}, domain.ErrNotFound
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
		return domain.ThemeSource{}, fmt.Errorf("%w: ThemerrDB returned HTTP %d", domain.ErrUnavailable, res.StatusCode)
	}
	var payload response
	decoder := json.NewDecoder(io.LimitReader(res.Body, 1<<20))
	if err := decoder.Decode(&payload); err != nil {
		return domain.ThemeSource{}, fmt.Errorf("%w: decode ThemerrDB response: %v", domain.ErrUnavailable, err)
	}
	themeURL, err := validateYouTubeURL(payload.YouTubeThemeURL)
	if err != nil {
		return domain.ThemeSource{}, err
	}
	return domain.ThemeSource{Provider: "themerrdb", ExternalID: tmdbID, URL: themeURL}, nil
}

func validateYouTubeURL(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
		return "", fmt.Errorf("%w: ThemerrDB theme URL is invalid", domain.ErrUnavailable)
	}
	host := strings.ToLower(parsed.Hostname())
	switch host {
	case "youtube.com", "www.youtube.com", "m.youtube.com", "music.youtube.com", "youtu.be":
	default:
		return "", fmt.Errorf("%w: ThemerrDB theme URL is not a supported YouTube URL", domain.ErrUnavailable)
	}
	return parsed.String(), nil
}
