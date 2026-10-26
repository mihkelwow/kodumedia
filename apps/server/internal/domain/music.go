package domain

// MusicArtist is a catalog projection built from audio-file tags. Artists and
// albums are intentionally not persisted as synthetic media items, so rescans
// can rebuild the hierarchy without migration work.
type MusicArtist struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AlbumCount int    `json:"albumCount"`
	TrackCount int    `json:"trackCount"`
	ArtworkURL string `json:"artworkUrl,omitempty"`
}

type MusicAlbum struct {
	ID         string      `json:"id"`
	Artist     string      `json:"artist"`
	Title      string      `json:"title"`
	Year       int         `json:"year,omitempty"`
	TrackCount int         `json:"trackCount"`
	ArtworkURL string      `json:"artworkUrl,omitempty"`
	Tracks     []MediaItem `json:"tracks,omitempty"`
}

type MusicLanding struct {
	TotalTracks    int           `json:"totalTracks"`
	RecentlyPlayed []MediaItem   `json:"recentlyPlayed"`
	RecentlyAdded  []MediaItem   `json:"recentlyAdded"`
	Artists        []MusicArtist `json:"artists"`
	Albums         []MusicAlbum  `json:"albums"`
}

type MusicArtistDetails struct {
	Artist MusicArtist  `json:"artist"`
	Albums []MusicAlbum `json:"albums"`
}

type MusicAlbumDetails struct {
	Album MusicAlbum `json:"album"`
}

type MusicFacet struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type MusicTrackFacets struct {
	Artists      []MusicFacet `json:"artists"`
	Albums       []MusicFacet `json:"albums"`
	Genres       []MusicFacet `json:"genres"`
	Styles       []MusicFacet `json:"styles"`
	Years        []MusicFacet `json:"years"`
	Codecs       []MusicFacet `json:"codecs"`
	Lossless     int          `json:"lossless"`
	HiRes        int          `json:"hiRes"`
	Unclassified int          `json:"unclassified"`
}

type MusicTrackQuery struct {
	LibraryID string
	Search    string
	Artist    string
	Album     string
	Genre     string
	Style     string
	Year      int
	Codec     string
	Quality   string
	Sort      string
	Order     SortOrder
	Limit     int
	Offset    int
}

type MusicTrackPage struct {
	Items  []MediaItem      `json:"items"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
	Facets MusicTrackFacets `json:"facets"`
}
