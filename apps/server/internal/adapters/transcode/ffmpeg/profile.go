package ffmpeg

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type RuntimeCapabilities struct {
	Readrate             bool
	ReadrateInitialBurst bool
	CUDA                 bool
	QSV                  bool
	D3D11VA              bool
	H264QSV              bool
	HEVCQSV              bool
	AV1QSV               bool
	H264NVENC            bool
	HEVCNVENC            bool
	AV1NVENC             bool
	H264AMF              bool
	HEVCAMF              bool
	AV1AMF               bool
	LibVMAF              bool
	LibX264              bool
	LibX265              bool
	LibSVTAV1            bool
	LibAOMAV1            bool
}

type Profile interface {
	ID() string
	Arguments(request ports.TranscodeStartRequest, outputDir string, capabilities RuntimeCapabilities) []string
}

type SoftwareH264AACProfile struct {
	VideoThreads   int
	SegmentSeconds int
}

func (SoftwareH264AACProfile) ID() string { return "adaptive-video-aac" }

func (profile SoftwareH264AACProfile) Arguments(request ports.TranscodeStartRequest, outputDir string, capabilities RuntimeCapabilities) []string {
	_ = outputDir
	segmentPath := "segment_%06d.m4s"
	playlistPath := "index.m3u8"
	transportStream := strings.EqualFold(strings.TrimSpace(request.Plan.Container), "mpegts-hls")
	if transportStream {
		segmentPath = "segment_%06d.ts"
	}
	segmentSeconds := profile.segmentSeconds()
	threads := profile.videoThreads()
	filterThreads := threads
	if filterThreads > 8 {
		filterThreads = 8
	}

	args := []string{
		"-hide_banner", "-loglevel", "warning", "-nostdin", "-y",
		"-fflags", "+genpts+discardcorrupt", "-err_detect", "ignore_err",
		"-filter_threads", strconv.Itoa(filterThreads),
		"-filter_complex_threads", strconv.Itoa(filterThreads),
	}
	inputStartMS := request.Plan.StartPositionMS
	if request.CacheResumeFromMS > 0 {
		inputStartMS = request.CacheResumeFromMS
	}
	if inputStartMS > 0 {
		args = append(args, "-ss", formatSeconds(inputStartMS), "-accurate_seek")
	}
	readRate := request.Plan.ReadRate
	initialBurst := request.Plan.InitialBurstSeconds
	if readRate <= 0 {
		switch {
		case request.Plan.VideoAction == domain.TrackActionTranscode:
			readRate, initialBurst = 1.30, 24
		case request.Plan.AudioAction == domain.TrackActionTranscode:
			readRate, initialBurst = 1.60, 30
		default:
			readRate, initialBurst = 2.00, 30
		}
	}
	if capabilities.Readrate {
		args = append(args, "-readrate", strconv.FormatFloat(readRate, 'f', 2, 64))
		if capabilities.ReadrateInitialBurst && initialBurst > 0 {
			args = append(args, "-readrate_initial_burst", strconv.Itoa(initialBurst))
		}
	} else {
		args = append(args, "-re")
	}
	if strings.EqualFold(request.Plan.VideoDecoder, "cuda") {
		args = append(args, "-hwaccel", "cuda", "-hwaccel_output_format", "cuda", "-extra_hw_frames", "16")
	}
	args = append(args, "-threads", strconv.Itoa(threads), "-i", request.Media.File.Path)

	externalImageSubtitle := request.Subtitle != nil && request.Subtitle.RequiresBurnIn && request.Subtitle.Source != domain.SubtitleSourceEmbedded && strings.TrimSpace(request.Subtitle.Path) != ""
	if externalImageSubtitle {
		if inputStartMS > 0 {
			args = append(args, "-ss", formatSeconds(inputStartMS))
		}
		args = append(args, "-i", request.Subtitle.Path)
	}

	videoMap, videoFilter := buildVideoFilter(request, externalImageSubtitle)
	if request.Plan.VideoAction == domain.TrackActionDrop {
		args = append(args, "-vn")
	} else {
		switch request.Plan.VideoAction {
		case domain.TrackActionCopy:
			args = append(args, "-map", videoMap, "-c:v", "copy")
			if strings.EqualFold(request.Plan.VideoCodec, "hevc") || strings.EqualFold(request.Plan.VideoCodec, "h265") {
				args = append(args, "-tag:v", "hvc1")
			}
		default:
			if videoFilter != "" && strings.HasPrefix(videoMap, "[") {
				args = append(args, "-filter_complex", videoFilter, "-map", videoMap)
			} else {
				args = append(args, "-map", videoMap)
				if videoFilter != "" {
					args = append(args, "-vf", videoFilter)
				}
			}
			encoder, presetArgs := selectVideoEncoder(request.Plan, capabilities)
			args = append(args, "-c:v", encoder)
			args = append(args, presetArgs...)
			if strings.HasPrefix(encoder, "lib") {
				args = append(args, "-threads:v", strconv.Itoa(threads))
			}
			pixelFormat := request.Plan.PixelFormat
			if pixelFormat == "" {
				pixelFormat = "yuv420p"
			}
			args = append(args, "-pix_fmt", pixelFormat)
			if request.Plan.MaxVideoBitrate > 0 {
				buffer := request.Plan.RateControlBuffer
				if buffer <= 0 {
					buffer = request.Plan.MaxVideoBitrate * 2
				}
				args = append(args, "-maxrate:v", strconv.FormatInt(request.Plan.MaxVideoBitrate, 10), "-bufsize:v", strconv.FormatInt(buffer, 10))
			}
			args = append(args, "-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%d)", segmentSeconds), "-sc_threshold", "0")
			if strings.EqualFold(request.Plan.VideoCodec, "hevc") || strings.EqualFold(request.Plan.VideoCodec, "h265") {
				args = append(args, "-tag:v", "hvc1")
			}
			if request.Plan.PreserveHDR {
				if value := strings.TrimSpace(request.Media.Technical.ColorPrimaries); value != "" {
					args = append(args, "-color_primaries", value)
				}
				if value := strings.TrimSpace(request.Media.Technical.ColorTransfer); value != "" {
					args = append(args, "-color_trc", value)
				}
				if value := strings.TrimSpace(request.Media.Technical.ColorSpace); value != "" {
					args = append(args, "-colorspace", value)
				}
			}
		}
	}

	if request.Plan.AudioAction == domain.TrackActionDrop {
		args = append(args, "-an")
	} else {
		audioMap := "0:a:0?"
		if request.Plan.SelectedAudioTrackIndex != nil {
			audioMap = fmt.Sprintf("0:%d?", *request.Plan.SelectedAudioTrackIndex)
		}
		args = append(args, "-map", audioMap)
		switch request.Plan.AudioAction {
		case domain.TrackActionCopy:
			args = append(args, "-c:a", "copy")
		default:
			args = append(args, "-af", "aresample=async=1:first_pts=0", "-c:a", "aac", "-b:a", "192k", "-ac", "2", "-ar", "48000")
		}
	}

	hlsFlags := "independent_segments+temp_file"
	if request.CacheAppend {
		hlsFlags = "append_list+independent_segments+temp_file"
	}
	args = append(args,
		"-sn", "-dn", "-map_metadata", "-1",
		"-max_muxing_queue_size", "2048", "-max_interleave_delta", "0",
		"-avoid_negative_ts", "make_zero", "-muxdelay", "0",
		"-progress", "progress.log", "-stats_period", "0.5",
		"-f", "hls", "-hls_init_time", "1", "-hls_time", strconv.Itoa(segmentSeconds),
		"-hls_list_size", "0", "-hls_playlist_type", "event", "-hls_flags", hlsFlags,
	)
	if transportStream {
		args = append(args, "-hls_segment_type", "mpegts")
	} else {
		args = append(args, "-hls_segment_type", "fmp4", "-hls_fmp4_init_filename", "init.mp4")
	}
	if request.CacheStartNumber > 0 && !request.CacheAppend {
		args = append(args, "-start_number", strconv.Itoa(request.CacheStartNumber))
	}
	args = append(args, "-hls_segment_filename", segmentPath, playlistPath)
	return args
}

func selectVideoEncoder(plan domain.PlaybackPlan, capabilities RuntimeCapabilities) (string, []string) {
	codec := normalizeOutputCodec(plan.VideoCodec)
	preference := strings.ToLower(strings.TrimSpace(plan.VideoEncoder))
	if preference == "" || preference == "auto" {
		preference = "software"
	}

	if preference != "software" && capabilityAdvertisesEncoder(capabilities, codec, preference) {
		encoder := encoderName(codec, preference)
		if encoder != "" {
			arguments := hardwareQualityArguments(preference, plan)
			arguments = append(arguments, encoderProfileArguments(codec, plan.PixelFormat)...)
			return encoder, arguments
		}
	}

	preset := strings.TrimSpace(plan.VideoPreset)
	if preset == "" {
		preset = "medium"
	}
	crf := plan.VideoCRF
	if crf <= 0 {
		crf = defaultSoftwareCRF(plan.QualityProfile)
	}
	var encoder string
	var arguments []string
	switch codec {
	case "av1":
		if capabilities.LibSVTAV1 {
			encoder = "libsvtav1"
			arguments = []string{"-preset", strconv.Itoa(svtAV1Preset(plan.QualityProfile)), "-crf", strconv.Itoa(crf)}
		} else {
			encoder = "libaom-av1"
			arguments = []string{"-cpu-used", strconv.Itoa(aomAV1CPUUsed(plan.QualityProfile)), "-crf", strconv.Itoa(crf), "-b:v", "0", "-row-mt", "1"}
		}
	case "hevc":
		encoder = "libx265"
		arguments = []string{"-preset", preset, "-crf", strconv.Itoa(crf)}
	default:
		encoder = "libx264"
		arguments = []string{"-preset", preset, "-crf", strconv.Itoa(crf)}
	}
	arguments = append(arguments, encoderProfileArguments(codec, plan.PixelFormat)...)
	return encoder, arguments
}

func normalizeOutputCodec(value string) string {
	codec := strings.ToLower(strings.TrimSpace(value))
	if codec == "h265" {
		return "hevc"
	}
	if codec == "" {
		return "h264"
	}
	return codec
}

func defaultSoftwareCRF(profile string) int {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "maximum":
		return 16
	case "balanced":
		return 18
	case "data-saver":
		return 23
	default:
		return 17
	}
}

func hardwareQualityArguments(family string, plan domain.PlaybackPlan) []string {
	quality := mappedHardwareQuality(family, plan)
	switch family {
	case "qsv":
		return []string{
			"-preset", qsvPreset(plan.QualityProfile),
			"-global_quality", strconv.Itoa(quality),
		}
	case "nvenc":
		return []string{
			"-preset", nvencPreset(plan.QualityProfile),
			"-tune", "hq",
			"-rc", "vbr",
			"-cq", strconv.Itoa(quality),
			"-b:v", "0",
		}
	case "amf":
		return []string{
			"-quality", amfPreset(plan.QualityProfile),
			"-rc", "cqp",
			"-qp_i", strconv.Itoa(quality),
			"-qp_p", strconv.Itoa(quality),
		}
	default:
		return nil
	}
}

func mappedHardwareQuality(family string, plan domain.PlaybackPlan) int {
	profile := strings.ToLower(strings.TrimSpace(plan.QualityProfile))
	softwareBaseline := defaultSoftwareCRF(profile)
	requested := plan.VideoCRF
	if requested <= 0 {
		requested = softwareBaseline
	}
	delta := requested - softwareBaseline
	base := 18
	switch family {
	case "qsv":
		switch profile {
		case "maximum":
			base = 16
		case "balanced":
			base = 21
		case "data-saver":
			base = 25
		default:
			base = 18
		}
	case "nvenc":
		switch profile {
		case "maximum":
			base = 15
		case "balanced":
			base = 19
		case "data-saver":
			base = 23
		default:
			base = 17
		}
	case "amf":
		switch profile {
		case "maximum":
			base = 16
		case "balanced":
			base = 20
		case "data-saver":
			base = 24
		default:
			base = 18
		}
	}
	return clampQuality(base + delta)
}

func clampQuality(value int) int {
	if value < 12 {
		return 12
	}
	if value > 30 {
		return 30
	}
	return value
}

func qsvPreset(profile string) string {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "maximum":
		return "slow"
	case "balanced":
		return "fast"
	case "data-saver":
		return "veryfast"
	default:
		return "medium"
	}
}

func nvencPreset(profile string) string {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "maximum":
		return "p7"
	case "balanced":
		return "p5"
	case "data-saver":
		return "p4"
	default:
		return "p6"
	}
}

func amfPreset(profile string) string {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "balanced":
		return "balanced"
	case "data-saver":
		return "speed"
	default:
		return "quality"
	}
}

func svtAV1Preset(profile string) int {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "maximum":
		return 6
	case "balanced":
		return 8
	case "data-saver":
		return 10
	default:
		return 7
	}
}

func aomAV1CPUUsed(profile string) int {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "maximum":
		return 4
	case "balanced":
		return 6
	case "data-saver":
		return 8
	default:
		return 5
	}
}

func encoderProfileArguments(codec, pixelFormat string) []string {
	if normalizeOutputCodec(codec) != "hevc" || !isTenBitPixelFormat(pixelFormat) {
		return nil
	}
	return []string{"-profile:v", "main10"}
}

func isTenBitPixelFormat(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(value, "10") || strings.HasPrefix(value, "p010")
}

func (profile SoftwareH264AACProfile) videoThreads() int {
	if profile.VideoThreads > 0 {
		if profile.VideoThreads > 16 {
			return 16
		}
		return profile.VideoThreads
	}
	logical := runtime.NumCPU()
	if logical <= 2 {
		return logical
	}
	threads := logical / 2
	if threads < 2 {
		threads = 2
	}
	if threads > 8 {
		threads = 8
	}
	return threads
}
func (profile SoftwareH264AACProfile) segmentSeconds() int {
	if profile.SegmentSeconds >= 1 && profile.SegmentSeconds <= 10 {
		return profile.SegmentSeconds
	}
	return 2
}

func buildVideoFilter(request ports.TranscodeStartRequest, externalImageSubtitle bool) (mapValue, filter string) {
	if request.Plan.VideoAction == domain.TrackActionDrop || request.Plan.VideoAction == domain.TrackActionCopy {
		return "0:v:0?", ""
	}
	filters := make([]string, 0, 10)
	if strings.EqualFold(request.Plan.VideoDecoder, "cuda") {
		sourceFormat := "nv12"
		if request.Media.Technical.BitDepth > 8 || request.Media.Technical.HDRFormat != "" {
			sourceFormat = "p010le"
		}
		filters = append(filters, "hwdownload", "format="+sourceFormat)
	}

	scaled := false
	if request.Plan.ToneMap {
		linear := "zscale=t=linear:npl=100"
		if request.Plan.TargetWidth > 0 && request.Plan.TargetHeight > 0 {
			// Resize while converting to linear light. Tone mapping then works on
			// the final output resolution instead of needlessly processing every
			// 4K source pixel in float precision.
			linear += fmt.Sprintf(":w=%d:h=%d:filter=lanczos", request.Plan.TargetWidth, request.Plan.TargetHeight)
			scaled = true
		}
		filters = append(filters,
			linear,
			"format=gbrpf32le",
			"zscale=p=bt709",
			"tonemap=mobius:desat=0",
			"zscale=t=bt709:m=bt709:r=tv",
		)
	}
	if !scaled && request.Plan.TargetWidth > 0 && request.Plan.TargetHeight > 0 {
		filters = append(filters, fmt.Sprintf("scale=%d:%d:flags=lanczos", request.Plan.TargetWidth, request.Plan.TargetHeight))
	} else if !scaled && request.Plan.TargetWidth <= 0 && request.Plan.TargetHeight <= 0 {
		filters = append(filters, "scale=trunc(iw/2)*2:trunc(ih/2)*2")
	}
	pixelFormat := strings.TrimSpace(request.Plan.PixelFormat)
	if pixelFormat == "" {
		pixelFormat = "yuv420p"
	}
	filters = append(filters, "format="+pixelFormat, "setpts=PTS-STARTPTS")
	chain := strings.Join(filters, ",")
	if request.Subtitle != nil && request.Subtitle.RequiresBurnIn {
		subtitleInput := "0:s:0"
		if request.Subtitle.Source == domain.SubtitleSourceEmbedded {
			subtitleInput = fmt.Sprintf("0:s:%d", request.Subtitle.StreamOrdinal)
		} else if externalImageSubtitle {
			subtitleInput = "1:s:0"
		}
		return "[vout]", fmt.Sprintf("[0:v:0][%s]overlay=shortest=1,%s[vout]", subtitleInput, chain)
	}
	return "0:v:0?", chain
}

func formatSeconds(milliseconds int64) string {
	if milliseconds <= 0 {
		return "0"
	}
	return strconv.FormatFloat(float64(milliseconds)/1000, 'f', 3, 64)
}
