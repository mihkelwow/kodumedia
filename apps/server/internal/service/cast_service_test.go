package service

import (
	"context"
	"errors"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type castPlaybackStub struct {
	capabilities domain.ClientCapabilities
	preferences  domain.PlaybackPreferences
	stopped      string
}

func (stub *castPlaybackStub) Start(_ context.Context, _ string, capabilities domain.ClientCapabilities, preferences domain.PlaybackPreferences) (domain.PlaybackSession, error) {
	stub.capabilities = capabilities
	stub.preferences = preferences
	return domain.PlaybackSession{
		ID:          "playback-1",
		PlaylistURL: "/api/v1/playback/sessions/playback-1/index.m3u8",
		Plan: domain.PlaybackPlan{
			Mode:              domain.PlaybackModeTranscode,
			Container:         "mp4",
			StartPositionMS:   90_000,
			InitialPositionMS: 0,
			DurationMS:        600_000,
		},
	}, nil
}

func (stub *castPlaybackStub) Stop(sessionID string) error {
	stub.stopped = sessionID
	return nil
}

type castControllerStub struct {
	device       domain.CastDevice
	loaded       ports.CastLoadRequest
	seekPosition int64
}

func (stub *castControllerStub) Discover(context.Context) ([]domain.CastDevice, error) {
	return []domain.CastDevice{stub.device}, nil
}

func (stub *castControllerStub) ReachableMediaURL(_ context.Context, _ string, mediaURL string) (string, error) {
	return mediaURL, nil
}

func (stub *castControllerStub) Load(_ context.Context, _ string, request ports.CastLoadRequest) error {
	stub.loaded = request
	return nil
}

func (*castControllerStub) Play(context.Context, string) error  { return nil }
func (*castControllerStub) Pause(context.Context, string) error { return nil }
func (*castControllerStub) Stop(context.Context, string) error  { return nil }

func (stub *castControllerStub) Seek(_ context.Context, _ string, positionMS int64) error {
	stub.seekPosition = positionMS
	return nil
}

func (*castControllerStub) SetVolume(context.Context, string, float64) error { return nil }
func (*castControllerStub) SetMuted(context.Context, string, bool) error     { return nil }
func (*castControllerStub) Status(context.Context, string) (domain.CastDeviceStatus, error) {
	return domain.CastDeviceStatus{State: domain.CastStatePlaying, PositionMS: 30_000, DurationMS: 510_000, Volume: 0.65}, nil
}

func TestCastServiceStartsSamsungDLNAWithBurnedSubtitlesAndAbsoluteTimeline(t *testing.T) {
	playback := &castPlaybackStub{}
	controller := &castControllerStub{device: domain.CastDevice{
		ID: "tv-1", Name: "Elutoa teler", Manufacturer: "Samsung", Model: "RU7172",
		Protocol: domain.CastProtocolDLNA, CanSeek: true, CanSetVolume: true,
	}}
	service := NewCastService(playback, controller)

	session, err := service.Start(context.Background(), CastStartRequest{
		DeviceID: "tv-1", MediaID: "movie-1", BaseURL: "http://192.168.1.20:8096",
		Title: "Film", Preferences: domain.PlaybackPreferences{StartPositionMS: 90_000, SubtitleID: "sub-et"},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if !playback.preferences.ForceSubtitleBurn {
		t.Fatal("expected selected subtitle to be burned into the cast stream")
	}
	if playback.capabilities.MaximumWidth != 3840 || playback.capabilities.MaximumHeight != 2160 {
		t.Fatalf("expected Samsung 4K capabilities, got %dx%d", playback.capabilities.MaximumWidth, playback.capabilities.MaximumHeight)
	}
	if controller.loaded.MediaURL != "http://192.168.1.20:8096/api/v1/playback/sessions/playback-1/index.m3u8" {
		t.Fatalf("unexpected media URL: %s", controller.loaded.MediaURL)
	}
	if controller.loaded.PositionMS != 0 || session.PositionMS != 90_000 {
		t.Fatalf("expected receiver-relative 0 and absolute 90000 positions, got load=%d session=%d", controller.loaded.PositionMS, session.PositionMS)
	}
	if err := service.StopSource(session.SourceID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("active DLNA source deletion error = %v; want conflict", err)
	}

	updated, err := service.Command(context.Background(), session.ID, domain.CastCommand{Action: domain.CastCommandSeek, PositionMS: 120_000})
	if err != nil {
		t.Fatalf("seek command returned error: %v", err)
	}
	if controller.seekPosition != 30_000 {
		t.Fatalf("receiver seek = %d; want 30000", controller.seekPosition)
	}
	if updated.PositionMS != 120_000 {
		t.Fatalf("absolute status position = %d; want 120000", updated.PositionMS)
	}

	if err := service.Stop(context.Background(), session.ID); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if playback.stopped != "playback-1" {
		t.Fatalf("stopped playback session = %q; want playback-1", playback.stopped)
	}
}

func TestAbsoluteCastURLRejectsPathsOutsidePlaybackAPI(t *testing.T) {
	if _, err := absoluteCastURL("http://192.168.1.20:8096", "https://example.com/video.mp4"); err == nil {
		t.Fatal("expected absolute external URL to be rejected")
	}
	if _, err := absoluteCastURL("http://192.168.1.20:8096", "/admin/secrets"); err == nil {
		t.Fatal("expected non-API path to be rejected")
	}
}
