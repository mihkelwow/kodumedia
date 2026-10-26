package domain

type SubtitleSource string

type SubtitleKind string

const (
	SubtitleSourceEmbedded   SubtitleSource = "embedded"
	SubtitleSourceExternal   SubtitleSource = "external"
	SubtitleSourceDownloaded SubtitleSource = "downloaded"

	SubtitleKindText  SubtitleKind = "text"
	SubtitleKindImage SubtitleKind = "image"
)

type SubtitleTrack struct {
	ID              string         `json:"id"`
	Source          SubtitleSource `json:"source"`
	Kind            SubtitleKind   `json:"kind"`
	Format          string         `json:"format"`
	Language        string         `json:"language,omitempty"`
	Title           string         `json:"title,omitempty"`
	Path            string         `json:"-"`
	StreamIndex     int            `json:"streamIndex,omitempty"`
	StreamOrdinal   int            `json:"streamOrdinal,omitempty"`
	Default         bool           `json:"default,omitempty"`
	Forced          bool           `json:"forced,omitempty"`
	HearingImpaired bool           `json:"hearingImpaired,omitempty"`
	RequiresBurnIn  bool           `json:"requiresBurnIn,omitempty"`
	URL             string         `json:"url,omitempty"`
}

type SubtitleSearchResult struct {
	ID              string  `json:"id"`
	FileID          int64   `json:"fileId"`
	FileName        string  `json:"fileName"`
	Language        string  `json:"language"`
	Release         string  `json:"release,omitempty"`
	HearingImpaired bool    `json:"hearingImpaired,omitempty"`
	Trusted         bool    `json:"trusted,omitempty"`
	Rating          float64 `json:"rating,omitempty"`
	Downloads       int     `json:"downloads,omitempty"`
	MatchType       string  `json:"matchType,omitempty"`
}

type SubtitleDownloadRequest struct {
	FileID   int64  `json:"fileId"`
	FileName string `json:"fileName"`
	Language string `json:"language"`
}
