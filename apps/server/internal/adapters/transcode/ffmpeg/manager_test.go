package ffmpeg

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

func TestWaitForPlaylistRequiresInitAndFirstFragment(t *testing.T) {
	directory := t.TempDir()
	runtime := &runtimeSession{dir: directory, done: make(chan struct{})}
	manager := &Manager{}

	playlist := "#EXTM3U\n#EXT-X-MAP:URI=\"init.mp4\"\n#EXTINF:2.0,\nsegment_000000.m4s\n"
	if err := os.WriteFile(filepath.Join(directory, "index.m3u8"), []byte(playlist), 0o600); err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(90 * time.Millisecond)
		_ = os.WriteFile(filepath.Join(directory, "init.mp4"), []byte("init"), 0o600)
		_ = os.WriteFile(filepath.Join(directory, "segment_000000.m4s"), []byte("segment"), 0o600)
		close(runtime.done)
	}()

	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.waitForPlaylist(ctx, runtime); err != nil {
		t.Fatalf("expected playable output to be accepted, got %v", err)
	}
	if elapsed := time.Since(started); elapsed < 70*time.Millisecond {
		t.Fatalf("readiness returned before init and segment were available: %v", elapsed)
	}
}

func TestPlayableOutputWaitsForRequestedDLNABuffer(t *testing.T) {
	directory := t.TempDir()
	for index := 0; index < 3; index++ {
		name := fmt.Sprintf("segment_%06d.ts", index)
		if err := os.WriteFile(filepath.Join(directory, name), []byte("segment"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	playlist := "#EXTM3U\n#EXTINF:2.0,\nsegment_000000.ts\n#EXTINF:2.0,\nsegment_000001.ts\n#EXTINF:2.0,\nsegment_000002.ts\n"
	if err := os.WriteFile(filepath.Join(directory, "index.m3u8"), []byte(playlist), 0o600); err != nil {
		t.Fatal(err)
	}
	ready, err := playableOutputReadyWithBuffer(directory, 12)
	if err != nil {
		t.Fatal(err)
	}
	if ready {
		t.Fatal("six seconds of media must not satisfy a twelve-second DLNA startup buffer")
	}
	playlist += "#EXTINF:6.0,\nsegment_000003.ts\n"
	if err := os.WriteFile(filepath.Join(directory, "segment_000003.ts"), []byte("segment"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "index.m3u8"), []byte(playlist), 0o600); err != nil {
		t.Fatal(err)
	}
	ready, err = playableOutputReadyWithBuffer(directory, 12)
	if err != nil || !ready {
		t.Fatalf("twelve seconds of complete segments should be ready, ready=%v err=%v", ready, err)
	}
}

func TestPlayableOutputRejectsTraversalURI(t *testing.T) {
	directory := t.TempDir()
	playlist := "#EXTM3U\n#EXT-X-MAP:URI=\"../init.mp4\"\n#EXTINF:2.0,\nsegment_000000.m4s\n"
	if err := os.WriteFile(filepath.Join(directory, "index.m3u8"), []byte(playlist), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "segment_000000.m4s"), []byte("segment"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := playableOutputReady(directory); err == nil {
		t.Fatal("expected unsafe playlist URI to be rejected")
	}
}

func TestManagerProducesPlayableFMP4AssetsInSessionDirectory(t *testing.T) {
	binary, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not available in PATH")
	}

	work := t.TempDir()
	source := filepath.Join(work, "source.mkv")
	generate := exec.Command(
		binary,
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc2=size=320x180:rate=24",
		"-f", "lavfi", "-i", "sine=frequency=880:sample_rate=48000",
		"-t", "4",
		"-c:v", "mpeg4", "-q:v", "5",
		"-c:a", "aac", "-b:a", "96k",
		source,
	)
	if output, generateErr := generate.CombinedOutput(); generateErr != nil {
		t.Fatalf("create source fixture: %v\n%s", generateErr, output)
	}

	root := filepath.Join(work, "sessions")
	manager, err := NewManager(binary, root, 30*time.Second, SoftwareH264AACProfile{VideoThreads: 2, SegmentSeconds: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	session, err := manager.Start(ctx, ports.TranscodeStartRequest{
		Media: domain.MediaItem{ID: "media_test", File: domain.MediaFile{Path: source}},
		Plan: domain.PlaybackPlan{
			Mode:             domain.PlaybackModeTranscode,
			VideoAction:      domain.TrackActionTranscode,
			AudioAction:      domain.TrackActionTranscode,
			TranscodeProfile: "software-h264-aac",
		},
	})
	if err != nil {
		t.Fatalf("start transcode: %v", err)
	}

	playlistFile, err := manager.Open(ctx, session.ID, "index.m3u8")
	if err != nil {
		t.Fatalf("open playlist: %v", err)
	}
	playlistData, err := io.ReadAll(playlistFile)
	_ = playlistFile.Close()
	if err != nil {
		t.Fatal(err)
	}
	initURI, segmentURI := firstPlaylistFiles(string(playlistData))
	if initURI != "init.mp4" {
		t.Fatalf("unexpected init URI %q in playlist:\n%s", initURI, playlistData)
	}
	if !strings.HasPrefix(segmentURI, "segment_") {
		t.Fatalf("unexpected segment URI %q in playlist:\n%s", segmentURI, playlistData)
	}

	for _, relative := range []string{initURI, segmentURI} {
		file, openErr := manager.Open(ctx, session.ID, relative)
		if openErr != nil {
			t.Fatalf("open generated file %q: %v", relative, openErr)
		}
		info, statErr := file.Stat()
		_ = file.Close()
		if statErr != nil {
			t.Fatal(statErr)
		}
		if info.Size() == 0 {
			t.Fatalf("generated file %q is empty", relative)
		}
	}

	decodeCtx, decodeCancel := context.WithTimeout(context.Background(), 10*time.Second)
	decode := exec.CommandContext(
		decodeCtx,
		binary,
		"-hide_banner", "-loglevel", "error",
		"-i", filepath.Join(root, session.ID, "index.m3u8"),
		"-t", "0.5", "-f", "null", "-",
	)
	if output, decodeErr := decode.CombinedOutput(); decodeErr != nil {
		decodeCancel()
		t.Fatalf("decode generated HLS: %v\n%s", decodeErr, output)
	}
	decodeCancel()

	if err := manager.Stop(session.ID); err != nil {
		t.Fatalf("stop session: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, session.ID)); !os.IsNotExist(err) {
		t.Fatalf("session directory was not removed, stat error: %v", err)
	}
}

func TestParseRuntimeCapabilitiesSupportsModernAndLegacyHelp(t *testing.T) {
	modern := parseRuntimeCapabilities(`
-readrate <speed>    read input at specified rate
-readrate_initial_burst <seconds> initial burst duration
`)
	if !modern.Readrate || !modern.ReadrateInitialBurst {
		t.Fatalf("expected modern readrate capabilities, got %+v", modern)
	}

	legacy := parseRuntimeCapabilities(`
-re                 read input at native frame rate
-readrate <speed>   read input at specified rate
`)
	if !legacy.Readrate || legacy.ReadrateInitialBurst {
		t.Fatalf("expected legacy readrate without initial burst, got %+v", legacy)
	}

	veryOld := parseRuntimeCapabilities(`-re read input at native frame rate`)
	if veryOld.Readrate || veryOld.ReadrateInitialBurst {
		t.Fatalf("expected no readrate capabilities, got %+v", veryOld)
	}

	hardware := parseRuntimeCapabilities("Hardware acceleration methods:\nqsv\ncuda\nd3d11va\n")
	if !hardware.QSV || !hardware.CUDA || !hardware.D3D11VA {
		t.Fatalf("expected dedicated hardware acceleration methods, got %+v", hardware)
	}
}

func TestRuntimeCapabilitiesIncludesDedicatedHWAccelsProbe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell fixture is only used by host-side adapter tests")
	}
	directory := t.TempDir()
	binary := filepath.Join(directory, "fake-ffmpeg")
	script := `#!/bin/sh
case "$*" in
  *"-h full"*) printf '%s\n' '-readrate <speed>' ;;
  *"-encoders"*) printf '%s\n' ' V..... h264_nvenc' ;;
  *"-filters"*) printf '%s\n' ' ... libvmaf' ;;
  *"-hwaccels"*) printf '%s\n' 'Hardware acceleration methods:' 'cuda' 'qsv' 'd3d11va' ;;
esac
`
	if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	manager := &Manager{
		binary:          func() string { return binary },
		capabilityCache: map[string]RuntimeCapabilities{},
	}
	capabilities := manager.runtimeCapabilities()
	if !capabilities.CUDA || !capabilities.QSV || !capabilities.D3D11VA {
		t.Fatalf("dedicated -hwaccels output was not merged: %+v", capabilities)
	}
	if !capabilities.H264NVENC || !capabilities.Readrate || !capabilities.LibVMAF {
		t.Fatalf("parallel capability probes were not merged: %+v", capabilities)
	}
}

func generateCacheFixture(t *testing.T, binary, path string, seconds int) {
	t.Helper()
	command := exec.Command(
		binary,
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc2=size=320x180:rate=24",
		"-f", "lavfi", "-i", "sine=frequency=660:sample_rate=48000",
		"-t", fmt.Sprintf("%d", seconds),
		"-c:v", "mpeg4", "-q:v", "5",
		"-c:a", "aac", "-b:a", "96k",
		path,
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("create cache source fixture: %v\n%s", err, output)
	}
}

func cacheTestRequest(source string, durationMS int64, policy string) ports.TranscodeStartRequest {
	return ports.TranscodeStartRequest{
		Media: domain.MediaItem{
			ID:   "cache-media",
			File: domain.MediaFile{Path: source},
			Technical: domain.TechnicalMetadata{
				ProbeVersion: 2,
				Container:    "mkv",
				DurationMS:   durationMS,
				VideoCodec:   "mpeg4",
				AudioCodec:   "aac",
				Width:        320,
				Height:       180,
			},
		},
		Plan: domain.PlaybackPlan{
			Mode:               domain.PlaybackModeTranscode,
			VideoAction:        domain.TrackActionTranscode,
			AudioAction:        domain.TrackActionTranscode,
			VideoCodec:         "h264",
			VideoEncoder:       "software",
			HardwarePreference: "software",
			VideoPreset:        "veryfast",
			VideoCRF:           18,
			PixelFormat:        "yuv420p",
			TranscodeProfile:   "adaptive-video-aac",
			QualityProfile:     "balanced",
			CachePolicy:        policy,
			CacheMaxBytes:      1 << 30,
		},
	}
}

func runtimeForSession(t *testing.T, manager *Manager, id string) *runtimeSession {
	t.Helper()
	manager.mu.RLock()
	runtime := manager.sessions[id]
	manager.mu.RUnlock()
	if runtime == nil {
		t.Fatalf("runtime session %s not found", id)
	}
	return runtime
}

func waitRuntime(t *testing.T, runtime *runtimeSession, timeout time.Duration) {
	t.Helper()
	select {
	case <-runtime.done:
	case <-time.After(timeout):
		t.Fatalf("ffmpeg runtime did not finish in %s", timeout)
	}
}

func TestCompletedJITCacheIsReusedAtAbsoluteResumePosition(t *testing.T) {
	binary, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not available in PATH")
	}
	work := t.TempDir()
	source := filepath.Join(work, "cache-source.mkv")
	generateCacheFixture(t, binary, source, 4)
	manager, err := NewManager(binary, filepath.Join(work, "sessions"), 30*time.Second, SoftwareH264AACProfile{VideoThreads: 2, SegmentSeconds: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	request := cacheTestRequest(source, 4_000, "watched")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	first, err := manager.Start(ctx, request)
	if err != nil {
		t.Fatalf("start first cached transcode: %v", err)
	}
	firstRuntime := runtimeForSession(t, manager, first.ID)
	waitRuntime(t, firstRuntime, 15*time.Second)
	if firstRuntime.getErr() != nil {
		t.Fatalf("first cache producer failed: %v", firstRuntime.getErr())
	}
	if err := manager.Stop(first.ID); err != nil {
		t.Fatal(err)
	}

	request.Plan.StartPositionMS = 2_000
	second, err := manager.Start(ctx, request)
	if err != nil {
		t.Fatalf("reuse completed cache: %v", err)
	}
	if !second.Plan.CacheHit || second.Plan.InitialPositionMS != 2_000 || second.Plan.StartPositionMS != 0 {
		t.Fatalf("unexpected completed-cache plan: %+v", second.Plan)
	}
	secondRuntime := runtimeForSession(t, manager, second.ID)
	if secondRuntime.command != nil || !secondRuntime.persistent {
		t.Fatalf("cache hit unexpectedly started a new encoder: %+v", secondRuntime)
	}
	file, err := manager.Open(ctx, second.ID, "index.m3u8")
	if err != nil {
		t.Fatalf("open cached playlist: %v", err)
	}
	_ = file.Close()
	if err := manager.Stop(second.ID); err != nil {
		t.Fatal(err)
	}
}

func TestJITCacheCreatedFromResumePositionIsReusableForLaterPositions(t *testing.T) {
	binary, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not available in PATH")
	}
	work := t.TempDir()
	source := filepath.Join(work, "range-cache-source.mkv")
	generateCacheFixture(t, binary, source, 10)
	manager, err := NewManager(binary, filepath.Join(work, "sessions"), 30*time.Second, SoftwareH264AACProfile{VideoThreads: 2, SegmentSeconds: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	request := cacheTestRequest(source, 10_000, "watched")
	request.Plan.StartPositionMS = 3_200
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	first, err := manager.Start(ctx, request)
	if err != nil {
		t.Fatalf("start ranged cache producer: %v", err)
	}
	if first.Plan.StartPositionMS != 2_000 || first.Plan.InitialPositionMS != 1_200 {
		t.Fatalf("unexpected aligned range plan: %+v", first.Plan)
	}
	firstRuntime := runtimeForSession(t, manager, first.ID)
	waitRuntime(t, firstRuntime, 20*time.Second)
	if firstRuntime.getErr() != nil {
		t.Fatalf("ranged cache producer failed: %v", firstRuntime.getErr())
	}
	if err := manager.Stop(first.ID); err != nil {
		t.Fatal(err)
	}

	request.Plan.StartPositionMS = 5_100
	second, err := manager.Start(ctx, request)
	if err != nil {
		t.Fatalf("reuse ranged cache: %v", err)
	}
	if !second.Plan.CacheHit || second.Plan.StartPositionMS != 2_000 || second.Plan.InitialPositionMS != 3_100 {
		t.Fatalf("unexpected ranged cache hit plan: %+v", second.Plan)
	}
	secondRuntime := runtimeForSession(t, manager, second.ID)
	if secondRuntime.command != nil || !secondRuntime.persistent {
		t.Fatalf("ranged cache hit unexpectedly started an encoder: %+v", secondRuntime)
	}
	if err := manager.Stop(second.ID); err != nil {
		t.Fatal(err)
	}
}

func TestPartialJITCacheCanResumeAndProduceDecodablePlaylist(t *testing.T) {
	binary, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not available in PATH")
	}
	work := t.TempDir()
	source := filepath.Join(work, "partial-source.mkv")
	generateCacheFixture(t, binary, source, 8)
	manager, err := NewManager(binary, filepath.Join(work, "sessions"), 30*time.Second, SoftwareH264AACProfile{VideoThreads: 2, SegmentSeconds: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	request := cacheTestRequest(source, 8_000, "watched")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	first, err := manager.Start(ctx, request)
	if err != nil {
		t.Fatalf("start partial cache producer: %v", err)
	}
	firstRuntime := runtimeForSession(t, manager, first.ID)
	if err := manager.Stop(first.ID); err != nil {
		t.Fatal(err)
	}
	waitRuntime(t, firstRuntime, 10*time.Second)

	entries, err := os.ReadDir(filepath.Join(work, "optimized-media"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected one partial cache entry, entries=%d err=%v", len(entries), err)
	}
	cacheDir := filepath.Join(work, "optimized-media", entries[0].Name())
	metadata, err := readCacheMetadata(cacheDir)
	if err != nil || metadata.Complete || metadata.SegmentCount == 0 {
		t.Fatalf("expected reusable partial cache, metadata=%+v err=%v", metadata, err)
	}

	second, err := manager.Start(ctx, request)
	if err != nil {
		t.Fatalf("resume partial cache: %v", err)
	}
	secondRuntime := runtimeForSession(t, manager, second.ID)
	if !secondRuntime.request.CacheAppend || secondRuntime.request.CacheResumeFromMS <= 0 {
		t.Fatalf("expected append resume request, got %+v", secondRuntime.request)
	}
	waitRuntime(t, secondRuntime, 20*time.Second)
	if secondRuntime.getErr() != nil {
		t.Fatalf("resumed cache producer failed: %v\n%s", secondRuntime.getErr(), readLogTail(filepath.Join(secondRuntime.dir, "ffmpeg.log"), 4000))
	}
	if err := manager.Stop(second.ID); err != nil {
		t.Fatal(err)
	}
	if !cachedOutputReady(cacheDir) {
		t.Fatalf("resumed cache was not finalized: %+v", inspectPlaylistCache(cacheDir))
	}

	decodeCtx, decodeCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer decodeCancel()
	decode := exec.CommandContext(decodeCtx, binary, "-hide_banner", "-loglevel", "error", "-i", filepath.Join(cacheDir, "index.m3u8"), "-t", "6", "-f", "null", "-")
	if output, err := decode.CombinedOutput(); err != nil {
		t.Fatalf("decode resumed cache: %v\n%s\nplaylist:\n%s", err, output, mustReadTestFile(t, filepath.Join(cacheDir, "index.m3u8")))
	}
}

func mustReadTestFile(t *testing.T, path string) string {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(payload)
}

func TestConcurrentClientsShareOneJITEncoder(t *testing.T) {
	binary, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not available in PATH")
	}
	work := t.TempDir()
	source := filepath.Join(work, "shared-source.mkv")
	generateCacheFixture(t, binary, source, 30)
	manager, err := NewManager(binary, filepath.Join(work, "sessions"), 30*time.Second, SoftwareH264AACProfile{VideoThreads: 1, SegmentSeconds: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	request := cacheTestRequest(source, 30_000, "watched")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	first, err := manager.Start(ctx, request)
	if err != nil {
		t.Fatalf("start first shared client: %v", err)
	}
	firstRuntime := runtimeForSession(t, manager, first.ID)
	second, err := manager.Start(ctx, request)
	if err != nil {
		t.Fatalf("start second shared client: %v", err)
	}
	secondRuntime := runtimeForSession(t, manager, second.ID)
	if firstRuntime != secondRuntime {
		t.Fatal("concurrent clients started separate encoder runtimes")
	}
	manager.mu.RLock()
	refs := firstRuntime.refCount
	jobs := len(manager.activeJobs)
	manager.mu.RUnlock()
	if refs != 2 || jobs != 1 {
		t.Fatalf("unexpected shared encoder state: refs=%d jobs=%d", refs, jobs)
	}
	if err := manager.Stop(first.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-firstRuntime.done:
		t.Fatal("encoder stopped while another client was still attached")
	default:
	}
	if err := manager.Stop(second.ID); err != nil {
		t.Fatal(err)
	}
	waitRuntime(t, firstRuntime, 10*time.Second)
}

func TestCacheLimitEvictsOldestInactiveEntryAndProtectsActive(t *testing.T) {
	root := t.TempDir()
	create := func(name string, size int, accessed time.Time) {
		directory := filepath.Join(root, name)
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "segment_000000.m4s"), make([]byte, size), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := writeCacheMetadata(directory, cacheMetadata{Version: 1, Key: name, LastAccessAt: accessed}); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	create("old", 128, now.Add(-2*time.Hour))
	create("active", 128, now.Add(-3*time.Hour))
	create("new", 128, now.Add(-time.Hour))
	total := directorySize(root)
	oldSize := directorySize(filepath.Join(root, "old"))
	enforceCacheLimit(root, total-oldSize+1, 0, 0, map[string]struct{}{"active": {}})
	if _, err := os.Stat(filepath.Join(root, "old")); !os.IsNotExist(err) {
		t.Fatalf("oldest inactive cache was not evicted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "active")); err != nil {
		t.Fatalf("active cache was evicted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "new")); err != nil {
		t.Fatalf("new cache was evicted: %v", err)
	}
}

func TestExecutableFingerprintUsesContentInsteadOfPathOrModifiedTime(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "ffmpeg-a.exe")
	second := filepath.Join(root, "ffmpeg-b.exe")
	content := []byte("same bundled ffmpeg build")
	if err := os.WriteFile(first, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(first, time.Unix(1_600_000_000, 0), time.Unix(1_600_000_000, 0)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(second, time.Unix(1_700_000_000, 0), time.Unix(1_700_000_000, 0)); err != nil {
		t.Fatal(err)
	}
	firstFingerprint := executableFingerprint(first)
	secondFingerprint := executableFingerprint(second)
	if firstFingerprint != secondFingerprint {
		t.Fatalf("identical executable content produced different fingerprints: %q != %q", firstFingerprint, secondFingerprint)
	}
	if err := os.WriteFile(second, []byte("different bundled ffmpeg build"), 0o600); err != nil {
		t.Fatal(err)
	}
	if changed := executableFingerprint(second); changed == firstFingerprint {
		t.Fatal("changed executable content reused the old fingerprint")
	}
}

func TestManagerStartupRemovesOrphanedSessionDirectories(t *testing.T) {
	root := filepath.Join(t.TempDir(), "transcodes")
	orphan := filepath.Join(root, "session-from-crash", "segment_000000.m4s")
	if err := os.MkdirAll(filepath.Dir(orphan), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(orphan, []byte("orphaned segment"), 0o600); err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager("ffmpeg", root, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("startup preserved orphaned transcode sessions: %+v", entries)
	}
}

func TestSoftwareHEVCMain10ProducesDecodableTenBitHLS(t *testing.T) {
	binary, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not available in PATH")
	}
	if output, err := exec.Command(binary, "-hide_banner", "-encoders").CombinedOutput(); err != nil || !strings.Contains(string(output), "libx265") {
		t.Skip("ffmpeg does not provide libx265")
	}
	probe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe is not available in PATH")
	}

	work := t.TempDir()
	source := filepath.Join(work, "main10-source.mkv")
	generateCacheFixture(t, binary, source, 3)
	manager, err := NewManager(binary, filepath.Join(work, "sessions"), 30*time.Second, SoftwareH264AACProfile{VideoThreads: 2, SegmentSeconds: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	request := cacheTestRequest(source, 3_000, "off")
	request.Media.Technical.BitDepth = 10
	request.Plan.VideoCodec = "hevc"
	request.Plan.VideoCodecCandidates = []string{"hevc"}
	request.Plan.PreferredVideoCodec = "hevc"
	request.Plan.VideoEncoder = "software"
	request.Plan.HardwarePreference = "software"
	request.Plan.VideoPreset = "ultrafast"
	request.Plan.PixelFormat = "yuv420p10le"
	request.Plan.QualityProfile = "original"

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	session, err := manager.Start(ctx, request)
	if err != nil {
		t.Fatalf("start HEVC Main10 session: %v", err)
	}
	runtime := runtimeForSession(t, manager, session.ID)
	waitRuntime(t, runtime, 30*time.Second)
	if runtime.getErr() != nil {
		t.Fatalf("HEVC Main10 encoder failed: %v\n%s", runtime.getErr(), readLogTail(filepath.Join(runtime.dir, "ffmpeg.log"), 5000))
	}

	command := exec.CommandContext(ctx, probe,
		"-hide_banner", "-loglevel", "error",
		"-show_entries", "stream=codec_name,profile,pix_fmt",
		"-of", "default=noprint_wrappers=1",
		filepath.Join(runtime.dir, "index.m3u8"),
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("probe HEVC Main10 output: %v\n%s", err, output)
	}
	text := strings.ToLower(string(output))
	if !strings.Contains(text, "codec_name=hevc") || !strings.Contains(text, "pix_fmt=yuv420p10le") {
		t.Fatalf("unexpected HEVC Main10 stream metadata:\n%s", output)
	}
}
