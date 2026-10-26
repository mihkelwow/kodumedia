package service

import (
	"context"
	"fmt"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type MediaOption func(*MediaService)

func WithMediaWatch(watch *WatchService) MediaOption {
	return func(service *MediaService) { service.watch = watch }
}

type MediaService struct {
	catalog ports.Catalog
	watch   *WatchService
}

func NewMediaService(catalog ports.Catalog, options ...MediaOption) *MediaService {
	service := &MediaService{catalog: catalog}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *MediaService) List(ctx context.Context, query domain.MediaQuery) (domain.MediaPage, error) {
	page, err := s.catalog.ListMedia(ctx, query)
	if err != nil {
		return domain.MediaPage{}, err
	}
	if s.watch != nil {
		if err := s.watch.Decorate(ctx, page.Items); err != nil {
			return domain.MediaPage{}, err
		}
	}
	return page, nil
}

func (s *MediaService) Get(ctx context.Context, id string) (domain.MediaItem, error) {
	item, err := s.catalog.GetMedia(ctx, id)
	if err != nil {
		return domain.MediaItem{}, err
	}
	items := []domain.MediaItem{item}
	if s.watch != nil {
		if err := s.watch.Decorate(ctx, items); err != nil {
			return domain.MediaItem{}, err
		}
	}
	return items[0], nil
}

func (s *MediaService) GetSeriesDetails(ctx context.Context, id string) (domain.SeriesDetails, error) {
	series, err := s.catalog.GetMedia(ctx, id)
	if err != nil {
		return domain.SeriesDetails{}, err
	}
	if series.Type != domain.MediaTypeSeries {
		return domain.SeriesDetails{}, fmt.Errorf("%w: media item is not a series", domain.ErrInvalidInput)
	}
	episodes, err := s.listAllSeriesEpisodes(ctx, id)
	if err != nil {
		return domain.SeriesDetails{}, err
	}
	SortEpisodesChronologically(episodes)
	if s.watch != nil {
		items := append([]domain.MediaItem{series}, episodes...)
		if err := s.watch.Decorate(ctx, items); err != nil {
			return domain.SeriesDetails{}, err
		}
		series = items[0]
		episodes = items[1:]
	}
	seasonIndexes := map[int]int{}
	seasons := make([]domain.SeasonGroup, 0, series.SeasonCount)
	for _, episode := range episodes {
		index, exists := seasonIndexes[episode.Season]
		if !exists {
			title := fmt.Sprintf("Hooaeg %d", episode.Season)
			if episode.Season == 0 {
				title = "Eriepisoodid"
			}
			index = len(seasons)
			seasonIndexes[episode.Season] = index
			seasons = append(seasons, domain.SeasonGroup{Number: episode.Season, Title: title, Episodes: []domain.MediaItem{}})
		}
		seasons[index].Episodes = append(seasons[index].Episodes, episode)
		seasons[index].EpisodeCount = len(seasons[index].Episodes)
	}
	series.SeasonCount = len(seasons)
	series.EpisodeCount = len(episodes)
	availability := seriesAvailability(series, episodes)
	var playTarget *domain.MediaItem
	if s.watch != nil {
		playTarget, err = s.watch.SeriesPlayTarget(ctx, id, episodes)
		if err != nil {
			return domain.SeriesDetails{}, err
		}
	} else if len(episodes) > 0 {
		copyItem := episodes[0]
		playTarget = &copyItem
	}
	return domain.SeriesDetails{Series: series, Seasons: seasons, PlayTarget: playTarget, Availability: availability}, nil
}

func seriesAvailability(series domain.MediaItem, episodes []domain.MediaItem) domain.SeriesAvailability {
	result := domain.SeriesAvailability{
		ReleasedSeasonCount:   series.Metadata.OfficialSeasonCount,
		ReleasedEpisodeCount:  series.Metadata.OfficialEpisodeCount,
		AvailableSeasonCount:  series.SeasonCount,
		AvailableEpisodeCount: len(episodes),
		LatestReleasedEpisode: series.Metadata.LatestEpisode,
	}
	var latest *domain.MediaItem
	for index := range episodes {
		episode := &episodes[index]
		if latest == nil || episodeAfter(*episode, *latest) {
			latest = episode
		}
	}
	if latest != nil {
		result.LatestAvailableSeason = latest.Season
		result.LatestAvailableEpisode = latest.Episode
		result.LatestAvailableDate = latest.ReleaseDate
	}
	if result.ReleasedSeasonCount == 0 {
		result.ReleasedSeasonCount = series.SeasonCount
	}
	if result.ReleasedEpisodeCount == 0 {
		result.ReleasedEpisodeCount = len(episodes)
	}
	if result.LatestReleasedEpisode != nil {
		latestReleased := result.LatestReleasedEpisode
		result.NewEpisodeAvailable = latest == nil || latestReleased.Season > latest.Season ||
			(latestReleased.Season == latest.Season && latestReleased.Episode > latest.Episode)
	}
	return result
}

func episodeAfter(left, right domain.MediaItem) bool {
	if left.Season != right.Season {
		return left.Season > right.Season
	}
	if left.Episode != right.Episode {
		return left.Episode > right.Episode
	}
	leftDate, rightDate := time.Time{}, time.Time{}
	if left.ReleaseDate != nil {
		leftDate = *left.ReleaseDate
	}
	if right.ReleaseDate != nil {
		rightDate = *right.ReleaseDate
	}
	return leftDate.After(rightDate)
}

func (s *MediaService) listAllSeriesEpisodes(ctx context.Context, seriesID string) ([]domain.MediaItem, error) {
	const pageSize = 500
	episodes := make([]domain.MediaItem, 0, 64)
	for offset := 0; ; offset += pageSize {
		page, err := s.catalog.ListMedia(ctx, domain.MediaQuery{SeriesID: seriesID, Types: []domain.MediaType{domain.MediaTypeEpisode}, Sort: domain.MediaSortTitle, Order: domain.SortAscending, Limit: pageSize, Offset: offset})
		if err != nil {
			return nil, err
		}
		episodes = append(episodes, page.Items...)
		if len(episodes) >= page.Total || len(page.Items) == 0 {
			break
		}
	}
	return episodes, nil
}
