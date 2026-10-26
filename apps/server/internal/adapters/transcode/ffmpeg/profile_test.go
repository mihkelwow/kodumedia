package ffmpeg

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

func TestSoftwareProfileCopiesCompatibleTracksWithoutVideoFilter(t *testing.T) {
	request := ports.TranscodeStartRequest{
		Media: domain.MediaItem{File: domain.MediaFile{Path: `C:\Media\movie.mkv`}},
		Plan: domain.PlaybackPlan{
			VideoAction: domain.TrackActionCopy,
			AudioAction: domain.TrackActionCopy,
		},
	}
	args := (SoftwareH264AACProfile{}).Arguments(request, t.TempDir(), RuntimeCapabilities{Readrate: true, ReadrateInitialBurst: true})
	if slices.Contains(args, "-vf") || slices.Contains(args, "-filter_complex") {
		t.Fatalf("copy profile must not add video filters: %v", args)
	}
	assertArgPair(t, args, "-c:v", "copy")
	assertArgPair(t, args, "-c:a", "copy")
}

func TestSoftwareProfileTranscodesSelectedAudioAndToneMapsHDR(t *testing.T) {
	selected := 3
	request := ports.TranscodeStartRequest{
		Media: domain.MediaItem{File: domain.MediaFile{Path: `C:\Media\movie.mkv`}},
		Plan: domain.PlaybackPlan{
			VideoAction:             domain.TrackActionTranscode,
			AudioAction:             domain.TrackActionTranscode,
			SelectedAudioTrackIndex: &selected,
			ToneMap:                 true,
			TargetWidth:             1920,
			TargetHeight:            1080,
		},
	}
	args := (SoftwareH264AACProfile{}).Arguments(request, t.TempDir(), RuntimeCapabilities{Readrate: true, ReadrateInitialBurst: true})
	assertArgPair(t, args, "-map", "0:3?")
	assertArgPair(t, args, "-c:v", "libx264")
	assertArgPair(t, args, "-c:a", "aac")
	assertArgPair(t, args, "-af", "aresample=async=1:first_pts=0")
	filter := valueAfter(args, "-vf")
	if !strings.Contains(filter, "tonemap=mobius") || !strings.Contains(filter, "zscale=t=linear:npl=100:w=1920:h=1080:filter=lanczos") || !strings.Contains(filter, "setpts=PTS-STARTPTS") {
		t.Fatalf("expected HDR tone-map with early target-resolution scaling, got %q", filter)
	}
}

func TestProfileUsesCUDAHardwareDecodeBeforeCPUToneMap(t *testing.T) {
	request := ports.TranscodeStartRequest{
		Media: domain.MediaItem{
			File:      domain.MediaFile{Path: `C:\Media\hdr.mkv`},
			Technical: domain.TechnicalMetadata{VideoCodec: "hevc", BitDepth: 10, HDRFormat: "hdr10"},
		},
		Plan: domain.PlaybackPlan{
			VideoAction:    domain.TrackActionTranscode,
			AudioAction:    domain.TrackActionTranscode,
			VideoCodec:     "h264",
			VideoEncoder:   "nvenc",
			VideoDecoder:   "cuda",
			ToneMap:        true,
			TargetWidth:    1920,
			TargetHeight:   1080,
			PixelFormat:    "yuv420p",
			QualityProfile: "balanced",
		},
	}
	args := (SoftwareH264AACProfile{}).Arguments(request, t.TempDir(), RuntimeCapabilities{Readrate: true, CUDA: true, H264NVENC: true})
	assertArgPair(t, args, "-hwaccel", "cuda")
	assertArgPair(t, args, "-hwaccel_output_format", "cuda")
	assertArgPair(t, args, "-c:v", "h264_nvenc")
	filter := valueAfter(args, "-vf")
	if !strings.HasPrefix(filter, "hwdownload,format=p010le,") {
		t.Fatalf("expected CUDA frames to be downloaded as 10-bit before CPU filters, got %q", filter)
	}
	if !strings.Contains(filter, "zscale=t=linear:npl=100:w=1920:h=1080:filter=lanczos") || !strings.Contains(filter, "tonemap=mobius") {
		t.Fatalf("expected early-scale HDR tone mapping, got %q", filter)
	}
}

func TestSoftwareProfileBurnsImageSubtitleThroughComplexFilter(t *testing.T) {
	request := ports.TranscodeStartRequest{
		Media: domain.MediaItem{File: domain.MediaFile{Path: `C:\Media\movie.mkv`}},
		Plan: domain.PlaybackPlan{
			VideoAction:     domain.TrackActionTranscode,
			AudioAction:     domain.TrackActionTranscode,
			SubtitleAction:  domain.TrackActionBurn,
			StartPositionMS: 65_500,
		},
		Subtitle: &domain.SubtitleTrack{
			Source:         domain.SubtitleSourceExternal,
			Kind:           domain.SubtitleKindImage,
			Path:           `C:\Media\movie.sup`,
			RequiresBurnIn: true,
		},
	}
	args := (SoftwareH264AACProfile{}).Arguments(request, t.TempDir(), RuntimeCapabilities{Readrate: true, ReadrateInitialBurst: true})
	filterIndex := slices.Index(args, "-filter_complex")
	mapIndex := slices.Index(args, "-map")
	if filterIndex < 0 || mapIndex < 0 || filterIndex > mapIndex {
		t.Fatalf("complex filter must be declared before its map: %v", args)
	}
	if filter := valueAfter(args, "-filter_complex"); !strings.Contains(filter, "[1:s:0]overlay") {
		t.Fatalf("expected external subtitle overlay, got %q", filter)
	}
	if countArgPair(args, "-ss", "65.500") != 2 {
		t.Fatalf("both media and external subtitle inputs must seek together: %v", args)
	}
}

func TestSoftwareProfileSupportsAudioOnlyOutput(t *testing.T) {
	request := ports.TranscodeStartRequest{
		Media: domain.MediaItem{File: domain.MediaFile{Path: `C:\Media\album.ape`}},
		Plan:  domain.PlaybackPlan{VideoAction: domain.TrackActionDrop, AudioAction: domain.TrackActionTranscode},
	}
	args := (SoftwareH264AACProfile{}).Arguments(request, t.TempDir(), RuntimeCapabilities{Readrate: true, ReadrateInitialBurst: true})
	if !slices.Contains(args, "-vn") {
		t.Fatalf("audio-only profile must disable video: %v", args)
	}
	assertArgPair(t, args, "-c:a", "aac")
	if got := valueAfter(args, "-hls_segment_filename"); filepath.Base(got) != "segment_%06d.m4s" {
		t.Fatalf("unexpected segment path %q", got)
	}
}

func TestSoftwareProfilePacesInputAndCapsWorkerThreads(t *testing.T) {
	request := ports.TranscodeStartRequest{
		Media: domain.MediaItem{File: domain.MediaFile{Path: `C:\Media\movie.mkv`}},
		Plan: domain.PlaybackPlan{
			VideoAction: domain.TrackActionTranscode,
			AudioAction: domain.TrackActionTranscode,
		},
	}
	args := (SoftwareH264AACProfile{VideoThreads: 6}).Arguments(request, t.TempDir(), RuntimeCapabilities{Readrate: true, ReadrateInitialBurst: true})
	assertArgPair(t, args, "-readrate", "1.30")
	assertArgPair(t, args, "-readrate_initial_burst", "24")
	assertArgPair(t, args, "-threads", "6")
	assertArgPair(t, args, "-threads:v", "6")
	assertArgPair(t, args, "-hls_time", "2")
	if got := valueAfter(args, "-hls_fmp4_init_filename"); got != "init.mp4" {
		t.Fatalf("unexpected init fragment path %q", got)
	}
	if got := valueAfter(args, "-hls_segment_filename"); got != "segment_%06d.m4s" {
		t.Fatalf("segment URI must remain relative, got %q", got)
	}
	if got := args[len(args)-1]; got != "index.m3u8" {
		t.Fatalf("playlist path must remain relative, got %q", got)
	}
}

func TestSoftwareProfileUsesMPEGTSForDLNACast(t *testing.T) {
	request := ports.TranscodeStartRequest{
		Media: domain.MediaItem{File: domain.MediaFile{Path: `C:\Media\movie.mkv`}},
		Plan: domain.PlaybackPlan{
			Container: "mpegts-hls", VideoAction: domain.TrackActionCopy, AudioAction: domain.TrackActionCopy,
		},
	}
	args := (SoftwareH264AACProfile{}).Arguments(request, t.TempDir(), RuntimeCapabilities{Readrate: true, ReadrateInitialBurst: true})
	assertArgPair(t, args, "-hls_segment_type", "mpegts")
	if got := valueAfter(args, "-hls_segment_filename"); got != "segment_%06d.ts" {
		t.Fatalf("DLNA segment URI must use MPEG-TS, got %q", got)
	}
	if slices.Contains(args, "-hls_fmp4_init_filename") {
		t.Fatalf("MPEG-TS output must not reference an fMP4 init segment: %v", args)
	}
}

func TestSoftwareProfileAppliesPlanVBVLimits(t *testing.T) {
	request := ports.TranscodeStartRequest{
		Media: domain.MediaItem{File: domain.MediaFile{Path: `C:\Media\movie.mkv`}},
		Plan: domain.PlaybackPlan{
			VideoAction:       domain.TrackActionTranscode,
			AudioAction:       domain.TrackActionTranscode,
			MaxVideoBitrate:   20_000_000,
			RateControlBuffer: 40_000_000,
		},
	}
	args := (SoftwareH264AACProfile{}).Arguments(request, t.TempDir(), RuntimeCapabilities{Readrate: true})
	assertArgPair(t, args, "-maxrate:v", "20000000")
	assertArgPair(t, args, "-bufsize:v", "40000000")
}

func TestSoftwareProfileOmitsUnsupportedInitialBurst(t *testing.T) {
	request := ports.TranscodeStartRequest{
		Media: domain.MediaItem{File: domain.MediaFile{Path: `C:\Media\movie.mkv`}},
		Plan: domain.PlaybackPlan{
			VideoAction: domain.TrackActionTranscode,
			AudioAction: domain.TrackActionTranscode,
		},
	}
	args := (SoftwareH264AACProfile{}).Arguments(request, t.TempDir(), RuntimeCapabilities{Readrate: true})
	assertArgPair(t, args, "-readrate", "1.30")
	if slices.Contains(args, "-readrate_initial_burst") {
		t.Fatalf("unsupported startup burst option must be omitted: %v", args)
	}
}

func TestSoftwareProfileFallsBackToRealtimeInputForLegacyFFmpeg(t *testing.T) {
	request := ports.TranscodeStartRequest{
		Media: domain.MediaItem{File: domain.MediaFile{Path: `C:\Media\movie.mkv`}},
		Plan: domain.PlaybackPlan{
			VideoAction: domain.TrackActionTranscode,
			AudioAction: domain.TrackActionTranscode,
		},
	}
	args := (SoftwareH264AACProfile{}).Arguments(request, t.TempDir(), RuntimeCapabilities{})
	if !slices.Contains(args, "-re") {
		t.Fatalf("legacy ffmpeg must use -re compatibility pacing: %v", args)
	}
	if slices.Contains(args, "-readrate") || slices.Contains(args, "-readrate_initial_burst") {
		t.Fatalf("legacy ffmpeg must not receive unsupported readrate options: %v", args)
	}
}

func assertArgPair(t *testing.T, args []string, key, value string) {
	t.Helper()
	if countArgPair(args, key, value) == 0 {
		t.Fatalf("missing argument pair %q %q in %v", key, value, args)
	}
}

func countArgPair(args []string, key, value string) int {
	count := 0
	for index := 0; index+1 < len(args); index++ {
		if args[index] == key && args[index+1] == value {
			count++
		}
	}
	return count
}

func valueAfter(args []string, key string) string {
	for index := 0; index+1 < len(args); index++ {
		if args[index] == key {
			return args[index+1]
		}
	}
	return ""
}

func TestSoftwareProfileTagsCopiedHEVCAsHVC1(t *testing.T) {
	request := ports.TranscodeStartRequest{
		Media: domain.MediaItem{File: domain.MediaFile{Path: `C:\Media\hdr.mkv`}},
		Plan: domain.PlaybackPlan{
			VideoCodec:  "hevc",
			VideoAction: domain.TrackActionCopy,
			AudioAction: domain.TrackActionTranscode,
			PreserveHDR: true,
		},
	}
	args := (SoftwareH264AACProfile{}).Arguments(request, t.TempDir(), RuntimeCapabilities{Readrate: true})
	assertArgPair(t, args, "-c:v", "copy")
	assertArgPair(t, args, "-tag:v", "hvc1")
}

func TestHEVCMain10ProfileIsExplicitForSoftwareAndHardwareEncoders(t *testing.T) {
	base := domain.PlaybackPlan{
		VideoAction:    domain.TrackActionTranscode,
		VideoCodec:     "hevc",
		QualityProfile: "original",
		VideoCRF:       17,
		PixelFormat:    "yuv420p10le",
	}

	software := base
	software.VideoEncoder = "software"
	encoder, args := selectVideoEncoder(software, RuntimeCapabilities{LibX265: true})
	if encoder != "libx265" {
		t.Fatalf("software encoder = %q; want libx265", encoder)
	}
	assertArgPair(t, args, "-profile:v", "main10")

	for _, test := range []struct {
		name       string
		family     string
		capability RuntimeCapabilities
		encoder    string
	}{
		{name: "qsv", family: "qsv", capability: RuntimeCapabilities{HEVCQSV: true}, encoder: "hevc_qsv"},
		{name: "nvenc", family: "nvenc", capability: RuntimeCapabilities{HEVCNVENC: true}, encoder: "hevc_nvenc"},
		{name: "amf", family: "amf", capability: RuntimeCapabilities{HEVCAMF: true}, encoder: "hevc_amf"},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan := base
			plan.VideoEncoder = test.family
			plan.PixelFormat = "p010le"
			encoder, args := selectVideoEncoder(plan, test.capability)
			if encoder != test.encoder {
				t.Fatalf("encoder = %q; want %q", encoder, test.encoder)
			}
			assertArgPair(t, args, "-profile:v", "main10")
		})
	}
}

func TestHardwareEncoderQualityProfilesUseFamilySpecificScales(t *testing.T) {
	tests := []struct {
		name       string
		family     string
		codec      string
		profile    string
		crf        int
		capability RuntimeCapabilities
		presetKey  string
		wantPreset string
		qualityKey string
		wantValue  string
	}{
		{name: "qsv maximum", family: "qsv", codec: "h264", profile: "maximum", crf: 16, capability: RuntimeCapabilities{H264QSV: true}, presetKey: "-preset", wantPreset: "slow", qualityKey: "-global_quality", wantValue: "16"},
		{name: "nvenc original", family: "nvenc", codec: "hevc", profile: "original", crf: 17, capability: RuntimeCapabilities{HEVCNVENC: true}, presetKey: "-preset", wantPreset: "p6", qualityKey: "-cq", wantValue: "17"},
		{name: "amf balanced", family: "amf", codec: "h264", profile: "balanced", crf: 18, capability: RuntimeCapabilities{H264AMF: true}, presetKey: "-quality", wantPreset: "balanced", qualityKey: "-qp_i", wantValue: "20"},
		{name: "nvenc data saver", family: "nvenc", codec: "h264", profile: "data-saver", crf: 23, capability: RuntimeCapabilities{H264NVENC: true}, presetKey: "-preset", wantPreset: "p4", qualityKey: "-cq", wantValue: "23"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := domain.PlaybackPlan{
				VideoCodec:     test.codec,
				VideoEncoder:   test.family,
				QualityProfile: test.profile,
				VideoCRF:       test.crf,
				PixelFormat:    "yuv420p",
			}
			_, args := selectVideoEncoder(plan, test.capability)
			assertArgPair(t, args, test.presetKey, test.wantPreset)
			assertArgPair(t, args, test.qualityKey, test.wantValue)
		})
	}
}
