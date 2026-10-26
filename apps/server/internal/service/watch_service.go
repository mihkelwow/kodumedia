package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

const (
	continueMinimumProgress = 0.01
	continueMaximumProgress = 0.95
)

// WatchService owns viewing-history semantics. The catalog remains immutable
// from playback progress updates, while the dedicated watch store survives
// rescans and application upgrades.
type WatchService struct {
	catalog ports.Catalog
	store   ports.WatchStore
	now     func() time.Time
}

func NewWatchService(catalog ports.Catalog, store ports.WatchStore) *WatchService {
	return &WatchService{catalog: catalog, store: store, now: time.Now}
}

func (s *WatchService) Update(ctx context.Context, mediaID string, update domain.WatchUpdate) (domain.WatchState, error) {
	if s == nil || s.store == nil {
		return domain.WatchState{}, domain.ErrUnavailable
	}
	item, err := s.catalog.GetMedia(ctx, mediaID)
	if err != nil {
		return domain.WatchState{}, err
	}
	if item.Type == domain.MediaTypeSeries || item.File.Path == "" {
		return domain.WatchState{}, fmt.Errorf("%w: media item is not directly playable", domain.ErrInvalidInput)
	}
	duration := update.DurationMS
	if duration <= 0 {
		duration = item.Technical.DurationMS
	}
	if duration < 0 {
		duration = 0
	}
	position := update.PositionMS
	if position < 0 {
		position = 0
	}
	if duration > 0 && position > duration {
		position = duration
	}
	progress := 0.0
	if duration > 0 {
		progress = float64(position) / float64(duration)
		if progress > 1 {
			progress = 1
		}
	}
	completed := update.Completed || (duration > 0 && progress > continueMaximumProgress)
	if completed && duration > 0 {
		position = duration
		progress = 1
	}
	now := s.now().UTC()
	state := domain.WatchState{
		MediaID:       item.ID,
		SeriesID:      item.SeriesID,
		MediaType:     item.Type,
		Season:        item.Season,
		Episode:       item.Episode,
		PositionMS:    position,
		DurationMS:    duration,
		Progress:      progress,
		Completed:     completed,
		LastWatchedAt: now,
		UpdatedAt:     now,
	}
	if err := s.store.Upsert(ctx, state); err != nil {
		return domain.WatchState{}, err
	}
	return state, nil
}

func (s *WatchService) States(ctx context.Context) ([]domain.WatchState, error) {
	if s == nil || s.store == nil {
		return []domain.WatchState{}, nil
	}
	return s.store.List(ctx)
}

func (s *WatchService) PruneOrphans(ctx context.Context) (int, error) {
	if s == nil || s.store == nil || s.catalog == nil {
		return 0, nil
	}
	states, err := s.store.List(ctx)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, state := range states {
		if err := ctx.Err(); err != nil {
			return removed, err
		}
		if _, getErr := s.catalog.GetMedia(ctx, state.MediaID); getErr == nil {
			continue
		} else if !errors.Is(getErr, domain.ErrNotFound) {
			return removed, getErr
		}
		if deleteErr := s.store.Delete(ctx, state.MediaID); deleteErr == nil {
			removed++
		} else if !errors.Is(deleteErr, domain.ErrNotFound) {
			return removed, deleteErr
		}
	}
	return removed, nil
}

func (s *WatchService) Get(ctx context.Context, mediaID string) (*domain.WatchState, error) {
	if s == nil || s.store == nil {
		return nil, nil
	}
	state, err := s.store.Get(ctx, mediaID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *WatchService) Decorate(ctx context.Context, items []domain.MediaItem) error {
	if s == nil || s.store == nil || len(items) == 0 {
		return nil
	}
	states, err := s.store.List(ctx)
	if err != nil {
		return err
	}
	byMedia := make(map[string]domain.WatchState, len(states))
	seriesStates := make(map[string][]domain.WatchState)
	for _, state := range states {
		current := state
		if item, getErr := s.catalog.GetMedia(ctx, state.MediaID); getErr == nil && item.SeriesID != "" {
			current.SeriesID = item.SeriesID
		}
		byMedia[current.MediaID] = current
		if current.SeriesID != "" {
			seriesStates[current.SeriesID] = append(seriesStates[current.SeriesID], current)
		}
	}
	for index := range items {
		item := &items[index]
		if item.Type == domain.MediaTypeSeries {
			state, found := aggregateSeriesWatch(*item, seriesStates[item.ID])
			if found {
				copyState := state
				item.Watch = &copyState
			}
			continue
		}
		if state, found := byMedia[item.ID]; found {
			copyState := state
			item.Watch = &copyState
		}
	}
	return nil
}

// aggregateSeriesWatch returns the most useful checkpoint for a series card.
// An actively unfinished episode wins. Otherwise the card shows the share of
// server-available episodes that have been completed, without mutating the
// episode-level records used by Continue Watching.
func aggregateSeriesWatch(series domain.MediaItem, states []domain.WatchState) (domain.WatchState, bool) {
	if len(states) == 0 {
		return domain.WatchState{}, false
	}
	latest := states[0]
	var latestIncomplete *domain.WatchState
	completedEpisodes := 0
	for index := range states {
		state := states[index]
		if state.LastWatchedAt.After(latest.LastWatchedAt) {
			latest = state
		}
		if state.Completed {
			completedEpisodes++
		}
		if state.Completed || state.Progress < continueMinimumProgress || state.Progress > continueMaximumProgress {
			continue
		}
		if latestIncomplete == nil || state.LastWatchedAt.After(latestIncomplete.LastWatchedAt) {
			copyState := state
			latestIncomplete = &copyState
		}
	}
	if latestIncomplete != nil {
		return *latestIncomplete, true
	}
	totalEpisodes := series.EpisodeCount
	if totalEpisodes <= 0 {
		totalEpisodes = len(states)
	}
	progress := 0.0
	if totalEpisodes > 0 {
		progress = float64(completedEpisodes) / float64(totalEpisodes)
		if progress > 1 {
			progress = 1
		}
	}
	latest.MediaID = series.ID
	latest.SeriesID = series.ID
	latest.MediaType = domain.MediaTypeSeries
	latest.PositionMS = int64(completedEpisodes)
	latest.DurationMS = int64(totalEpisodes)
	latest.Progress = progress
	latest.Completed = totalEpisodes > 0 && completedEpisodes >= totalEpisodes
	return latest, true
}

func (s *WatchService) ContinueWatching(ctx context.Context, limit int) ([]domain.MediaItem, error) {
	if s == nil || s.store == nil {
		return []domain.MediaItem{}, nil
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > 50 {
		limit = 50
	}
	states, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]domain.MediaItem, 0, limit)
	seenSeries := map[string]struct{}{}
	for _, state := range states {
		if state.Completed || state.Progress < continueMinimumProgress || state.Progress > continueMaximumProgress {
			continue
		}
		item, getErr := s.catalog.GetMedia(ctx, state.MediaID)
		if getErr != nil {
			continue
		}
		if item.Type == domain.MediaTypeMusic {
			continue
		}
		seriesID := item.SeriesID
		if seriesID != "" {
			if _, exists := seenSeries[seriesID]; exists {
				continue
			}
			seenSeries[seriesID] = struct{}{}
		}
		copyState := state
		copyState.SeriesID = seriesID
		item.Watch = &copyState
		result = append(result, item)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (s *WatchService) SeriesPlayTarget(ctx context.Context, seriesID string, episodes []domain.MediaItem) (*domain.MediaItem, error) {
	if len(episodes) == 0 {
		return nil, nil
	}
	if s == nil || s.store == nil {
		copyItem := episodes[0]
		return &copyItem, nil
	}
	states, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	stateByMedia := make(map[string]domain.WatchState, len(states))
	episodeIDs := make(map[string]struct{}, len(episodes))
	for _, episode := range episodes {
		episodeIDs[episode.ID] = struct{}{}
	}
	var latestIncomplete *domain.WatchState
	for _, state := range states {
		stateByMedia[state.MediaID] = state
		if _, belongsToSeries := episodeIDs[state.MediaID]; !belongsToSeries || state.Completed || state.Progress < continueMinimumProgress || state.Progress > continueMaximumProgress {
			continue
		}
		if latestIncomplete == nil || state.LastWatchedAt.After(latestIncomplete.LastWatchedAt) {
			copyState := state
			copyState.SeriesID = seriesID
			latestIncomplete = &copyState
		}
	}
	if latestIncomplete != nil {
		for _, episode := range episodes {
			if episode.ID == latestIncomplete.MediaID {
				copyItem := episode
				copyState := *latestIncomplete
				copyItem.Watch = &copyState
				return &copyItem, nil
			}
		}
	}
	for _, episode := range episodes {
		state, watched := stateByMedia[episode.ID]
		if !watched || !state.Completed {
			copyItem := episode
			if watched {
				copyState := state
				copyItem.Watch = &copyState
			}
			return &copyItem, nil
		}
	}
	copyItem := episodes[0]
	if state, found := stateByMedia[copyItem.ID]; found {
		copyState := state
		copyItem.Watch = &copyState
	}
	return &copyItem, nil
}

func SortEpisodesChronologically(items []domain.MediaItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Season != items[j].Season {
			return items[i].Season < items[j].Season
		}
		if items[i].Episode != items[j].Episode {
			return items[i].Episode < items[j].Episode
		}
		return items[i].SortTitle < items[j].SortTitle
	})
}
