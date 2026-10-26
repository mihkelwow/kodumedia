package domain

import "time"

type PlaybackMode string

const (
	PlaybackModeDirect    PlaybackMode = "direct"
	PlaybackModeRemux     PlaybackMode = "remux"
	PlaybackModeTranscode PlaybackMode = "transcode"
)

type VideoDecodeCapability struct {
	Codec          string  `json:"codec"`
	CodecProfile   string  `json:"codecProfile,omitempty"`
	CodecLevel     int     `json:"codecLevel,omitempty"`
	Source         bool    `json:"source,omitempty"`
	ContentType    string  `json:"contentType,omitempty"`
	Width          int     `json:"width,omitempty"`
	Height         int     `json:"height,omitempty"`
	FrameRate      float64 `json:"frameRate,omitempty"`
	Bitrate        int64   `json:"bitrate,omitempty"`
	BitDepth       int     `json:"bitDepth,omitempty"`
	HDRFormat      string  `json:"hdrFormat,omitempty"`
	Supported      bool    `json:"supported"`
	Smooth         bool    `json:"smooth"`
	PowerEfficient bool    `json:"powerEfficient"`
}

type ClientCapabilities struct {
	ClientName           string                  `json:"clientName,omitempty"`
	ClientVersion        string                  `json:"clientVersion,omitempty"`
	SupportedMIMEs       []string                `json:"supportedMimes"`
	VideoCodecs          []string                `json:"videoCodecs"`
	AudioCodecs          []string                `json:"audioCodecs"`
	SubtitleFormats      []string                `json:"subtitleFormats"`
	HDRFormats           []string                `json:"hdrFormats,omitempty"`
	SupportsMSE          bool                    `json:"supportsMse"`
	SupportsHLS          bool                    `json:"supportsHls"`
	SupportsPIP          bool                    `json:"supportsPip,omitempty"`
	MaximumWidth         int                     `json:"maximumWidth,omitempty"`
	MaximumHeight        int                     `json:"maximumHeight,omitempty"`
	MaximumBitrate       int64                   `json:"maximumBitrate,omitempty"`
	MaximumAudioChannels int                     `json:"maximumAudioChannels,omitempty"`
	DecodeCapabilities   []VideoDecodeCapability `json:"decodeCapabilities,omitempty"`
}

type HDRMode string

const (
	HDRModeAuto HDRMode = "auto"
	HDRModeHDR  HDRMode = "hdr"
	HDRModeSDR  HDRMode = "sdr"
)

type PlaybackPreferences struct {
	ForceTranscode    bool    `json:"forceTranscode,omitempty"`
	ForceSubtitleBurn bool    `json:"forceSubtitleBurn,omitempty"`
	StartPositionMS   int64   `json:"startPositionMs,omitempty"`
	AudioTrackIndex   *int    `json:"audioTrackIndex,omitempty"`
	SubtitleID        string  `json:"subtitleId,omitempty"`
	HDRMode           HDRMode `json:"hdrMode,omitempty"`
	RecoveryLevel     int     `json:"recoveryLevel,omitempty"`
}

type TrackAction string

const (
	TrackActionCopy      TrackAction = "copy"
	TrackActionTranscode TrackAction = "transcode"
	TrackActionDrop      TrackAction = "drop"
	TrackActionExternal  TrackAction = "external"
	TrackActionBurn      TrackAction = "burn"
)

type PlaybackPlan struct {
	Mode                      PlaybackMode `json:"mode"`
	Reason                    string       `json:"reason"`
	Container                 string       `json:"container"`
	VideoCodec                string       `json:"videoCodec,omitempty"`
	AudioCodec                string       `json:"audioCodec,omitempty"`
	VideoAction               TrackAction  `json:"videoAction,omitempty"`
	AudioAction               TrackAction  `json:"audioAction,omitempty"`
	SubtitleAction            TrackAction  `json:"subtitleAction,omitempty"`
	SelectedAudioTrackIndex   *int         `json:"selectedAudioTrackIndex,omitempty"`
	SelectedSubtitleID        string       `json:"selectedSubtitleId,omitempty"`
	ToneMap                   bool         `json:"toneMap,omitempty"`
	PreserveHDR               bool         `json:"preserveHdr,omitempty"`
	SourceHDRFormat           string       `json:"sourceHdrFormat,omitempty"`
	TargetWidth               int          `json:"targetWidth,omitempty"`
	TargetHeight              int          `json:"targetHeight,omitempty"`
	StartPositionMS           int64        `json:"startPositionMs,omitempty"`
	DurationMS                int64        `json:"durationMs,omitempty"`
	DirectURL                 string       `json:"directUrl,omitempty"`
	PlaylistURL               string       `json:"playlistUrl,omitempty"`
	TranscodeProfile          string       `json:"transcodeProfile,omitempty"`
	QualityProfile            string       `json:"qualityProfile,omitempty"`
	VideoEncoder              string       `json:"videoEncoder,omitempty"`
	VideoDecoder              string       `json:"videoDecoder,omitempty"`
	VideoPreset               string       `json:"videoPreset,omitempty"`
	EncoderFallbackReason     string       `json:"encoderFallbackReason,omitempty"`
	VideoCodecCandidates      []string     `json:"-"`
	PreferredVideoCodec       string       `json:"-"`
	HardwarePreference        string       `json:"-"`
	VideoCRF                  int          `json:"videoCrf,omitempty"`
	PixelFormat               string       `json:"pixelFormat,omitempty"`
	CachePolicy               string       `json:"cachePolicy,omitempty"`
	CacheMaxBytes             int64        `json:"cacheMaxBytes,omitempty"`
	CacheFreeReserveBytes     int64        `json:"cacheFreeReserveBytes,omitempty"`
	CacheFreeReservePercent   int          `json:"cacheFreeReservePercent,omitempty"`
	GracePeriodSeconds        int          `json:"gracePeriodSeconds,omitempty"`
	VMAFEnabled               bool         `json:"vmafEnabled,omitempty"`
	VMAFTarget                int          `json:"vmafTarget,omitempty"`
	VMAFSampleSeconds         int          `json:"vmafSampleSeconds,omitempty"`
	EncoderFallback           bool         `json:"encoderFallback,omitempty"`
	AllowSoftwareAV1          bool         `json:"allowSoftwareAv1,omitempty"`
	RecoveryLevel             int          `json:"recoveryLevel,omitempty"`
	StartupBufferSeconds      int          `json:"startupBufferSeconds,omitempty"`
	RebufferBufferSeconds     int          `json:"rebufferBufferSeconds,omitempty"`
	DesiredBufferSeconds      int          `json:"desiredBufferSeconds,omitempty"`
	LowBufferSeconds          int          `json:"lowBufferSeconds,omitempty"`
	ReadRate                  float64      `json:"readRate,omitempty"`
	InitialBurstSeconds       int          `json:"initialBurstSeconds,omitempty"`
	MinimumEncoderSpeed       float64      `json:"minimumEncoderSpeed,omitempty"`
	MaxVideoBitrate           int64        `json:"maxVideoBitrate,omitempty"`
	RateControlBuffer         int64        `json:"rateControlBuffer,omitempty"`
	CacheHit                  bool         `json:"cacheHit,omitempty"`
	InitialPositionMS         int64        `json:"initialPositionMs,omitempty"`
	MinimumReadyBufferSeconds int          `json:"-"`
	TranscodeRevision         string       `json:"-"`
	EncoderBuild              string       `json:"-"`
	ExcludedVideoEncoders     []string     `json:"-"`
	DisableHardwareDecode     bool         `json:"-"`
}

type PlaybackRuntimeStats struct {
	SessionID           string    `json:"sessionId"`
	State               string    `json:"state"`
	ProducerAlive       bool      `json:"producerAlive"`
	EncoderSpeed        float64   `json:"encoderSpeed,omitempty"`
	AverageEncoderSpeed float64   `json:"averageEncoderSpeed,omitempty"`
	OutputPositionMS    int64     `json:"outputPositionMs,omitempty"`
	GeneratedDurationMS int64     `json:"generatedDurationMs,omitempty"`
	SegmentCount        int       `json:"segmentCount,omitempty"`
	EstimatedBitrate    int64     `json:"estimatedBitrate,omitempty"`
	MinimumEncoderSpeed float64   `json:"minimumEncoderSpeed,omitempty"`
	FallbackRecommended bool      `json:"fallbackRecommended,omitempty"`
	FallbackReason      string    `json:"fallbackReason,omitempty"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type PlaybackSession struct {
	ID           string       `json:"id"`
	MediaID      string       `json:"mediaId"`
	Plan         PlaybackPlan `json:"plan"`
	PlaylistURL  string       `json:"playlistUrl,omitempty"`
	CreatedAt    time.Time    `json:"createdAt"`
	LastAccessAt time.Time    `json:"lastAccessAt"`
}

type PlaybackOptions struct {
	MediaID     string          `json:"mediaId"`
	DurationMS  int64           `json:"durationMs,omitempty"`
	AudioTracks []MediaTrack    `json:"audioTracks"`
	Subtitles   []SubtitleTrack `json:"subtitles"`
}
