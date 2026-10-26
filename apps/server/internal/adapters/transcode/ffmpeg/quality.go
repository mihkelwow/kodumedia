package ffmpeg

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/adapters/atomicfile"
	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/platform/childprocess"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type qualityRecord struct {
	Key            string    `json:"key"`
	SourcePath     string    `json:"sourcePath"`
	SourceSize     int64     `json:"sourceSize"`
	SourceModified int64     `json:"sourceModifiedNs"`
	Profile        string    `json:"profile"`
	Codec          string    `json:"codec"`
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	CRF            int       `json:"crf"`
	VMAF           float64   `json:"vmaf"`
	MeasuredAt     time.Time `json:"measuredAt"`
}
type qualityFile struct {
	Version int                      `json:"version"`
	Records map[string]qualityRecord `json:"records"`
}
type qualityStore struct {
	path    string
	mu      sync.Mutex
	loaded  bool
	records map[string]qualityRecord
}

func newQualityStore(path string) *qualityStore {
	return &qualityStore{path: path, records: map[string]qualityRecord{}}
}
func (s *qualityStore) loadLocked() {
	if s.loaded {
		return
	}
	s.loaded = true
	payload, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var file qualityFile
	if json.Unmarshal(payload, &file) == nil && file.Version == 1 && file.Records != nil {
		s.records = file.Records
	}
}
func (s *qualityStore) get(key string) (qualityRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	r, ok := s.records[key]
	return r, ok
}
func (s *qualityStore) put(record qualityRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	s.records[record.Key] = record
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(qualityFile{Version: 1, Records: s.records}, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(payload, '\n'), 0o600); err != nil {
		return err
	}
	if err := atomicfile.Replace(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
func qualityAnalysisKey(request ports.TranscodeStartRequest, info os.FileInfo) string {
	value := strings.Join([]string{request.Media.File.Path, strconv.FormatInt(info.Size(), 10), strconv.FormatInt(info.ModTime().UnixNano(), 10), request.Plan.QualityProfile, request.Plan.VideoCodec, request.Plan.VideoEncoder, request.Plan.VideoPreset, strconv.Itoa(request.Plan.TargetWidth), strconv.Itoa(request.Plan.TargetHeight)}, "\x00")
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:16])
}
func (m *Manager) applyQualityAdvice(request ports.TranscodeStartRequest, info os.FileInfo) ports.TranscodeStartRequest {
	if !request.Plan.VMAFEnabled || request.Plan.VideoAction != domain.TrackActionTranscode || request.Plan.ToneMap || m.quality == nil {
		return request
	}
	record, ok := m.quality.get(qualityAnalysisKey(request, info))
	if !ok || record.VMAF <= 0 || request.Plan.VMAFTarget <= 0 {
		return request
	}
	difference := float64(request.Plan.VMAFTarget) - record.VMAF
	adjusted := request.Plan.VideoCRF
	switch {
	case difference > 1:
		adjusted -= int(math.Min(3, math.Ceil(difference/3)))
	case difference < -3:
		adjusted++
	}
	if adjusted < 12 {
		adjusted = 12
	}
	if adjusted > 28 {
		adjusted = 28
	}
	if adjusted != request.Plan.VideoCRF {
		request.Plan.VideoCRF = adjusted
		request.Plan.Reason += fmt.Sprintf(" VMAF %.1f põhjal kasutab KoduMedia CRF %d.", record.VMAF, adjusted)
	}
	return request
}
func (m *Manager) analyzeVMAF(request ports.TranscodeStartRequest, info os.FileInfo, cacheDir string, capabilities RuntimeCapabilities) {
	if !request.Plan.VMAFEnabled || !capabilities.LibVMAF || m.quality == nil || request.Plan.VideoAction != domain.TrackActionTranscode {
		return
	}
	if request.Media.Technical.HDRFormat != "" || request.Plan.ToneMap || request.Plan.PreserveHDR {
		return
	}
	sample := request.Plan.VMAFSampleSeconds
	if sample <= 0 {
		sample = 12
	}
	if sample > 60 {
		sample = 60
	}
	state := inspectPlaylistCache(cacheDir)
	if state.SegmentCount == 0 || state.DurationMS < int64(sample)*1000 || !fileNonEmpty(filepath.Join(cacheDir, "init.mp4")) {
		return
	}
	analysisKey := qualityAnalysisKey(request, info)
	if existing, ok := m.quality.get(analysisKey); ok && existing.CRF == request.Plan.VideoCRF && existing.VMAF > 0 {
		return
	}
	jobKey := analysisKey + "|" + strconv.Itoa(request.Plan.VideoCRF)
	m.analysisMu.Lock()
	if _, running := m.analysisJobs[jobKey]; running {
		m.analysisMu.Unlock()
		return
	}
	m.analysisJobs[jobKey] = struct{}{}
	m.analysisMu.Unlock()
	defer func() { m.analysisMu.Lock(); delete(m.analysisJobs, jobKey); m.analysisMu.Unlock() }()
	playlist, cleanup, err := completedPlaylistSnapshot(cacheDir)
	if err != nil {
		return
	}
	defer cleanup()
	width, height := request.Plan.TargetWidth, request.Plan.TargetHeight
	if width <= 0 {
		width = request.Media.Technical.Width
	}
	if height <= 0 {
		height = request.Media.Technical.Height
	}
	if width <= 0 || height <= 0 {
		return
	}
	logPath := filepath.Join(cacheDir, ".vmaf.json")
	filter := fmt.Sprintf("[0:v]setpts=PTS-STARTPTS,scale=%d:%d:flags=lanczos,format=yuv420p[ref];[1:v]setpts=PTS-STARTPTS,scale=%d:%d:flags=lanczos,format=yuv420p[dist];[dist][ref]libvmaf=log_fmt=json:log_path='%s':n_threads=2", width, height, width, height, escapeFilterPath(logPath))
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sample+45)*time.Second)
	defer cancel()
	arguments := []string{"-hide_banner", "-loglevel", "error", "-nostdin"}
	if request.Plan.StartPositionMS > 0 {
		arguments = append(arguments, "-ss", formatSeconds(request.Plan.StartPositionMS))
	}
	arguments = append(arguments,
		"-t", strconv.Itoa(sample), "-i", request.Media.File.Path,
		"-t", strconv.Itoa(sample), "-i", playlist,
		"-lavfi", filter, "-f", "null", "-",
	)
	command := exec.CommandContext(ctx, m.executable(), arguments...)
	childprocess.Hide(command)
	if output, err := command.CombinedOutput(); err != nil {
		slog.Debug("VMAF analysis skipped", "error", err, "output", strings.TrimSpace(string(output)))
		_ = os.Remove(logPath)
		return
	}
	value, err := parseVMAF(logPath)
	_ = os.Remove(logPath)
	if err != nil || value <= 0 {
		return
	}
	record := qualityRecord{Key: analysisKey, SourcePath: request.Media.File.Path, SourceSize: info.Size(), SourceModified: info.ModTime().UnixNano(), Profile: request.Plan.QualityProfile, Codec: request.Plan.VideoCodec, Width: width, Height: height, CRF: request.Plan.VideoCRF, VMAF: value, MeasuredAt: time.Now().UTC()}
	if err := m.quality.put(record); err != nil {
		return
	}
	_ = updateCacheMetadata(cacheDir, func(metadata *cacheMetadata) { metadata.VMAF = value; metadata.UpdatedAt = time.Now().UTC() })
	slog.Info("VMAF analysis completed", "media", request.Media.ID, "score", value, "crf", request.Plan.VideoCRF)
}
func completedPlaylistSnapshot(cacheDir string) (string, func(), error) {
	original := filepath.Join(cacheDir, "index.m3u8")
	payload, err := os.ReadFile(original)
	if err != nil {
		return "", func() {}, err
	}
	if len(payload) == 0 {
		return "", func() {}, fmt.Errorf("empty HLS playlist")
	}
	if strings.Contains(string(payload), "#EXT-X-ENDLIST") {
		return original, func() {}, nil
	}
	snapshot := filepath.Join(cacheDir, ".vmaf-snapshot.m3u8")
	content := strings.TrimRight(string(payload), "\r\n") + "\n#EXT-X-ENDLIST\n"
	if err := os.WriteFile(snapshot, []byte(content), 0o600); err != nil {
		return "", func() {}, err
	}
	return snapshot, func() { _ = os.Remove(snapshot) }, nil
}
func parseVMAF(path string) (float64, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var decoded struct {
		PooledMetrics struct {
			VMAF struct {
				Mean float64 `json:"mean"`
			} `json:"vmaf"`
		} `json:"pooled_metrics"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return 0, err
	}
	if decoded.PooledMetrics.VMAF.Mean <= 0 {
		return 0, fmt.Errorf("VMAF result did not contain a mean score")
	}
	return decoded.PooledMetrics.VMAF.Mean, nil
}
func escapeFilterPath(path string) string {
	value := filepath.ToSlash(path)
	value = strings.ReplaceAll(value, ":", "\\:")
	value = strings.ReplaceAll(value, "'", "\\'")
	return value
}
