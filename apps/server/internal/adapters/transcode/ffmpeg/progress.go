package ffmpeg

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
)

type encoderProgress struct {
	OutTimeUS int64
	Speed     float64
	Ended     bool
}

func readEncoderProgress(directory string) encoderProgress {
	file, err := os.Open(filepath.Join(directory, "progress.log"))
	if err != nil {
		return encoderProgress{}
	}
	defer file.Close()
	result := encoderProgress{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if !ok {
			continue
		}
		switch key {
		case "out_time_us", "out_time_ms":
			if parsed, parseErr := strconv.ParseInt(strings.TrimSpace(value), 10, 64); parseErr == nil {
				result.OutTimeUS = parsed
			}
		case "speed":
			result.Speed = parseSpeed(value)
		case "progress":
			result.Ended = strings.TrimSpace(value) == "end"
		}
	}
	return result
}

func shouldAssessSpeed(plan domain.PlaybackPlan, source domain.TechnicalMetadata) bool {
	if plan.VideoAction != domain.TrackActionTranscode {
		return false
	}
	if source.DurationMS > 0 && source.DurationMS < int64(30*time.Second/time.Millisecond) {
		return false
	}
	return plan.MinimumEncoderSpeed > 0
}

func (m *Manager) assessStartupSpeed(runtime *runtimeSession, plan domain.PlaybackPlan, source domain.TechnicalMetadata) (float64, bool) {
	if !shouldAssessSpeed(plan, source) {
		return 0, false
	}
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		progress := readEncoderProgress(runtime.dir)
		if progress.OutTimeUS >= 4_000_000 && progress.Speed > 0 {
			return progress.Speed, true
		}
		select {
		case <-runtime.done:
			progress = readEncoderProgress(runtime.dir)
			return progress.Speed, progress.Speed > 0
		case <-time.After(125 * time.Millisecond):
		}
	}
	progress := readEncoderProgress(runtime.dir)
	return progress.Speed, progress.Speed > 0
}

func isUnsustainableSpeed(plan domain.PlaybackPlan, speed float64, known bool) bool {
	minimum := plan.MinimumEncoderSpeed
	if minimum <= 0 {
		minimum = 1.15
	}
	return known && speed > 0 && speed < minimum
}

// recentProductionSpeed measures newly produced media time against wall time.
// FFmpeg's reported speed is cumulative from process start, so a very fast
// initial burst can keep that number high long after current production has
// slowed down. Buffer protection needs the recent rate instead.
func recentProductionSpeed(previousOutputUS int64, previousAt time.Time, outputUS int64, sampledAt time.Time) (float64, bool) {
	if previousAt.IsZero() || !sampledAt.After(previousAt) || outputUS < previousOutputUS {
		return 0, false
	}
	elapsed := sampledAt.Sub(previousAt).Seconds()
	if elapsed <= 0 {
		return 0, false
	}
	return (float64(outputUS-previousOutputUS) / 1_000_000) / elapsed, true
}

func (r *runtimeSession) updateStats(update func(*domain.PlaybackRuntimeStats)) {
	r.statsMu.Lock()
	update(&r.stats)
	r.stats.UpdatedAt = time.Now().UTC()
	r.statsMu.Unlock()
}

func (r *runtimeSession) snapshotStats() domain.PlaybackRuntimeStats {
	r.statsMu.RLock()
	defer r.statsMu.RUnlock()
	return r.stats
}

func (m *Manager) monitorRuntime(runtime *runtimeSession) {
	if runtime == nil {
		return
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	const sampleWindow = 8
	samples := make([]float64, 0, sampleWindow)
	belowSince := time.Time{}
	nextMeasuredSegment := -1
	generatedBytes := int64(0)
	initMeasured := false
	previousOutputUS := int64(0)
	previousSampleAt := time.Time{}

	update := func(alive bool) {
		sampledAt := time.Now()
		progress := readEncoderProgress(runtime.dir)
		playlist := inspectPlaylistCache(runtime.dir)
		generatedBytes, nextMeasuredSegment, initMeasured = measureGeneratedBytes(
			runtime.dir,
			playlist,
			generatedBytes,
			nextMeasuredSegment,
			initMeasured,
		)
		recentSpeed, recentKnown := recentProductionSpeed(previousOutputUS, previousSampleAt, progress.OutTimeUS, sampledAt)
		previousOutputUS = progress.OutTimeUS
		previousSampleAt = sampledAt
		if recentKnown {
			samples = append(samples, recentSpeed)
			if len(samples) > sampleWindow {
				samples = samples[len(samples)-sampleWindow:]
			}
		}
		average := 0.0
		for _, sample := range samples {
			average += sample
		}
		if len(samples) > 0 {
			average /= float64(len(samples))
		}

		displayedSpeed := recentSpeed
		if !recentKnown {
			displayedSpeed = progress.Speed
		}
		minimum := runtime.request.Plan.MinimumEncoderSpeed
		fallback := false
		reason := ""
		if alive && minimum > 0 && progress.OutTimeUS >= 8_000_000 && len(samples) > 0 && average < minimum {
			if belowSince.IsZero() {
				belowSince = time.Now()
			}
			if time.Since(belowSince) >= 6*time.Second {
				fallback = true
				reason = fmt.Sprintf("Encoder'i keskmine kiirus %.2fx jäi alla %.2fx turvavaru.", average, minimum)
			}
		} else {
			belowSince = time.Time{}
		}

		bitrate := bitrateForBytes(generatedBytes, playlist.DurationMS)
		runtime.updateStats(func(stats *domain.PlaybackRuntimeStats) {
			stats.SessionID = runtime.session.ID
			stats.State = "running"
			if !alive || progress.Ended {
				stats.State = "completed"
				if runtime.getErr() != nil {
					stats.State = "failed"
				}
			}
			stats.ProducerAlive = alive
			stats.EncoderSpeed = displayedSpeed
			stats.AverageEncoderSpeed = average
			stats.OutputPositionMS = progress.OutTimeUS / 1000
			stats.GeneratedDurationMS = playlist.DurationMS
			stats.SegmentCount = playlist.SegmentCount
			stats.EstimatedBitrate = bitrate
			stats.MinimumEncoderSpeed = minimum
			stats.FallbackRecommended = fallback
			stats.FallbackReason = reason
		})
	}

	update(true)
	for {
		select {
		case <-runtime.done:
			update(false)
			return
		case <-ticker.C:
			update(true)
		}
	}
}

func measureGeneratedBytes(
	directory string,
	state playlistCacheState,
	currentBytes int64,
	nextSegment int,
	initMeasured bool,
) (int64, int, bool) {
	if !initMeasured {
		if info, err := os.Stat(filepath.Join(directory, "init.mp4")); err == nil && info.Size() > 0 {
			currentBytes += info.Size()
			initMeasured = true
		}
	}
	if nextSegment < 0 {
		nextSegment = state.NextNumber - state.SegmentCount
		if nextSegment < 0 {
			nextSegment = 0
		}
	}
	for nextSegment < state.NextNumber {
		path := filepath.Join(directory, fmt.Sprintf("segment_%06d.m4s", nextSegment))
		info, err := os.Stat(path)
		if err != nil || info.Size() <= 0 {
			break
		}
		currentBytes += info.Size()
		nextSegment++
	}
	return currentBytes, nextSegment, initMeasured
}

func bitrateForBytes(bytes, durationMS int64) int64 {
	if bytes <= 0 || durationMS <= 0 {
		return 0
	}
	return bytes * 8 * 1000 / durationMS
}

func (m *Manager) waitForAppendedSegment(ctx context.Context, runtime *runtimeSession, startNumber int) error {
	if startNumber <= 0 {
		return nil
	}
	name := filepath.Join(runtime.dir, fmt.Sprintf("segment_%06d.m4s", startNumber))
	timer := time.NewTimer(8 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(75 * time.Millisecond)
	defer ticker.Stop()
	for {
		if info, err := os.Stat(name); err == nil && info.Size() > 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-runtime.done:
			if runtime.getErr() != nil {
				return m.outputUnavailable(runtime, filepath.Base(name))
			}
			return nil
		case <-timer.C:
			return fmt.Errorf("%w: resumed encoder did not create a new segment", domain.ErrUnavailable)
		case <-ticker.C:
		}
	}
}
