package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

const notificationPageSize = 500
const notificationHistoryLimit = 1000

type NotificationService struct {
	catalog  ports.Catalog
	store    ports.NotificationRepository
	now      func() time.Time
	syncMu   sync.Mutex
	lastSync time.Time
}

func NewNotificationService(catalog ports.Catalog, store ports.NotificationRepository) *NotificationService {
	return &NotificationService{catalog: catalog, store: store, now: time.Now}
}

func (s *NotificationService) List(ctx context.Context, limit, offset int) (domain.NotificationPage, error) {
	if err := s.sync(ctx); err != nil {
		return domain.NotificationPage{}, err
	}
	items, err := s.store.List(ctx)
	if err != nil {
		return domain.NotificationPage{}, err
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	unread := 0
	for _, item := range items {
		if item.ReadAt == nil {
			unread++
		}
	}
	total := len(items)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return domain.NotificationPage{
		Items:  append([]domain.Notification(nil), items[offset:end]...),
		Total:  total,
		Unread: unread,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *NotificationService) MarkAllRead(ctx context.Context) error {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	return s.store.MarkAllRead(ctx, s.now().UTC())
}

func (s *NotificationService) sync(ctx context.Context) error {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	now := s.now().UTC()
	if !s.lastSync.IsZero() && now.Sub(s.lastSync) < 30*time.Second {
		return nil
	}
	seriesItems, err := s.listAll(ctx, []domain.MediaType{domain.MediaTypeSeries})
	if err != nil {
		return err
	}
	episodes, err := s.listAll(ctx, []domain.MediaType{domain.MediaTypeEpisode})
	if err != nil {
		return err
	}
	existing, err := s.store.List(ctx)
	if err != nil {
		return err
	}

	latestLocal := make(map[string]domain.MediaItem)
	for _, episode := range episodes {
		if episode.SeriesID == "" {
			continue
		}
		current, found := latestLocal[episode.SeriesID]
		if !found || notificationEpisodeAfter(episode.Season, episode.Episode, current.Season, current.Episode) {
			latestLocal[episode.SeriesID] = episode
		}
	}

	items := append([]domain.Notification(nil), existing...)
	indexByID := make(map[string]int, len(items))
	changed := false
	for index := range items {
		indexByID[items[index].ID] = index
		if items[index].Kind != domain.NotificationKindSeriesEpisodeReleased || items[index].ResolvedAt != nil {
			continue
		}
		local, found := latestLocal[items[index].SeriesID]
		if found && !notificationEpisodeAfter(items[index].Season, items[index].Episode, local.Season, local.Episode) {
			resolved := now
			items[index].ResolvedAt = &resolved
			changed = true
		}
	}

	for _, series := range seriesItems {
		latest := series.Metadata.LatestEpisode
		if latest == nil || latest.Season <= 0 || latest.Episode <= 0 {
			continue
		}
		if latest.ReleaseDate != nil && latest.ReleaseDate.After(now) {
			continue
		}
		local, found := latestLocal[series.ID]
		if found && !notificationEpisodeAfter(latest.Season, latest.Episode, local.Season, local.Episode) {
			continue
		}
		id := notificationID(series.ID, latest.Season, latest.Episode)
		if index, exists := indexByID[id]; exists {
			item := &items[index]
			if item.SeriesTitle != series.Title || item.EpisodeTitle != latest.Title || !sameOptionalTime(item.ReleaseDate, latest.ReleaseDate) || item.ResolvedAt != nil {
				item.SeriesTitle = series.Title
				item.EpisodeTitle = latest.Title
				item.ReleaseDate = latest.ReleaseDate
				item.ResolvedAt = nil
				changed = true
			}
			continue
		}
		items = append(items, domain.Notification{
			ID:           id,
			Kind:         domain.NotificationKindSeriesEpisodeReleased,
			SeriesID:     series.ID,
			SeriesTitle:  series.Title,
			Season:       latest.Season,
			Episode:      latest.Episode,
			EpisodeTitle: latest.Title,
			ReleaseDate:  latest.ReleaseDate,
			CreatedAt:    now,
		})
		indexByID[id] = len(items) - 1
		changed = true
	}

	// Preserve a bounded history so the notifications page remains lightweight.
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	if len(items) > notificationHistoryLimit {
		items = items[:notificationHistoryLimit]
		changed = true
	}
	if changed {
		if err := s.store.Replace(ctx, items); err != nil {
			return err
		}
	}
	s.lastSync = now
	return nil
}

func (s *NotificationService) listAll(ctx context.Context, types []domain.MediaType) ([]domain.MediaItem, error) {
	items := make([]domain.MediaItem, 0, notificationPageSize)
	for offset := 0; ; offset += notificationPageSize {
		page, err := s.catalog.ListMedia(ctx, domain.MediaQuery{
			Types:  types,
			Sort:   domain.MediaSortTitle,
			Order:  domain.SortAscending,
			Limit:  notificationPageSize,
			Offset: offset,
		})
		if err != nil {
			return nil, err
		}
		items = append(items, page.Items...)
		if len(items) >= page.Total || len(page.Items) == 0 {
			return items, nil
		}
	}
}

func notificationID(seriesID string, season, episode int) string {
	return fmt.Sprintf("series:%s:s%02de%02d", seriesID, season, episode)
}

func notificationEpisodeAfter(leftSeason, leftEpisode, rightSeason, rightEpisode int) bool {
	if leftSeason != rightSeason {
		return leftSeason > rightSeason
	}
	return leftEpisode > rightEpisode
}

func sameOptionalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}
