package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/platform/childprocess"
)

const audioAnalysisCacheSchema = 2

type audioAnalysisCacheEntry struct {
	SchemaVersion int       `json:"schemaVersion"`
	CreatedAt     time.Time `json:"createdAt"`
	BPM           float64   `json:"bpm,omitempty"`
	BeatStrength  float64   `json:"beatStrength,omitempty"`
	ZeroCrossRate float64   `json:"zeroCrossRate,omitempty"`
	Brightness    float64   `json:"brightness,omitempty"`
	DynamicRange  float64   `json:"dynamicRange,omitempty"`
	Genre         string    `json:"genre"`
	Style         string    `json:"style"`
	Mood          string    `json:"mood,omitempty"`
	Confidence    string    `json:"confidence"`
}

func (p *MusicMetadataProvider) analyzeAudioClassification(ctx context.Context, item domain.MediaItem) (inferredMusicClassification, error) {
	key := stableID(
		"music-audio-analysis-v1",
		filepath.Clean(item.File.Path),
		strconv.FormatInt(item.File.SizeBytes, 10),
		item.File.ModifiedAt.UTC().Format(time.RFC3339Nano),
	)
	cachePath := filepath.Join(p.cacheDir, "audio-analysis", key+".json")
	if payload, err := os.ReadFile(cachePath); err == nil {
		var cached audioAnalysisCacheEntry
		if json.Unmarshal(payload, &cached) == nil && cached.SchemaVersion == audioAnalysisCacheSchema && cached.Genre != "" && cached.Style != "" {
			return inferredMusicClassification{
				Genres: []string{cached.Genre}, Styles: []string{cached.Style},
				Moods: optionalMusicValue(cached.Mood), Source: "audio-analysis", Confidence: cached.Confidence,
			}, nil
		}
	}

	binaryPath := "ffmpeg"
	if p.ffmpeg != nil && strings.TrimSpace(p.ffmpeg()) != "" {
		binaryPath = strings.TrimSpace(p.ffmpeg())
	}
	offsetSeconds := 20.0
	if item.Technical.DurationMS > 0 {
		duration := float64(item.Technical.DurationMS) / 1000
		offsetSeconds = math.Max(0, math.Min(duration*0.2, math.Max(0, duration-28)))
	}
	command := exec.CommandContext(
		ctx, binaryPath,
		"-v", "error",
		"-ss", fmt.Sprintf("%.3f", offsetSeconds),
		"-i", item.File.Path,
		"-t", "24",
		"-vn", "-sn", "-dn",
		"-ac", "1", "-ar", "8000",
		"-f", "s16le", "pipe:1",
	)
	childprocess.Hide(command)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	stdout, err := command.StdoutPipe()
	if err != nil {
		return inferredMusicClassification{}, err
	}
	if err := command.Start(); err != nil {
		return inferredMusicClassification{}, fmt.Errorf("start audio analysis: %w", err)
	}
	payload, readErr := io.ReadAll(io.LimitReader(stdout, 8000*24*2+4096))
	waitErr := command.Wait()
	if readErr != nil {
		return inferredMusicClassification{}, readErr
	}
	if waitErr != nil {
		return inferredMusicClassification{}, fmt.Errorf("audio analysis failed: %w: %s", waitErr, strings.TrimSpace(stderr.String()))
	}
	if len(payload) < 8000*4*2 {
		return inferredMusicClassification{}, errors.New("audio analysis sample was too short")
	}

	entry, err := classifyPCM(payload, 8000)
	if err != nil {
		return inferredMusicClassification{}, err
	}
	entry.SchemaVersion = audioAnalysisCacheSchema
	entry.CreatedAt = p.now().UTC()
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err == nil {
		if encoded, marshalErr := json.MarshalIndent(entry, "", "  "); marshalErr == nil {
			_ = os.WriteFile(cachePath, append(encoded, '\n'), 0o600)
		}
	}
	return inferredMusicClassification{
		Genres: []string{entry.Genre}, Styles: []string{entry.Style},
		Moods: optionalMusicValue(entry.Mood), Source: "audio-analysis", Confidence: entry.Confidence,
	}, nil
}

func classifyPCM(payload []byte, sampleRate int) (audioAnalysisCacheEntry, error) {
	count := len(payload) / 2
	if count < sampleRate*4 {
		return audioAnalysisCacheEntry{}, errors.New("not enough PCM samples")
	}
	samples := make([]float64, count)
	var sumAbs, diffAbs float64
	zeroCrossings := 0
	previous := 0.0
	for index := 0; index < count; index++ {
		value := float64(int16(binary.LittleEndian.Uint16(payload[index*2:]))) / 32768
		samples[index] = value
		sumAbs += math.Abs(value)
		if index > 0 {
			diffAbs += math.Abs(value - previous)
			if (value >= 0) != (previous >= 0) {
				zeroCrossings++
			}
		}
		previous = value
	}
	meanAbs := sumAbs / float64(count)
	if meanAbs < 0.0004 {
		return audioAnalysisCacheEntry{Genre: "Electronic", Style: "Ambient", Mood: "Calm", Confidence: "low"}, nil
	}
	brightness := (diffAbs / float64(count-1)) / math.Max(meanAbs, 0.00001)
	zcr := float64(zeroCrossings) / float64(count-1)

	frameSize := sampleRate / 100 // 10 ms envelope.
	if frameSize < 16 {
		frameSize = 16
	}
	frames := count / frameSize
	envelope := make([]float64, frames)
	for frame := 0; frame < frames; frame++ {
		var energy float64
		start := frame * frameSize
		for _, sample := range samples[start : start+frameSize] {
			energy += sample * sample
		}
		envelope[frame] = math.Sqrt(energy / float64(frameSize))
	}
	rmsValues := append([]float64(nil), envelope...)
	sort.Float64s(rmsValues)
	low := percentileSorted(rmsValues, 0.2)
	high := percentileSorted(rmsValues, 0.9)
	dynamicRange := high / math.Max(low, 0.0001)

	onsets := make([]float64, frames)
	var onsetEnergy float64
	for index := 2; index < frames; index++ {
		baseline := (envelope[index-1] + envelope[index-2]) / 2
		delta := envelope[index] - baseline
		if delta > 0 {
			onsets[index] = delta
			onsetEnergy += delta * delta
		}
	}
	bpm, beatStrength := estimateTempo(onsets, onsetEnergy, 100)

	entry := audioAnalysisCacheEntry{
		BPM: bpm, BeatStrength: beatStrength, ZeroCrossRate: zcr,
		Brightness: brightness, DynamicRange: dynamicRange, Confidence: "low",
	}
	switch {
	case beatStrength >= 0.18 && bpm >= 158:
		entry.Genre, entry.Style, entry.Mood = "Electronic", "Drum & Bass", "Energetic"
		entry.Confidence = "estimated"
	case beatStrength >= 0.2 && bpm >= 138:
		entry.Genre, entry.Style, entry.Mood = "Electronic", "Trance", "Energetic"
		entry.Confidence = "estimated"
	case beatStrength >= 0.2 && bpm >= 116:
		entry.Genre, entry.Style, entry.Mood = "Electronic", "House", "Energetic"
		entry.Confidence = "estimated"
	case beatStrength < 0.08 && brightness < 1.4 && dynamicRange < 3.0:
		entry.Genre, entry.Style, entry.Mood = "Electronic", "Ambient", "Calm"
	case zcr > 0.12 && brightness > 1.8 && dynamicRange > 2.5:
		entry.Genre, entry.Style, entry.Mood = "Rock", "Rock", "Energetic"
	case dynamicRange > 4.5 && beatStrength < 0.14:
		entry.Genre, entry.Style, entry.Mood = "Singer-Songwriter", "Acoustic", "Organic"
	default:
		entry.Genre, entry.Style, entry.Mood = "Pop", "Pop", "Balanced"
	}
	return entry, nil
}

func estimateTempo(onsets []float64, energy float64, frameRate int) (float64, float64) {
	if energy <= 0 || len(onsets) < frameRate*4 {
		return 0, 0
	}
	bestLag := 0
	best := 0.0
	for bpm := 60; bpm <= 200; bpm++ {
		lag := int(math.Round(float64(frameRate*60) / float64(bpm)))
		if lag <= 0 || lag >= len(onsets)/2 {
			continue
		}
		var correlation float64
		for index := lag; index < len(onsets); index++ {
			correlation += onsets[index] * onsets[index-lag]
		}
		correlation /= energy
		if correlation > best {
			best, bestLag = correlation, lag
		}
	}
	if bestLag == 0 {
		return 0, 0
	}
	bpm := float64(frameRate*60) / float64(bestLag)
	for bpm < 80 {
		bpm *= 2
	}
	for bpm > 190 {
		bpm /= 2
	}
	return math.Round(bpm*10) / 10, best
}

func percentileSorted(values []float64, fraction float64) float64 {
	if len(values) == 0 {
		return 0
	}
	index := int(math.Round(float64(len(values)-1) * fraction))
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func optionalMusicValue(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return []string{strings.TrimSpace(value)}
}
