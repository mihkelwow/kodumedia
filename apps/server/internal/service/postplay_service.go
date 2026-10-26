package service

import (
	"context"
	"sort"
	"strings"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

const postPlayRecommendationLimit = 3

type PostPlayService struct {
	catalog ports.Catalog
	watch   *WatchService
}

func NewPostPlayService(catalog ports.Catalog, watch *WatchService) *PostPlayService {
	return &PostPlayService{catalog: catalog, watch: watch}
}

func (s *PostPlayService) Get(ctx context.Context, mediaID string) (domain.PostPlay, error) {
	current, err := s.catalog.GetMedia(ctx, mediaID)
	if err != nil {
		return domain.PostPlay{}, err
	}
	if current.Type != domain.MediaTypeEpisode || current.SeriesID == "" {
		return domain.PostPlay{Recommendations: []domain.PostPlayRecommendation{}}, nil
	}

	allItems, err := s.listAll(ctx)
	if err != nil {
		return domain.PostPlay{}, err
	}
	itemsByID := make(map[string]domain.MediaItem, len(allItems))
	episodesBySeries := make(map[string][]domain.MediaItem)
	for _, item := range allItems {
		itemsByID[item.ID] = item
		if item.Type == domain.MediaTypeEpisode && item.SeriesID != "" {
			episodesBySeries[item.SeriesID] = append(episodesBySeries[item.SeriesID], item)
		}
	}
	for seriesID := range episodesBySeries {
		SortEpisodesChronologically(episodesBySeries[seriesID])
	}

	if next := nextSeriesEpisode(current, episodesBySeries[current.SeriesID]); next != nil {
		if series, found := itemsByID[current.SeriesID]; found {
			if next.SeriesTitle == "" {
				next.SeriesTitle = series.Title
			}
			if next.Metadata.BackdropURL == "" {
				next.Metadata.BackdropURL = series.Metadata.BackdropURL
			}
			if next.Metadata.PosterURL == "" {
				next.Metadata.PosterURL = series.Metadata.PosterURL
			}
		}
		items := []domain.MediaItem{*next}
		if s.watch != nil {
			if err := s.watch.Decorate(ctx, items); err != nil {
				return domain.PostPlay{}, err
			}
		}
		return domain.PostPlay{NextEpisode: &items[0], Recommendations: []domain.PostPlayRecommendation{}}, nil
	}

	source := current
	if series, found := itemsByID[current.SeriesID]; found {
		source = series
	}
	states := map[string]domain.WatchState{}
	if s.watch != nil {
		watchStates, watchErr := s.watch.States(ctx)
		if watchErr != nil {
			return domain.PostPlay{}, watchErr
		}
		for _, state := range watchStates {
			states[state.MediaID] = state
		}
	}

	type scoredRecommendation struct {
		recommendation domain.PostPlayRecommendation
		score          float64
		related        bool
	}
	scored := make([]scoredRecommendation, 0, len(allItems))
	for _, candidate := range allItems {
		if candidate.ID == source.ID || candidate.ID == current.SeriesID {
			continue
		}
		var target *domain.MediaItem
		switch candidate.Type {
		case domain.MediaTypeMovie:
			if watchedAtLeastOnePercent(states[candidate.ID]) {
				continue
			}
			copyItem := candidate
			target = &copyItem
		case domain.MediaTypeSeries:
			episodes := episodesBySeries[candidate.ID]
			oldest := oldestUnwatchedEpisode(episodes, states)
			if oldest == nil {
				continue
			}
			target = oldest
		default:
			continue
		}

		shared, relatedCount := genreRelationship(source.Metadata.Genres, candidate.Metadata.Genres)
		related := len(shared) > 0 || relatedCount > 0
		score := float64(len(shared))*40 + float64(relatedCount)*12 + candidate.Metadata.IMDbRating*2
		if candidate.Year > 0 {
			score += float64(candidate.Year-1900) / 250
		}
		if candidate.Metadata.PosterURL != "" || candidate.Metadata.BackdropURL != "" {
			score += 2
		}
		reason := "Vaatamata soovitus"
		if len(shared) > 0 {
			reason = "Sama žanr: " + strings.Join(shared, ", ")
		} else if relatedCount > 0 {
			reason = "Sarnane vaatamiselamus"
		} else if candidate.Metadata.IMDbRating > 0 {
			reason = "Vaatamata ja kõrgelt hinnatud"
		}
		candidateCopy := candidate
		targetCopy := *target
		scored = append(scored, scoredRecommendation{
			recommendation: domain.PostPlayRecommendation{
				Item:         candidateCopy,
				PlayTarget:   &targetCopy,
				SharedGenres: shared,
				Reason:       reason,
			},
			score:   score,
			related: related,
		})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].related != scored[j].related {
			return scored[i].related
		}
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return strings.ToLower(scored[i].recommendation.Item.SortTitle) < strings.ToLower(scored[j].recommendation.Item.SortTitle)
	})
	limit := postPlayRecommendationLimit
	if len(scored) < limit {
		limit = len(scored)
	}
	result := make([]domain.PostPlayRecommendation, 0, limit)
	for _, value := range scored[:limit] {
		result = append(result, value.recommendation)
	}
	if s.watch != nil && len(result) > 0 {
		decorated := make([]domain.MediaItem, 0, len(result)*2)
		for _, recommendation := range result {
			decorated = append(decorated, recommendation.Item)
			if recommendation.PlayTarget != nil {
				decorated = append(decorated, *recommendation.PlayTarget)
			}
		}
		if err := s.watch.Decorate(ctx, decorated); err != nil {
			return domain.PostPlay{}, err
		}
		index := 0
		for recommendationIndex := range result {
			result[recommendationIndex].Item = decorated[index]
			index++
			if result[recommendationIndex].PlayTarget != nil {
				target := decorated[index]
				result[recommendationIndex].PlayTarget = &target
				index++
			}
		}
	}
	return domain.PostPlay{Recommendations: result}, nil
}

func (s *PostPlayService) listAll(ctx context.Context) ([]domain.MediaItem, error) {
	const pageSize = 500
	items := make([]domain.MediaItem, 0, 512)
	for offset := 0; ; offset += pageSize {
		page, err := s.catalog.ListMedia(ctx, domain.MediaQuery{
			Types:  []domain.MediaType{domain.MediaTypeMovie, domain.MediaTypeSeries, domain.MediaTypeEpisode},
			Sort:   domain.MediaSortTitle,
			Order:  domain.SortAscending,
			Limit:  pageSize,
			Offset: offset,
		})
		if err != nil {
			return nil, err
		}
		items = append(items, page.Items...)
		if len(page.Items) == 0 || len(items) >= page.Total {
			break
		}
	}
	return items, nil
}

func nextSeriesEpisode(current domain.MediaItem, episodes []domain.MediaItem) *domain.MediaItem {
	for index := range episodes {
		if episodes[index].ID == current.ID {
			if index+1 < len(episodes) {
				copyItem := episodes[index+1]
				return &copyItem
			}
			return nil
		}
	}
	for _, episode := range episodes {
		if episode.Season > current.Season || (episode.Season == current.Season && episode.Episode > current.Episode) {
			copyItem := episode
			return &copyItem
		}
	}
	return nil
}

func oldestUnwatchedEpisode(episodes []domain.MediaItem, states map[string]domain.WatchState) *domain.MediaItem {
	for _, episode := range episodes {
		state, found := states[episode.ID]
		if !found || (!state.Completed && state.Progress < continueMinimumProgress) {
			copyItem := episode
			return &copyItem
		}
	}
	return nil
}

func watchedAtLeastOnePercent(state domain.WatchState) bool {
	return state.Completed || state.Progress >= continueMinimumProgress
}

var relatedGenreGroups = [][]string{
	{"drama", "crime", "thriller", "mystery"},
	{"science fiction", "sci-fi", "fantasy", "adventure"},
	{"action", "adventure", "war", "western"},
	{"comedy", "family", "animation", "romance"},
	{"horror", "thriller", "mystery"},
	{"documentary", "history", "biography"},
}

func genreRelationship(source, candidate []string) ([]string, int) {
	sourceSet := make(map[string]string, len(source))
	candidateSet := make(map[string]struct{}, len(candidate))
	for _, genre := range source {
		normalized := strings.ToLower(strings.TrimSpace(genre))
		if normalized != "" {
			sourceSet[normalized] = genre
		}
	}
	for _, genre := range candidate {
		normalized := strings.ToLower(strings.TrimSpace(genre))
		if normalized != "" {
			candidateSet[normalized] = struct{}{}
		}
	}
	shared := make([]string, 0, 3)
	for normalized, original := range sourceSet {
		if _, found := candidateSet[normalized]; found {
			shared = append(shared, original)
		}
	}
	sort.Strings(shared)
	related := 0
	for _, group := range relatedGenreGroups {
		sourceInGroup := false
		candidateInGroup := false
		for _, genre := range group {
			if _, found := sourceSet[genre]; found {
				sourceInGroup = true
			}
			if _, found := candidateSet[genre]; found {
				candidateInGroup = true
			}
		}
		if sourceInGroup && candidateInGroup {
			related++
		}
	}
	return shared, related
}
