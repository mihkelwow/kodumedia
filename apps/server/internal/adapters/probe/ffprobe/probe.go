package ffprobe

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/platform/childprocess"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type Probe struct {
	binary func() string
}

func New(binary string) *Probe               { return NewDynamic(func() string { return binary }) }
func NewDynamic(binary func() string) *Probe { return &Probe{binary: binary} }

func (p *Probe) executable() string {
	if p.binary != nil {
		if value := strings.TrimSpace(p.binary()); value != "" {
			return value
		}
	}
	return "ffprobe"
}

func (p *Probe) Available(ctx context.Context) bool {
	command := exec.CommandContext(ctx, p.executable(), "-version")
	childprocess.Hide(command)
	return command.Run() == nil
}

type probeDisposition struct {
	Default         int `json:"default"`
	Forced          int `json:"forced"`
	HearingImpaired int `json:"hearing_impaired"`
}

type probeSideData struct {
	SideDataType string `json:"side_data_type"`
}

type probeStream struct {
	Index            int               `json:"index"`
	CodecName        string            `json:"codec_name"`
	CodecType        string            `json:"codec_type"`
	Profile          string            `json:"profile"`
	Level            int               `json:"level"`
	AverageFrameRate string            `json:"avg_frame_rate"`
	RealFrameRate    string            `json:"r_frame_rate"`
	BitRate          string            `json:"bit_rate"`
	Width            int               `json:"width"`
	Height           int               `json:"height"`
	Channels         int               `json:"channels"`
	ChannelLayout    string            `json:"channel_layout"`
	SampleRate       string            `json:"sample_rate"`
	PixelFormat      string            `json:"pix_fmt"`
	BitsPerRawSample string            `json:"bits_per_raw_sample"`
	ColorPrimaries   string            `json:"color_primaries"`
	ColorTransfer    string            `json:"color_transfer"`
	ColorSpace       string            `json:"color_space"`
	Tags             map[string]string `json:"tags"`
	Disposition      probeDisposition  `json:"disposition"`
	SideDataList     []probeSideData   `json:"side_data_list"`
}

type probeOutput struct {
	Streams []probeStream `json:"streams"`
	Format  struct {
		FormatName string            `json:"format_name"`
		Duration   string            `json:"duration"`
		BitRate    string            `json:"bit_rate"`
		Tags       map[string]string `json:"tags"`
	} `json:"format"`
}

func (p *Probe) Probe(ctx context.Context, path string) (ports.ProbeResult, error) {
	command := exec.CommandContext(
		ctx,
		p.executable(),
		"-v", "error",
		"-show_entries", "format=format_name,duration,bit_rate:format_tags=title,artist,album,date,track,disc,album_artist,genre,style,subgenre,grouping,contentgroup,mood,isrc,musicbrainz_recordingid,musicbrainz_trackid",
		"-show_entries", "stream=index,codec_type,codec_name,profile,level,avg_frame_rate,r_frame_rate,bit_rate,width,height,channels,channel_layout,sample_rate,pix_fmt,bits_per_raw_sample,color_primaries,color_transfer,color_space:stream_tags=language,title:stream_disposition=default,forced,hearing_impaired:stream_side_data=side_data_type",
		"-of", "json",
		path,
	)
	childprocess.Hide(command)
	output, err := command.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return ports.ProbeResult{}, fmt.Errorf("ffprobe failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return ports.ProbeResult{}, fmt.Errorf("run ffprobe: %w", err)
	}

	var decoded probeOutput
	if err := json.Unmarshal(output, &decoded); err != nil {
		return ports.ProbeResult{}, fmt.Errorf("decode ffprobe output: %w", err)
	}

	technical := domain.TechnicalMetadata{
		ProbeVersion: 3,
		Container:    normalizeContainer(decoded.Format.FormatName),
		DurationMS:   milliseconds(decoded.Format.Duration),
		Bitrate:      integer64(decoded.Format.BitRate),
		Tracks:       make([]domain.MediaTrack, 0, len(decoded.Streams)),
	}
	ordinals := map[string]int{}
	for _, stream := range decoded.Streams {
		ordinal := ordinals[stream.CodecType]
		ordinals[stream.CodecType] = ordinal + 1
		bitDepth := integer(stream.BitsPerRawSample)
		if bitDepth == 0 {
			bitDepth = bitDepthFromPixelFormat(stream.PixelFormat)
		}
		hdr := detectHDR(stream)
		frameRate := rationalFrameRate(firstNonEmpty(stream.AverageFrameRate, stream.RealFrameRate))
		track := domain.MediaTrack{
			Index:           stream.Index,
			StreamOrdinal:   ordinal,
			Kind:            stream.CodecType,
			Codec:           normalizeCodec(stream.CodecName),
			CodecProfile:    stream.Profile,
			CodecLevel:      stream.Level,
			FrameRate:       frameRate,
			Bitrate:         integer64(stream.BitRate),
			Language:        normalizeLanguage(firstNonEmpty(stream.Tags["language"], stream.Tags["LANGUAGE"])),
			Title:           firstNonEmpty(stream.Tags["title"], stream.Tags["TITLE"]),
			Channels:        stream.Channels,
			ChannelLayout:   stream.ChannelLayout,
			SampleRate:      integer(stream.SampleRate),
			Width:           stream.Width,
			Height:          stream.Height,
			BitDepth:        bitDepth,
			PixelFormat:     stream.PixelFormat,
			ColorPrimaries:  stream.ColorPrimaries,
			ColorTransfer:   stream.ColorTransfer,
			ColorSpace:      stream.ColorSpace,
			HDRFormat:       hdr,
			Default:         stream.Disposition.Default != 0,
			Forced:          stream.Disposition.Forced != 0,
			HearingImpaired: stream.Disposition.HearingImpaired != 0,
		}
		technical.Tracks = append(technical.Tracks, track)
		switch stream.CodecType {
		case "video":
			if technical.VideoCodec == "" {
				technical.VideoCodec = track.Codec
				technical.VideoProfile = stream.Profile
				technical.VideoLevel = stream.Level
				technical.FrameRate = frameRate
				technical.Width = stream.Width
				technical.Height = stream.Height
				technical.BitDepth = bitDepth
				technical.PixelFormat = stream.PixelFormat
				technical.ColorPrimaries = stream.ColorPrimaries
				technical.ColorTransfer = stream.ColorTransfer
				technical.ColorSpace = stream.ColorSpace
				technical.HDRFormat = hdr
			}
		case "audio":
			if technical.AudioCodec == "" {
				technical.AudioCodec = track.Codec
				technical.AudioChannels = stream.Channels
				technical.AudioSampleRate = track.SampleRate
				technical.AudioBitDepth = bitDepth
			}
		}
	}

	return ports.ProbeResult{Technical: technical, Tags: normalizedTags(decoded.Format.Tags)}, nil
}

func detectHDR(stream probeStream) string {
	for _, data := range stream.SideDataList {
		value := strings.ToLower(data.SideDataType)
		switch {
		case strings.Contains(value, "dovi") || strings.Contains(value, "dolby vision"):
			return "dolby-vision"
		case strings.Contains(value, "hdr dynamic metadata smpte2094-40") || strings.Contains(value, "hdr10+"):
			return "hdr10plus"
		}
	}
	switch strings.ToLower(stream.ColorTransfer) {
	case "arib-std-b67":
		return "hlg"
	case "smpte2084":
		return "hdr10"
	}
	return ""
}

func bitDepthFromPixelFormat(value string) int {
	value = strings.ToLower(value)
	for _, depth := range []int{16, 14, 12, 10, 9} {
		if strings.Contains(value, strconv.Itoa(depth)) {
			return depth
		}
	}
	if value != "" {
		return 8
	}
	return 0
}

func rationalFrameRate(value string) float64 {
	value = strings.TrimSpace(value)
	if value == "" || value == "0/0" {
		return 0
	}
	if numerator, denominator, ok := strings.Cut(value, "/"); ok {
		n, nErr := strconv.ParseFloat(strings.TrimSpace(numerator), 64)
		d, dErr := strconv.ParseFloat(strings.TrimSpace(denominator), 64)
		if nErr == nil && dErr == nil && d > 0 {
			return n / d
		}
	}
	parsed, _ := strconv.ParseFloat(value, 64)
	return parsed
}

func milliseconds(value string) int64 {
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil || seconds <= 0 {
		return 0
	}
	return int64(seconds * 1000)
}

func integer64(value string) int64 { parsed, _ := strconv.ParseInt(value, 10, 64); return parsed }
func integer(value string) int     { parsed, _ := strconv.Atoi(strings.TrimSpace(value)); return parsed }

func normalizedTags(tags map[string]string) map[string]string {
	result := make(map[string]string, len(tags))
	for key, value := range tags {
		result[strings.ToLower(key)] = strings.TrimSpace(value)
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeLanguage(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	aliases := map[string]string{"est": "et", "eng": "en", "fin": "fi", "rus": "ru", "deu": "de", "ger": "de", "fra": "fr", "fre": "fr", "spa": "es", "ita": "it", "jpn": "ja", "kor": "ko", "zho": "zh", "chi": "zh"}
	if mapped := aliases[value]; mapped != "" {
		return mapped
	}
	return value
}

func normalizeContainer(formatName string) string {
	parts := strings.Split(strings.ToLower(formatName), ",")
	for _, part := range parts {
		switch strings.TrimSpace(part) {
		case "mov", "mp4", "m4a", "3gp", "3g2", "mj2":
			return "mp4"
		case "matroska":
			return "mkv"
		case "webm":
			return "webm"
		case "mp3":
			return "mp3"
		case "flac":
			return "flac"
		case "ogg":
			return "ogg"
		case "wav":
			return "wav"
		case "avi":
			return "avi"
		case "asf":
			return "asf"
		case "mpegts":
			return "mpegts"
		case "mpeg":
			return "mpeg"
		}
	}
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return ""
}

func normalizeCodec(codec string) string {
	switch strings.ToLower(strings.TrimSpace(codec)) {
	case "h264", "avc1":
		return "h264"
	case "hevc", "h265":
		return "hevc"
	case "av1":
		return "av1"
	case "vp09", "vp9":
		return "vp9"
	case "aac":
		return "aac"
	case "ac3":
		return "ac3"
	case "eac3":
		return "eac3"
	case "dts", "dca":
		return "dts"
	case "truehd":
		return "truehd"
	case "opus":
		return "opus"
	case "vorbis":
		return "vorbis"
	case "mp3":
		return "mp3"
	case "flac":
		return "flac"
	case "subrip":
		return "srt"
	case "ass", "ssa", "webvtt", "mov_text", "hdmv_pgs_subtitle", "dvd_subtitle", "dvb_subtitle":
		return strings.ToLower(strings.TrimSpace(codec))
	default:
		return strings.ToLower(strings.TrimSpace(codec))
	}
}

var _ ports.MediaProbe = (*Probe)(nil)
