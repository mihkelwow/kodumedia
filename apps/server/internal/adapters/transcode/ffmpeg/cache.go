package ffmpeg

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/adapters/atomicfile"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

const cacheMetadataName = ".kodu-cache.json"

var cacheMetadataMu sync.Mutex

type cacheMetadata struct {
	Version           int       `json:"version"`
	Key               string    `json:"key"`
	VariantKey        string    `json:"variantKey,omitempty"`
	SourcePath        string    `json:"sourcePath"`
	SourceSize        int64     `json:"sourceSize"`
	SourceModifiedNS  int64     `json:"sourceModifiedNs"`
	Complete          bool      `json:"complete"`
	BaseStartMS       int64     `json:"baseStartMs,omitempty"`
	CachedDurationMS  int64     `json:"cachedDurationMs"`
	SegmentCount      int       `json:"segmentCount"`
	QualityProfile    string    `json:"qualityProfile"`
	VideoCodec        string    `json:"videoCodec"`
	VideoEncoder      string    `json:"videoEncoder"`
	TargetWidth       int       `json:"targetWidth"`
	TargetHeight      int       `json:"targetHeight"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	LastAccessAt      time.Time `json:"lastAccessAt"`
	SizeBytes         int64     `json:"sizeBytes"`
	VMAF              float64   `json:"vmaf,omitempty"`
	TranscodeRevision string    `json:"transcodeRevision,omitempty"`
	EncoderBuild      string    `json:"encoderBuild,omitempty"`
}
type playlistCacheState struct {
	DurationMS   int64
	SegmentCount int
	NextNumber   int
	Complete     bool
}

func inspectPlaylistCache(directory string) playlistCacheState {
	payload, err := os.ReadFile(filepath.Join(directory, "index.m3u8"))
	if err != nil {
		return playlistCacheState{}
	}
	state := playlistCacheState{Complete: strings.Contains(string(payload), "#EXT-X-ENDLIST")}
	for _, raw := range strings.Split(string(payload), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "#EXTINF:") {
			v := strings.TrimSuffix(strings.TrimPrefix(line, "#EXTINF:"), ",")
			seconds, _ := strconv.ParseFloat(v, 64)
			state.DurationMS += int64(seconds * 1000)
			continue
		}
		if strings.HasPrefix(line, "segment_") {
			state.SegmentCount++
			base := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(line), "segment_"), filepath.Ext(line))
			if n, e := strconv.Atoi(base); e == nil && n >= state.NextNumber {
				state.NextNumber = n + 1
			}
		}
	}
	return state
}
func readCacheMetadata(directory string) (cacheMetadata, error) {
	payload, err := os.ReadFile(filepath.Join(directory, cacheMetadataName))
	if err != nil {
		return cacheMetadata{}, err
	}
	var m cacheMetadata
	if err := json.Unmarshal(payload, &m); err != nil {
		return cacheMetadata{}, err
	}
	return m, nil
}
func writeCacheMetadata(directory string, metadata cacheMetadata) error {
	cacheMetadataMu.Lock()
	defer cacheMetadataMu.Unlock()
	return writeCacheMetadataUnlocked(directory, metadata)
}
func writeCacheMetadataUnlocked(directory string, metadata cacheMetadata) error {
	payload, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(directory, cacheMetadataName)
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, append(payload, '\n'), 0o600); err != nil {
		return err
	}
	if err := atomicfile.Replace(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}
func buildCacheMetadata(key string, request ports.TranscodeStartRequest, info fs.FileInfo, state playlistCacheState, createdAt time.Time) cacheMetadata {
	now := time.Now().UTC()
	if createdAt.IsZero() {
		createdAt = now
	}
	return cacheMetadata{Version: 2, Key: key, VariantKey: transcodeVariantKey(request, info), SourcePath: request.Media.File.Path, SourceSize: info.Size(), SourceModifiedNS: info.ModTime().UnixNano(), Complete: state.Complete, BaseStartMS: request.Plan.StartPositionMS, CachedDurationMS: state.DurationMS, SegmentCount: state.SegmentCount, QualityProfile: request.Plan.QualityProfile, VideoCodec: request.Plan.VideoCodec, VideoEncoder: request.Plan.VideoEncoder, TargetWidth: request.Plan.TargetWidth, TargetHeight: request.Plan.TargetHeight, CreatedAt: createdAt, UpdatedAt: now, LastAccessAt: now, TranscodeRevision: request.Plan.TranscodeRevision, EncoderBuild: request.Plan.EncoderBuild}
}
func cacheSourceMatches(metadata cacheMetadata, request ports.TranscodeStartRequest, info fs.FileInfo) bool {
	if metadata.Version < 1 || metadata.Version > 2 {
		return false
	}
	baseStart := metadata.BaseStartMS
	if metadata.Version == 1 {
		baseStart = 0
	}
	return metadata.SourcePath == request.Media.File.Path &&
		metadata.SourceSize == info.Size() &&
		metadata.SourceModifiedNS == info.ModTime().UnixNano() &&
		baseStart == request.Plan.StartPositionMS
}

type completedCacheCandidate struct {
	Path     string
	Metadata cacheMetadata
}

func findCompletedCache(root, variantKey string, requestedStartMS int64) (completedCacheCandidate, bool) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return completedCacheCandidate{}, false
	}
	requestedStartMS = nonNegativeMS(requestedStartMS)
	best := completedCacheCandidate{}
	found := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		directory := filepath.Join(root, entry.Name())
		metadata, err := readCacheMetadata(directory)
		if err != nil || !metadata.Complete || metadata.VariantKey != variantKey {
			continue
		}
		end := metadata.BaseStartMS + metadata.CachedDurationMS
		if requestedStartMS < metadata.BaseStartMS || requestedStartMS >= end {
			continue
		}
		if !cachedOutputReady(directory) {
			continue
		}
		if !found || metadata.BaseStartMS > best.Metadata.BaseStartMS {
			best = completedCacheCandidate{Path: directory, Metadata: metadata}
			found = true
		}
	}
	return best, found
}

func cachedOutputReady(directory string) bool {
	metadata, err := readCacheMetadata(directory)
	if err != nil || !metadata.Complete {
		return false
	}
	state := inspectPlaylistCache(directory)
	return state.Complete && state.SegmentCount > 0 && fileNonEmpty(filepath.Join(directory, "init.mp4"))
}
func preparePartialCache(cacheDir, sessionDir string, request *ports.TranscodeStartRequest, info fs.FileInfo) (bool, error) {
	metadata, err := readCacheMetadata(cacheDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if metadata.Complete || !cacheSourceMatches(metadata, *request, info) {
		return false, nil
	}
	state := inspectPlaylistCache(cacheDir)
	if state.SegmentCount == 0 || state.DurationMS <= 0 {
		return false, nil
	}
	if err := copyCacheMedia(cacheDir, sessionDir); err != nil {
		return false, err
	}
	playlist := filepath.Join(sessionDir, "index.m3u8")
	payload, err := os.ReadFile(playlist)
	if err != nil {
		return false, err
	}
	cleaned := strings.ReplaceAll(string(payload), "#EXT-X-ENDLIST\n", "")
	cleaned = strings.ReplaceAll(cleaned, "#EXT-X-ENDLIST", "")
	if err := os.WriteFile(playlist, []byte(cleaned), 0o600); err != nil {
		return false, err
	}
	request.CacheResumeFromMS = metadata.BaseStartMS + state.DurationMS
	request.CacheStartNumber = state.NextNumber
	request.CacheAppend = true
	return true, nil
}
func publishCacheSnapshot(source, destination, key string, request ports.TranscodeStartRequest, info fs.FileInfo, forceComplete bool) error {
	state := inspectPlaylistCache(source)
	if state.SegmentCount == 0 || !fileNonEmpty(filepath.Join(source, "init.mp4")) {
		return fmt.Errorf("no reusable HLS segments were generated")
	}
	if forceComplete && !state.Complete {
		return fmt.Errorf("playback cache is not complete")
	}
	created := time.Time{}
	if previous, err := readCacheMetadata(destination); err == nil {
		created = previous.CreatedAt
	}
	temporary := destination + ".tmp"
	_ = os.RemoveAll(temporary)
	if err := copyCacheMedia(source, temporary); err != nil {
		_ = os.RemoveAll(temporary)
		return err
	}
	metadata := buildCacheMetadata(key, request, info, state, created)
	metadata.SizeBytes = directorySize(temporary)
	if err := writeCacheMetadata(temporary, metadata); err != nil {
		_ = os.RemoveAll(temporary)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		_ = os.RemoveAll(temporary)
		return err
	}
	_ = os.RemoveAll(destination)
	if err := os.Rename(temporary, destination); err != nil {
		_ = os.RemoveAll(temporary)
		return err
	}
	return nil
}
func touchCache(directory string) {
	cacheMetadataMu.Lock()
	defer cacheMetadataMu.Unlock()
	metadata, err := readCacheMetadata(directory)
	if err != nil {
		return
	}
	metadata.LastAccessAt = time.Now().UTC()
	metadata.UpdatedAt = metadata.LastAccessAt
	_ = writeCacheMetadataUnlocked(directory, metadata)
}
func updateCacheMetadata(directory string, update func(*cacheMetadata)) error {
	cacheMetadataMu.Lock()
	defer cacheMetadataMu.Unlock()
	metadata, err := readCacheMetadata(directory)
	if err != nil {
		return err
	}
	update(&metadata)
	return writeCacheMetadataUnlocked(directory, metadata)
}
func copyCacheMedia(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return os.MkdirAll(destination, 0o755)
		}
		base := filepath.Base(path)
		if entry.IsDir() {
			return os.MkdirAll(filepath.Join(destination, relative), 0o755)
		}
		if base == "ffmpeg.log" || base == "progress.log" || strings.HasSuffix(base, ".tmp") || base == cacheMetadataName {
			return nil
		}
		if base != "index.m3u8" && base != "init.mp4" && !strings.HasPrefix(base, "segment_") {
			return nil
		}
		target := filepath.Join(destination, relative)
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			_ = input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		inClose := input.Close()
		outClose := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inClose != nil {
			return inClose
		}
		return outClose
	})
}
func directorySize(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return nil
		}
		if info, ie := e.Info(); ie == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

type cacheCandidate struct {
	Path       string
	Size       int64
	LastAccess time.Time
}

func enforceCacheLimit(root string, maximumBytes, reserveBytes int64, reservePercent int, active map[string]struct{}) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	candidates := make([]cacheCandidate, 0, len(entries))
	var total int64
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		size := cachedDirectorySize(path)
		total += size
		if _, inUse := active[entry.Name()]; inUse {
			continue
		}
		last := time.Time{}
		if metadata, e := readCacheMetadata(path); e == nil {
			last = metadata.LastAccessAt
			if strings.TrimSpace(metadata.SourcePath) != "" {
				if source, statErr := os.Stat(metadata.SourcePath); statErr != nil ||
					source.Size() != metadata.SourceSize ||
					source.ModTime().UnixNano() != metadata.SourceModifiedNS {
					if os.RemoveAll(path) == nil {
						total -= size
					}
					continue
				}
			}
		}
		if last.IsZero() {
			if info, e := entry.Info(); e == nil {
				last = info.ModTime()
			}
		}
		candidates = append(candidates, cacheCandidate{path, size, last})
	}
	effectiveMaximum := maximumBytes
	if maximumBytes <= 0 {
		effectiveMaximum = 0
	} else if freeBytes, totalBytes, spaceErr := diskSpace(root); spaceErr == nil {
		reserve := reserveBytes
		if reservePercent > 0 {
			percentReserve := int64(totalBytes / 100 * uint64(reservePercent))
			if percentReserve > reserve {
				reserve = percentReserve
			}
		}
		allowedByReserve := total + int64(freeBytes) - reserve
		if allowedByReserve < 0 {
			allowedByReserve = 0
		}
		if allowedByReserve < effectiveMaximum {
			effectiveMaximum = allowedByReserve
		}
	}
	if total <= effectiveMaximum {
		return
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].LastAccess.Before(candidates[j].LastAccess) })
	for _, c := range candidates {
		if total <= effectiveMaximum {
			break
		}
		if os.RemoveAll(c.Path) == nil {
			total -= c.Size
		}
	}
}

func cachedDirectorySize(path string) int64 {
	if metadata, err := readCacheMetadata(path); err == nil && metadata.SizeBytes > 0 {
		if info, statErr := os.Stat(filepath.Join(path, cacheMetadataName)); statErr == nil {
			return metadata.SizeBytes + info.Size()
		}
		return metadata.SizeBytes
	}
	return directorySize(path)
}

func cacheRootSize(root string) int64 {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	var total int64
	for _, entry := range entries {
		if entry.IsDir() {
			total += cachedDirectorySize(filepath.Join(root, entry.Name()))
		}
	}
	return total
}
func playlistComplete(directory string) bool { return inspectPlaylistCache(directory).Complete }
