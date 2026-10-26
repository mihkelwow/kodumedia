package ffmpeg

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/platform/childprocess"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

const (
	qualityPipelineRevision = "quality-engine-v5"
	cacheRangeAlignmentMS   = int64(2_000)
	playlistStartupTimeout  = 35 * time.Second
	generatedFileWait       = 8 * time.Second
	processStopTimeout      = 4 * time.Second
	capabilityProbeTimeout  = 8 * time.Second
)

type runtimeSession struct {
	session             domain.PlaybackSession
	dir                 string
	command             *exec.Cmd
	done                chan struct{}
	errMu               sync.RWMutex
	err                 error
	persistent          bool
	cacheDir            string
	cacheKey            string
	cachePolicy         string
	cacheMaxBytes       int64
	cacheReserveBytes   int64
	cacheReservePercent int
	gracePeriod         time.Duration
	refCount            int
	stopTimer           *time.Timer
	request             ports.TranscodeStartRequest
	statsMu             sync.RWMutex
	stats               domain.PlaybackRuntimeStats
}

type Manager struct {
	binary   func() string
	root     string
	idleTTL  time.Duration
	profiles map[string]Profile

	mu         sync.RWMutex
	sessions   map[string]*runtimeSession
	activeJobs map[string]*runtimeSession
	stop       chan struct{}
	stopped    chan struct{}

	capabilityMu    sync.Mutex
	capabilityCache map[string]RuntimeCapabilities
	encoderMu       sync.Mutex
	encoderCache    map[string]encoderProbeCacheEntry
	quality         *qualityStore
	analysisMu      sync.Mutex
	analysisJobs    map[string]struct{}
}

func NewManager(binary, root string, idleTTL time.Duration, profiles ...Profile) (*Manager, error) {
	return NewManagerDynamic(func() string { return binary }, root, idleTTL, profiles...)
}

func NewManagerDynamic(binary func() string, root string, idleTTL time.Duration, profiles ...Profile) (*Manager, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create transcode directory: %w", err)
	}
	// Playback sessions cannot survive a server restart. Removing every old
	// session directory here is safe and also recovers space after crashes,
	// power loss, or forced termination.
	if entries, err := os.ReadDir(root); err == nil {
		for _, entry := range entries {
			_ = os.RemoveAll(filepath.Join(root, entry.Name()))
		}
	}
	profileMap := make(map[string]Profile, len(profiles))
	for _, profile := range profiles {
		profileMap[profile.ID()] = profile
	}
	manager := &Manager{
		binary:          binary,
		root:            root,
		idleTTL:         idleTTL,
		profiles:        profileMap,
		sessions:        map[string]*runtimeSession{},
		activeJobs:      map[string]*runtimeSession{},
		stop:            make(chan struct{}),
		stopped:         make(chan struct{}),
		capabilityCache: map[string]RuntimeCapabilities{},
		encoderCache:    map[string]encoderProbeCacheEntry{},
		quality:         newQualityStore(filepath.Join(filepath.Dir(root), "quality", "analysis.json")),
		analysisJobs:    map[string]struct{}{},
	}
	go manager.cleanupLoop()
	return manager, nil
}

func (r *runtimeSession) setErr(err error) {
	r.errMu.Lock()
	r.err = err
	r.errMu.Unlock()
}

func (r *runtimeSession) getErr() error {
	r.errMu.RLock()
	defer r.errMu.RUnlock()
	return r.err
}

func (m *Manager) executable() string {
	if m.binary == nil {
		return "ffmpeg"
	}
	if value := strings.TrimSpace(m.binary()); value != "" {
		return value
	}
	return "ffmpeg"
}

func (m *Manager) Available(ctx context.Context) bool {
	command := exec.CommandContext(ctx, m.executable(), "-version")
	childprocess.Hide(command)
	return command.Run() == nil
}

func (m *Manager) runtimeCapabilities() RuntimeCapabilities {
	executable := m.executable()
	key := executableFingerprint(executable)
	m.capabilityMu.Lock()
	if capabilities, ok := m.capabilityCache[key]; ok {
		m.capabilityMu.Unlock()
		return capabilities
	}
	m.capabilityMu.Unlock()
	type probeResult struct {
		index  int
		output []byte
	}
	probes := [][]string{
		{"-hide_banner", "-h", "full"},
		{"-hide_banner", "-encoders"},
		{"-hide_banner", "-filters"},
		{"-hide_banner", "-hwaccels"},
	}
	results := make(chan probeResult, len(probes))
	for index, arguments := range probes {
		go func(index int, arguments []string) {
			results <- probeResult{index: index, output: runCapabilityProbe(executable, arguments...)}
		}(index, arguments)
	}
	outputs := make([][]byte, len(probes))
	for range probes {
		result := <-results
		outputs[result.index] = result.output
	}
	capabilityOutput := make([]string, len(outputs))
	for index, output := range outputs {
		capabilityOutput[index] = string(output)
	}
	capabilities := parseRuntimeCapabilities(strings.Join(capabilityOutput, "\n"))
	m.capabilityMu.Lock()
	if existing, ok := m.capabilityCache[key]; ok {
		m.capabilityMu.Unlock()
		return existing
	}
	m.capabilityCache[key] = capabilities
	m.capabilityMu.Unlock()
	slog.Info("ffmpeg runtime capabilities detected", "executable", executable, "readrate", capabilities.Readrate, "readrateInitialBurst", capabilities.ReadrateInitialBurst, "cuda", capabilities.CUDA, "qsv", capabilities.QSV, "d3d11va", capabilities.D3D11VA, "libvmaf", capabilities.LibVMAF)
	return capabilities
}

func runCapabilityProbe(executable string, arguments ...string) []byte {
	ctx, cancel := context.WithTimeout(context.Background(), capabilityProbeTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, executable, arguments...)
	childprocess.Hide(command)
	output, _ := command.CombinedOutput()
	return output
}

func parseRuntimeCapabilities(help string) RuntimeCapabilities {
	capabilities := RuntimeCapabilities{}
	for _, field := range strings.Fields(help) {
		switch strings.TrimSpace(field) {
		case "-readrate":
			capabilities.Readrate = true
		case "-readrate_initial_burst":
			capabilities.ReadrateInitialBurst = true
		}
	}
	// The burst option is an extension of readrate. Treat malformed custom help
	// output conservatively and never emit the child option without its parent.
	if capabilities.ReadrateInitialBurst {
		capabilities.Readrate = true
	}
	lower := strings.ToLower(help)
	capabilities.CUDA = containsCapabilityLine(lower, "cuda")
	capabilities.QSV = containsCapabilityLine(lower, "qsv")
	capabilities.D3D11VA = containsCapabilityLine(lower, "d3d11va")
	capabilities.H264QSV = strings.Contains(lower, "h264_qsv")
	capabilities.HEVCQSV = strings.Contains(lower, "hevc_qsv")
	capabilities.AV1QSV = strings.Contains(lower, "av1_qsv")
	capabilities.H264NVENC = strings.Contains(lower, "h264_nvenc")
	capabilities.HEVCNVENC = strings.Contains(lower, "hevc_nvenc")
	capabilities.AV1NVENC = strings.Contains(lower, "av1_nvenc")
	capabilities.H264AMF = strings.Contains(lower, "h264_amf")
	capabilities.HEVCAMF = strings.Contains(lower, "hevc_amf")
	capabilities.AV1AMF = strings.Contains(lower, "av1_amf")
	capabilities.LibVMAF = strings.Contains(lower, "libvmaf")
	capabilities.LibX264 = strings.Contains(lower, "libx264")
	capabilities.LibX265 = strings.Contains(lower, "libx265")
	capabilities.LibSVTAV1 = strings.Contains(lower, "libsvtav1")
	capabilities.LibAOMAV1 = strings.Contains(lower, "libaom-av1") || strings.Contains(lower, "libaom_av1")
	return capabilities
}

func containsCapabilityLine(output, value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, line := range strings.Split(output, "\n") {
		if strings.ToLower(strings.TrimSpace(line)) == value {
			return true
		}
	}
	return false
}

func (m *Manager) Start(ctx context.Context, request ports.TranscodeStartRequest) (domain.PlaybackSession, error) {
	profile, ok := m.profiles[request.Plan.TranscodeProfile]
	if !ok && request.Plan.TranscodeProfile == "software-h264-aac" {
		profile, ok = m.profiles["adaptive-video-aac"]
	}
	if !ok {
		return domain.PlaybackSession{}, fmt.Errorf("%w: transcode profile %q", domain.ErrUnsupported, request.Plan.TranscodeProfile)
	}
	if request.Media.File.Path == "" {
		return domain.PlaybackSession{}, fmt.Errorf("%w: media path is empty", domain.ErrInvalidInput)
	}
	sourceInfo, err := os.Stat(request.Media.File.Path)
	if err != nil {
		return domain.PlaybackSession{}, fmt.Errorf("source file unavailable: %w", err)
	}

	capabilities := m.runtimeCapabilities()
	request.Plan.TranscodeRevision = qualityPipelineRevision
	request.Plan.EncoderBuild = executableFingerprint(m.executable())
	requestedStartMS := nonNegativeMS(request.Plan.StartPositionMS)
	if cacheLookupEligible(request) {
		baseStart := alignedCacheStartMS(requestedStartMS)
		request.Plan.StartPositionMS = baseStart
		request.Plan.InitialPositionMS = requestedStartMS - baseStart
	}
	request.Plan = m.resolveEncoder(ctx, request.Plan, request.Media.Technical, capabilities)
	request = m.applyQualityAdvice(request, sourceInfo)

	id := newSessionID()
	cacheDir, cacheKey := "", ""
	cacheWritable := false
	if cacheLookupEligible(request) {
		cacheRoot := filepath.Join(filepath.Dir(m.root), "optimized-media")
		variantKey := transcodeVariantKey(request, sourceInfo)
		if candidate, found := findCompletedCache(cacheRoot, variantKey, requestedStartMS); found {
			cacheDir, cacheKey = candidate.Path, candidate.Metadata.Key
			request.Plan.StartPositionMS = candidate.Metadata.BaseStartMS
			request.Plan.InitialPositionMS = requestedStartMS - candidate.Metadata.BaseStartMS
		} else {
			cacheKey = transcodeCacheKey(request, sourceInfo)
			cacheDir = filepath.Join(cacheRoot, cacheKey)
		}
		if cachedOutputReady(cacheDir) {
			metadata, _ := readCacheMetadata(cacheDir)
			now := time.Now().UTC()
			done := make(chan struct{})
			close(done)
			cachedPlan := request.Plan
			cachedPlan.CacheHit = true
			cachedPlan.Reason = "KoduMedia kasutab varem vaatamise ajal loodud optimeeritud kvaliteedivoogu."
			session := newPlaybackSession(id, request.Media.ID, cachedPlan, now)
			cachedRequest := request
			cachedRequest.Plan = cachedPlan
			runtime := &runtimeSession{
				session: session, dir: cacheDir, done: done, persistent: true,
				cacheDir: cacheDir, cacheKey: cacheKey, refCount: 1, request: cachedRequest,
				stats: domain.PlaybackRuntimeStats{
					SessionID: id, State: "cached", ProducerAlive: false,
					GeneratedDurationMS: metadata.CachedDurationMS,
					SegmentCount:        metadata.SegmentCount, MinimumEncoderSpeed: cachedPlan.MinimumEncoderSpeed,
					UpdatedAt: now,
				},
			}
			m.mu.Lock()
			m.sessions[id] = runtime
			m.mu.Unlock()
			touchCache(cacheDir)
			go m.enforceCacheLimit(request.Plan.CacheMaxBytes, request.Plan.CacheFreeReserveBytes, request.Plan.CacheFreeReservePercent)
			return session, nil
		}
		cacheWritable = cacheWriteEligible(request)
		if cacheWritable {
			m.mu.Lock()
			if active := m.activeJobs[cacheKey]; active != nil {
				if active.stopTimer != nil {
					active.stopTimer.Stop()
					active.stopTimer = nil
				}
				active.refCount++
				sharedPlan := active.session.Plan
				sharedPlan.StartPositionMS = request.Plan.StartPositionMS
				sharedPlan.InitialPositionMS = request.Plan.InitialPositionMS
				session := newPlaybackSession(id, request.Media.ID, sharedPlan, time.Now().UTC())
				m.sessions[id] = active
				m.mu.Unlock()
				if err := m.waitForPlayableOutput(ctx, active); err != nil {
					m.removeSession(id, false)
					return domain.PlaybackSession{}, err
				}
				return session, nil
			}
			m.mu.Unlock()
		} else {
			cacheDir, cacheKey = "", ""
		}
	}

	dir := filepath.Join(m.root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return domain.PlaybackSession{}, fmt.Errorf("create session directory: %w", err)
	}
	if cacheDir != "" && (request.Plan.CachePolicy == "watched" || request.Plan.CachePolicy == "finish") {
		if resumed, resumeErr := preparePartialCache(cacheDir, dir, &request, sourceInfo); resumeErr != nil {
			slog.Warn("restore partial optimized cache", "cache", cacheDir, "error", resumeErr)
		} else if resumed {
			slog.Info("resuming partial optimized cache", "cache", cacheDir, "fromMs", request.CacheResumeFromMS)
		}
	}
	logFile, err := os.OpenFile(filepath.Join(dir, "ffmpeg.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		_ = os.RemoveAll(dir)
		return domain.PlaybackSession{}, fmt.Errorf("create ffmpeg log: %w", err)
	}
	command := exec.Command(m.executable(), profile.Arguments(request, dir, capabilities)...)
	childprocess.Hide(command)
	command.Dir = dir
	command.Stdout, command.Stderr = logFile, logFile
	if err := command.Start(); err != nil {
		_ = logFile.Close()
		_ = os.RemoveAll(dir)
		return domain.PlaybackSession{}, fmt.Errorf("start ffmpeg: %w", err)
	}

	now := time.Now().UTC()
	session := newPlaybackSession(id, request.Media.ID, request.Plan, now)
	runtime := &runtimeSession{
		session: session, dir: dir, command: command, done: make(chan struct{}),
		cacheDir: cacheDir, cacheKey: cacheKey, cachePolicy: request.Plan.CachePolicy,
		cacheMaxBytes: request.Plan.CacheMaxBytes, cacheReserveBytes: request.Plan.CacheFreeReserveBytes,
		cacheReservePercent: request.Plan.CacheFreeReservePercent, gracePeriod: time.Duration(request.Plan.GracePeriodSeconds) * time.Second,
		refCount: 1, request: request,
		stats: domain.PlaybackRuntimeStats{
			SessionID: id, State: "starting", ProducerAlive: true,
			MinimumEncoderSpeed: request.Plan.MinimumEncoderSpeed, UpdatedAt: now,
		},
	}
	m.mu.Lock()
	m.sessions[id] = runtime
	if cacheKey != "" {
		m.activeJobs[cacheKey] = runtime
	}
	m.mu.Unlock()

	go m.monitorRuntime(runtime)

	go func() {
		runtime.setErr(command.Wait())
		_ = logFile.Close()
		complete := runtime.getErr() == nil && playlistComplete(runtime.dir)
		if runtime.cacheDir != "" && (complete || runtime.cachePolicy == "watched" || runtime.cachePolicy == "finish") {
			if err := publishCacheSnapshot(runtime.dir, runtime.cacheDir, runtime.cacheKey, runtime.request, sourceInfo, complete); err != nil {
				if complete {
					slog.Warn("publish optimized playback cache", "session", id, "error", err)
				}
			} else {
				go m.analyzeVMAF(runtime.request, sourceInfo, runtime.cacheDir, capabilities)
			}
		}
		m.mu.Lock()
		if runtime.cacheKey != "" && m.activeJobs[runtime.cacheKey] == runtime {
			delete(m.activeJobs, runtime.cacheKey)
		}
		m.mu.Unlock()
		close(runtime.done)
		m.mu.Lock()
		refs := runtime.refCount
		if refs == 0 && runtime.stopTimer != nil {
			runtime.stopTimer.Stop()
			runtime.stopTimer = nil
		}
		m.mu.Unlock()
		if refs == 0 && !runtime.persistent {
			removeDirectoryWithRetry(runtime.dir)
		}
		if runtime.cacheDir != "" {
			go m.enforceCacheLimit(runtime.cacheMaxBytes, runtime.cacheReserveBytes, runtime.cacheReservePercent)
		}
	}()

	if err := m.waitForPlayableOutput(ctx, runtime); err != nil {
		failure := err.Error()
		m.removeSession(id, true)
		if request.Plan.VideoAction == domain.TrackActionTranscode {
			if cacheDir != "" {
				_ = os.RemoveAll(cacheDir)
			}
			request.CacheAppend = false
			request.CacheResumeFromMS = 0
			request.CacheStartNumber = 0
			if retry, ok := m.retryAfterEncoderFailure(request.Plan, request.Media.Technical, capabilities, failure); ok {
				request.Plan = retry
				return m.Start(ctx, request)
			}
		}
		return domain.PlaybackSession{}, err
	}
	if request.CacheAppend {
		if err := m.waitForAppendedSegment(ctx, runtime, request.CacheStartNumber); err != nil {
			m.removeSession(id, true)
			request.CacheAppend = false
			request.CacheResumeFromMS = 0
			request.CacheStartNumber = 0
			_ = os.RemoveAll(cacheDir)
			return m.Start(ctx, request)
		}
	}
	if speed, known := m.assessStartupSpeed(runtime, request.Plan, request.Media.Technical); isUnsustainableSpeed(request.Plan, speed, known) {
		m.removeSession(id, true)
		if cacheDir != "" {
			_ = os.RemoveAll(cacheDir)
		}
		request.CacheAppend = false
		request.CacheResumeFromMS = 0
		request.CacheStartNumber = 0
		reason := fmt.Sprintf("Encoder'i käivituskiirus %.2fx jäi alla %.2fx turvavaru.", speed, request.Plan.MinimumEncoderSpeed)
		if retry, ok := nextStabilityPlan(request.Plan, request.Media.Technical, reason); ok {
			request.Plan = retry
			return m.Start(ctx, request)
		}
		return domain.PlaybackSession{}, fmt.Errorf("%w: %s", domain.ErrUnavailable, reason)
	}
	slog.Info("playback session ready", "session", id, "media", request.Media.ID, "mode", request.Plan.Mode, "encoder", request.Plan.VideoEncoder)
	return session, nil
}

func (m *Manager) Open(ctx context.Context, sessionID, relativePath string) (fs.File, error) {
	m.mu.RLock()
	runtime, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return nil, domain.ErrNotFound
	}

	clean := filepath.Clean(filepath.FromSlash(relativePath))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, domain.ErrInvalidInput
	}
	path := filepath.Join(runtime.dir, clean)
	if !isWithin(runtime.dir, path) {
		return nil, domain.ErrInvalidInput
	}

	file, err := m.openGeneratedFile(ctx, runtime, clean, path)
	if err != nil {
		return nil, err
	}
	if countsAsPlaybackActivity(clean) {
		m.Touch(sessionID)
	}
	return file, nil
}

func (m *Manager) openGeneratedFile(ctx context.Context, runtime *runtimeSession, relativePath, absolutePath string) (fs.File, error) {
	open := func() (fs.File, error) {
		file, err := os.Open(absolutePath)
		if errors.Is(err, os.ErrNotExist) {
			return nil, domain.ErrNotFound
		}
		return file, err
	}

	file, err := open()
	if err == nil || !errors.Is(err, domain.ErrNotFound) || !isExpectedGeneratedFile(relativePath) {
		return file, err
	}

	// A playlist can become visible a few milliseconds before an atomically
	// renamed segment. Briefly wait instead of turning a harmless producer /
	// consumer race into a fatal fragLoadError in hls.js.
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(generatedFileWait)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-runtime.done:
			file, finalErr := open()
			if finalErr == nil {
				return file, nil
			}
			return nil, m.outputUnavailable(runtime, relativePath)
		case <-timer.C:
			return nil, fmt.Errorf("%w: generated playback file %q was not ready in time", domain.ErrUnavailable, relativePath)
		case <-ticker.C:
			file, retryErr := open()
			if retryErr == nil {
				return file, nil
			}
			if !errors.Is(retryErr, domain.ErrNotFound) {
				return nil, retryErr
			}
		}
	}
}

func (m *Manager) Touch(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if runtime, ok := m.sessions[sessionID]; ok {
		runtime.session.LastAccessAt = time.Now().UTC()
	}
}

func (m *Manager) Stats(sessionID string) (domain.PlaybackRuntimeStats, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return domain.PlaybackRuntimeStats{}, domain.ErrInvalidInput
	}
	m.mu.RLock()
	runtime, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return domain.PlaybackRuntimeStats{}, domain.ErrNotFound
	}
	stats := runtime.snapshotStats()
	stats.SessionID = sessionID
	if stats.UpdatedAt.IsZero() {
		stats.UpdatedAt = time.Now().UTC()
	}
	return stats, nil
}

func (m *Manager) Stop(sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return domain.ErrInvalidInput
	}
	m.mu.RLock()
	_, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return domain.ErrNotFound
	}
	m.removeSession(sessionID, false)
	return nil
}

func (m *Manager) Close() error {
	select {
	case <-m.stop:
		return nil
	default:
		close(m.stop)
	}
	<-m.stopped
	m.mu.RLock()
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	jobs := make(map[*runtimeSession]struct{}, len(m.activeJobs))
	for _, runtime := range m.activeJobs {
		jobs[runtime] = struct{}{}
	}
	m.mu.RUnlock()
	for _, id := range ids {
		m.removeSession(id, true)
	}
	for runtime := range jobs {
		m.stopRuntime(runtime, true)
	}
	return nil
}

func (m *Manager) waitForPlayableOutput(ctx context.Context, runtime *runtimeSession) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(playlistStartupTimeout)
	defer timeout.Stop()

	for {
		ready, err := playableOutputReadyWithBuffer(runtime.dir, runtime.request.Plan.MinimumReadyBufferSeconds)
		if err != nil {
			return err
		}
		if ready {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-runtime.done:
			ready, _ := playableOutputReadyWithBuffer(runtime.dir, runtime.request.Plan.MinimumReadyBufferSeconds)
			if ready {
				return nil
			}
			return m.outputUnavailable(runtime, "initial HLS output")
		case <-timeout.C:
			return fmt.Errorf("%w: ffmpeg did not create a playable HLS fragment in time", domain.ErrUnavailable)
		case <-ticker.C:
		}
	}
}

// waitForPlaylist is retained as a narrow compatibility helper for existing
// adapter tests. Readiness now means a playlist plus every file needed to play
// its first fragment, not merely a non-empty index.m3u8.
func (m *Manager) waitForPlaylist(ctx context.Context, runtime *runtimeSession) error {
	return m.waitForPlayableOutput(ctx, runtime)
}

func playableOutputReady(directory string) (bool, error) {
	return playableOutputReadyWithBuffer(directory, 0)
}

func playableOutputReadyWithBuffer(directory string, minimumBufferSeconds int) (bool, error) {
	playlistPath := filepath.Join(directory, "index.m3u8")
	data, err := os.ReadFile(playlistPath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read generated playlist: %w", err)
	}
	if len(data) == 0 {
		return false, nil
	}

	playlist := string(data)
	initURI, segmentURI := firstPlaylistFiles(playlist)
	if segmentURI == "" {
		return false, nil
	}
	bufferedSeconds, lastSegmentURI := playlistBufferedDuration(playlist)
	if minimumBufferSeconds > 0 && bufferedSeconds < float64(minimumBufferSeconds) && !strings.Contains(playlist, "#EXT-X-ENDLIST") {
		return false, nil
	}
	for _, uri := range []string{initURI, segmentURI, lastSegmentURI} {
		if uri == "" {
			continue
		}
		clean, ok := safePlaylistPath(uri)
		if !ok {
			return false, fmt.Errorf("%w: unsafe generated playlist URI %q", domain.ErrInvalidInput, uri)
		}
		info, statErr := os.Stat(filepath.Join(directory, clean))
		if errors.Is(statErr, os.ErrNotExist) || (statErr == nil && info.Size() == 0) {
			return false, nil
		}
		if statErr != nil {
			return false, fmt.Errorf("inspect generated HLS file: %w", statErr)
		}
	}
	return true, nil
}

func playlistBufferedDuration(playlist string) (float64, string) {
	var total float64
	var pending float64
	lastSegment := ""
	for _, rawLine := range strings.Split(playlist, "\n") {
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, "#EXTINF:") {
			value := strings.TrimSuffix(strings.TrimPrefix(line, "#EXTINF:"), ",")
			pending, _ = strconv.ParseFloat(value, 64)
			continue
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lastSegment = line
		if pending > 0 {
			total += pending
			pending = 0
		}
	}
	return total, lastSegment
}

func firstPlaylistFiles(playlist string) (initURI, segmentURI string) {
	for _, rawLine := range strings.Split(playlist, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#EXT-X-MAP:") && initURI == "" {
			initURI = attributeValue(line, "URI")
			continue
		}
		if !strings.HasPrefix(line, "#") && segmentURI == "" {
			segmentURI = line
		}
	}
	return initURI, segmentURI
}

func attributeValue(line, name string) string {
	needle := name + "=\""
	start := strings.Index(line, needle)
	if start < 0 {
		return ""
	}
	start += len(needle)
	end := strings.IndexByte(line[start:], '"')
	if end < 0 {
		return ""
	}
	return line[start : start+end]
}

func safePlaylistPath(raw string) (string, bool) {
	withoutQuery := raw
	if index := strings.IndexAny(withoutQuery, "?#"); index >= 0 {
		withoutQuery = withoutQuery[:index]
	}
	decoded, err := url.PathUnescape(withoutQuery)
	if err != nil {
		return "", false
	}
	clean := filepath.Clean(filepath.FromSlash(decoded))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", false
	}
	return clean, true
}

func (m *Manager) outputUnavailable(runtime *runtimeSession, file string) error {
	message := strings.TrimSpace(readLogTail(filepath.Join(runtime.dir, "ffmpeg.log"), 2400))
	if message == "" {
		if runtimeErr := runtime.getErr(); runtimeErr != nil {
			message = runtimeErr.Error()
		} else {
			message = "ffmpeg stopped before producing the requested output"
		}
	}
	return fmt.Errorf("%w: %s (%s)", domain.ErrUnavailable, message, file)
}

func readLogTail(path string, maximum int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if maximum > 0 && len(data) > maximum {
		data = data[len(data)-maximum:]
	}
	return string(data)
}

func (m *Manager) cleanupLoop() {
	defer close(m.stopped)
	if m.idleTTL <= 0 {
		<-m.stop
		return
	}
	interval := m.idleTTL / 4
	if interval > 15*time.Second {
		interval = 15 * time.Second
	}
	if interval < 2*time.Second {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			m.cleanupIdle()
		}
	}
}

func (m *Manager) cleanupIdle() {
	cutoff := time.Now().UTC().Add(-m.idleTTL)
	m.mu.RLock()
	ids := make([]string, 0)
	for id, runtime := range m.sessions {
		if runtime.session.LastAccessAt.Before(cutoff) {
			ids = append(ids, id)
		}
	}
	m.mu.RUnlock()
	for _, id := range ids {
		slog.Info("stopping idle playback session", "session", id)
		m.removeSession(id, false)
	}
}

func (m *Manager) removeSession(id string, immediate bool) {
	m.mu.Lock()
	runtime, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
		if runtime.refCount > 0 {
			runtime.refCount--
		}
	}
	if !ok {
		m.mu.Unlock()
		return
	}
	remaining := runtime.refCount
	if remaining > 0 || runtime.persistent {
		m.mu.Unlock()
		return
	}
	if runtime.stopTimer != nil {
		runtime.stopTimer.Stop()
		runtime.stopTimer = nil
	}
	if !immediate && runtime.command != nil {
		if runtime.cachePolicy == "finish" {
			m.mu.Unlock()
			return
		}
		if runtime.gracePeriod > 0 {
			runtime.stopTimer = time.AfterFunc(runtime.gracePeriod, func() {
				m.mu.Lock()
				if runtime.refCount != 0 {
					m.mu.Unlock()
					return
				}
				runtime.stopTimer = nil
				m.mu.Unlock()
				m.stopRuntime(runtime, true)
			})
			m.mu.Unlock()
			return
		}
	}
	m.mu.Unlock()
	m.stopRuntime(runtime, true)
}

func (m *Manager) stopRuntime(runtime *runtimeSession, removeFiles bool) {
	if runtime == nil {
		return
	}
	if runtime.command != nil && runtime.command.Process != nil {
		select {
		case <-runtime.done:
		default:
			_ = runtime.command.Process.Kill()
			select {
			case <-runtime.done:
			case <-time.After(processStopTimeout):
				slog.Warn("ffmpeg process did not exit after kill", "session", runtime.session.ID, "pid", runtime.command.Process.Pid)
			}
		}
	}
	if removeFiles && !runtime.persistent {
		removeDirectoryWithRetry(runtime.dir)
	}
	slog.Info("playback session stopped", "session", runtime.session.ID)
}

func removeDirectoryWithRetry(path string) {
	for attempt := 0; attempt < 4; attempt++ {
		if err := os.RemoveAll(path); err == nil {
			return
		}
		time.Sleep(time.Duration(attempt+1) * 75 * time.Millisecond)
	}
}

func isExpectedGeneratedFile(relativePath string) bool {
	base := strings.ToLower(filepath.Base(relativePath))
	if base == "index.m3u8" || base == "init.mp4" {
		return true
	}
	extension := strings.ToLower(filepath.Ext(base))
	return strings.HasPrefix(base, "segment_") && (extension == ".m4s" || extension == ".ts")
}

func countsAsPlaybackActivity(relativePath string) bool {
	base := strings.ToLower(filepath.Base(relativePath))
	if base == "init.mp4" {
		return true
	}
	extension := strings.ToLower(filepath.Ext(base))
	return strings.HasPrefix(base, "segment_") && (extension == ".m4s" || extension == ".ts")
}

func cacheLookupEligible(request ports.TranscodeStartRequest) bool {
	if request.Plan.VideoAction != domain.TrackActionTranscode || request.Plan.CachePolicy == "" || request.Plan.CachePolicy == "off" {
		return false
	}
	return request.Subtitle == nil || !request.Subtitle.RequiresBurnIn
}

func cacheWriteEligible(request ports.TranscodeStartRequest) bool {
	return cacheLookupEligible(request)
}

func alignedCacheStartMS(value int64) int64 {
	value = nonNegativeMS(value)
	if value == 0 || cacheRangeAlignmentMS <= 0 {
		return value
	}
	return value - value%cacheRangeAlignmentMS
}

func nonNegativeMS(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func transcodeVariantKey(request ports.TranscodeStartRequest, info fs.FileInfo) string {
	value := strings.Join([]string{
		request.Media.File.Path, strconv.FormatInt(info.Size(), 10), strconv.FormatInt(info.ModTime().UnixNano(), 10),
		request.Plan.Container, request.Plan.QualityProfile, request.Plan.VideoCodec, request.Plan.VideoEncoder, request.Plan.VideoDecoder, request.Plan.VideoPreset,
		strconv.Itoa(request.Plan.VideoCRF), request.Plan.PixelFormat, strconv.Itoa(request.Plan.TargetWidth), strconv.Itoa(request.Plan.TargetHeight),
		request.Plan.AudioCodec, string(request.Plan.AudioAction), selectedTrackKey(request.Plan.SelectedAudioTrackIndex),
		strconv.FormatBool(request.Plan.ToneMap), strconv.FormatBool(request.Plan.PreserveHDR), request.Plan.SourceHDRFormat,
		request.Plan.TranscodeRevision, request.Plan.EncoderBuild,
	}, "\x00")
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:16])
}

func transcodeCacheKey(request ports.TranscodeStartRequest, info fs.FileInfo) string {
	variant := transcodeVariantKey(request, info)
	value := variant + "\x00" + strconv.FormatInt(request.Plan.StartPositionMS, 10)
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:16])
}

var executableFingerprintCache = struct {
	sync.Mutex
	values map[string]string
}{values: map[string]string{}}

func executableFingerprint(executable string) string {
	resolved := executable
	if path, err := exec.LookPath(executable); err == nil {
		resolved = path
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(executable))
	}
	statKey := strings.Join([]string{
		strings.ToLower(filepath.Clean(resolved)),
		strconv.FormatInt(info.Size(), 10),
		strconv.FormatInt(info.ModTime().UnixNano(), 10),
	}, "|")
	executableFingerprintCache.Lock()
	if cached := executableFingerprintCache.values[statKey]; cached != "" {
		executableFingerprintCache.Unlock()
		return cached
	}
	executableFingerprintCache.Unlock()
	file, err := os.Open(resolved)
	if err != nil {
		return statKey
	}
	sum := sha256.New()
	_, copyErr := io.Copy(sum, file)
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		return statKey
	}
	fingerprint := hex.EncodeToString(sum.Sum(nil))
	executableFingerprintCache.Lock()
	executableFingerprintCache.values[statKey] = fingerprint
	executableFingerprintCache.Unlock()
	return fingerprint
}

func selectedTrackKey(index *int) string {
	if index == nil {
		return "default"
	}
	return strconv.Itoa(*index)
}
func fileNonEmpty(path string) bool { info, err := os.Stat(path); return err == nil && info.Size() > 0 }

func newPlaybackSession(id, mediaID string, plan domain.PlaybackPlan, now time.Time) domain.PlaybackSession {
	playlistURL := "/api/v1/playback/sessions/" + id + "/index.m3u8"
	plan.PlaylistURL = playlistURL
	return domain.PlaybackSession{ID: id, MediaID: mediaID, Plan: plan, PlaylistURL: playlistURL, CreatedAt: now, LastAccessAt: now}
}

func (m *Manager) enforceCacheLimit(maximumBytes, reserveBytes int64, reservePercent int) {
	active := map[string]struct{}{}
	m.mu.RLock()
	for key := range m.activeJobs {
		active[key] = struct{}{}
	}
	for _, runtime := range m.sessions {
		if runtime.cacheKey != "" {
			active[runtime.cacheKey] = struct{}{}
		}
	}
	m.mu.RUnlock()
	enforceCacheLimit(filepath.Join(filepath.Dir(m.root), "optimized-media"), maximumBytes, reserveBytes, reservePercent, active)
}

// ApplyStoragePolicy immediately enforces a saved cache policy. It is safe to
// call while playback is active because active cache keys are always excluded.
func (m *Manager) ApplyStoragePolicy(policy string, maximumBytes, reserveBytes int64, reservePercent int) {
	if strings.EqualFold(strings.TrimSpace(policy), "off") {
		maximumBytes = 0
	}
	m.enforceCacheLimit(maximumBytes, reserveBytes, reservePercent)
}

func (m *Manager) OptimizedCacheUsage() int64 {
	return cacheRootSize(filepath.Join(filepath.Dir(m.root), "optimized-media"))
}

func (m *Manager) ClearOptimizedCache() int64 {
	root := filepath.Join(filepath.Dir(m.root), "optimized-media")
	before := cacheRootSize(root)
	m.enforceCacheLimit(0, 0, 0)
	after := cacheRootSize(root)
	return before - after
}

func newSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

func isWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

var _ ports.TranscodeSessionManager = (*Manager)(nil)
