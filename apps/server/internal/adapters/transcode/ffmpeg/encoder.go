package ffmpeg

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/platform/childprocess"
)

const (
	encoderProbeTimeout     = 15 * time.Second
	encoderNegativeCacheTTL = 30 * time.Second
)

type encoderProbeCacheEntry struct {
	Available bool
	CheckedAt time.Time
}

func encoderName(codec, family string) string {
	codec = strings.ToLower(strings.TrimSpace(codec))
	if codec == "h265" {
		codec = "hevc"
	}
	switch family {
	case "qsv":
		if codec == "av1" {
			return "av1_qsv"
		}
		if codec == "hevc" {
			return "hevc_qsv"
		}
		return "h264_qsv"
	case "nvenc":
		if codec == "av1" {
			return "av1_nvenc"
		}
		if codec == "hevc" {
			return "hevc_nvenc"
		}
		return "h264_nvenc"
	case "amf":
		if codec == "av1" {
			return "av1_amf"
		}
		if codec == "hevc" {
			return "hevc_amf"
		}
		return "h264_amf"
	default:
		return ""
	}
}

func capabilityAdvertisesEncoder(c RuntimeCapabilities, codec, family string) bool {
	codec = strings.ToLower(strings.TrimSpace(codec))
	if codec == "h265" {
		codec = "hevc"
	}
	switch family {
	case "qsv":
		if codec == "av1" {
			return c.AV1QSV
		}
		if codec == "hevc" {
			return c.HEVCQSV
		}
		return c.H264QSV
	case "nvenc":
		if codec == "av1" {
			return c.AV1NVENC
		}
		if codec == "hevc" {
			return c.HEVCNVENC
		}
		return c.H264NVENC
	case "amf":
		if codec == "av1" {
			return c.AV1AMF
		}
		if codec == "hevc" {
			return c.HEVCAMF
		}
		return c.H264AMF
	}
	return false
}

func encoderFamilies(preference string, source domain.TechnicalMetadata) []string {
	switch strings.ToLower(strings.TrimSpace(preference)) {
	case "qsv":
		return []string{"qsv"}
	case "nvenc":
		return []string{"nvenc"}
	case "amf":
		return []string{"amf"}
	case "software":
		return nil
	default:
		// Prefer a discrete NVIDIA encoder whenever one is available. The old
		// SDR order selected Intel QSV before an RTX-class GPU, leaving the
		// stronger real-time encoder idle exactly when a long-lived buffer needed
		// sustained headroom. Missing families are skipped by capability checks.
		return []string{"nvenc", "qsv", "amf"}
	}
}

func (m *Manager) resolveEncoder(ctx context.Context, plan domain.PlaybackPlan, source domain.TechnicalMetadata, capabilities RuntimeCapabilities) domain.PlaybackPlan {
	if plan.VideoAction != domain.TrackActionTranscode {
		return plan
	}
	preference := strings.ToLower(strings.TrimSpace(plan.HardwarePreference))
	if preference == "" {
		preference = strings.ToLower(strings.TrimSpace(plan.VideoEncoder))
	}
	if preference == "" {
		preference = "auto"
	}
	candidates := normalizedCodecCandidates(plan)

	if preference != "software" {
		for _, codec := range candidates {
			for _, family := range encoderFamilies(preference, source) {
				if !capabilityAdvertisesEncoder(capabilities, codec, family) {
					continue
				}
				name := encoderName(codec, family)
				if name == "" || encoderExcluded(plan, name) {
					continue
				}
				candidate := configureSelectedCodec(plan, source, codec, family, true)
				candidate.VideoDecoder = hardwareDecoderFor(candidate, source, capabilities)
				if m.encoderAvailable(ctx, name, candidate, capabilities) {
					candidate.EncoderFallbackReason = strings.TrimSpace(plan.EncoderFallbackReason)
					return candidate
				}
			}
		}
	}

	preferredCodec := strings.ToLower(strings.TrimSpace(plan.PreferredVideoCodec))
	if preferredCodec == "h265" {
		preferredCodec = "hevc"
	}
	softwareCandidates := candidates
	if preferredCodec == "" || preferredCodec == "auto" {
		// In automatic mode a CPU encoder should remain predictable and
		// real-time friendly. HEVC/AV1 are still selected automatically when a
		// validated hardware encoder is available above.
		softwareCandidates = prioritizeCodec(candidates, "h264")
	}
	for _, codec := range softwareCandidates {
		if codec == "av1" && !plan.AllowSoftwareAV1 {
			continue
		}
		if !softwareEncoderAvailable(capabilities, codec) {
			continue
		}
		selected := configureSelectedCodec(plan, source, codec, "software", false)
		selected.VideoDecoder = ""
		if preference != "auto" && preference != "software" {
			selected.EncoderFallback = true
			selected.EncoderFallbackReason = fmt.Sprintf("Valitud %s hardware encoder ei käivitunud.", strings.ToUpper(preference))
			selected.Reason = selected.EncoderFallbackReason + " KoduMedia kasutab kvaliteetset tarkvaralist encoder'it."
		} else if len(plan.ExcludedVideoEncoders) > 0 {
			selected.EncoderFallback = true
			selected.EncoderFallbackReason = "Kõik sobivad hardware encoder'id ebaõnnestusid."
			selected.Reason = selected.EncoderFallbackReason + " KoduMedia kasutab tarkvaralist fallback'i."
		} else if preferredCodec == "av1" && codec != "av1" {
			selected.EncoderFallback = true
			selected.EncoderFallbackReason = "Riistvaralist AV1 encoder'it ei leitud."
			selected.Reason = selected.EncoderFallbackReason + " KoduMedia kasutab reaalajas töökindlat H.264 kvaliteediprofiili."
		}
		return selected
	}

	fallback, _ := nextStabilityPlan(plan, source, "Valitud videocodec'i encoder puudub FFmpegist.")
	fallback.HardwarePreference = "software"
	fallback.VideoEncoder = "software"
	fallback.VideoDecoder = ""
	fallback.Reason = "Valitud videocodec'i encoder puudub FFmpegist; KoduMedia kasutab H.264 tarkvaralist fallback'i."
	return fallback
}

func encoderExcluded(plan domain.PlaybackPlan, encoder string) bool {
	encoder = strings.ToLower(strings.TrimSpace(encoder))
	for _, value := range plan.ExcludedVideoEncoders {
		if strings.EqualFold(strings.TrimSpace(value), encoder) {
			return true
		}
	}
	return false
}

func hardwareDecoderFor(plan domain.PlaybackPlan, source domain.TechnicalMetadata, capabilities RuntimeCapabilities) string {
	if plan.DisableHardwareDecode || plan.SubtitleAction == domain.TrackActionBurn {
		return ""
	}
	codec := strings.ToLower(strings.TrimSpace(source.VideoCodec))
	if codec == "h265" {
		codec = "hevc"
	}
	if plan.VideoEncoder == "nvenc" && capabilities.CUDA {
		switch codec {
		case "h264", "hevc", "av1":
			return "cuda"
		}
	}
	return ""
}

func normalizedCodecCandidates(plan domain.PlaybackPlan) []string {
	values := append([]string(nil), plan.VideoCodecCandidates...)
	if len(values) == 0 {
		values = []string{plan.VideoCodec}
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		codec := strings.ToLower(strings.TrimSpace(value))
		if codec == "h265" {
			codec = "hevc"
		}
		if codec != "h264" && codec != "hevc" && codec != "av1" {
			continue
		}
		if _, ok := seen[codec]; ok {
			continue
		}
		seen[codec] = struct{}{}
		result = append(result, codec)
	}
	if len(result) == 0 {
		result = []string{"h264"}
	}
	return result
}

func prioritizeCodec(values []string, preferred string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == preferred {
			result = append(result, value)
		}
	}
	for _, value := range values {
		if value != preferred {
			result = append(result, value)
		}
	}
	return result
}

func softwareEncoderAvailable(capabilities RuntimeCapabilities, codec string) bool {
	switch codec {
	case "av1":
		return capabilities.LibSVTAV1 || capabilities.LibAOMAV1
	case "hevc":
		return capabilities.LibX265
	default:
		return capabilities.LibX264
	}
}

func selectedPixelFormat(source domain.TechnicalMetadata, plan domain.PlaybackPlan, codec string, hardware bool) string {
	if plan.ToneMap || codec == "h264" {
		return "yuv420p"
	}
	if (source.HDRFormat != "" && plan.PreserveHDR) || source.BitDepth > 8 {
		if hardware {
			return "p010le"
		}
		return "yuv420p10le"
	}
	return "yuv420p"
}

func configureSelectedCodec(plan domain.PlaybackPlan, source domain.TechnicalMetadata, codec, encoder string, hardware bool) domain.PlaybackPlan {
	originalCodec := strings.ToLower(strings.TrimSpace(plan.VideoCodec))
	if originalCodec == "h265" {
		originalCodec = "hevc"
	}
	plan.VideoCodec = codec
	plan.VideoEncoder = encoder
	if codec != originalCodec {
		plan.EncoderFallback = true
	}
	if source.HDRFormat != "" && plan.PreserveHDR && codec == "h264" {
		plan.PreserveHDR = false
		plan.ToneMap = true
		plan.EncoderFallback = true
		plan.Reason = "HDR-võimelist reaalaja encoder'it ei leitud; KoduMedia teeb kvaliteetse HDR→SDR H.264 fallback'i."
	}
	plan.PixelFormat = selectedPixelFormat(source, plan, codec, hardware)
	if plan.ToneMap {
		plan.PreserveHDR = false
		plan.PixelFormat = "yuv420p"
	}
	return plan
}

func (m *Manager) encoderAvailable(ctx context.Context, encoder string, plan domain.PlaybackPlan, capabilities RuntimeCapabilities) bool {
	_, encoderArguments := selectVideoEncoder(plan, capabilities)
	pixelFormat := selectedPixelFormat(domain.TechnicalMetadata{HDRFormat: plan.SourceHDRFormat, BitDepth: bitDepthForPixelFormat(plan.PixelFormat)}, plan, plan.VideoCodec, plan.VideoEncoder != "software")
	if strings.TrimSpace(plan.PixelFormat) != "" {
		pixelFormat = plan.PixelFormat
	}
	key := encoderProbeKey(m.executable(), encoder, pixelFormat, encoderArguments)
	now := time.Now()
	m.encoderMu.Lock()
	if entry, ok := m.encoderCache[key]; ok {
		if entry.Available || now.Sub(entry.CheckedAt) < encoderNegativeCacheTTL {
			m.encoderMu.Unlock()
			return entry.Available
		}
		delete(m.encoderCache, key)
	}
	m.encoderMu.Unlock()

	probeCtx, cancel := context.WithTimeout(ctx, encoderProbeTimeout)
	defer cancel()
	args := []string{
		"-hide_banner", "-loglevel", "error", "-nostdin", "-y",
		"-f", "lavfi", "-i", "color=c=black:s=1920x1080:r=1",
		"-frames:v", "1", "-an", "-c:v", encoder,
	}
	args = append(args, encoderArguments...)
	if strings.TrimSpace(pixelFormat) != "" {
		args = append(args, "-pix_fmt", pixelFormat)
	}
	args = append(args, "-f", "null", "-")
	command := exec.CommandContext(probeCtx, m.executable(), args...)
	childprocess.Hide(command)
	output, err := command.CombinedOutput()
	available := err == nil
	m.encoderMu.Lock()
	m.encoderCache[key] = encoderProbeCacheEntry{Available: available, CheckedAt: now}
	m.encoderMu.Unlock()
	if !available {
		slog.Warn("hardware encoder probe failed", "encoder", encoder, "error", err, "output", strings.TrimSpace(string(output)))
	}
	return available
}

func encoderProbeKey(executable, encoder, pixelFormat string, encoderArguments []string) string {
	return strings.ToLower(strings.TrimSpace(strings.Join([]string{
		executableFingerprint(executable),
		encoder,
		pixelFormat,
		strings.Join(encoderArguments, "\x00"),
	}, "|")))
}

func (m *Manager) markEncoderUnavailable(plan domain.PlaybackPlan, capabilities RuntimeCapabilities, reason string) {
	family := strings.ToLower(strings.TrimSpace(plan.VideoEncoder))
	if family == "" || family == "auto" || family == "software" {
		return
	}
	encoder := encoderName(plan.VideoCodec, family)
	if encoder == "" {
		return
	}
	_, encoderArguments := selectVideoEncoder(plan, capabilities)
	key := encoderProbeKey(m.executable(), encoder, plan.PixelFormat, encoderArguments)
	m.encoderMu.Lock()
	m.encoderCache[key] = encoderProbeCacheEntry{Available: false, CheckedAt: time.Now()}
	m.encoderMu.Unlock()
	slog.Warn("hardware encoder disabled temporarily", "encoder", encoder, "reason", reason)
}

func bitDepthForPixelFormat(value string) int {
	if isTenBitPixelFormat(value) {
		return 10
	}
	return 8
}

func (m *Manager) retryAfterEncoderFailure(plan domain.PlaybackPlan, source domain.TechnicalMetadata, capabilities RuntimeCapabilities, failure string) (domain.PlaybackPlan, bool) {
	failure = strings.TrimSpace(failure)
	if failure == "" {
		failure = "encoder ei loonud esitusvoogu"
	}
	if plan.VideoDecoder != "" && !plan.DisableHardwareDecode {
		retry := plan
		retry.VideoDecoder = ""
		retry.DisableHardwareDecode = true
		retry.EncoderFallback = true
		retry.EncoderFallbackReason = fmt.Sprintf("Hardware decoder %s ebaõnnestus; sama hardware encoder proovitakse tarkvaralise dekodeerimisega.", strings.ToUpper(plan.VideoDecoder))
		retry.Reason = retry.EncoderFallbackReason
		return retry, true
	}
	family := strings.ToLower(strings.TrimSpace(plan.VideoEncoder))
	if family != "" && family != "auto" && family != "software" {
		m.markEncoderUnavailable(plan, capabilities, failure)
		encoder := encoderName(plan.VideoCodec, family)
		retry := plan
		if encoder != "" && !encoderExcluded(retry, encoder) {
			retry.ExcludedVideoEncoders = append(append([]string(nil), retry.ExcludedVideoEncoders...), encoder)
		}
		retry.VideoEncoder = retry.HardwarePreference
		if retry.VideoEncoder == "" {
			retry.VideoEncoder = "auto"
		}
		retry.VideoDecoder = ""
		retry.DisableHardwareDecode = false
		retry.EncoderFallback = true
		retry.EncoderFallbackReason = fmt.Sprintf("%s ei käivitunud selle faili jaoks; KoduMedia proovib järgmist encoder'it.", strings.ToUpper(family))
		retry.Reason = retry.EncoderFallbackReason
		return retry, true
	}
	return nextStabilityPlan(plan, source, failure)
}

func nextStabilityPlan(plan domain.PlaybackPlan, source domain.TechnicalMetadata, reason string) (domain.PlaybackPlan, bool) {
	nextLevel := plan.RecoveryLevel + 1
	if nextLevel < 1 {
		nextLevel = 1
	}
	if nextLevel > 3 {
		return plan, false
	}
	plan.RecoveryLevel = nextLevel
	plan.VideoCodec = "h264"
	plan.VideoCodecCandidates = []string{"h264"}
	plan.PreferredVideoCodec = "h264"
	if strings.TrimSpace(plan.HardwarePreference) == "" {
		plan.HardwarePreference = "auto"
	}
	plan.VideoEncoder = plan.HardwarePreference
	plan.VideoDecoder = ""
	plan.DisableHardwareDecode = false
	plan.AllowSoftwareAV1 = false
	plan.PixelFormat = "yuv420p"
	plan.PreserveHDR = false
	if source.HDRFormat != "" {
		plan.ToneMap = true
	}
	plan.EncoderFallback = true
	plan.EncoderFallbackReason = strings.TrimSpace(reason)

	maxWidth, maxHeight := 1920, 1080
	if nextLevel == 2 {
		maxWidth, maxHeight = 1280, 720
	} else if nextLevel >= 3 {
		maxWidth, maxHeight = 854, 480
	}
	if source.Width > 0 && source.Height > 0 {
		ratio := minRatio(maxWidth, maxHeight, source.Width, source.Height)
		if ratio > 1 {
			ratio = 1
		}
		plan.TargetWidth = evenSize(int(float64(source.Width) * ratio))
		plan.TargetHeight = evenSize(int(float64(source.Height) * ratio))
	}

	switch nextLevel {
	case 1:
		plan.QualityProfile = "balanced"
		plan.VideoPreset = "veryfast"
		plan.VideoCRF = 19
		plan.StartupBufferSeconds = 24
		plan.RebufferBufferSeconds = 36
		plan.DesiredBufferSeconds = 90
		plan.LowBufferSeconds = 24
		plan.ReadRate = 2.50
		plan.InitialBurstSeconds = 60
		plan.MinimumEncoderSpeed = 1.15
		plan.Reason = "Esmane kvaliteediprofiil ei püsinud reaalajas; KoduMedia kasutab stabiilset 1080p H.264 profiili."
	case 2:
		plan.QualityProfile = "data-saver"
		plan.VideoPreset = "superfast"
		plan.VideoCRF = 23
		plan.StartupBufferSeconds = 20
		plan.RebufferBufferSeconds = 30
		plan.DesiredBufferSeconds = 75
		plan.LowBufferSeconds = 20
		plan.ReadRate = 2.75
		plan.InitialBurstSeconds = 50
		plan.MinimumEncoderSpeed = 1.05
		plan.Reason = "1080p profiil ei püsinud reaalajas; KoduMedia kasutab töökindlat 720p H.264 profiili."
	default:
		plan.QualityProfile = "data-saver"
		plan.VideoPreset = "ultrafast"
		plan.VideoCRF = 26
		plan.StartupBufferSeconds = 16
		plan.RebufferBufferSeconds = 24
		plan.DesiredBufferSeconds = 60
		plan.LowBufferSeconds = 16
		plan.ReadRate = 3.00
		plan.InitialBurstSeconds = 40
		plan.MinimumEncoderSpeed = 1.02
		plan.Reason = "720p profiil ei püsinud reaalajas; KoduMedia kasutab viimase töökindla fallback'ina 480p H.264 profiili."
	}
	if plan.EncoderFallbackReason != "" {
		plan.Reason = plan.EncoderFallbackReason + " " + plan.Reason
	}
	plan.MinimumReadyBufferSeconds = plan.StartupBufferSeconds
	plan.MaxVideoBitrate, plan.RateControlBuffer = fallbackRateControl(plan.TargetWidth, plan.TargetHeight, source.Bitrate)
	return plan, true
}

func fallbackPlan(plan domain.PlaybackPlan, source domain.TechnicalMetadata) domain.PlaybackPlan {
	fallback, ok := nextStabilityPlan(plan, source, "")
	if !ok {
		return plan
	}
	return fallback
}

func fallbackRateControl(width, height int, sourceBitrate int64) (int64, int64) {
	maximum := int64(20_000_000)
	if width > 1920 || height > 1080 {
		maximum = 52_000_000
	} else if width <= 1280 && height <= 720 {
		maximum = 9_000_000
	}
	if sourceBitrate > 0 && sourceBitrate < maximum {
		candidate := sourceBitrate * 130 / 100
		if candidate > 4_000_000 {
			maximum = candidate
		}
	}
	return maximum, maximum * 2
}

func minRatio(maxWidth, maxHeight, width, height int) float64 {
	if width <= 0 || height <= 0 {
		return 1
	}
	wr := float64(maxWidth) / float64(width)
	hr := float64(maxHeight) / float64(height)
	if wr < hr {
		return wr
	}
	return hr
}

func evenSize(value int) int {
	if value <= 0 {
		return 0
	}
	return value - value%2
}

func parseSpeed(value string) float64 {
	value = strings.TrimSpace(strings.TrimSuffix(value, "x"))
	parsed, _ := strconv.ParseFloat(value, 64)
	return parsed
}
