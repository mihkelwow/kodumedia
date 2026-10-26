package domain

import "time"

type MediaType string

const (
	MediaTypeMovie   MediaType = "movie"
	MediaTypeSeries  MediaType = "series"
	MediaTypeEpisode MediaType = "episode"
	MediaTypeMusic   MediaType = "music"
	MediaTypeUnknown MediaType = "unknown"
)

type MediaFile struct {
	Path         string    `json:"path,omitempty"`
	RelativePath string    `json:"relativePath"`
	Name         string    `json:"name"`
	Extension    string    `json:"extension"`
	SizeBytes    int64     `json:"sizeBytes"`
	ModifiedAt   time.Time `json:"modifiedAt"`
}

type MediaTrack struct {
	Index           int     `json:"index"`
	StreamOrdinal   int     `json:"streamOrdinal,omitempty"`
	Kind            string  `json:"kind"`
	Codec           string  `json:"codec"`
	CodecProfile    string  `json:"codecProfile,omitempty"`
	CodecLevel      int     `json:"codecLevel,omitempty"`
	FrameRate       float64 `json:"frameRate,omitempty"`
	Bitrate         int64   `json:"bitrate,omitempty"`
	Language        string  `json:"language,omitempty"`
	Title           string  `json:"title,omitempty"`
	Channels        int     `json:"channels,omitempty"`
	ChannelLayout   string  `json:"channelLayout,omitempty"`
	SampleRate      int     `json:"sampleRate,omitempty"`
	Width           int     `json:"width,omitempty"`
	Height          int     `json:"height,omitempty"`
	BitDepth        int     `json:"bitDepth,omitempty"`
	PixelFormat     string  `json:"pixelFormat,omitempty"`
	ColorPrimaries  string  `json:"colorPrimaries,omitempty"`
	ColorTransfer   string  `json:"colorTransfer,omitempty"`
	ColorSpace      string  `json:"colorSpace,omitempty"`
	HDRFormat       string  `json:"hdrFormat,omitempty"`
	Default         bool    `json:"default,omitempty"`
	Forced          bool    `json:"forced,omitempty"`
	HearingImpaired bool    `json:"hearingImpaired,omitempty"`
}

type TechnicalMetadata struct {
	ProbeVersion    int          `json:"probeVersion,omitempty"`
	Container       string       `json:"container,omitempty"`
	DurationMS      int64        `json:"durationMs,omitempty"`
	Bitrate         int64        `json:"bitrate,omitempty"`
	Width           int          `json:"width,omitempty"`
	Height          int          `json:"height,omitempty"`
	VideoCodec      string       `json:"videoCodec,omitempty"`
	VideoProfile    string       `json:"videoProfile,omitempty"`
	VideoLevel      int          `json:"videoLevel,omitempty"`
	FrameRate       float64      `json:"frameRate,omitempty"`
	AudioCodec      string       `json:"audioCodec,omitempty"`
	AudioChannels   int          `json:"audioChannels,omitempty"`
	AudioSampleRate int          `json:"audioSampleRate,omitempty"`
	AudioBitDepth   int          `json:"audioBitDepth,omitempty"`
	BitDepth        int          `json:"bitDepth,omitempty"`
	PixelFormat     string       `json:"pixelFormat,omitempty"`
	ColorPrimaries  string       `json:"colorPrimaries,omitempty"`
	ColorTransfer   string       `json:"colorTransfer,omitempty"`
	ColorSpace      string       `json:"colorSpace,omitempty"`
	HDRFormat       string       `json:"hdrFormat,omitempty"`
	Tracks          []MediaTrack `json:"tracks,omitempty"`
}

type PersonCredit struct {
	Name      string `json:"name"`
	Character string `json:"character,omitempty"`
	Role      string `json:"role,omitempty"`
	ImageURL  string `json:"imageUrl,omitempty"`
}

type ExternalIDs struct {
	IMDb                    string `json:"imdb,omitempty"`
	TMDb                    string `json:"tmdb,omitempty"`
	TVDb                    string `json:"tvdb,omitempty"`
	SoundCloud              string `json:"soundcloud,omitempty"` // Legacy catalog compatibility.
	MusicBrainzRecording    string `json:"musicBrainzRecording,omitempty"`
	MusicBrainzRelease      string `json:"musicBrainzRelease,omitempty"`
	MusicBrainzReleaseGroup string `json:"musicBrainzReleaseGroup,omitempty"`
	AcoustID                string `json:"acoustId,omitempty"`
}

type EpisodeReleaseInfo struct {
	Season      int        `json:"season"`
	Episode     int        `json:"episode"`
	Title       string     `json:"title,omitempty"`
	ReleaseDate *time.Time `json:"releaseDate,omitempty"`
}

type TrailerCandidate struct {
	YouTubeKey string `json:"youtubeKey"`
	Name       string `json:"name,omitempty"`
}

type EditorialMetadata struct {
	Summary              string              `json:"summary,omitempty"`
	Tagline              string              `json:"tagline,omitempty"`
	Genres               []string            `json:"genres,omitempty"`
	Cast                 []PersonCredit      `json:"cast,omitempty"`
	Directors            []PersonCredit      `json:"directors,omitempty"`
	PosterURL            string              `json:"posterUrl,omitempty"`
	BackdropURL          string              `json:"backdropUrl,omitempty"`
	LogoURL              string              `json:"logoUrl,omitempty"`
	TrailerYouTubeKey    string              `json:"trailerYoutubeKey,omitempty"`
	TrailerName          string              `json:"trailerName,omitempty"`
	TrailerCandidates    []TrailerCandidate  `json:"trailerCandidates,omitempty"`
	HeroAssetsVersion    int                 `json:"heroAssetsVersion,omitempty"`
	IMDbRating           float64             `json:"imdbRating,omitempty"`
	IMDbVotes            int64               `json:"imdbVotes,omitempty"`
	MetadataSource       string              `json:"metadataSource,omitempty"`
	OfficialSeasonCount  int                 `json:"officialSeasonCount,omitempty"`
	OfficialEpisodeCount int                 `json:"officialEpisodeCount,omitempty"`
	LatestEpisode        *EpisodeReleaseInfo `json:"latestEpisode,omitempty"`
	UpdatedAt            *time.Time          `json:"updatedAt,omitempty"`
}

type MusicClassification struct {
	Genres      []string `json:"genres,omitempty"`
	Styles      []string `json:"styles,omitempty"`
	Moods       []string `json:"moods,omitempty"`
	Source      string   `json:"source,omitempty"`
	GenreSource string   `json:"genreSource,omitempty"`
	StyleSource string   `json:"styleSource,omitempty"`
	MoodSource  string   `json:"moodSource,omitempty"`
	Confidence  string   `json:"confidence,omitempty"`
	SourceURL   string   `json:"sourceUrl,omitempty"`
}

type MediaItem struct {
	ID                  string               `json:"id"`
	LibraryID           string               `json:"libraryId"`
	SeriesID            string               `json:"seriesId,omitempty"`
	Type                MediaType            `json:"type"`
	Title               string               `json:"title"`
	SortTitle           string               `json:"sortTitle"`
	Year                int                  `json:"year,omitempty"`
	ReleaseDate         *time.Time           `json:"releaseDate,omitempty"`
	Season              int                  `json:"season,omitempty"`
	Episode             int                  `json:"episode,omitempty"`
	EpisodeTitle        string               `json:"episodeTitle,omitempty"`
	SeriesTitle         string               `json:"seriesTitle,omitempty"`
	SeasonCount         int                  `json:"seasonCount,omitempty"`
	EpisodeCount        int                  `json:"episodeCount,omitempty"`
	Artist              string               `json:"artist,omitempty"`
	Album               string               `json:"album,omitempty"`
	TrackNumber         int                  `json:"trackNumber,omitempty"`
	DiscNumber          int                  `json:"discNumber,omitempty"`
	MusicClassification *MusicClassification `json:"musicClassification,omitempty"`
	File                MediaFile            `json:"file"`
	Technical           TechnicalMetadata    `json:"technical"`
	ExternalIDs         ExternalIDs          `json:"externalIds"`
	Metadata            EditorialMetadata    `json:"metadata"`
	AddedAt             time.Time            `json:"addedAt"`
	UpdatedAt           time.Time            `json:"updatedAt"`
	Watch               *WatchState          `json:"watch,omitempty"`
}

type MediaSort string

const (
	MediaSortAddedAt     MediaSort = "addedAt"
	MediaSortReleaseDate MediaSort = "releaseDate"
	MediaSortTitle       MediaSort = "title"
	MediaSortRating      MediaSort = "rating"
	MediaSortYear        MediaSort = "year"
	MediaSortDuration    MediaSort = "duration"
	MediaSortModifiedAt  MediaSort = "modifiedAt"
)

type SortOrder string

const (
	SortAscending  SortOrder = "asc"
	SortDescending SortOrder = "desc"
)

type MediaQuery struct {
	LibraryID string
	SeriesID  string
	Season    int
	Types     []MediaType
	Search    string
	Genre     string
	Genres    []string
	YearFrom  int
	YearTo    int
	Sort      MediaSort
	Order     SortOrder
	Limit     int
	Offset    int
}

type PostPlayRecommendation struct {
	Item         MediaItem  `json:"item"`
	PlayTarget   *MediaItem `json:"playTarget,omitempty"`
	SharedGenres []string   `json:"sharedGenres,omitempty"`
	Reason       string     `json:"reason,omitempty"`
}

type PostPlay struct {
	NextEpisode     *MediaItem               `json:"nextEpisode,omitempty"`
	Recommendations []PostPlayRecommendation `json:"recommendations"`
}

type MediaPage struct {
	Items  []MediaItem `json:"items"`
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}
