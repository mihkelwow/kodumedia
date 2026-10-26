package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type MusicService struct {
	catalog ports.Catalog
	watch   *WatchService
}

func NewMusicService(catalog ports.Catalog, watch *WatchService) *MusicService {
	return &MusicService{catalog: catalog, watch: watch}
}

func (s *MusicService) Landing(ctx context.Context, libraryID string) (domain.MusicLanding, error) {
	tracks, err := s.listTracks(ctx, libraryID)
	if err != nil {
		return domain.MusicLanding{}, err
	}
	artists, albums := buildMusicHierarchy(tracks)
	recentlyAdded := append([]domain.MediaItem(nil), tracks...)
	sort.SliceStable(recentlyAdded, func(i, j int) bool { return recentlyAdded[i].AddedAt.After(recentlyAdded[j].AddedAt) })
	if len(recentlyAdded) > 20 {
		recentlyAdded = recentlyAdded[:20]
	}

	recentlyPlayed := make([]domain.MediaItem, 0, 20)
	if s.watch != nil {
		states, stateErr := s.watch.States(ctx)
		if stateErr != nil {
			return domain.MusicLanding{}, stateErr
		}
		byID := make(map[string]domain.MediaItem, len(tracks))
		for _, track := range tracks {
			byID[track.ID] = track
		}
		for _, state := range states {
			track, found := byID[state.MediaID]
			if !found || track.Type != domain.MediaTypeMusic {
				continue
			}
			copyState := state
			track.Watch = &copyState
			recentlyPlayed = append(recentlyPlayed, track)
			if len(recentlyPlayed) >= 20 {
				break
			}
		}
	}
	return domain.MusicLanding{TotalTracks: len(tracks), RecentlyPlayed: recentlyPlayed, RecentlyAdded: recentlyAdded, Artists: artists, Albums: albums}, nil
}

func (s *MusicService) Artist(ctx context.Context, libraryID, artistName string) (domain.MusicArtistDetails, error) {
	tracks, err := s.listTracks(ctx, libraryID)
	if err != nil {
		return domain.MusicArtistDetails{}, err
	}
	filtered := make([]domain.MediaItem, 0)
	for _, track := range tracks {
		if strings.EqualFold(strings.TrimSpace(track.Artist), strings.TrimSpace(artistName)) {
			filtered = append(filtered, track)
		}
	}
	if len(filtered) == 0 {
		return domain.MusicArtistDetails{}, domain.ErrNotFound
	}
	artists, albums := buildMusicHierarchy(filtered)
	return domain.MusicArtistDetails{Artist: artists[0], Albums: albums}, nil
}

func (s *MusicService) Album(ctx context.Context, libraryID, albumID string) (domain.MusicAlbumDetails, error) {
	tracks, err := s.listTracks(ctx, libraryID)
	if err != nil {
		return domain.MusicAlbumDetails{}, err
	}
	_, albums := buildMusicHierarchy(tracks)
	for _, album := range albums {
		if album.ID == albumID {
			return domain.MusicAlbumDetails{Album: album}, nil
		}
	}
	return domain.MusicAlbumDetails{}, domain.ErrNotFound
}

func (s *MusicService) Tracks(ctx context.Context, query domain.MusicTrackQuery) (domain.MusicTrackPage, error) {
	tracks, err := s.listTracks(ctx, query.LibraryID)
	if err != nil {
		return domain.MusicTrackPage{}, err
	}
	if s.watch != nil {
		if err := s.watch.Decorate(ctx, tracks); err != nil {
			return domain.MusicTrackPage{}, err
		}
	}
	facets := buildMusicTrackFacets(tracks)
	filtered := make([]domain.MediaItem, 0, len(tracks))
	for _, track := range tracks {
		if musicTrackMatches(track, query) {
			filtered = append(filtered, track)
		}
	}
	sortMusicTracks(filtered, query.Sort, query.Order)
	limit := query.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	start := offset
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	items := append([]domain.MediaItem(nil), filtered[start:end]...)
	return domain.MusicTrackPage{Items: items, Total: len(filtered), Limit: limit, Offset: offset, Facets: facets}, nil
}

func musicTrackMatches(track domain.MediaItem, query domain.MusicTrackQuery) bool {
	if value := strings.TrimSpace(query.Artist); value != "" && !strings.EqualFold(track.Artist, value) {
		return false
	}
	if value := strings.TrimSpace(query.Album); value != "" && !strings.EqualFold(track.Album, value) {
		return false
	}
	if query.Year > 0 && track.Year != query.Year {
		return false
	}
	if value := strings.TrimSpace(query.Codec); value != "" && !strings.EqualFold(track.Technical.AudioCodec, value) {
		return false
	}
	classification := track.MusicClassification
	if value := strings.TrimSpace(query.Genre); value != "" && !containsFold(musicGenres(track), value) {
		return false
	}
	if value := strings.TrimSpace(query.Style); value != "" {
		if classification == nil || !containsFold(classification.Styles, value) {
			return false
		}
	}
	switch strings.ToLower(strings.TrimSpace(query.Quality)) {
	case "lossless":
		if !isLosslessTrack(track) {
			return false
		}
	case "hires":
		if !isHiResTrack(track) {
			return false
		}
	case "unclassified":
		if classification != nil && (len(classification.Genres) > 0 || len(classification.Styles) > 0) {
			return false
		}
	}
	if search := normalizeMusicSearch(query.Search); search != "" {
		values := []string{track.Title, track.Artist, track.Album, track.File.Name, track.File.RelativePath, strings.Join(musicGenres(track), " ")}
		if classification != nil {
			values = append(values, strings.Join(classification.Styles, " "), strings.Join(classification.Moods, " "))
		}
		haystack := normalizeMusicSearch(strings.Join(values, " "))
		for _, token := range strings.Fields(search) {
			if !strings.Contains(haystack, token) {
				return false
			}
		}
	}
	return true
}

func sortMusicTracks(items []domain.MediaItem, sortBy string, order domain.SortOrder) {
	sortBy = strings.ToLower(strings.TrimSpace(sortBy))
	if sortBy == "" {
		sortBy = "title"
	}
	if order == "" {
		order = domain.SortAscending
	}
	compare := func(left, right domain.MediaItem) int {
		switch sortBy {
		case "artist":
			if value := strings.Compare(strings.ToLower(left.Artist), strings.ToLower(right.Artist)); value != 0 {
				return value
			}
		case "album":
			if value := strings.Compare(strings.ToLower(left.Album), strings.ToLower(right.Album)); value != 0 {
				return value
			}
			if left.DiscNumber != right.DiscNumber {
				return compareIntValue(left.DiscNumber, right.DiscNumber)
			}
			if left.TrackNumber != right.TrackNumber {
				return compareIntValue(left.TrackNumber, right.TrackNumber)
			}
		case "year":
			if left.Year != right.Year {
				return compareIntValue(left.Year, right.Year)
			}
		case "addedat":
			if !left.AddedAt.Equal(right.AddedAt) {
				if left.AddedAt.Before(right.AddedAt) {
					return -1
				}
				return 1
			}
		case "duration":
			if left.Technical.DurationMS != right.Technical.DurationMS {
				if left.Technical.DurationMS < right.Technical.DurationMS {
					return -1
				}
				return 1
			}
		case "quality":
			if left.Technical.AudioSampleRate != right.Technical.AudioSampleRate {
				return compareIntValue(left.Technical.AudioSampleRate, right.Technical.AudioSampleRate)
			}
			if left.Technical.AudioBitDepth != right.Technical.AudioBitDepth {
				return compareIntValue(left.Technical.AudioBitDepth, right.Technical.AudioBitDepth)
			}
			if left.Technical.Bitrate != right.Technical.Bitrate {
				if left.Technical.Bitrate < right.Technical.Bitrate {
					return -1
				}
				return 1
			}
		case "lastplayed":
			leftTime, rightTime := musicLastPlayed(left), musicLastPlayed(right)
			if !leftTime.Equal(rightTime) {
				if leftTime.Before(rightTime) {
					return -1
				}
				return 1
			}
		}
		if value := strings.Compare(strings.ToLower(left.Title), strings.ToLower(right.Title)); value != 0 {
			return value
		}
		return strings.Compare(left.ID, right.ID)
	}
	sort.SliceStable(items, func(i, j int) bool {
		value := compare(items[i], items[j])
		if order == domain.SortDescending {
			return value > 0
		}
		return value < 0
	})
}

func buildMusicTrackFacets(tracks []domain.MediaItem) domain.MusicTrackFacets {
	artists, albums, genres, styles, years, codecs := map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}
	lossless, hiRes, unclassified := 0, 0, 0
	for _, track := range tracks {
		addFacet(artists, track.Artist)
		addFacet(albums, track.Album)
		for _, value := range musicGenres(track) {
			addFacet(genres, value)
		}
		if track.MusicClassification != nil {
			for _, value := range track.MusicClassification.Styles {
				addFacet(styles, value)
			}
		}
		if track.Year > 0 {
			addFacet(years, strconv.Itoa(track.Year))
		}
		addFacet(codecs, strings.ToUpper(track.Technical.AudioCodec))
		if isLosslessTrack(track) {
			lossless++
		}
		if isHiResTrack(track) {
			hiRes++
		}
		if track.MusicClassification == nil || (len(track.MusicClassification.Genres) == 0 && len(track.MusicClassification.Styles) == 0) {
			unclassified++
		}
	}
	return domain.MusicTrackFacets{Artists: facetList(artists), Albums: facetList(albums), Genres: facetList(genres), Styles: facetList(styles), Years: facetList(years), Codecs: facetList(codecs), Lossless: lossless, HiRes: hiRes, Unclassified: unclassified}
}

func musicGenres(track domain.MediaItem) []string {
	if track.MusicClassification != nil && len(track.MusicClassification.Genres) > 0 {
		return track.MusicClassification.Genres
	}
	return track.Metadata.Genres
}

func isLosslessTrack(track domain.MediaItem) bool {
	codec := strings.ToLower(track.Technical.AudioCodec)
	ext := strings.ToLower(track.File.Extension)
	switch codec {
	case "flac", "alac", "wavpack", "ape", "tta", "pcm_s16le", "pcm_s24le", "pcm_s32le", "pcm_f32le", "pcm_f64le", "truehd", "mlp":
		return true
	}
	switch ext {
	case ".wav", ".aif", ".aiff", ".aifc", ".flac", ".ape", ".wv", ".tta":
		return true
	}
	return false
}

func isHiResTrack(track domain.MediaItem) bool {
	return isLosslessTrack(track) && (track.Technical.AudioBitDepth >= 24 || track.Technical.AudioSampleRate > 48000)
}

func addFacet(target map[string]int, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		target[value]++
	}
}
func facetList(values map[string]int) []domain.MusicFacet {
	result := make([]domain.MusicFacet, 0, len(values))
	for value, count := range values {
		result = append(result, domain.MusicFacet{Value: value, Count: count})
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return strings.ToLower(result[i].Value) < strings.ToLower(result[j].Value)
	})
	return result
}
func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}
func normalizeMusicSearch(value string) string {
	return strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, value)), " ")
}
func compareIntValue(left, right int) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
func musicLastPlayed(item domain.MediaItem) time.Time {
	if item.Watch != nil {
		return item.Watch.LastWatchedAt
	}
	return time.Time{}
}

func (s *MusicService) listTracks(ctx context.Context, libraryID string) ([]domain.MediaItem, error) {
	if strings.TrimSpace(libraryID) == "" {
		return nil, fmt.Errorf("%w: library id is required", domain.ErrInvalidInput)
	}
	items := make([]domain.MediaItem, 0, 256)
	for offset := 0; ; offset += 500 {
		page, err := s.catalog.ListMedia(ctx, domain.MediaQuery{LibraryID: libraryID, Types: []domain.MediaType{domain.MediaTypeMusic}, Sort: domain.MediaSortTitle, Order: domain.SortAscending, Limit: 500, Offset: offset})
		if err != nil {
			return nil, err
		}
		for _, item := range page.Items {
			items = append(items, decorateMusicArtwork(item))
		}
		if len(items) >= page.Total || len(page.Items) == 0 {
			break
		}
	}
	return items, nil
}

func buildMusicHierarchy(tracks []domain.MediaItem) ([]domain.MusicArtist, []domain.MusicAlbum) {
	type albumBuilder struct {
		album        domain.MusicAlbum
		artworkTrack string
	}
	artistAlbums := map[string]map[string]struct{}{}
	artistTracks := map[string]int{}
	artistName := map[string]string{}
	artistArtwork := map[string]string{}
	albums := map[string]*albumBuilder{}
	for _, track := range tracks {
		artist := strings.TrimSpace(track.Artist)
		if artist == "" {
			artist = "Unknown Artist"
		}
		albumTitle := strings.TrimSpace(track.Album)
		if albumTitle == "" {
			albumTitle = "Unknown Album"
		}
		artistKey := strings.ToLower(artist)
		albumID := stableID("music-album", track.LibraryID, artistKey, strings.ToLower(albumTitle))
		if artistAlbums[artistKey] == nil {
			artistAlbums[artistKey] = map[string]struct{}{}
		}
		artistAlbums[artistKey][albumID] = struct{}{}
		artistTracks[artistKey]++
		artistName[artistKey] = artist
		builder := albums[albumID]
		if builder == nil {
			builder = &albumBuilder{album: domain.MusicAlbum{ID: albumID, Artist: artist, Title: albumTitle, Year: track.Year, Tracks: []domain.MediaItem{}}}
			albums[albumID] = builder
		}
		builder.album.Tracks = append(builder.album.Tracks, track)
		if builder.album.Year == 0 || (track.Year > 0 && track.Year < builder.album.Year) {
			builder.album.Year = track.Year
		}
		if builder.artworkTrack == "" {
			builder.artworkTrack = track.ID
		}
		if artistArtwork[artistKey] == "" {
			artistArtwork[artistKey] = musicArtworkURL(track.ID)
		}
	}
	albumList := make([]domain.MusicAlbum, 0, len(albums))
	for _, builder := range albums {
		sort.SliceStable(builder.album.Tracks, func(i, j int) bool {
			left, right := builder.album.Tracks[i], builder.album.Tracks[j]
			if left.TrackNumber != right.TrackNumber {
				if left.TrackNumber == 0 {
					return false
				}
				if right.TrackNumber == 0 {
					return true
				}
				return left.TrackNumber < right.TrackNumber
			}
			return strings.ToLower(left.Title) < strings.ToLower(right.Title)
		})
		builder.album.TrackCount = len(builder.album.Tracks)
		builder.album.ArtworkURL = musicArtworkURL(builder.artworkTrack)
		albumList = append(albumList, builder.album)
	}
	sort.SliceStable(albumList, func(i, j int) bool {
		if !strings.EqualFold(albumList[i].Artist, albumList[j].Artist) {
			return strings.ToLower(albumList[i].Artist) < strings.ToLower(albumList[j].Artist)
		}
		if albumList[i].Year != albumList[j].Year {
			return albumList[i].Year < albumList[j].Year
		}
		return strings.ToLower(albumList[i].Title) < strings.ToLower(albumList[j].Title)
	})
	artists := make([]domain.MusicArtist, 0, len(artistName))
	for key, name := range artistName {
		artists = append(artists, domain.MusicArtist{ID: stableID("music-artist", key), Name: name, AlbumCount: len(artistAlbums[key]), TrackCount: artistTracks[key], ArtworkURL: artistArtwork[key]})
	}
	sort.SliceStable(artists, func(i, j int) bool { return strings.ToLower(artists[i].Name) < strings.ToLower(artists[j].Name) })
	return artists, albumList
}

func decorateMusicArtwork(item domain.MediaItem) domain.MediaItem {
	if item.Type == domain.MediaTypeMusic && item.Metadata.PosterURL == "" {
		item.Metadata.PosterURL = musicArtworkURL(item.ID)
	}
	return item
}

const musicArtworkCacheVersion = "2"

func musicArtworkURL(mediaID string) string {
	return "/api/v1/media/" + mediaID + "/artwork?v=" + musicArtworkCacheVersion
}
