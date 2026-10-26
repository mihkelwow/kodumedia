package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/service"
)

func TestUpdateThemeSettingsStoresModeAndReturnsFixedPlaybackValues(t *testing.T) {
	repository := &httpTestSettingsRepository{value: domain.ServerSettings{
		FFmpegPath:  "ffmpeg.exe",
		FFprobePath: "ffprobe.exe",
		ThemeMusic:  domain.ThemeMusicSettings{SeriesPlayback: domain.ThemePlaybackAlways},
	}}
	settings := service.NewSettingsService(repository, httpTestInspector{}, "0.8.21")
	handler := NewRouter(Dependencies{Settings: settings})
	request := httptest.NewRequest(http.MethodPut, "/api/v1/themes/settings", strings.NewReader(`{"seriesPlayback":"first-open"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if repository.value.ThemeMusic.SeriesPlayback != domain.ThemePlaybackFirstOpen {
		t.Fatalf("theme mode was not stored: %+v", repository.value.ThemeMusic)
	}
	var payload domain.ThemePlaybackConfiguration
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.SeriesPlayback != domain.ThemePlaybackFirstOpen || payload.Volume != 0.15 || payload.FadeInMS != 2000 || payload.FadeOutMS != 2000 {
		t.Fatalf("unexpected playback configuration: %+v", payload)
	}
}

func TestUpdateThemeSettingsRejectsUnknownMode(t *testing.T) {
	repository := &httpTestSettingsRepository{value: domain.ServerSettings{ThemeMusic: domain.ThemeMusicSettings{SeriesPlayback: domain.ThemePlaybackAlways}}}
	settings := service.NewSettingsService(repository, httpTestInspector{}, "0.8.21")
	handler := NewRouter(Dependencies{Settings: settings})
	request := httptest.NewRequest(http.MethodPut, "/api/v1/themes/settings", strings.NewReader(`{"seriesPlayback":"sometimes"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", response.Code, response.Body.String())
	}
	if repository.value.ThemeMusic.SeriesPlayback != domain.ThemePlaybackAlways {
		t.Fatalf("invalid mode changed stored settings: %+v", repository.value.ThemeMusic)
	}
}
