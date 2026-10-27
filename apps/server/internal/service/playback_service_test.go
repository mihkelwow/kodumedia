package service

import (
	"strings"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestPlaybackPlannerDirectPlay(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{
		ID:        "media-1",
		Type:      domain.MediaTypeMovie,
		Technical: domain.TechnicalMetadata{Container: "mp4", VideoCodec: "h264", AudioCodec: "aac", Width: 1920, Height: 1080},
	}
	capabilities := domain.ClientCapabilities{
		SupportedMIMEs: []string{"video/mp4; codecs=avc1.42E01E,mp4a.40.2"},
		VideoCodecs:    []string{"h264"},
		AudioCodecs:    []string{"aac"},
		SupportsHLS:    true,
	}
	plan := planner.Plan(item, capabilities, domain.PlaybackPreferences{}, nil)
	if plan.Mode != domain.PlaybackModeDirect {
		t.Fatalf("expected direct play, got %s", plan.Mode)
	}
}

func TestPlaybackPlannerRemuxesCompatibleTracks(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{
		ID:        "media-1",
		Type:      domain.MediaTypeMovie,
		Technical: domain.TechnicalMetadata{Container: "mkv", VideoCodec: "h264", AudioCodec: "aac"},
	}
	capabilities := domain.ClientCapabilities{
		SupportedMIMEs: []string{"video/mp4"},
		VideoCodecs:    []string{"h264"},
		AudioCodecs:    []string{"aac"},
		SupportsHLS:    true,
	}
	plan := planner.Plan(item, capabilities, domain.PlaybackPreferences{}, nil)
	if plan.Mode != domain.PlaybackModeRemux {
		t.Fatalf("expected remux, got %s", plan.Mode)
	}
	if plan.VideoAction != domain.TrackActionCopy || plan.AudioAction != domain.TrackActionCopy {
		t.Fatalf("expected both tracks copied, got video=%s audio=%s", plan.VideoAction, plan.AudioAction)
	}
}

func TestPlaybackPlannerTranscodesUnsupportedCodec(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{
		ID:        "media-1",
		Type:      domain.MediaTypeMovie,
		Technical: domain.TechnicalMetadata{Container: "mkv", VideoCodec: "hevc", AudioCodec: "eac3"},
	}
	capabilities := domain.ClientCapabilities{
		SupportedMIMEs: []string{"video/mp4"},
		VideoCodecs:    []string{"h264"},
		AudioCodecs:    []string{"aac"},
		SupportsHLS:    true,
	}
	plan := planner.Plan(item, capabilities, domain.PlaybackPreferences{}, nil)
	if plan.Mode != domain.PlaybackModeTranscode {
		t.Fatalf("expected transcode, got %s", plan.Mode)
	}
	if plan.VideoAction != domain.TrackActionTranscode || plan.AudioAction != domain.TrackActionTranscode {
		t.Fatalf("expected both tracks transcoded, got video=%s audio=%s", plan.VideoAction, plan.AudioAction)
	}
}

func TestPlaybackPlannerToneMapsHDRWhenClientDoesNotAdvertiseIt(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{
		ID: "hdr-1",
		Technical: domain.TechnicalMetadata{
			Container: "mkv", VideoCodec: "hevc", AudioCodec: "eac3", HDRFormat: "hdr10", Width: 3840, Height: 2160,
		},
	}
	plan := planner.Plan(item, domain.ClientCapabilities{VideoCodecs: []string{"h264"}, AudioCodecs: []string{"aac"}, MaximumWidth: 1920, MaximumHeight: 1080}, domain.PlaybackPreferences{}, nil)
	if plan.Mode != domain.PlaybackModeTranscode || !plan.ToneMap {
		t.Fatalf("expected HDR transcode with tone mapping, got %+v", plan)
	}
	if plan.TargetWidth != 1920 || plan.TargetHeight != 1080 {
		t.Fatalf("expected 1080p software fallback, got %dx%d", plan.TargetWidth, plan.TargetHeight)
	}
}

func TestPlaybackPlannerKeepsSupportedHDRAsDirectPlay(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{ID: "hdr-2", Technical: domain.TechnicalMetadata{Container: "mp4", VideoCodec: "h264", AudioCodec: "aac", HDRFormat: "hlg"}}
	caps := domain.ClientCapabilities{SupportedMIMEs: []string{"video/mp4"}, VideoCodecs: []string{"h264"}, AudioCodecs: []string{"aac"}, HDRFormats: []string{"hlg"}}
	plan := planner.Plan(item, caps, domain.PlaybackPreferences{}, nil)
	if plan.Mode != domain.PlaybackModeDirect || plan.ToneMap {
		t.Fatalf("expected supported HLG direct play, got %+v", plan)
	}
}

func TestPlaybackPlannerBurnsImageSubtitle(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{ID: "sub-1", Technical: domain.TechnicalMetadata{Container: "mp4", VideoCodec: "h264", AudioCodec: "aac"}}
	subtitle := &domain.SubtitleTrack{ID: "pgs", Kind: domain.SubtitleKindImage, RequiresBurnIn: true}
	plan := planner.Plan(item, domain.ClientCapabilities{SupportedMIMEs: []string{"video/mp4"}, VideoCodecs: []string{"h264"}, AudioCodecs: []string{"aac"}}, domain.PlaybackPreferences{SubtitleID: subtitle.ID}, subtitle)
	if plan.Mode != domain.PlaybackModeTranscode || plan.SubtitleAction != domain.TrackActionBurn || plan.VideoAction != domain.TrackActionTranscode {
		t.Fatalf("expected image subtitle burn-in, got %+v", plan)
	}
}

func TestPlaybackPlannerCanForceBurnTextSubtitleForTelevisionCast(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{ID: "sub-cast-1", Technical: domain.TechnicalMetadata{Container: "mp4", VideoCodec: "h264", AudioCodec: "aac"}}
	subtitle := &domain.SubtitleTrack{ID: "srt-et", Kind: domain.SubtitleKindText, Format: "srt"}
	preferences := domain.PlaybackPreferences{SubtitleID: subtitle.ID, ForceSubtitleBurn: true}
	plan := planner.Plan(item, domain.ClientCapabilities{SupportedMIMEs: []string{"video/mp4"}, VideoCodecs: []string{"h264"}, AudioCodecs: []string{"aac"}}, preferences, subtitle)
	if plan.Mode != domain.PlaybackModeTranscode || plan.SubtitleAction != domain.TrackActionBurn || plan.VideoAction != domain.TrackActionTranscode {
		t.Fatalf("expected forced cast subtitle burn-in, got %+v", plan)
	}
}

func TestPlaybackPlannerHandlesAudioOnlyFile(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{ID: "audio-1", Technical: domain.TechnicalMetadata{Container: "ape", AudioCodec: "ape"}}
	plan := planner.Plan(item, domain.ClientCapabilities{AudioCodecs: []string{"aac"}}, domain.PlaybackPreferences{}, nil)
	if plan.Mode != domain.PlaybackModeTranscode || plan.VideoAction != domain.TrackActionDrop || plan.AudioAction != domain.TrackActionTranscode {
		t.Fatalf("expected audio-only AAC transcode, got %+v", plan)
	}
}

func TestPlaybackPlannerForceTranscodeOverridesCompatibleSource(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{ID: "fallback-1", Technical: domain.TechnicalMetadata{Container: "mp4", VideoCodec: "h264", AudioCodec: "aac"}}
	caps := domain.ClientCapabilities{SupportedMIMEs: []string{"video/mp4"}, VideoCodecs: []string{"h264"}, AudioCodecs: []string{"aac"}}
	plan := planner.Plan(item, caps, domain.PlaybackPreferences{ForceTranscode: true}, nil)
	if plan.Mode != domain.PlaybackModeTranscode || plan.VideoAction != domain.TrackActionTranscode || plan.AudioAction != domain.TrackActionTranscode {
		t.Fatalf("expected complete fallback transcode, got %+v", plan)
	}
}

func TestPlaybackPlannerHDRFallbackReasonDescribesSDRInsteadOfGenericClientRequest(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{
		ID: "hdr-fallback-reason",
		Technical: domain.TechnicalMetadata{
			Container: "mkv", VideoCodec: "hevc", AudioCodec: "eac3", HDRFormat: "hdr10", Width: 3840, Height: 2160,
		},
	}
	caps := domain.ClientCapabilities{
		SupportedMIMEs: []string{"video/mp4"},
		VideoCodecs:    []string{"h264"},
		AudioCodecs:    []string{"aac"},
		MaximumWidth:   3840,
		MaximumHeight:  2160,
	}
	plan := planner.Plan(item, caps, domain.PlaybackPreferences{HDRMode: domain.HDRModeSDR, ForceTranscode: true}, nil)
	if !plan.ToneMap || !strings.Contains(plan.Reason, "SDR fallback") {
		t.Fatalf("expected an explicit SDR fallback reason, got %+v", plan)
	}
}

func TestPlaybackPlannerCarriesAuthoritativeSourceDuration(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{
		ID:   "duration-1",
		Type: domain.MediaTypeMovie,
		Technical: domain.TechnicalMetadata{
			Container: "mkv", VideoCodec: "hevc", AudioCodec: "eac3", DurationMS: 6_543_210,
		},
	}
	plan := planner.Plan(item, domain.ClientCapabilities{VideoCodecs: []string{"h264"}, AudioCodecs: []string{"aac"}}, domain.PlaybackPreferences{}, nil)
	if plan.DurationMS != item.Technical.DurationMS {
		t.Fatalf("plan duration = %d; want %d", plan.DurationMS, item.Technical.DurationMS)
	}
}

func TestPlaybackTechnicalDataNeedsProbeForSuspiciousLargeMovieDuration(t *testing.T) {
	item := domain.MediaItem{
		Type: domain.MediaTypeMovie,
		File: domain.MediaFile{SizeBytes: 2 << 30},
		Technical: domain.TechnicalMetadata{
			ProbeVersion: 2, Container: "mkv", DurationMS: 14_000,
			Tracks: []domain.MediaTrack{{Index: 0, Kind: "video", Codec: "h264"}},
		},
	}
	if !playbackTechnicalDataNeedsProbe(item) {
		t.Fatal("expected large movie with a 14 second cached duration to be reprobed")
	}
	item.Technical.DurationMS = 90 * 60 * 1000
	if playbackTechnicalDataNeedsProbe(item) {
		t.Fatal("did not expect a normal feature-film duration to be reprobed")
	}
}

func TestPlaybackPlannerManualHDRAttemptsHEVCRemuxWhenCapabilityProbeIsConservative(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{
		ID: "hdr-manual",
		Technical: domain.TechnicalMetadata{
			Container: "mkv", VideoCodec: "hevc", AudioCodec: "eac3", HDRFormat: "hdr10", Width: 3840, Height: 2160,
		},
	}
	caps := domain.ClientCapabilities{
		SupportedMIMEs: []string{"video/mp4"},
		VideoCodecs:    []string{"h264"},
		AudioCodecs:    []string{"aac"},
		SupportsHLS:    true,
		MaximumWidth:   3840,
		MaximumHeight:  2160,
	}
	plan := planner.Plan(item, caps, domain.PlaybackPreferences{HDRMode: domain.HDRModeHDR}, nil)
	if plan.Mode != domain.PlaybackModeTranscode && plan.Mode != domain.PlaybackModeRemux {
		t.Fatalf("expected adaptive HDR session, got %+v", plan)
	}
	if plan.VideoAction != domain.TrackActionCopy || plan.ToneMap || !plan.PreserveHDR {
		t.Fatalf("expected HEVC HDR passthrough attempt, got %+v", plan)
	}
}

func TestPlaybackPlannerForceSDRToneMapsEvenWhenClientAdvertisesHDR(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{
		ID: "hdr-sdr",
		Technical: domain.TechnicalMetadata{
			Container: "mp4", VideoCodec: "hevc", AudioCodec: "aac", HDRFormat: "hdr10", Width: 3840, Height: 2160,
		},
	}
	caps := domain.ClientCapabilities{
		SupportedMIMEs: []string{"video/mp4"},
		VideoCodecs:    []string{"hevc"},
		AudioCodecs:    []string{"aac"},
		HDRFormats:     []string{"hdr10"},
		MaximumWidth:   3840,
		MaximumHeight:  2160,
	}
	plan := planner.Plan(item, caps, domain.PlaybackPreferences{HDRMode: domain.HDRModeSDR}, nil)
	if !plan.ToneMap || plan.PreserveHDR || plan.VideoAction != domain.TrackActionTranscode {
		t.Fatalf("expected explicit SDR tone mapping, got %+v", plan)
	}
}

func TestPlaybackPlannerAutoPreservesHEVCHDROnCapableClient(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{
		ID: "hdr-auto",
		Technical: domain.TechnicalMetadata{
			Container: "mkv", VideoCodec: "hevc", AudioCodec: "aac", HDRFormat: "hdr10", Width: 3840, Height: 2160,
		},
	}
	caps := domain.ClientCapabilities{
		SupportedMIMEs: []string{"video/mp4"},
		VideoCodecs:    []string{"hevc"},
		AudioCodecs:    []string{"aac"},
		HDRFormats:     []string{"hdr10"},
		MaximumWidth:   3840,
		MaximumHeight:  2160,
	}
	plan := planner.Plan(item, caps, domain.PlaybackPreferences{HDRMode: domain.HDRModeAuto}, nil)
	if plan.Mode != domain.PlaybackModeRemux || plan.VideoAction != domain.TrackActionCopy || plan.ToneMap || !plan.PreserveHDR {
		t.Fatalf("expected automatic HDR passthrough remux, got %+v", plan)
	}
}

func TestQualityOriginalPreserves4KWhenClientSupportsOutput(t *testing.T) {
	planner := NewPlaybackPlanner(func() domain.QualitySettings {
		return domain.QualitySettings{Profile: "original", PreferredCodec: "auto", HardwarePreference: "auto", CachePolicy: "watched", CacheMaxGB: 50}
	})
	item := domain.MediaItem{ID: "quality-4k", Technical: domain.TechnicalMetadata{Container: "mkv", VideoCodec: "vp9", AudioCodec: "opus", Width: 3840, Height: 2160, BitDepth: 10, FrameRate: 24}}
	caps := domain.ClientCapabilities{
		VideoCodecs: []string{"h264", "hevc", "av1"}, AudioCodecs: []string{"aac"}, MaximumWidth: 3840, MaximumHeight: 2160,
		DecodeCapabilities: []domain.VideoDecodeCapability{
			{Codec: "hevc", Width: 3840, Height: 2160, FrameRate: 24, BitDepth: 10, Supported: true, Smooth: true, PowerEfficient: true},
			{Codec: "h264", Width: 3840, Height: 2160, FrameRate: 24, BitDepth: 8, Supported: true, Smooth: true},
		},
	}
	plan := planner.Plan(item, caps, domain.PlaybackPreferences{}, nil)
	if plan.TargetWidth != 3840 || plan.TargetHeight != 2160 {
		t.Fatalf("original profile unexpectedly reduced 4K source to %dx%d", plan.TargetWidth, plan.TargetHeight)
	}
	if len(plan.VideoCodecCandidates) == 0 || plan.VideoCodecCandidates[0] != "hevc" {
		t.Fatalf("expected HEVC-first 4K candidates, got %#v", plan.VideoCodecCandidates)
	}
}

func TestQualityProfilesApplyOnlyExplicitResolutionLimits(t *testing.T) {
	item := domain.MediaItem{ID: "quality-limits", Technical: domain.TechnicalMetadata{Container: "mkv", VideoCodec: "vp9", AudioCodec: "opus", Width: 3840, Height: 2160, FrameRate: 24}}
	caps := domain.ClientCapabilities{VideoCodecs: []string{"h264"}, AudioCodecs: []string{"aac"}, MaximumWidth: 3840, MaximumHeight: 2160}
	tests := []struct {
		profile string
		width   int
		height  int
	}{
		{profile: "maximum", width: 3840, height: 2160},
		{profile: "balanced", width: 1920, height: 1080},
		{profile: "data-saver", width: 1280, height: 720},
	}
	for _, test := range tests {
		t.Run(test.profile, func(t *testing.T) {
			planner := NewPlaybackPlanner(func() domain.QualitySettings {
				return domain.QualitySettings{Profile: test.profile, PreferredCodec: "auto", HardwarePreference: "auto", CachePolicy: "watched", CacheMaxGB: 50}
			})
			plan := planner.Plan(item, caps, domain.PlaybackPreferences{}, nil)
			if plan.TargetWidth != test.width || plan.TargetHeight != test.height {
				t.Fatalf("%s dimensions = %dx%d; want %dx%d", test.profile, plan.TargetWidth, plan.TargetHeight, test.width, test.height)
			}
		})
	}
}

func TestPlannerCopiesSupportedVideoAndOnlyTranscodesAudio(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{ID: "audio-only-transcode", Technical: domain.TechnicalMetadata{Container: "mkv", VideoCodec: "hevc", AudioCodec: "truehd", Width: 3840, Height: 2160, BitDepth: 10}}
	caps := domain.ClientCapabilities{
		SupportedMIMEs: []string{"video/mp4"}, VideoCodecs: []string{"hevc"}, AudioCodecs: []string{"aac"}, MaximumWidth: 3840, MaximumHeight: 2160,
		DecodeCapabilities: []domain.VideoDecodeCapability{{Codec: "hevc", Source: true, Width: 3840, Height: 2160, BitDepth: 10, Supported: true, Smooth: true, PowerEfficient: true}},
	}
	plan := planner.Plan(item, caps, domain.PlaybackPreferences{}, nil)
	if plan.VideoAction != domain.TrackActionCopy || plan.AudioAction != domain.TrackActionTranscode {
		t.Fatalf("expected original video copy with audio-only transcode, got %+v", plan)
	}
	if plan.TargetWidth != 0 || plan.TargetHeight != 0 {
		t.Fatalf("video-copy plan must not scale source, got %dx%d", plan.TargetWidth, plan.TargetHeight)
	}
}

func TestOutputSelectionFallsBackToSupported1080pInsteadOfUnsupported4K(t *testing.T) {
	planner := NewPlaybackPlanner(func() domain.QualitySettings {
		return domain.QualitySettings{Profile: "original", PreferredCodec: "auto", HardwarePreference: "auto", CachePolicy: "watched", CacheMaxGB: 50}
	})
	item := domain.MediaItem{ID: "quality-client-limit", Technical: domain.TechnicalMetadata{Container: "mkv", VideoCodec: "mpeg2video", AudioCodec: "ac3", Width: 3840, Height: 2160, FrameRate: 24}}
	caps := domain.ClientCapabilities{
		VideoCodecs: []string{"h264"}, AudioCodecs: []string{"aac"}, MaximumWidth: 3840, MaximumHeight: 2160,
		DecodeCapabilities: []domain.VideoDecodeCapability{
			{Codec: "h264", Width: 1920, Height: 1080, FrameRate: 24, BitDepth: 8, Supported: true, Smooth: true},
		},
	}
	plan := planner.Plan(item, caps, domain.PlaybackPreferences{}, nil)
	if plan.TargetWidth != 1920 || plan.TargetHeight != 1080 || plan.VideoCodec != "h264" {
		t.Fatalf("expected supported 1080p H.264 output, got %+v", plan)
	}
}

func TestPlaybackPlannerAppliesLightweightDirectBufferPolicy(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{
		ID: "buffer-direct",
		Technical: domain.TechnicalMetadata{
			Container: "mp4", VideoCodec: "h264", AudioCodec: "aac", Width: 1920, Height: 1080, DurationMS: 7_200_000,
		},
	}
	caps := domain.ClientCapabilities{
		SupportedMIMEs: []string{"video/mp4"}, VideoCodecs: []string{"h264"}, AudioCodecs: []string{"aac"},
	}
	plan := planner.Plan(item, caps, domain.PlaybackPreferences{}, nil)
	if plan.Mode != domain.PlaybackModeDirect {
		t.Fatalf("mode = %s; want direct", plan.Mode)
	}
	if plan.StartupBufferSeconds != 4 || plan.RebufferBufferSeconds != 8 || plan.DesiredBufferSeconds != 20 || plan.LowBufferSeconds != 4 {
		t.Fatalf("unexpected direct buffer policy: %+v", plan)
	}
}

func TestPlaybackPlannerAppliesRemuxBufferPolicy(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{
		ID: "buffer-remux",
		Technical: domain.TechnicalMetadata{
			Container: "mkv", VideoCodec: "h264", AudioCodec: "aac", Width: 1920, Height: 1080, DurationMS: 7_200_000,
		},
	}
	caps := domain.ClientCapabilities{
		SupportedMIMEs: []string{"video/mp4"}, VideoCodecs: []string{"h264"}, AudioCodecs: []string{"aac"},
	}
	plan := planner.Plan(item, caps, domain.PlaybackPreferences{}, nil)
	if plan.Mode != domain.PlaybackModeRemux {
		t.Fatalf("mode = %s; want remux", plan.Mode)
	}
	if plan.StartupBufferSeconds != 20 || plan.RebufferBufferSeconds != 30 || plan.DesiredBufferSeconds != 60 || plan.LowBufferSeconds != 16 || plan.MinimumReadyBufferSeconds != 20 {
		t.Fatalf("unexpected remux buffer policy: %+v", plan)
	}
	if plan.ReadRate != 3.0 || plan.InitialBurstSeconds != 45 {
		t.Fatalf("unexpected remux producer policy: %+v", plan)
	}
}

func TestPlaybackPlannerUsesBufferedMPEGTSForDLNARemux(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{
		ID: "dlna-remux",
		Technical: domain.TechnicalMetadata{
			Container: "mkv", VideoCodec: "h264", AudioCodec: "aac", Width: 1920, Height: 1080, DurationMS: 7_200_000,
		},
	}
	caps := domain.ClientCapabilities{
		ClientName: "Samsung RU7172 DLNA", SupportedMIMEs: []string{"video/mp4", "application/vnd.apple.mpegurl"},
		VideoCodecs: []string{"h264", "hevc"}, AudioCodecs: []string{"aac", "ac3"}, SupportsHLS: true,
		MaximumWidth: 3840, MaximumHeight: 2160,
	}
	plan := planner.Plan(item, caps, domain.PlaybackPreferences{}, nil)
	if plan.Mode != domain.PlaybackModeRemux || plan.Container != "mpegts-hls" {
		t.Fatalf("expected MPEG-TS DLNA remux, got %+v", plan)
	}
	if plan.VideoAction != domain.TrackActionCopy || plan.MinimumReadyBufferSeconds != 20 {
		t.Fatalf("unexpected DLNA remux policy: %+v", plan)
	}
	if plan.CachePolicy != "off" {
		t.Fatalf("DLNA session must not reuse an fMP4 cache: %+v", plan)
	}
}

func TestPlaybackPlannerCapsRequiredDLNATranscodeAt1080pH264(t *testing.T) {
	planner := NewPlaybackPlanner()
	item := domain.MediaItem{
		ID: "dlna-subtitle",
		Technical: domain.TechnicalMetadata{
			Container: "mkv", VideoCodec: "hevc", AudioCodec: "eac3", HDRFormat: "hdr10",
			Width: 3840, Height: 2160, DurationMS: 7_200_000, Bitrate: 50_000_000,
		},
	}
	caps := domain.ClientCapabilities{
		ClientName: "Samsung RU7172 DLNA", SupportedMIMEs: []string{"video/mp4", "application/vnd.apple.mpegurl"},
		VideoCodecs: []string{"h264", "hevc"}, AudioCodecs: []string{"aac", "ac3", "eac3"}, HDRFormats: []string{"hdr10"},
		SupportsHLS: true, MaximumWidth: 3840, MaximumHeight: 2160, MaximumBitrate: 80_000_000,
	}
	subtitle := &domain.SubtitleTrack{ID: "et", Kind: domain.SubtitleKindText, Format: "srt"}
	plan := planner.Plan(item, caps, domain.PlaybackPreferences{SubtitleID: "et", ForceSubtitleBurn: true}, subtitle)
	if plan.Mode != domain.PlaybackModeTranscode || plan.VideoCodec != "h264" || plan.TargetWidth > 1920 || plan.TargetHeight > 1080 {
		t.Fatalf("expected conservative DLNA transcode, got %+v", plan)
	}
	if !plan.ToneMap || plan.PreserveHDR || plan.Container != "mpegts-hls" || plan.MinimumReadyBufferSeconds != 20 {
		t.Fatalf("unexpected DLNA HDR/buffer policy: %+v", plan)
	}
}

func TestPlaybackPlannerApplies1080pTranscodeHeadroomAndVBV(t *testing.T) {
	planner := NewPlaybackPlanner(func() domain.QualitySettings {
		return domain.QualitySettings{Profile: "original", PreferredCodec: "h264", HardwarePreference: "software", CachePolicy: "watched", CacheMaxGB: 50}
	})
	item := domain.MediaItem{
		ID: "buffer-1080",
		Technical: domain.TechnicalMetadata{
			Container: "mkv", VideoCodec: "mpeg2video", AudioCodec: "ac3", Width: 1920, Height: 1080, DurationMS: 7_200_000, Bitrate: 12_000_000,
		},
	}
	caps := domain.ClientCapabilities{VideoCodecs: []string{"h264"}, AudioCodecs: []string{"aac"}, MaximumWidth: 1920, MaximumHeight: 1080}
	plan := planner.Plan(item, caps, domain.PlaybackPreferences{}, nil)
	if plan.VideoAction != domain.TrackActionTranscode {
		t.Fatalf("expected video transcode, got %+v", plan)
	}
	if plan.StartupBufferSeconds != 24 || plan.RebufferBufferSeconds != 36 || plan.DesiredBufferSeconds != 90 || plan.LowBufferSeconds != 24 || plan.MinimumReadyBufferSeconds != 24 {
		t.Fatalf("unexpected 1080p buffer policy: %+v", plan)
	}
	if plan.MinimumEncoderSpeed != 1.20 || plan.ReadRate != 2.50 || plan.InitialBurstSeconds != 60 {
		t.Fatalf("unexpected 1080p encoder headroom: %+v", plan)
	}
	if plan.MaxVideoBitrate <= 0 || plan.RateControlBuffer != plan.MaxVideoBitrate*2 {
		t.Fatalf("expected VBV limits, got max=%d buffer=%d", plan.MaxVideoBitrate, plan.RateControlBuffer)
	}
}

func TestPlaybackPlannerApplies4KTranscodeSafetyPolicy(t *testing.T) {
	planner := NewPlaybackPlanner(func() domain.QualitySettings {
		return domain.QualitySettings{Profile: "original", PreferredCodec: "h264", HardwarePreference: "software", CachePolicy: "watched", CacheMaxGB: 50}
	})
	item := domain.MediaItem{
		ID: "buffer-4k",
		Technical: domain.TechnicalMetadata{
			Container: "mkv", VideoCodec: "mpeg2video", AudioCodec: "ac3", Width: 3840, Height: 2160, DurationMS: 7_200_000, Bitrate: 40_000_000,
		},
	}
	caps := domain.ClientCapabilities{VideoCodecs: []string{"h264"}, AudioCodecs: []string{"aac"}, MaximumWidth: 3840, MaximumHeight: 2160}
	plan := planner.Plan(item, caps, domain.PlaybackPreferences{}, nil)
	if plan.StartupBufferSeconds != 30 || plan.RebufferBufferSeconds != 45 || plan.DesiredBufferSeconds != 120 || plan.LowBufferSeconds != 30 || plan.MinimumReadyBufferSeconds != 30 {
		t.Fatalf("unexpected 4K buffer policy: %+v", plan)
	}
	if plan.MinimumEncoderSpeed != 1.25 || plan.ReadRate != 2.25 || plan.InitialBurstSeconds != 75 {
		t.Fatalf("unexpected 4K encoder headroom: %+v", plan)
	}
}

func TestPlaybackPlannerRecoveryProfilesStepDownPredictably(t *testing.T) {
	planner := NewPlaybackPlanner(func() domain.QualitySettings {
		return domain.QualitySettings{Profile: "original", PreferredCodec: "auto", HardwarePreference: "auto", CachePolicy: "watched", CacheMaxGB: 50}
	})
	item := domain.MediaItem{
		ID: "recovery",
		Technical: domain.TechnicalMetadata{
			Container: "mkv", VideoCodec: "hevc", AudioCodec: "eac3", Width: 3840, Height: 2160, DurationMS: 7_200_000,
		},
	}
	caps := domain.ClientCapabilities{VideoCodecs: []string{"h264"}, AudioCodecs: []string{"aac"}, MaximumWidth: 3840, MaximumHeight: 2160}

	balanced := planner.Plan(item, caps, domain.PlaybackPreferences{RecoveryLevel: 1}, nil)
	if balanced.QualityProfile != "balanced" || balanced.VideoCodec != "h264" || balanced.TargetWidth > 1920 || balanced.TargetHeight > 1080 {
		t.Fatalf("unexpected recovery level 1 plan: %+v", balanced)
	}
	if balanced.RecoveryLevel != 1 || balanced.MinimumEncoderSpeed != 1.15 {
		t.Fatalf("unexpected recovery level 1 policy: %+v", balanced)
	}

	dataSaver := planner.Plan(item, caps, domain.PlaybackPreferences{RecoveryLevel: 2}, nil)
	if dataSaver.QualityProfile != "data-saver" || dataSaver.VideoCodec != "h264" || dataSaver.TargetWidth > 1280 || dataSaver.TargetHeight > 720 {
		t.Fatalf("unexpected recovery level 2 plan: %+v", dataSaver)
	}
	if dataSaver.RecoveryLevel != 2 || dataSaver.MinimumEncoderSpeed != 1.10 {
		t.Fatalf("unexpected recovery level 2 policy: %+v", dataSaver)
	}
}
