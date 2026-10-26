package ffmpeg

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func testSuccessfulExecutable(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is only used by host-side adapter tests")
	}
	path := filepath.Join(t.TempDir(), "fake-ffmpeg")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestResolveEncoderChoosesHardwareAV1FromAutomaticCandidates(t *testing.T) {
	executable := testSuccessfulExecutable(t)
	manager := &Manager{binary: func() string { return executable }, encoderCache: map[string]encoderProbeCacheEntry{}}
	plan := domain.PlaybackPlan{
		VideoAction:          domain.TrackActionTranscode,
		VideoCodec:           "av1",
		VideoCodecCandidates: []string{"av1", "hevc", "h264"},
		PreferredVideoCodec:  "auto",
		HardwarePreference:   "auto",
		VideoEncoder:         "auto",
		PreserveHDR:          true,
		PixelFormat:          "yuv420p10le",
	}
	resolved := manager.resolveEncoder(context.Background(), plan, domain.TechnicalMetadata{HDRFormat: "hdr10", BitDepth: 10}, RuntimeCapabilities{AV1QSV: true, LibX264: true})
	if resolved.VideoCodec != "av1" || resolved.VideoEncoder != "qsv" {
		t.Fatalf("expected AV1 QSV, got %+v", resolved)
	}
	if resolved.PixelFormat != "p010le" || !resolved.PreserveHDR || resolved.ToneMap {
		t.Fatalf("expected 10-bit HDR hardware output, got %+v", resolved)
	}
}

func TestResolveEncoderUsesSoftwareH264WhenAutomaticHardwareIsUnavailable(t *testing.T) {
	manager := &Manager{encoderCache: map[string]encoderProbeCacheEntry{}}
	plan := domain.PlaybackPlan{
		VideoAction:          domain.TrackActionTranscode,
		VideoCodec:           "hevc",
		VideoCodecCandidates: []string{"hevc", "av1", "h264"},
		PreferredVideoCodec:  "auto",
		HardwarePreference:   "auto",
		VideoEncoder:         "auto",
		PreserveHDR:          true,
	}
	resolved := manager.resolveEncoder(context.Background(), plan, domain.TechnicalMetadata{HDRFormat: "hdr10", BitDepth: 10, Width: 3840, Height: 2160}, RuntimeCapabilities{LibX264: true, LibX265: true, LibSVTAV1: true})
	if resolved.VideoCodec != "h264" || resolved.VideoEncoder != "software" {
		t.Fatalf("expected predictable software H.264 fallback, got %+v", resolved)
	}
	if !resolved.ToneMap || resolved.PreserveHDR || resolved.PixelFormat != "yuv420p" {
		t.Fatalf("HDR must become SDR when automatic software fallback is H.264, got %+v", resolved)
	}
}

func TestResolveEncoderHonorsExplicitSoftwareHEVC(t *testing.T) {
	manager := &Manager{encoderCache: map[string]encoderProbeCacheEntry{}}
	plan := domain.PlaybackPlan{
		VideoAction:          domain.TrackActionTranscode,
		VideoCodec:           "hevc",
		VideoCodecCandidates: []string{"hevc", "h264"},
		PreferredVideoCodec:  "hevc",
		HardwarePreference:   "software",
		VideoEncoder:         "software",
		PixelFormat:          "yuv420p10le",
	}
	resolved := manager.resolveEncoder(context.Background(), plan, domain.TechnicalMetadata{BitDepth: 10}, RuntimeCapabilities{LibX265: true, LibX264: true})
	if resolved.VideoCodec != "hevc" || resolved.VideoEncoder != "software" || resolved.PixelFormat != "yuv420p10le" {
		t.Fatalf("expected explicit software HEVC Main10, got %+v", resolved)
	}
}

func TestResolveEncoderPrefersNVENCForHDRAndUsesCUDAHardwareDecode(t *testing.T) {
	executable := testSuccessfulExecutable(t)
	manager := &Manager{binary: func() string { return executable }, encoderCache: map[string]encoderProbeCacheEntry{}}
	plan := domain.PlaybackPlan{
		VideoAction:          domain.TrackActionTranscode,
		VideoCodec:           "h264",
		VideoCodecCandidates: []string{"h264"},
		PreferredVideoCodec:  "auto",
		HardwarePreference:   "auto",
		VideoEncoder:         "auto",
		ToneMap:              true,
		PixelFormat:          "yuv420p",
	}
	resolved := manager.resolveEncoder(context.Background(), plan, domain.TechnicalMetadata{VideoCodec: "hevc", HDRFormat: "hdr10", BitDepth: 10, Width: 3840, Height: 2160}, RuntimeCapabilities{H264QSV: true, H264NVENC: true, CUDA: true, LibX264: true})
	if resolved.VideoEncoder != "nvenc" {
		t.Fatalf("expected NVENC to be preferred for 4K HDR, got %+v", resolved)
	}
	if resolved.VideoDecoder != "cuda" {
		t.Fatalf("expected CUDA hardware decode, got %+v", resolved)
	}
}

func TestRetryAfterHardwareFailureFirstDisablesDecoderThenExcludesEncoder(t *testing.T) {
	manager := &Manager{encoderCache: map[string]encoderProbeCacheEntry{}}
	plan := domain.PlaybackPlan{
		VideoAction: domain.TrackActionTranscode, VideoCodec: "h264", VideoEncoder: "nvenc", VideoDecoder: "cuda",
		HardwarePreference: "auto", VideoCodecCandidates: []string{"h264"},
	}
	retry, ok := manager.retryAfterEncoderFailure(plan, domain.TechnicalMetadata{VideoCodec: "hevc"}, RuntimeCapabilities{H264NVENC: true}, "decode failed")
	if !ok || retry.VideoDecoder != "" || !retry.DisableHardwareDecode || len(retry.ExcludedVideoEncoders) != 0 {
		t.Fatalf("expected same encoder with software decode retry, got %+v", retry)
	}
	retry.VideoEncoder = "nvenc"
	retry, ok = manager.retryAfterEncoderFailure(retry, domain.TechnicalMetadata{VideoCodec: "hevc"}, RuntimeCapabilities{H264NVENC: true}, "encode failed")
	if !ok || len(retry.ExcludedVideoEncoders) != 1 || retry.ExcludedVideoEncoders[0] != "h264_nvenc" {
		t.Fatalf("expected failed NVENC encoder to be excluded, got %+v", retry)
	}
}

func TestNextStabilityPlanProgressesTo720And480(t *testing.T) {
	source := domain.TechnicalMetadata{Width: 3840, Height: 2160, HDRFormat: "hdr10", Bitrate: 18_000_000}
	base := domain.PlaybackPlan{VideoAction: domain.TrackActionTranscode, HardwarePreference: "auto"}
	balanced, ok := nextStabilityPlan(base, source, "slow")
	if !ok || balanced.RecoveryLevel != 1 || balanced.TargetWidth != 1920 || balanced.TargetHeight != 1080 || balanced.HardwarePreference != "auto" {
		t.Fatalf("unexpected balanced fallback: %+v", balanced)
	}
	dataSaver, ok := nextStabilityPlan(balanced, source, "still slow")
	if !ok || dataSaver.RecoveryLevel != 2 || dataSaver.TargetWidth != 1280 || dataSaver.TargetHeight != 720 {
		t.Fatalf("unexpected 720p fallback: %+v", dataSaver)
	}
	emergency, ok := nextStabilityPlan(dataSaver, source, "still slow")
	if !ok || emergency.RecoveryLevel != 3 || emergency.TargetWidth != 852 || emergency.TargetHeight != 480 || emergency.VideoPreset != "ultrafast" {
		t.Fatalf("unexpected 480p fallback: %+v", emergency)
	}
	if _, ok := nextStabilityPlan(emergency, source, "no more"); ok {
		t.Fatal("did not expect another fallback after recovery level 3")
	}
}
