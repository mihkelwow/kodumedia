package jsoncatalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/mihkelwow/kodumedia/internal/adapters/atomicfile"
	"github.com/mihkelwow/kodumedia/internal/domain"
)

const schemaVersion = 1

type persistedState struct {
	Version    int                         `json:"version"`
	Libraries  map[string]domain.Library   `json:"libraries"`
	Media      map[string]domain.MediaItem `json:"media"`
	MediaPaths map[string]string           `json:"mediaPaths"`
}

type Store struct {
	path       string
	mu         sync.RWMutex
	state      persistedState
	queryMu    sync.RWMutex
	queryCache map[string][]string
}

func Open(path string) (*Store, error) {
	store := &Store{
		path:       path,
		queryCache: map[string][]string{},
		state: persistedState{
			Version:    schemaVersion,
			Libraries:  map[string]domain.Library{},
			Media:      map[string]domain.MediaItem{},
			MediaPaths: map[string]string{},
		},
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create catalog directory: %w", err)
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read catalog: %w", err)
	}
	if len(data) == 0 {
		return store, nil
	}

	if err := json.Unmarshal(data, &store.state); err != nil {
		return nil, fmt.Errorf("decode catalog: %w", err)
	}
	if store.state.Version != schemaVersion {
		return nil, fmt.Errorf("unsupported catalog schema version %d", store.state.Version)
	}
	if store.state.Libraries == nil {
		store.state.Libraries = map[string]domain.Library{}
	}
	if store.state.Media == nil {
		store.state.Media = map[string]domain.MediaItem{}
	}
	if store.state.MediaPaths == nil {
		store.state.MediaPaths = map[string]string{}
	}
	repaired := false
	for id, library := range store.state.Libraries {
		if library.Sources == nil {
			library.Sources = []domain.LibrarySource{}
			store.state.Libraries[id] = library
			repaired = true
		}
	}
	latestSeriesModified := make(map[string]time.Time)
	for id, item := range store.state.Media {
		item.File.Path = store.state.MediaPaths[id]
		store.state.Media[id] = item
		if item.Type == domain.MediaTypeEpisode && item.SeriesID != "" && item.File.ModifiedAt.After(latestSeriesModified[item.SeriesID]) {
			latestSeriesModified[item.SeriesID] = item.File.ModifiedAt
		}
	}
	// Series parent records do not have their own file. Older catalogs therefore
	// stored a zero or stale modified time and could not be sorted by the newest
	// episode on disk. Repair that derived value once when opening the catalog.
	for id, modifiedAt := range latestSeriesModified {
		item, exists := store.state.Media[id]
		if !exists || item.Type != domain.MediaTypeSeries || !modifiedAt.After(item.File.ModifiedAt) {
			continue
		}
		item.File.ModifiedAt = modifiedAt
		store.state.Media[id] = item
		repaired = true
	}
	if repaired {
		if err := store.persistLocked(); err != nil {
			return nil, fmt.Errorf("repair catalog collections: %w", err)
		}
	}
	return store, nil
}

func (s *Store) ListLibraries(_ context.Context) ([]domain.Library, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	libraries := make([]domain.Library, 0, len(s.state.Libraries))
	for _, library := range s.state.Libraries {
		libraries = append(libraries, cloneLibrary(library))
	}
	sort.Slice(libraries, func(i, j int) bool {
		if libraries[i].System != libraries[j].System {
			return libraries[i].System
		}
		return strings.ToLower(libraries[i].Name) < strings.ToLower(libraries[j].Name)
	})
	return libraries, nil
}

func (s *Store) GetLibrary(_ context.Context, id string) (domain.Library, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	library, ok := s.state.Libraries[id]
	if !ok {
		return domain.Library{}, domain.ErrNotFound
	}
	return cloneLibrary(library), nil
}

func (s *Store) CreateLibrary(_ context.Context, library domain.Library) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.state.Libraries[library.ID]; exists {
		return domain.ErrConflict
	}
	s.state.Libraries[library.ID] = cloneLibrary(library)
	s.invalidateQueryCache()
	return s.persistLocked()
}

func (s *Store) UpdateLibrary(_ context.Context, library domain.Library) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.state.Libraries[library.ID]; !exists {
		return domain.ErrNotFound
	}
	s.state.Libraries[library.ID] = cloneLibrary(library)
	s.invalidateQueryCache()
	return s.persistLocked()
}

func (s *Store) DeleteLibrary(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	library, exists := s.state.Libraries[id]
	if !exists {
		return domain.ErrNotFound
	}
	if library.System {
		return domain.ErrConflict
	}
	delete(s.state.Libraries, id)
	for mediaID, item := range s.state.Media {
		if item.LibraryID == id {
			delete(s.state.Media, mediaID)
			delete(s.state.MediaPaths, mediaID)
		}
	}
	s.invalidateQueryCache()
	return s.persistLocked()
}

func (s *Store) ListMedia(_ context.Context, query domain.MediaQuery) (domain.MediaPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cacheKey := mediaQueryCacheKey(query)
	s.queryMu.RLock()
	ids, cached := s.queryCache[cacheKey]
	if cached {
		ids = append([]string(nil), ids...)
	}
	s.queryMu.RUnlock()
	if !cached {
		matched := make([]domain.MediaItem, 0, len(s.state.Media))
		for _, item := range s.state.Media {
			if matchesQuery(item, query) {
				matched = append(matched, cloneMedia(item))
			}
		}
		sortMedia(matched, query.Sort, query.Order)
		ids = make([]string, 0, len(matched))
		for _, item := range matched {
			ids = append(ids, item.ID)
		}
		s.queryMu.Lock()
		if len(s.queryCache) >= 256 {
			s.queryCache = map[string][]string{}
		}
		s.queryCache[cacheKey] = append([]string(nil), ids...)
		s.queryMu.Unlock()
	}
	total := len(ids)
	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= total {
		ids = []string{}
	} else {
		end := offset + limit
		if end > total {
			end = total
		}
		ids = ids[offset:end]
	}
	items := make([]domain.MediaItem, 0, len(ids))
	for _, id := range ids {
		if item, exists := s.state.Media[id]; exists {
			items = append(items, cloneMedia(item))
		}
	}
	return domain.MediaPage{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *Store) GetMedia(_ context.Context, id string) (domain.MediaItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.state.Media[id]
	if !ok {
		return domain.MediaItem{}, domain.ErrNotFound
	}
	return cloneMedia(item), nil
}

func (s *Store) ReplaceLibraryMedia(_ context.Context, libraryID string, items []domain.MediaItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.state.Libraries[libraryID]; !exists {
		return domain.ErrNotFound
	}
	for id, item := range s.state.Media {
		if item.LibraryID == libraryID {
			delete(s.state.Media, id)
			delete(s.state.MediaPaths, id)
		}
	}
	for _, item := range items {
		copyItem := cloneMedia(item)
		s.state.Media[item.ID] = copyItem
		s.state.MediaPaths[item.ID] = item.File.Path
	}
	s.invalidateQueryCache()
	return s.persistLocked()
}

func (s *Store) UpdateMedia(_ context.Context, item domain.MediaItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.state.Media[item.ID]; !exists {
		return domain.ErrNotFound
	}
	s.state.Media[item.ID] = cloneMedia(item)
	s.state.MediaPaths[item.ID] = item.File.Path
	s.invalidateQueryCache()
	return s.persistLocked()
}

func (s *Store) UpdateMediaBatch(ctx context.Context, items []domain.MediaItem) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, item := range items {
		if _, exists := s.state.Media[item.ID]; !exists {
			return domain.ErrNotFound
		}
	}
	for _, item := range items {
		s.state.Media[item.ID] = cloneMedia(item)
		s.state.MediaPaths[item.ID] = item.File.Path
	}
	s.invalidateQueryCache()
	return s.persistLocked()
}

func mediaQueryCacheKey(query domain.MediaQuery) string {
	query.Limit = 0
	query.Offset = 0
	payload, _ := json.Marshal(query)
	return string(payload)
}

func (s *Store) invalidateQueryCache() {
	s.queryMu.Lock()
	s.queryCache = map[string][]string{}
	s.queryMu.Unlock()
}

func (s *Store) persistLocked() error {
	persisted := s.state
	persisted.Media = make(map[string]domain.MediaItem, len(s.state.Media))
	persisted.MediaPaths = make(map[string]string, len(s.state.Media))
	for id, item := range s.state.Media {
		persisted.MediaPaths[id] = item.File.Path
		copyItem := cloneMedia(item)
		copyItem.File.Path = ""
		copyItem.Watch = nil
		persisted.Media[id] = copyItem
	}

	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return fmt.Errorf("encode catalog: %w", err)
	}
	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("write temporary catalog: %w", err)
	}
	if err := atomicfile.Replace(tempPath, s.path); err != nil {
		return fmt.Errorf("replace catalog: %w", err)
	}
	return nil
}

func matchesQuery(item domain.MediaItem, query domain.MediaQuery) bool {
	if query.LibraryID != "" && item.LibraryID != query.LibraryID {
		return false
	}
	if query.SeriesID != "" && item.SeriesID != query.SeriesID {
		return false
	}
	if query.Season > 0 && item.Season != query.Season {
		return false
	}
	if len(query.Types) > 0 {
		matched := false
		for _, mediaType := range query.Types {
			if item.Type == mediaType {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if query.YearFrom > 0 && item.Year < query.YearFrom {
		return false
	}
	if query.YearTo > 0 && item.Year > query.YearTo {
		return false
	}
	requestedGenres := append([]string(nil), query.Genres...)
	if genre := strings.TrimSpace(query.Genre); genre != "" {
		requestedGenres = append(requestedGenres, genre)
	}
	if len(requestedGenres) > 0 {
		genreMatched := false
		for _, itemGenre := range item.Metadata.Genres {
			for _, requestedGenre := range requestedGenres {
				if strings.EqualFold(itemGenre, strings.TrimSpace(requestedGenre)) {
					genreMatched = true
					break
				}
			}
			if genreMatched {
				break
			}
		}
		if !genreMatched {
			return false
		}
	}
	if query.Search != "" {
		searchValues := []string{
			item.Title, item.SortTitle, item.SeriesTitle, item.EpisodeTitle, item.Artist, item.Album,
			item.File.Name, item.File.RelativePath, strings.Join(item.Metadata.Genres, " "), item.Metadata.Summary, item.Metadata.Tagline,
		}
		for _, credit := range item.Metadata.Cast {
			searchValues = append(searchValues, credit.Name, credit.Character, credit.Role)
		}
		for _, credit := range item.Metadata.Directors {
			searchValues = append(searchValues, credit.Name, credit.Role)
		}
		haystack := normalizeSearchText(strings.Join(searchValues, " "))
		for _, token := range strings.Fields(normalizeSearchText(query.Search)) {
			if !strings.Contains(haystack, token) {
				return false
			}
		}
	}
	return true
}

func normalizeSearchText(value string) string {
	return strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, value)), " ")
}

func sortMedia(items []domain.MediaItem, sortBy domain.MediaSort, order domain.SortOrder) {
	if sortBy == "" {
		sortBy = domain.MediaSortAddedAt
	}
	if order == "" {
		order = domain.SortDescending
	}
	less := func(i, j int) bool {
		left, right := items[i], items[j]
		var comparison int
		switch sortBy {
		case domain.MediaSortTitle:
			comparison = strings.Compare(strings.ToLower(left.SortTitle), strings.ToLower(right.SortTitle))
		case domain.MediaSortRating:
			comparison = compareFloat(left.Metadata.IMDbRating, right.Metadata.IMDbRating)
		case domain.MediaSortYear:
			comparison = compareInt(left.Year, right.Year)
		case domain.MediaSortDuration:
			comparison = compareInt64(left.Technical.DurationMS, right.Technical.DurationMS)
		case domain.MediaSortReleaseDate:
			comparison = compareTime(releaseSortTime(left), releaseSortTime(right))
		case domain.MediaSortModifiedAt:
			comparison = compareTime(left.File.ModifiedAt, right.File.ModifiedAt)
		default:
			comparison = compareTime(left.AddedAt, right.AddedAt)
		}
		if comparison == 0 {
			comparison = strings.Compare(strings.ToLower(left.SortTitle), strings.ToLower(right.SortTitle))
		}
		if order == domain.SortAscending {
			return comparison < 0
		}
		return comparison > 0
	}
	sort.SliceStable(items, less)
}

func releaseSortTime(item domain.MediaItem) time.Time {
	if item.ReleaseDate != nil {
		return *item.ReleaseDate
	}
	if item.Year > 0 {
		return time.Date(item.Year, time.January, 1, 0, 0, 0, 0, time.UTC)
	}
	return time.Time{}
}

func compareInt(left, right int) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func compareInt64(left, right int64) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func compareFloat(left, right float64) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func compareTime(left, right time.Time) int {
	switch {
	case left.Before(right):
		return -1
	case left.After(right):
		return 1
	default:
		return 0
	}
}

func cloneLibrary(library domain.Library) domain.Library {
	library.Sources = append(make([]domain.LibrarySource, 0, len(library.Sources)), library.Sources...)
	return library
}

func cloneMedia(item domain.MediaItem) domain.MediaItem {
	item.Technical.Tracks = append([]domain.MediaTrack(nil), item.Technical.Tracks...)
	item.Metadata.Genres = append([]string(nil), item.Metadata.Genres...)
	item.Metadata.Cast = append([]domain.PersonCredit(nil), item.Metadata.Cast...)
	item.Metadata.Directors = append([]domain.PersonCredit(nil), item.Metadata.Directors...)
	if item.Metadata.LatestEpisode != nil {
		latest := *item.Metadata.LatestEpisode
		item.Metadata.LatestEpisode = &latest
	}
	if item.Watch != nil {
		watch := *item.Watch
		item.Watch = &watch
	}
	return item
}
