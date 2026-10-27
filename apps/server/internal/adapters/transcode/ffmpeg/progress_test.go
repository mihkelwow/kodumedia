package ffmpeg

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

func TestRecentProductionSpeedIgnoresMisleadingLifetimeAverage(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	speed, known := recentProductionSpeed(60_000_000, start, 65_000_000, start.Add(2*time.Second))
	if !known || speed != 2.5 {
		t.Fatalf("recent speed = %.2fx, known=%v; want 2.50x", speed, known)
	}
	if _, known := recentProductionSpeed(0, time.Time{}, 65_000_000, start); known {
		t.Fatal("first sample must not invent a recent production speed")
	}
}

func TestUnsustainableSpeedUsesPlanSafetyMargin(t *testing.T) {
	plan := domain.PlaybackPlan{MinimumEncoderSpeed: 1.25}
	if !isUnsustainableSpeed(plan, 1.19, true) {
		t.Fatal("expected 1.19x to be rejected below a 1.25x safety margin")
	}
	if isUnsustainableSpeed(plan, 1.25, true) {
		t.Fatal("did not expect speed at the safety margin to be rejected")
	}
	if isUnsustainableSpeed(plan, 0, false) {
		t.Fatal("unknown speed must not be treated as a confirmed failure")
	}
}

func TestGeneratedBitrateCountsEachPublishedSegmentOnlyOnce(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "init.mp4"), make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 2; index++ {
		name := filepath.Join(directory, fmt.Sprintf("segment_%06d.m4s", index))
		if err := os.WriteFile(name, make([]byte, 1<<20), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	state := playlistCacheState{SegmentCount: 2, NextNumber: 2, DurationMS: 4_000}
	bytes, next, initMeasured := measureGeneratedBytes(directory, state, 0, -1, false)
	if next != 2 || !initMeasured {
		t.Fatalf("unexpected measurement cursor: next=%d init=%v", next, initMeasured)
	}
	secondBytes, secondNext, _ := measureGeneratedBytes(directory, state, bytes, next, initMeasured)
	if secondBytes != bytes || secondNext != next {
		t.Fatalf("published segments were counted twice: first=%d second=%d", bytes, secondBytes)
	}
	want := bytes * 8 * 1000 / state.DurationMS
	if got := bitrateForBytes(bytes, state.DurationMS); got != want {
		t.Fatalf("estimated bitrate = %d; want %d", got, want)
	}
}

func TestReadEncoderProgressUsesLatestProgressValues(t *testing.T) {
	directory := t.TempDir()
	content := "out_time_us=2000000\nspeed=1.10x\nprogress=continue\nout_time_us=6000000\nspeed=1.34x\nprogress=end\n"
	if err := os.WriteFile(filepath.Join(directory, "progress.log"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got := readEncoderProgress(directory)
	if got.OutTimeUS != 6_000_000 || got.Speed != 1.34 || !got.Ended {
		t.Fatalf("unexpected progress: %+v", got)
	}
}

func TestFallbackPlansRemainSubjectToEncoderSpeedSafetyMargin(t *testing.T) {
	plan := domain.PlaybackPlan{
		VideoAction:         domain.TrackActionTranscode,
		EncoderFallback:     true,
		MinimumEncoderSpeed: 1.15,
	}
	if !isUnsustainableSpeed(plan, 0.81, true) {
		t.Fatal("fallback transcode below the safety margin must be rejected")
	}
}
