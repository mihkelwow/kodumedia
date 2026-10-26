package themerrdb

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestProviderLooksUpSeriesByTMDbID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tv_shows/themoviedb/1428.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"youtube_theme_url":"https://www.youtube.com/watch?v=q0eTXZgTBUk"}`))
	}))
	defer server.Close()

	provider := NewWithClient(server.URL, server.Client(), "KoduMedia/test")
	source, err := provider.LookupSeriesTheme(context.Background(), "1428")
	if err != nil {
		t.Fatal(err)
	}
	if source.Provider != "themerrdb" || source.ExternalID != "1428" || source.URL == "" {
		t.Fatalf("unexpected source: %+v", source)
	}
}

func TestProviderTreatsMissingThemeAsNotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	provider := NewWithClient(server.URL, server.Client(), "")
	_, err := provider.LookupSeriesTheme(context.Background(), "1428")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestProviderRejectsNonYouTubeThemeURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"youtube_theme_url":"https://example.com/theme.mp3"}`))
	}))
	defer server.Close()
	provider := NewWithClient(server.URL, server.Client(), "")
	_, err := provider.LookupSeriesTheme(context.Background(), "1428")
	if !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("expected unavailable, got %v", err)
	}
}
