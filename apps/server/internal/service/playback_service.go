package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type PlaybackPlanner struct {
	quality func() domain.QualitySettings
}

func NewPlaybackPlanner(providers ...func() domain.QualitySettings) *PlaybackPlanner {
	planner := &PlaybackPlanner{}
	if len(providers) > 0 {
		planner.quality = providers[0]
	}
	return planner
}

func (p *PlaybackPlanner) qualitySettings() domain.QualitySettings {
	if p.quality != nil {
		return p.quality()
	}
	return domain.QualitySettings{Profile: "original", PreferredCodec: "auto", HardwarePreference: "auto", CachePolicy: "watched", CacheMaxGB: 50, CacheFreeReserveGB: 10, CacheFreeReservePercent: 10, GracePeriodSeconds: 15, VMAFEnabled: true, VMAFTarget: 95, VMAFSampleSeconds: 12}
}

func (p *PlaybackPlanner) Plan(item domain.MediaItem, capabilities domain.ClientCapabilities, preferences domain.PlaybackPreferences, subtitle *domain.SubtitleTrack) domain.PlaybackPlan {
	quality := p.qualitySettings()
	recoveryLevel := clampRecoveryLevel(preferences.RecoveryLevel)
	effectiveQuality := qualityForRecovery(quality, recoveryLevel)
	forceStability := recoveryLevel > 0
	video := firstTrack(item.Technical.Tracks, "video")
	audio := selectedAudioTrack(item.Technical.Tracks, preferences.AudioTrackIndex)
	videoCodec := item.Technical.VideoCodec
	if video != nil && video.Codec != "" {
		videoCodec = video.Codec
	}
	audioCodec := item.Technical.AudioCodec
	audioChannels := item.Technical.AudioChannels
	if audio != nil {
		audioCodec, audioChannels = audio.Codec, audio.Channels
	}

	hdr := strings.ToLower(strings.TrimSpace(item.Technical.HDRFormat))
	if video != nil && video.HDRFormat != "" {
		hdr = strings.ToLower(strings.TrimSpace(video.HDRFormat))
	}
	hdrMode := normalizeHDRMode(preferences.HDRMode)
	hasHDR := hdr != ""
	clientAdvertisesHDR := hasHDR && contains(capabilities.HDRFormats, hdr)
	forceHDR := hasHDR && hdrMode == domain.HDRModeHDR
	forceSDR := hasHDR && hdrMode == domain.HDRModeSDR
	wantsHDR := hasHDR && !forceSDR && (forceHDR || clientAdvertisesHDR)
	if (preferences.ForceTranscode || forceStability) && hasHDR {
		wantsHDR = false
	}
	toneMap := hasHDR && !wantsHDR

	withinLimits := withinClientLimits(item.Technical, audioChannels, capabilities)
	containerSupported := supportsContainer(capabilities.SupportedMIMEs, item.Technical.Container, videoCodec != "")
	videoSupported := videoCodec == "" || supportsVideoDecode(item.Technical, capabilities)
	if forceHDR && videoCodec != "" {
		videoSupported = true
	}
	audioSupported := audioCodec == "" || contains(capabilities.AudioCodecs, audioCodec)
	explicitAlternateAudio := preferences.AudioTrackIndex != nil && (audio == nil || !audio.Default && audio.StreamOrdinal != 0)
	burnSubtitle := subtitle != nil && (subtitle.RequiresBurnIn || preferences.ForceSubtitleBurn)
	dlnaPlayback := strings.Contains(strings.ToLower(strings.TrimSpace(capabilities.ClientName)), "dlna")

	if !preferences.ForceTranscode && !forceStability && !toneMap && !burnSubtitle && !explicitAlternateAudio && containerSupported && videoSupported && audioSupported && withinLimits && safeDirectContainer(item.Technical.Container, videoCodec != "") {
		plan := domain.PlaybackPlan{
			Mode: domain.PlaybackModeDirect, Reason: "Lähtefail sobib kliendile muutmata kujul.", Container: item.Technical.Container,
			VideoCodec: videoCodec, AudioCodec: audioCodec,
			VideoAction: actionForPresentCodec(videoCodec, domain.TrackActionCopy), AudioAction: actionForPresentCodec(audioCodec, domain.TrackActionCopy),
			SubtitleAction: subtitleAction(subtitle), SelectedSubtitleID: subtitleID(subtitle), StartPositionMS: nonNegative(preferences.StartPositionMS),
			DurationMS: item.Technical.DurationMS, DirectURL: "/api/v1/media/" + item.ID + "/stream",
			PreserveHDR: hasHDR, SourceHDRFormat: hdr, QualityProfile: effectiveQuality.Profile, RecoveryLevel: recoveryLevel,
		}
		if hasHDR {
			plan.Reason = "Klient toetab HDR-vormingut; lähtevideo esitatakse HDR-ina muutmata kujul."
		}
		if audio != nil {
			value := audio.Index
			plan.SelectedAudioTrackIndex = &value
		}
		applyPlaybackBufferPolicy(&plan, item.Technical)
		return plan
	}

	videoAction := domain.TrackActionDrop
	if videoCodec != "" {
		videoAction = domain.TrackActionTranscode
		if !preferences.ForceTranscode && !forceStability && !toneMap && !burnSubtitle && videoSupported && withinVideoLimits(item.Technical, capabilities) {
			videoAction = domain.TrackActionCopy
		}
	}
	audioAction := domain.TrackActionDrop
	if audioCodec != "" {
		audioAction = domain.TrackActionTranscode
		if !preferences.ForceTranscode && audioSupported && withinAudioLimits(audioChannels, capabilities) {
			audioAction = domain.TrackActionCopy
		}
	}
	if burnSubtitle || toneMap {
		videoAction = domain.TrackActionTranscode
	}

	targetWidth, targetHeight := 0, 0
	codecCandidates := []string(nil)
	dimensionProfile := effectiveQuality.Profile
	if preferences.ForceTranscode && recoveryLevel == 0 {
		dimensionProfile = "balanced"
	}
	if videoAction == domain.TrackActionTranscode {
		targetWidth, targetHeight, codecCandidates = qualityOutputSelection(item.Technical, capabilities, effectiveQuality, dimensionProfile, wantsHDR)
		if recoveryLevel >= 3 {
			targetWidth, targetHeight = dimensionsWithin(item.Technical.Width, item.Technical.Height, 854, 480)
		}
		// Real-time 4K HEVC encoding is not a safe universal DLNA fallback.
		// Preserve compatible 4K tracks through direct play/remux, but when a
		// subtitle or incompatible track requires encoding, use the broadly
		// supported 1080p H.264 path with enough encoder headroom.
		if dlnaPlayback {
			targetWidth, targetHeight = dimensionsWithin(item.Technical.Width, item.Technical.Height, 1920, 1080)
			codecCandidates = []string{"h264"}
			wantsHDR = false
			toneMap = hasHDR
		}
	}
	if len(codecCandidates) == 0 {
		codecCandidates = []string{"h264"}
	}
	codec := codecCandidates[0]
	encoder, preset, crf, pixelFormat := qualityVideoSettings(effectiveQuality, item.Technical, wantsHDR, codec)
	if preferences.ForceTranscode || forceStability {
		codecCandidates = []string{"h264"}
		codec = "h264"
		pixelFormat = "yuv420p"
		wantsHDR = false
		toneMap = hasHDR
	}

	preserveHDR := hasHDR && !toneMap && videoAction == domain.TrackActionCopy
	if hasHDR && videoAction == domain.TrackActionTranscode && wantsHDR {
		if (codec == "hevc" || codec == "av1") && supportsOutputCodec(capabilities, codec, targetWidth, targetHeight, 10, hdr) {
			preserveHDR = true
			pixelFormat = "yuv420p10le"
		} else {
			toneMap = true
			preserveHDR = false
			codec = "h264"
			pixelFormat = "yuv420p"
		}
	}
	if toneMap {
		videoAction = domain.TrackActionTranscode
		preserveHDR = false
		if codec != "h264" && !supportsOutputCodec(capabilities, codec, targetWidth, targetHeight, 8, "") {
			codec = "h264"
		}
		pixelFormat = "yuv420p"
	}

	mode := domain.PlaybackModeRemux
	reason := "Video- ja helirajad sobivad; server vahetab ainult konteineri HLS-voo vastu."
	if preferences.ForceTranscode || forceStability || videoAction == domain.TrackActionTranscode || audioAction == domain.TrackActionTranscode || burnSubtitle || toneMap {
		mode = domain.PlaybackModeTranscode
		reason = "Kvaliteedimootor loob kliendile sobiva kõrge kvaliteediga HLS-voo."
	}
	if preserveHDR && videoAction == domain.TrackActionCopy {
		reason = "Server säilitab HDR-video ning pakib selle brauserile sobivasse HLS-voogu."
	} else if preserveHDR && videoAction == domain.TrackActionTranscode {
		reason = "Server säilitab HDR-i ja lähteresolutsiooni kvaliteetses 10-bit voos."
	}
	if toneMap {
		reason = "HDR-vorming ei ole kliendis töökindlalt toetatud või kasutaja valis SDR-i; server teeb kvaliteetse HDR→SDR tone mapping'u."
	}
	if burnSubtitle {
		reason = "Pildisubtiitrid põletatakse töökindluse huvides videopilti."
	}
	if preferences.ForceTranscode {
		if hasHDR && toneMap {
			reason = "HDR-video esitatakse töökindla SDR fallback-voona (H.264/AAC)."
		} else {
			reason = "Klient palus pärast ebaõnnestunud esitust universaalset H.264/AAC transkodeerimist."
		}
	}
	if recoveryLevel == 1 {
		reason = "KoduMedia valis puhvri taastamiseks stabiilse 1080p H.264 kvaliteediprofiili."
	} else if recoveryLevel == 2 {
		reason = "KoduMedia valis korduva puhverdamise vältimiseks andmesäästliku 720p H.264 profiili."
	} else if recoveryLevel >= 3 {
		reason = "KoduMedia valis viimase töökindla fallback'ina 480p H.264 turvarežiimi."
	}

	plan := domain.PlaybackPlan{
		Mode: mode, Reason: reason, Container: "fmp4-hls",
		VideoCodec: targetCodec(videoCodec, videoAction, codec), AudioCodec: targetCodec(audioCodec, audioAction, "aac"),
		VideoAction: videoAction, AudioAction: audioAction, SubtitleAction: subtitleAction(subtitle),
		SelectedSubtitleID: subtitleID(subtitle), ToneMap: toneMap, PreserveHDR: preserveHDR, SourceHDRFormat: hdr,
		TargetWidth: targetWidth, TargetHeight: targetHeight,
		StartPositionMS: nonNegative(preferences.StartPositionMS), DurationMS: item.Technical.DurationMS,
		TranscodeProfile: "adaptive-video-aac", QualityProfile: effectiveQuality.Profile, RecoveryLevel: recoveryLevel,
		VideoEncoder: encoder, VideoPreset: preset, VideoCRF: crf, PixelFormat: pixelFormat,
		VideoCodecCandidates: codecCandidates, PreferredVideoCodec: effectiveQuality.PreferredCodec, HardwarePreference: effectiveQuality.HardwarePreference,
		CachePolicy: effectiveQuality.CachePolicy, CacheMaxBytes: int64(effectiveQuality.CacheMaxGB) << 30,
		CacheFreeReserveBytes:   int64(effectiveQuality.CacheFreeReserveGB) << 30,
		CacheFreeReservePercent: effectiveQuality.CacheFreeReservePercent,
		GracePeriodSeconds:      effectiveQuality.GracePeriodSeconds,
		VMAFEnabled:             effectiveQuality.VMAFEnabled, VMAFTarget: effectiveQuality.VMAFTarget, VMAFSampleSeconds: effectiveQuality.VMAFSampleSeconds,
		AllowSoftwareAV1: effectiveQuality.PreferredCodec == "av1" && effectiveQuality.HardwarePreference == "software",
	}
	if burnSubtitle {
		plan.SubtitleAction = domain.TrackActionBurn
	}
	if audio != nil {
		value := audio.Index
		plan.SelectedAudioTrackIndex = &value
	}
	applyPlaybackBufferPolicy(&plan, item.Technical)
	if dlnaPlayback {
		// Samsung and other television renderers are substantially more stable
		// with classic MPEG-TS HLS than fragmented-MP4 HLS. They also start
		// consuming immediately, so hold the AVTransport hand-off until a real
		// playback cushion exists instead of exposing a one-segment playlist.
		plan.Container = "mpegts-hls"
		plan.CachePolicy = "off"
		plan.MinimumReadyBufferSeconds = 12
		if plan.StartupBufferSeconds > plan.MinimumReadyBufferSeconds {
			plan.MinimumReadyBufferSeconds = plan.StartupBufferSeconds
		}
		if plan.MinimumReadyBufferSeconds > 20 {
			plan.MinimumReadyBufferSeconds = 20
		}
		if item.Technical.DurationMS > 0 && item.Technical.DurationMS < int64(90*time.Second/time.Millisecond) && plan.MinimumReadyBufferSeconds > 6 {
			plan.MinimumReadyBufferSeconds = 6
		}
	}
	return plan
}

func clampRecoveryLevel(value int) int {
	if value < 0 {
		return 0
	}
	if value > 3 {
		return 3
	}
	return value
}

func qualityForRecovery(settings domain.QualitySettings, level int) domain.QualitySettings {
	result := settings
	switch clampRecoveryLevel(level) {
	case 1:
		result.Profile = "balanced"
		result.PreferredCodec = "h264"
		result.VMAFEnabled = false
	case 2, 3:
		result.Profile = "data-saver"
		result.PreferredCodec = "h264"
		result.VMAFEnabled = false
	}
	return result
}

func applyPlaybackBufferPolicy(plan *domain.PlaybackPlan, technical domain.TechnicalMetadata) {
	if plan == nil {
		return
	}
	short := technical.DurationMS > 0 && technical.DurationMS < int64(90*time.Second/time.Millisecond)
	if plan.Mode == domain.PlaybackModeDirect {
		plan.StartupBufferSeconds = 4
		plan.RebufferBufferSeconds = 8
		plan.DesiredBufferSeconds = 20
		plan.LowBufferSeconds = 4
		if short {
			plan.StartupBufferSeconds, plan.RebufferBufferSeconds, plan.DesiredBufferSeconds = 2, 4, 10
		}
		return
	}

	if plan.VideoAction != domain.TrackActionTranscode {
		plan.StartupBufferSeconds = 20
		plan.RebufferBufferSeconds = 30
		plan.DesiredBufferSeconds = 60
		plan.LowBufferSeconds = 16
		plan.ReadRate = 3.0
		plan.InitialBurstSeconds = 45
		if plan.AudioAction == domain.TrackActionTranscode {
			plan.ReadRate = 2.5
		}
		if short {
			plan.StartupBufferSeconds, plan.RebufferBufferSeconds, plan.DesiredBufferSeconds = 3, 6, 12
			plan.LowBufferSeconds = 3
			plan.InitialBurstSeconds = 12
		}
		// Do not expose a short event playlist to the browser. The API returns
		// only after the producer has built the same cushion that the player
		// requires, including after a remote seek starts a fresh session.
		plan.MinimumReadyBufferSeconds = plan.StartupBufferSeconds
		return
	}

	width, height := plan.TargetWidth, plan.TargetHeight
	if width <= 0 {
		width = technical.Width
	}
	if height <= 0 {
		height = technical.Height
	}
	highComplexity := width > 1920 || height > 1080 || plan.PreserveHDR || plan.ToneMap || technical.HDRFormat != ""
	switch clampRecoveryLevel(plan.RecoveryLevel) {
	case 1:
		plan.StartupBufferSeconds = 24
		plan.RebufferBufferSeconds = 36
		plan.DesiredBufferSeconds = 90
		plan.LowBufferSeconds = 24
		plan.ReadRate = 2.50
		plan.InitialBurstSeconds = 60
		plan.MinimumEncoderSpeed = 1.15
	case 2:
		plan.StartupBufferSeconds = 20
		plan.RebufferBufferSeconds = 30
		plan.DesiredBufferSeconds = 75
		plan.LowBufferSeconds = 20
		plan.ReadRate = 2.75
		plan.InitialBurstSeconds = 50
		plan.MinimumEncoderSpeed = 1.10
	case 3:
		plan.StartupBufferSeconds = 16
		plan.RebufferBufferSeconds = 24
		plan.DesiredBufferSeconds = 60
		plan.LowBufferSeconds = 16
		plan.ReadRate = 3.00
		plan.InitialBurstSeconds = 40
		plan.MinimumEncoderSpeed = 1.02
	default:
		if highComplexity {
			plan.StartupBufferSeconds = 30
			plan.RebufferBufferSeconds = 45
			plan.DesiredBufferSeconds = 120
			plan.LowBufferSeconds = 30
			plan.ReadRate = 2.25
			plan.InitialBurstSeconds = 75
			plan.MinimumEncoderSpeed = 1.25
		} else {
			plan.StartupBufferSeconds = 24
			plan.RebufferBufferSeconds = 36
			plan.DesiredBufferSeconds = 90
			plan.LowBufferSeconds = 24
			plan.ReadRate = 2.50
			plan.InitialBurstSeconds = 60
			plan.MinimumEncoderSpeed = 1.20
		}
	}
	if short {
		plan.StartupBufferSeconds = minBufferInt(plan.StartupBufferSeconds, 4)
		plan.RebufferBufferSeconds = minBufferInt(plan.RebufferBufferSeconds, 8)
		plan.DesiredBufferSeconds = minBufferInt(plan.DesiredBufferSeconds, 15)
		plan.LowBufferSeconds = minBufferInt(plan.LowBufferSeconds, 4)
		plan.InitialBurstSeconds = minBufferInt(plan.InitialBurstSeconds, 12)
	}
	plan.MinimumReadyBufferSeconds = plan.StartupBufferSeconds
	plan.MaxVideoBitrate, plan.RateControlBuffer = qualityRateControl(plan.VideoCodec, width, height, plan.QualityProfile, technical.Bitrate)
}

func qualityRateControl(codec string, width, height int, profile string, sourceBitrate int64) (int64, int64) {
	pixels := int64(width) * int64(height)
	codec = strings.ToLower(strings.TrimSpace(codec))
	var maximum int64
	switch {
	case pixels > 1920*1080:
		switch codec {
		case "hevc", "h265":
			maximum = 35_000_000
		case "av1":
			maximum = 28_000_000
		default:
			maximum = 52_000_000
		}
	case pixels > 1280*720:
		switch codec {
		case "hevc", "h265":
			maximum = 14_000_000
		case "av1":
			maximum = 10_000_000
		default:
			maximum = 20_000_000
		}
	default:
		switch codec {
		case "hevc", "h265":
			maximum = 7_000_000
		case "av1":
			maximum = 5_500_000
		default:
			maximum = 9_000_000
		}
	}
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "maximum":
		maximum = maximum * 125 / 100
	case "data-saver":
		maximum = maximum * 70 / 100
	}
	if sourceBitrate > 0 && (profile == "original" || profile == "maximum") {
		candidate := sourceBitrate * 130 / 100
		if candidate > maximum {
			maximum = candidate
		}
	}
	const absoluteMaximum = int64(90_000_000)
	if maximum > absoluteMaximum {
		maximum = absoluteMaximum
	}
	return maximum, maximum * 2
}

func minBufferInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func qualityVideoSettings(settings domain.QualitySettings, technical domain.TechnicalMetadata, preserveHDR bool, codec string) (encoder, preset string, crf int, pixelFormat string) {
	codec = strings.ToLower(strings.TrimSpace(codec))
	if codec == "h265" {
		codec = "hevc"
	}
	encoder = settings.HardwarePreference
	if encoder == "" {
		encoder = "auto"
	}
	switch settings.Profile {
	case "maximum":
		preset, crf = "slow", 16
	case "balanced":
		preset, crf = "medium", 18
	case "data-saver":
		preset, crf = "faster", 23
	default:
		preset, crf = "medium", 17
	}
	pixelFormat = "yuv420p"
	if (codec == "hevc" || codec == "av1") && (preserveHDR || technical.BitDepth > 8) {
		pixelFormat = "yuv420p10le"
	}
	return
}

func qualityOutputSelection(technical domain.TechnicalMetadata, capabilities domain.ClientCapabilities, settings domain.QualitySettings, profile string, preserveHDR bool) (int, int, []string) {
	desiredWidth, desiredHeight := qualityTranscodeDimensions(technical, capabilities, profile)
	sizes := [][2]int{{desiredWidth, desiredHeight}}
	for _, maximum := range [][2]int{{1920, 1080}, {1280, 720}, {854, 480}} {
		width, height := dimensionsWithin(technical.Width, technical.Height, maximum[0], maximum[1])
		if width <= 0 || height <= 0 || (width == desiredWidth && height == desiredHeight) {
			continue
		}
		if profile == "data-saver" && (width > 1280 || height > 720) {
			continue
		}
		if profile == "balanced" && (width > 1920 || height > 1080) {
			continue
		}
		sizes = append(sizes, [2]int{width, height})
	}
	for _, size := range sizes {
		candidates := supportedCodecCandidates(settings.PreferredCodec, technical, capabilities, size[0], size[1], preserveHDR)
		if len(candidates) > 0 {
			return size[0], size[1], candidates
		}
	}
	// H.264/AAC remains the final universal fallback. Keeping the desired
	// dimensions here lets the encoder-speed fallback decide whether a 1080p
	// reduction is needed rather than silently lowering quality up front.
	return desiredWidth, desiredHeight, []string{"h264"}
}

func supportedCodecCandidates(preference string, technical domain.TechnicalMetadata, capabilities domain.ClientCapabilities, width, height int, preserveHDR bool) []string {
	preference = strings.ToLower(strings.TrimSpace(preference))
	if preference == "h265" {
		preference = "hevc"
	}
	order := []string{"h264", "hevc", "av1"}
	if preserveHDR || technical.BitDepth > 8 || width > 1920 || height > 1080 {
		order = []string{"hevc", "av1", "h264"}
	}
	if preference != "" && preference != "auto" {
		order = []string{preference, "h264"}
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(order))
	for _, codec := range order {
		if _, ok := seen[codec]; ok {
			continue
		}
		seen[codec] = struct{}{}
		bitDepth, hdr := 8, ""
		if codec != "h264" && (preserveHDR || technical.BitDepth > 8) {
			bitDepth = 10
			if preserveHDR {
				hdr = technical.HDRFormat
			}
		}
		if supportsOutputCodec(capabilities, codec, width, height, bitDepth, hdr) {
			result = append(result, codec)
		}
	}
	return result
}

func dimensionsWithin(width, height, maxWidth, maxHeight int) (int, int) {
	if width <= 0 || height <= 0 {
		return 0, 0
	}
	if maxWidth <= 0 || maxHeight <= 0 || (width <= maxWidth && height <= maxHeight) {
		return evenDimension(width), evenDimension(height)
	}
	ratio := minFloat(float64(maxWidth)/float64(width), float64(maxHeight)/float64(height))
	return evenDimension(int(float64(width) * ratio)), evenDimension(int(float64(height) * ratio))
}

func qualityTranscodeDimensions(technical domain.TechnicalMetadata, capabilities domain.ClientCapabilities, profile string) (int, int) {
	maxW, maxH := capabilities.MaximumWidth, capabilities.MaximumHeight
	switch profile {
	case "balanced":
		if maxW == 0 || maxW > 1920 {
			maxW = 1920
		}
		if maxH == 0 || maxH > 1080 {
			maxH = 1080
		}
	case "data-saver":
		if maxW == 0 || maxW > 1280 {
			maxW = 1280
		}
		if maxH == 0 || maxH > 720 {
			maxH = 720
		}
	}
	if maxW <= 0 || maxH <= 0 || technical.Width <= 0 || technical.Height <= 0 || (technical.Width <= maxW && technical.Height <= maxH) {
		return evenDimension(technical.Width), evenDimension(technical.Height)
	}
	ratio := minFloat(float64(maxW)/float64(technical.Width), float64(maxH)/float64(technical.Height))
	return evenDimension(int(float64(technical.Width) * ratio)), evenDimension(int(float64(technical.Height) * ratio))
}

func evenDimension(value int) int {
	if value <= 0 {
		return 0
	}
	return value - value%2
}
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func normalizeHDRMode(value domain.HDRMode) domain.HDRMode {
	switch value {
	case domain.HDRModeHDR, domain.HDRModeSDR:
		return value
	default:
		return domain.HDRModeAuto
	}
}

type PlaybackOption func(*PlaybackService)

func WithPlaybackProbe(probe ports.MediaProbe) PlaybackOption {
	return func(s *PlaybackService) { s.probe = probe }
}
func WithPlaybackSubtitles(subtitles *SubtitleService) PlaybackOption {
	return func(s *PlaybackService) { s.subtitles = subtitles }
}

type PlaybackService struct {
	catalog    ports.Catalog
	planner    *PlaybackPlanner
	transcoder ports.TranscodeSessionManager
	probe      ports.MediaProbe
	subtitles  *SubtitleService
}

func NewPlaybackService(catalog ports.Catalog, planner *PlaybackPlanner, transcoder ports.TranscodeSessionManager, options ...PlaybackOption) *PlaybackService {
	service := &PlaybackService{catalog: catalog, planner: planner, transcoder: transcoder}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *PlaybackService) Options(ctx context.Context, mediaID string) (domain.PlaybackOptions, error) {
	item, err := s.media(ctx, mediaID)
	if err != nil {
		return domain.PlaybackOptions{}, err
	}
	audio := make([]domain.MediaTrack, 0)
	for _, track := range item.Technical.Tracks {
		if track.Kind == "audio" {
			audio = append(audio, track)
		}
	}
	subtitles := []domain.SubtitleTrack{}
	if s.subtitles != nil {
		subtitles, err = s.subtitles.Discover(ctx, item)
		if err != nil {
			return domain.PlaybackOptions{}, err
		}
	}
	return domain.PlaybackOptions{MediaID: mediaID, DurationMS: item.Technical.DurationMS, AudioTracks: audio, Subtitles: subtitles}, nil
}

func (s *PlaybackService) Plan(ctx context.Context, mediaID string, capabilities domain.ClientCapabilities, preferences domain.PlaybackPreferences) (domain.PlaybackPlan, error) {
	item, err := s.media(ctx, mediaID)
	if err != nil {
		return domain.PlaybackPlan{}, err
	}
	subtitle, err := s.resolveSubtitle(ctx, item, preferences.SubtitleID)
	if err != nil {
		return domain.PlaybackPlan{}, err
	}
	return s.planner.Plan(item, capabilities, preferences, subtitle), nil
}

func (s *PlaybackService) Start(ctx context.Context, mediaID string, capabilities domain.ClientCapabilities, preferences domain.PlaybackPreferences) (domain.PlaybackSession, error) {
	item, err := s.media(ctx, mediaID)
	if err != nil {
		return domain.PlaybackSession{}, err
	}
	subtitle, err := s.resolveSubtitle(ctx, item, preferences.SubtitleID)
	if err != nil {
		return domain.PlaybackSession{}, err
	}
	plan := s.planner.Plan(item, capabilities, preferences, subtitle)
	if plan.Mode == domain.PlaybackModeDirect {
		return domain.PlaybackSession{MediaID: mediaID, Plan: plan}, nil
	}
	if s.transcoder == nil || !s.transcoder.Available(ctx) {
		return domain.PlaybackSession{}, fmt.Errorf("%w: FFmpeg pole saadaval; kontrolli Seaded → Süsteemi olek", domain.ErrUnavailable)
	}
	return s.transcoder.Start(ctx, ports.TranscodeStartRequest{Media: item, Plan: plan, Preferences: preferences, Subtitle: subtitle})
}

func (s *PlaybackService) Stop(sessionID string) error {
	if s.transcoder == nil || strings.TrimSpace(sessionID) == "" {
		return domain.ErrNotFound
	}
	return s.transcoder.Stop(sessionID)
}

func (s *PlaybackService) media(ctx context.Context, mediaID string) (domain.MediaItem, error) {
	item, err := s.catalog.GetMedia(ctx, mediaID)
	if err != nil {
		return domain.MediaItem{}, err
	}
	if item.Type == domain.MediaTypeSeries || strings.TrimSpace(item.File.Path) == "" {
		return domain.MediaItem{}, fmt.Errorf("%w: sellel kirjel puudub esitatav fail", domain.ErrUnsupported)
	}
	if s.probe != nil && playbackTechnicalDataNeedsProbe(item) {
		probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		result, probeErr := s.probe.Probe(probeCtx, item.File.Path)
		cancel()
		if probeErr == nil {
			item.Technical = result.Technical
			item.UpdatedAt = time.Now().UTC()
			_ = s.catalog.UpdateMedia(ctx, item)
		}
	}
	return item, nil
}

func playbackTechnicalDataNeedsProbe(item domain.MediaItem) bool {
	if item.Technical.ProbeVersion < 2 || item.Technical.Container == "" || len(item.Technical.Tracks) == 0 || item.Technical.DurationMS <= 0 {
		return true
	}
	// Old catalogs could contain a short, growing HLS duration instead of the
	// source duration. Re-probe large movie/episode files when the stored value
	// is implausibly short; FFprobe is authoritative and the corrected value is
	// persisted for subsequent starts.
	const largeMedia = int64(100 << 20)
	const suspiciousDuration = int64(2 * time.Minute / time.Millisecond)
	return (item.Type == domain.MediaTypeMovie || item.Type == domain.MediaTypeEpisode) &&
		item.File.SizeBytes >= largeMedia && item.Technical.DurationMS < suspiciousDuration
}

func (s *PlaybackService) resolveSubtitle(ctx context.Context, item domain.MediaItem, id string) (*domain.SubtitleTrack, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, nil
	}
	if s.subtitles == nil {
		return nil, fmt.Errorf("%w: subtiitrite teenus pole saadaval", domain.ErrUnavailable)
	}
	track, err := s.subtitles.Resolve(ctx, item, id)
	if err != nil {
		return nil, err
	}
	return &track, nil
}

func supportsVideoDecode(technical domain.TechnicalMetadata, caps domain.ClientCapabilities) bool {
	codec := strings.ToLower(strings.TrimSpace(technical.VideoCodec))
	if codec == "" {
		return true
	}
	if len(caps.DecodeCapabilities) == 0 {
		return contains(caps.VideoCodecs, codec)
	}
	hasSourceProbe := false
	for _, capability := range caps.DecodeCapabilities {
		if capability.Source && strings.EqualFold(capability.Codec, codec) {
			hasSourceProbe = true
			break
		}
	}
	videoBitrate := technical.Bitrate
	for _, track := range technical.Tracks {
		if track.Kind == "video" && track.Bitrate > 0 {
			videoBitrate = track.Bitrate
			break
		}
	}
	for _, capability := range caps.DecodeCapabilities {
		if hasSourceProbe && !capability.Source {
			continue
		}
		if !strings.EqualFold(capability.Codec, codec) || !capability.Supported || !capability.Smooth {
			continue
		}
		if capability.Width > 0 && technical.Width > capability.Width {
			continue
		}
		if capability.Height > 0 && technical.Height > capability.Height {
			continue
		}
		if capability.FrameRate > 0 && technical.FrameRate > capability.FrameRate+0.1 {
			continue
		}
		if capability.Bitrate > 0 && videoBitrate > capability.Bitrate {
			continue
		}
		if capability.BitDepth > 0 && technical.BitDepth > capability.BitDepth {
			continue
		}
		if technical.HDRFormat != "" && capability.HDRFormat == "" {
			continue
		}
		return true
	}
	return false
}

func supportsOutputCodec(caps domain.ClientCapabilities, codec string, width, height, bitDepth int, hdr string) bool {
	codec = strings.ToLower(strings.TrimSpace(codec))
	if codec == "" {
		return false
	}
	if len(caps.DecodeCapabilities) == 0 {
		return contains(caps.VideoCodecs, codec)
	}
	hasOutputProbe := false
	for _, capability := range caps.DecodeCapabilities {
		if !capability.Source && strings.EqualFold(capability.Codec, codec) {
			hasOutputProbe = true
			break
		}
	}
	for _, capability := range caps.DecodeCapabilities {
		if hasOutputProbe && capability.Source {
			continue
		}
		if !strings.EqualFold(capability.Codec, codec) || !capability.Supported || !capability.Smooth {
			continue
		}
		if capability.Width > 0 && width > capability.Width {
			continue
		}
		if capability.Height > 0 && height > capability.Height {
			continue
		}
		if capability.BitDepth > 0 && bitDepth > capability.BitDepth {
			continue
		}
		if hdr != "" && capability.HDRFormat == "" {
			continue
		}
		return true
	}
	return false
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(expected)) {
			return true
		}
	}
	return false
}

func supportsContainer(mimes []string, container string, hasVideo bool) bool {
	container = strings.ToLower(strings.TrimSpace(container))
	accepted := map[string][]string{
		"mp4": {"video/mp4", "audio/mp4"}, "webm": {"video/webm", "audio/webm"},
		"mp3": {"audio/mpeg"}, "flac": {"audio/flac"}, "ogg": {"audio/ogg", "video/ogg"}, "wav": {"audio/wav"},
	}[container]
	if len(accepted) == 0 {
		return false
	}
	for _, mime := range mimes {
		normalized := strings.ToLower(strings.TrimSpace(strings.Split(mime, ";")[0]))
		for _, candidate := range accepted {
			if normalized == candidate && (!hasVideo || strings.HasPrefix(candidate, "video/")) {
				return true
			}
		}
	}
	return false
}

func safeDirectContainer(container string, hasVideo bool) bool {
	container = strings.ToLower(strings.TrimSpace(container))
	if hasVideo {
		return container == "mp4" || container == "webm"
	}
	switch container {
	case "mp4", "webm", "mp3", "flac", "ogg", "wav":
		return true
	}
	return false
}

func withinClientLimits(technical domain.TechnicalMetadata, audioChannels int, caps domain.ClientCapabilities) bool {
	return withinVideoLimits(technical, caps) && withinAudioLimits(audioChannels, caps) && (caps.MaximumBitrate <= 0 || technical.Bitrate <= 0 || technical.Bitrate <= caps.MaximumBitrate)
}
func withinVideoLimits(technical domain.TechnicalMetadata, caps domain.ClientCapabilities) bool {
	if caps.MaximumWidth > 0 && technical.Width > caps.MaximumWidth {
		return false
	}
	if caps.MaximumHeight > 0 && technical.Height > caps.MaximumHeight {
		return false
	}
	return true
}
func withinAudioLimits(channels int, caps domain.ClientCapabilities) bool {
	return caps.MaximumAudioChannels <= 0 || channels <= 0 || channels <= caps.MaximumAudioChannels
}

func selectedAudioTrack(tracks []domain.MediaTrack, selected *int) *domain.MediaTrack {
	var first, defaultTrack *domain.MediaTrack
	for index := range tracks {
		if tracks[index].Kind != "audio" {
			continue
		}
		if first == nil {
			first = &tracks[index]
		}
		if tracks[index].Default && defaultTrack == nil {
			defaultTrack = &tracks[index]
		}
		if selected != nil && tracks[index].Index == *selected {
			return &tracks[index]
		}
	}
	if selected != nil {
		return first
	}
	if defaultTrack != nil {
		return defaultTrack
	}
	return first
}
func firstTrack(tracks []domain.MediaTrack, kind string) *domain.MediaTrack {
	for index := range tracks {
		if tracks[index].Kind == kind {
			return &tracks[index]
		}
	}
	return nil
}
func actionForPresentCodec(codec string, action domain.TrackAction) domain.TrackAction {
	if codec == "" {
		return domain.TrackActionDrop
	}
	return action
}
func targetCodec(source string, action domain.TrackAction, fallback string) string {
	if action == domain.TrackActionDrop {
		return ""
	}
	if action == domain.TrackActionCopy {
		return source
	}
	return fallback
}
func subtitleAction(track *domain.SubtitleTrack) domain.TrackAction {
	if track == nil {
		return domain.TrackActionDrop
	}
	if track.RequiresBurnIn {
		return domain.TrackActionBurn
	}
	return domain.TrackActionExternal
}
func subtitleID(track *domain.SubtitleTrack) string {
	if track == nil {
		return ""
	}
	return track.ID
}
func nonNegative(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func transcodeDimensions(technical domain.TechnicalMetadata, caps domain.ClientCapabilities) (int, int) {
	if technical.Width <= 0 || technical.Height <= 0 {
		return 0, 0
	}
	maxWidth, maxHeight := 1920, 1080
	if caps.MaximumWidth > 0 && caps.MaximumWidth < maxWidth {
		maxWidth = caps.MaximumWidth
	}
	if caps.MaximumHeight > 0 && caps.MaximumHeight < maxHeight {
		maxHeight = caps.MaximumHeight
	}
	width, height := technical.Width, technical.Height
	if width <= maxWidth && height <= maxHeight {
		return even(width), even(height)
	}
	ratio := float64(width) / float64(height)
	if float64(maxWidth)/float64(maxHeight) > ratio {
		width, height = int(float64(maxHeight)*ratio), maxHeight
	} else {
		width, height = maxWidth, int(float64(maxWidth)/ratio)
	}
	return even(width), even(height)
}
func even(value int) int {
	if value <= 0 {
		return 0
	}
	if value%2 != 0 {
		return value - 1
	}
	return value
}
