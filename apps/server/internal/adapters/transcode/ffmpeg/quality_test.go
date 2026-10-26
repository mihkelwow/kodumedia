package ffmpeg

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

func TestQualityAdviceAdjustsCRFPerEncoderAndSourceFingerprint(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "movie.mkv")
	if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(source)
	if err != nil {
		t.Fatal(err)
	}
	manager := &Manager{quality: newQualityStore(filepath.Join(root, "quality.json"))}
	request := ports.TranscodeStartRequest{
		Media: domain.MediaItem{File: domain.MediaFile{Path: source}},
		Plan: domain.PlaybackPlan{
			VideoAction:    domain.TrackActionTranscode,
			QualityProfile: "maximum", VideoCodec: "h264", VideoEncoder: "software",
			VideoPreset: "slow", VideoCRF: 18, TargetWidth: 1920, TargetHeight: 1080,
			VMAFEnabled: true, VMAFTarget: 95,
		},
	}
	key := qualityAnalysisKey(request, info)
	if err := manager.quality.put(qualityRecord{Key: key, CRF: 18, VMAF: 89, MeasuredAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	adjusted := manager.applyQualityAdvice(request, info)
	if adjusted.Plan.VideoCRF >= request.Plan.VideoCRF {
		t.Fatalf("low VMAF did not improve CRF: before=%d after=%d", request.Plan.VideoCRF, adjusted.Plan.VideoCRF)
	}

	hardware := request
	hardware.Plan.VideoEncoder = "qsv"
	if got := manager.applyQualityAdvice(hardware, info).Plan.VideoCRF; got != hardware.Plan.VideoCRF {
		t.Fatalf("software VMAF record leaked into another encoder: got CRF %d", got)
	}
}

func TestParseVMAFReadsPooledMean(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vmaf.json")
	if err := os.WriteFile(path, []byte(`{"pooled_metrics":{"vmaf":{"mean":96.75}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	value, err := parseVMAF(path)
	if err != nil {
		t.Fatal(err)
	}
	if value != 96.75 {
		t.Fatalf("VMAF = %v; want 96.75", value)
	}
}
