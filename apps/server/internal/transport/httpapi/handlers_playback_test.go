package httpapi

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type statsTestTranscoder struct {
	stats domain.PlaybackRuntimeStats
	err   error
}

func (f *statsTestTranscoder) Available(context.Context) bool { return true }
func (f *statsTestTranscoder) Start(context.Context, ports.TranscodeStartRequest) (domain.PlaybackSession, error) {
	return domain.PlaybackSession{}, domain.ErrUnsupported
}
func (f *statsTestTranscoder) Open(context.Context, string, string) (fs.File, error) {
	return nil, domain.ErrUnsupported
}
func (f *statsTestTranscoder) Touch(string) {}
func (f *statsTestTranscoder) Stats(string) (domain.PlaybackRuntimeStats, error) {
	return f.stats, f.err
}
func (f *statsTestTranscoder) Stop(string) error { return nil }
func (f *statsTestTranscoder) Close() error      { return nil }

func TestPlaybackStatsEndpointReturnsRuntimeDiagnostics(t *testing.T) {
	updatedAt := time.Date(2026, time.July, 19, 10, 0, 0, 0, time.UTC)
	transcoder := &statsTestTranscoder{stats: domain.PlaybackRuntimeStats{
		SessionID:           "session-1",
		State:               "running",
		ProducerAlive:       true,
		AverageEncoderSpeed: 1.34,
		GeneratedDurationMS: 42_000,
		SegmentCount:        21,
		EstimatedBitrate:    18_500_000,
		MinimumEncoderSpeed: 1.20,
		UpdatedAt:           updatedAt,
	}}
	handler := NewRouter(Dependencies{Transcoder: transcoder})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/playback/sessions/session-1/stats", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("runtime diagnostics must not be cached: %q", response.Header().Get("Cache-Control"))
	}
	var payload domain.PlaybackRuntimeStats
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.SessionID != "session-1" || payload.AverageEncoderSpeed != 1.34 || payload.SegmentCount != 21 {
		t.Fatalf("unexpected stats payload: %+v", payload)
	}
}

func TestPlaybackStatsEndpointRequiresTranscoder(t *testing.T) {
	handler := NewRouter(Dependencies{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/playback/sessions/session-1/stats", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d; want 503", response.Code)
	}
}
