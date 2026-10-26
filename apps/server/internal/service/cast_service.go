package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type CastPlayback interface {
	Start(ctx context.Context, mediaID string, capabilities domain.ClientCapabilities, preferences domain.PlaybackPreferences) (domain.PlaybackSession, error)
	Stop(sessionID string) error
}

type CastStartRequest struct {
	DeviceID    string
	MediaID     string
	BaseURL     string
	Title       string
	Subtitle    string
	Preferences domain.PlaybackPreferences
}

type CastSourceRequest struct {
	MediaID     string
	BaseURL     string
	Preferences domain.PlaybackPreferences
}

type CastService struct {
	playback   CastPlayback
	controller ports.CastController

	mu       sync.RWMutex
	sources  map[string]domain.CastSource
	sessions map[string]domain.CastSession
	byDevice map[string]string
}

func NewCastService(playback CastPlayback, controller ports.CastController) *CastService {
	return &CastService{
		playback: playback, controller: controller,
		sources: make(map[string]domain.CastSource), sessions: make(map[string]domain.CastSession), byDevice: make(map[string]string),
	}
}

func (s *CastService) Devices(ctx context.Context) ([]domain.CastDevice, error) {
	if s == nil || s.controller == nil {
		return nil, fmt.Errorf("%w: castimise teenus pole saadaval", domain.ErrUnavailable)
	}
	return s.controller.Discover(ctx)
}

func (s *CastService) PrepareBrowserSource(ctx context.Context, request CastSourceRequest) (domain.CastSource, error) {
	if s == nil || s.playback == nil {
		return domain.CastSource{}, fmt.Errorf("%w: castimise esitusvoog pole saadaval", domain.ErrUnavailable)
	}
	return s.prepareSource(ctx, request.MediaID, request.BaseURL, browserCastCapabilities(), request.Preferences)
}

func (s *CastService) Start(ctx context.Context, request CastStartRequest) (domain.CastSession, error) {
	if s == nil || s.playback == nil || s.controller == nil {
		return domain.CastSession{}, fmt.Errorf("%w: castimise teenus pole saadaval", domain.ErrUnavailable)
	}
	request.DeviceID = strings.TrimSpace(request.DeviceID)
	request.MediaID = strings.TrimSpace(request.MediaID)
	if request.DeviceID == "" || request.MediaID == "" {
		return domain.CastSession{}, fmt.Errorf("%w: seade ja meedia on kohustuslikud", domain.ErrInvalidInput)
	}
	devices, err := s.controller.Discover(ctx)
	if err != nil {
		return domain.CastSession{}, err
	}
	var device *domain.CastDevice
	for index := range devices {
		if devices[index].ID == request.DeviceID {
			copy := devices[index]
			device = &copy
			break
		}
	}
	if device == nil {
		return domain.CastSession{}, fmt.Errorf("%w: castimise seadet ei leitud", domain.ErrNotFound)
	}

	if existing := s.sessionForDevice(request.DeviceID); existing != nil {
		_ = s.Stop(ctx, existing.ID)
	}
	source, err := s.prepareSource(ctx, request.MediaID, request.BaseURL, capabilitiesForDevice(*device), request.Preferences)
	if err != nil {
		return domain.CastSession{}, err
	}
	reachableURL, err := s.controller.ReachableMediaURL(ctx, request.DeviceID, source.URL)
	if err != nil {
		s.stopSource(source.ID)
		return domain.CastSession{}, err
	}
	source.URL = reachableURL
	slog.Info("cast source ready",
		"device", device.Name,
		"model", device.Model,
		"media", request.MediaID,
		"mode", source.Plan.Mode,
		"container", source.Plan.Container,
		"videoCodec", source.Plan.VideoCodec,
		"videoAction", source.Plan.VideoAction,
		"audioCodec", source.Plan.AudioCodec,
		"audioAction", source.Plan.AudioAction,
		"readyBufferSeconds", source.Plan.MinimumReadyBufferSeconds,
	)
	s.mu.Lock()
	s.sources[source.ID] = source
	s.mu.Unlock()
	load := ports.CastLoadRequest{
		MediaURL:   source.URL,
		Metadata:   domain.CastMediaMetadata{Title: strings.TrimSpace(request.Title), Subtitle: strings.TrimSpace(request.Subtitle), ContentType: source.ContentType},
		PositionMS: source.StartPositionMS,
	}
	if err := s.controller.Load(ctx, request.DeviceID, load); err != nil {
		s.stopSource(source.ID)
		return domain.CastSession{}, err
	}
	now := time.Now().UTC()
	session := domain.CastSession{
		ID: newCastID(), Device: *device, MediaID: request.MediaID, SourceID: source.ID,
		State: domain.CastStatePlaying, PositionMS: source.BasePositionMS + source.StartPositionMS,
		DurationMS: source.DurationMS, Volume: 1, StartedAt: now, LastUpdateAt: now,
	}
	s.mu.Lock()
	s.sessions[session.ID] = session
	s.byDevice[request.DeviceID] = session.ID
	s.mu.Unlock()
	return session, nil
}

func (s *CastService) Active(mediaID string) (domain.CastSession, bool) {
	mediaID = strings.TrimSpace(mediaID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result domain.CastSession
	found := false
	for _, session := range s.sessions {
		if mediaID != "" && session.MediaID != mediaID {
			continue
		}
		if !found || session.StartedAt.After(result.StartedAt) {
			result = session
			found = true
		}
	}
	return result, found
}

func (s *CastService) Status(ctx context.Context, id string) (domain.CastSession, error) {
	session, source, err := s.sessionAndSource(id)
	if err != nil {
		return domain.CastSession{}, err
	}
	status, err := s.controller.Status(ctx, session.Device.ID)
	if err != nil {
		return domain.CastSession{}, err
	}
	session.State = status.State
	session.PositionMS = source.BasePositionMS + status.PositionMS
	if session.PositionMS < 0 {
		session.PositionMS = 0
	}
	if status.DurationMS > 0 && source.BasePositionMS == 0 {
		session.DurationMS = status.DurationMS
	} else if source.DurationMS > 0 {
		session.DurationMS = source.DurationMS
	}
	session.Volume = status.Volume
	session.Muted = status.Muted
	session.LastUpdateAt = time.Now().UTC()
	s.mu.Lock()
	s.sessions[id] = session
	s.mu.Unlock()
	return session, nil
}

func (s *CastService) Command(ctx context.Context, id string, command domain.CastCommand) (domain.CastSession, error) {
	session, source, err := s.sessionAndSource(id)
	if err != nil {
		return domain.CastSession{}, err
	}
	switch command.Action {
	case domain.CastCommandPlay:
		err = s.controller.Play(ctx, session.Device.ID)
	case domain.CastCommandPause:
		err = s.controller.Pause(ctx, session.Device.ID)
	case domain.CastCommandStop:
		if err = s.Stop(ctx, id); err == nil {
			session.State = domain.CastStateStopped
			session.LastUpdateAt = time.Now().UTC()
			return session, nil
		}
	case domain.CastCommandSeek:
		target := command.PositionMS - source.BasePositionMS
		if target < 0 {
			target = 0
		}
		err = s.controller.Seek(ctx, session.Device.ID, target)
	case domain.CastCommandSetVolume:
		err = s.controller.SetVolume(ctx, session.Device.ID, command.Volume)
	case domain.CastCommandSetMuted:
		err = s.controller.SetMuted(ctx, session.Device.ID, command.Muted)
	default:
		return domain.CastSession{}, fmt.Errorf("%w: tundmatu castimise käsk", domain.ErrInvalidInput)
	}
	if err != nil {
		return domain.CastSession{}, err
	}
	return s.Status(ctx, id)
}

func (s *CastService) Stop(ctx context.Context, id string) error {
	session, _, err := s.sessionAndSource(id)
	if err != nil {
		return err
	}
	controllerErr := s.controller.Stop(ctx, session.Device.ID)
	s.mu.Lock()
	delete(s.sessions, id)
	if s.byDevice[session.Device.ID] == id {
		delete(s.byDevice, session.Device.ID)
	}
	s.mu.Unlock()
	s.stopSource(session.SourceID)
	return controllerErr
}

func (s *CastService) StopSource(id string) error {
	s.mu.RLock()
	_, found := s.sources[id]
	inUse := false
	for _, session := range s.sessions {
		if session.SourceID == id {
			inUse = true
			break
		}
	}
	s.mu.RUnlock()
	if !found {
		return fmt.Errorf("%w: castimise esitusvoogu ei leitud", domain.ErrNotFound)
	}
	if inUse {
		return fmt.Errorf("%w: aktiivse teleriseansi esitusvoogu ei saa eraldi lõpetada", domain.ErrConflict)
	}
	s.stopSource(id)
	return nil
}

func (s *CastService) prepareSource(ctx context.Context, mediaID, baseURL string, capabilities domain.ClientCapabilities, preferences domain.PlaybackPreferences) (domain.CastSource, error) {
	mediaID = strings.TrimSpace(mediaID)
	if mediaID == "" {
		return domain.CastSource{}, fmt.Errorf("%w: meedia on kohustuslik", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(preferences.SubtitleID) != "" {
		preferences.ForceSubtitleBurn = true
	}
	playbackSession, err := s.playback.Start(ctx, mediaID, capabilities, preferences)
	if err != nil {
		return domain.CastSource{}, err
	}
	path := playbackSession.Plan.DirectURL
	contentType := mimeForContainer(playbackSession.Plan.Container)
	basePosition := int64(0)
	startPosition := playbackSession.Plan.StartPositionMS
	if playbackSession.Plan.Mode != domain.PlaybackModeDirect {
		path = playbackSession.PlaylistURL
		if path == "" {
			path = playbackSession.Plan.PlaylistURL
		}
		contentType = "application/vnd.apple.mpegurl"
		basePosition = playbackSession.Plan.StartPositionMS
		startPosition = playbackSession.Plan.InitialPositionMS
	}
	absoluteURL, err := absoluteCastURL(baseURL, path)
	if err != nil {
		if playbackSession.ID != "" {
			_ = s.playback.Stop(playbackSession.ID)
		}
		return domain.CastSource{}, err
	}
	source := domain.CastSource{
		ID: newCastID(), MediaID: mediaID, URL: absoluteURL, ContentType: contentType,
		PlaybackSessionID: playbackSession.ID, BasePositionMS: basePosition, StartPositionMS: startPosition,
		DurationMS: playbackSession.Plan.DurationMS, Plan: playbackSession.Plan, CreatedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	s.sources[source.ID] = source
	s.mu.Unlock()
	return source, nil
}

func (s *CastService) stopSource(id string) {
	s.mu.Lock()
	source, found := s.sources[id]
	if found {
		delete(s.sources, id)
	}
	s.mu.Unlock()
	if found && source.PlaybackSessionID != "" && s.playback != nil {
		_ = s.playback.Stop(source.PlaybackSessionID)
	}
}

func (s *CastService) sessionAndSource(id string) (domain.CastSession, domain.CastSource, error) {
	s.mu.RLock()
	session, found := s.sessions[strings.TrimSpace(id)]
	source, sourceFound := s.sources[session.SourceID]
	s.mu.RUnlock()
	if !found {
		return domain.CastSession{}, domain.CastSource{}, fmt.Errorf("%w: castimise seanssi ei leitud", domain.ErrNotFound)
	}
	if !sourceFound {
		return domain.CastSession{}, domain.CastSource{}, fmt.Errorf("%w: castimise esitusvoogu ei leitud", domain.ErrNotFound)
	}
	return session, source, nil
}

func (s *CastService) sessionForDevice(deviceID string) *domain.CastSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id := s.byDevice[deviceID]
	session, found := s.sessions[id]
	if !found {
		return nil
	}
	return &session
}

func absoluteCastURL(baseURL, mediaPath string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || base.Scheme == "" || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") {
		return "", fmt.Errorf("%w: vigane serveri võrguaadress", domain.ErrInvalidInput)
	}
	reference, err := url.Parse(strings.TrimSpace(mediaPath))
	if err != nil || reference.IsAbs() || !strings.HasPrefix(reference.Path, "/api/v1/") {
		return "", fmt.Errorf("%w: vigane castimise esitusallikas", domain.ErrInvalidInput)
	}
	return base.ResolveReference(reference).String(), nil
}

func mimeForContainer(container string) string {
	switch strings.ToLower(strings.TrimSpace(container)) {
	case "mp4", "mov", "m4v":
		return "video/mp4"
	case "webm":
		return "video/webm"
	case "mp3":
		return "audio/mpeg"
	case "flac":
		return "audio/flac"
	case "wav":
		return "audio/wav"
	default:
		return "video/mp4"
	}
}

func browserCastCapabilities() domain.ClientCapabilities {
	return domain.ClientCapabilities{
		ClientName: "KoduMedia Google Cast", ClientVersion: "1",
		SupportedMIMEs: []string{"video/mp4", "application/vnd.apple.mpegurl"},
		VideoCodecs:    []string{"h264"}, AudioCodecs: []string{"aac", "mp3"}, SubtitleFormats: []string{},
		SupportsHLS: true, MaximumWidth: 1920, MaximumHeight: 1080, MaximumBitrate: 24_000_000, MaximumAudioChannels: 6,
	}
}

var modernSamsungModel = regexp.MustCompile(`(?i)(?:^|[^A-Z])(RU|NU|Q[0-9A-Z]*|LS)[0-9]{2,}`)

func capabilitiesForDevice(device domain.CastDevice) domain.ClientCapabilities {
	capabilities := browserCastCapabilities()
	capabilities.ClientName = strings.TrimSpace(device.Manufacturer + " " + device.Model + " DLNA")
	if capabilities.ClientName == "DLNA" {
		capabilities.ClientName = "KoduMedia DLNA"
	}
	identity := strings.TrimSpace(device.Name + " " + device.Model)
	if strings.Contains(strings.ToLower(device.Manufacturer), "samsung") && modernSamsungModel.MatchString(identity) {
		capabilities.VideoCodecs = []string{"h264", "hevc", "h265"}
		capabilities.AudioCodecs = []string{"aac", "ac3", "eac3", "mp3"}
		capabilities.HDRFormats = []string{"hdr10", "hdr10+", "hlg"}
		capabilities.MaximumWidth = 3840
		capabilities.MaximumHeight = 2160
		capabilities.MaximumBitrate = 80_000_000
		capabilities.MaximumAudioChannels = 8
	}
	return capabilities
}

func newCastID() string {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err == nil {
		return hex.EncodeToString(value)
	}
	return fmt.Sprintf("cast-%d", time.Now().UnixNano())
}
