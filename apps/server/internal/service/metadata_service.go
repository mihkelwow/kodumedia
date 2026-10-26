package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type MetadataService struct {
	catalog   ports.Catalog
	providers []ports.MetadataProvider
	ratings   []ports.RatingsProvider
	language  func() string
	now       func() time.Time
}

func NewMetadataService(catalog ports.Catalog, language string, providers []ports.MetadataProvider, ratings []ports.RatingsProvider) *MetadataService {
	return NewDynamicMetadataService(catalog, func() string { return language }, providers, ratings)
}

func NewDynamicMetadataService(catalog ports.Catalog, language func() string, providers []ports.MetadataProvider, ratings []ports.RatingsProvider) *MetadataService {
	return &MetadataService{catalog: catalog, providers: providers, ratings: ratings, language: language, now: time.Now}
}

func (s *MetadataService) Available() bool {
	return s.EditorialAvailable() || s.RatingsAvailable()
}

func (s *MetadataService) EditorialAvailable() bool {
	for _, provider := range s.providers {
		if provider != nil && provider.Available() {
			return true
		}
	}
	return false
}

func (s *MetadataService) RatingsAvailable() bool {
	for _, provider := range s.ratings {
		if provider != nil && provider.Available() {
			return true
		}
	}
	return false
}

func (s *MetadataService) Refresh(ctx context.Context, mediaID string) (domain.MediaItem, error) {
	item, err := s.catalog.GetMedia(ctx, mediaID)
	if err != nil {
		return domain.MediaItem{}, err
	}
	var parent *domain.MediaItem
	if item.Type == domain.MediaTypeEpisode && item.SeriesID != "" {
		series, parentErr := s.catalog.GetMedia(ctx, item.SeriesID)
		if parentErr == nil {
			parent = &series
		}
	}
	updated, err := s.Enrich(ctx, item, parent)
	if err != nil {
		return domain.MediaItem{}, err
	}
	if err := s.catalog.UpdateMedia(ctx, updated); err != nil {
		return domain.MediaItem{}, err
	}
	if updated.Type == domain.MediaTypeSeries {
		_ = s.propagateSeriesTitle(ctx, updated)
	}
	return updated, nil
}

// Enrich applies editorial metadata first and then an IMDb rating when each
// provider is available. It does no persistence, making it reusable by scans
// and background maintenance jobs.
func (s *MetadataService) Enrich(ctx context.Context, item domain.MediaItem, parent *domain.MediaItem) (domain.MediaItem, error) {
	if item.Type == domain.MediaTypeMusic || item.Type == domain.MediaTypeUnknown {
		return domain.MediaItem{}, domain.ErrUnsupported
	}
	if parent != nil && item.Type == domain.MediaTypeEpisode {
		item.SeriesID = parent.ID
		item.SeriesTitle = parent.Title
	}

	changed := false
	if s.EditorialAvailable() {
		result, found, err := s.matchEditorial(ctx, item, parent)
		if err != nil {
			return domain.MediaItem{}, err
		}
		if found {
			applyMetadataResult(&item, result)
			changed = true
		}
	}

	withRating, ratingChanged, ratingErr := s.ApplyRatings(ctx, item)
	if ratingErr != nil {
		// A local ratings cache failure must not discard successfully fetched
		// titles, artwork or descriptions. Ratings can be repaired by the
		// maintenance job independently.
		if !changed {
			return domain.MediaItem{}, ratingErr
		}
	} else if ratingChanged {
		item = withRating
		changed = true
	}
	if !changed {
		return domain.MediaItem{}, fmt.Errorf("%w: no metadata provider is configured or no match was found", domain.ErrUnavailable)
	}
	if item.Type == domain.MediaTypeEpisode && parent != nil {
		if item.Metadata.BackdropURL == "" {
			item.Metadata.BackdropURL = parent.Metadata.BackdropURL
		}
		item.SeriesTitle = parent.Title
		item.SortTitle = episodeSortTitle(item)
	}
	s.markUpdated(&item)
	return item, nil
}

func (s *MetadataService) matchEditorial(ctx context.Context, item domain.MediaItem, parent *domain.MediaItem) (domain.MetadataResult, bool, error) {
	query := domain.MetadataMatchQuery{
		MediaID:     item.ID,
		Type:        item.Type,
		Title:       item.Title,
		SeriesTitle: item.SeriesTitle,
		Year:        item.Year,
		Season:      item.Season,
		Episode:     item.Episode,
		Language:    s.currentLanguage(),
		ExternalIDs: item.ExternalIDs,
		Candidates:  metadataTitleCandidates(item, parent),
	}
	if parent != nil {
		query.ParentExternalIDs = parent.ExternalIDs
		if query.Year == 0 {
			query.Year = parent.Year
		}
	}
	for _, provider := range s.providers {
		if provider == nil || !provider.Available() {
			continue
		}
		result, err := provider.Match(ctx, query)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrUnsupported) {
				continue
			}
			return domain.MetadataResult{}, false, fmt.Errorf("metadata provider %s: %w", provider.Name(), err)
		}
		return result, true, nil
	}
	return domain.MetadataResult{}, false, nil
}

// ApplyRatings updates only IMDb rating fields and does not make a network
// request. The dataset provider uses a compact local index.
func (s *MetadataService) ApplyRatings(ctx context.Context, item domain.MediaItem) (domain.MediaItem, bool, error) {
	if item.ExternalIDs.IMDb == "" {
		return item, false, nil
	}
	for _, provider := range s.ratings {
		if provider == nil || !provider.Available() {
			continue
		}
		rating, votes, found, err := provider.Lookup(ctx, item.ExternalIDs.IMDb)
		if err != nil {
			if errors.Is(err, domain.ErrUnavailable) {
				continue
			}
			return domain.MediaItem{}, false, fmt.Errorf("ratings provider %s: %w", provider.Name(), err)
		}
		if !found {
			continue
		}
		if item.Metadata.IMDbRating == rating && item.Metadata.IMDbVotes == votes {
			return item, false, nil
		}
		item.Metadata.IMDbRating = rating
		item.Metadata.IMDbVotes = votes
		s.markUpdated(&item)
		return item, true, nil
	}
	return item, false, nil
}

func (s *MetadataService) markUpdated(item *domain.MediaItem) {
	now := s.now().UTC()
	item.Metadata.UpdatedAt = &now
	item.UpdatedAt = now
}

func (s *MetadataService) currentLanguage() string {
	if s.language == nil {
		return domain.DefaultMetadataLanguage
	}
	if language := strings.TrimSpace(s.language()); language != "" {
		return language
	}
	return domain.DefaultMetadataLanguage
}

func (s *MetadataService) propagateSeriesTitle(ctx context.Context, series domain.MediaItem) error {
	items := make([]domain.MediaItem, 0, series.EpisodeCount)
	offset := 0
	for {
		page, err := s.catalog.ListMedia(ctx, domain.MediaQuery{SeriesID: series.ID, Types: []domain.MediaType{domain.MediaTypeEpisode}, Limit: 500, Offset: offset, Sort: domain.MediaSortTitle, Order: domain.SortAscending})
		if err != nil {
			return err
		}
		for _, episode := range page.Items {
			episode.SeriesTitle = series.Title
			episode.SortTitle = episodeSortTitle(episode)
			items = append(items, episode)
		}
		offset += len(page.Items)
		if offset >= page.Total || len(page.Items) == 0 {
			break
		}
	}
	if len(items) == 0 {
		return nil
	}
	return s.catalog.UpdateMediaBatch(ctx, items)
}

func needsEditorialMetadata(item domain.MediaItem) bool {
	if item.Type != domain.MediaTypeMovie && item.Type != domain.MediaTypeSeries && item.Type != domain.MediaTypeEpisode {
		return false
	}
	if item.Type == domain.MediaTypeEpisode && (item.Season <= 0 || item.Episode <= 0) {
		return false
	}
	if item.ExternalIDs.TMDb == "" || item.Metadata.MetadataSource == "" || item.Metadata.Summary == "" {
		return true
	}
	if item.Type == domain.MediaTypeMovie {
		return item.Metadata.PosterURL == "" || item.Metadata.HeroAssetsVersion < 2
	}
	if item.Type == domain.MediaTypeSeries {
		if item.Metadata.PosterURL == "" || item.Metadata.HeroAssetsVersion < 2 || item.Metadata.OfficialSeasonCount == 0 || item.Metadata.OfficialEpisodeCount == 0 || item.Metadata.LatestEpisode == nil {
			return true
		}
		return item.Metadata.UpdatedAt == nil || time.Since(*item.Metadata.UpdatedAt) >= 24*time.Hour
	}
	return false
}

func applyMetadataResult(item *domain.MediaItem, result domain.MetadataResult) {
	if result.Title != "" {
		if item.Type == domain.MediaTypeEpisode {
			item.EpisodeTitle = result.Title
			item.Title = result.Title
			item.SortTitle = episodeSortTitle(*item)
		} else {
			item.Title = result.Title
			item.SortTitle = normalizeSortTitle(result.Title)
		}
	}
	if result.Year > 0 {
		item.Year = result.Year
	}
	if result.ReleaseDate != nil {
		item.ReleaseDate = result.ReleaseDate
	}
	if result.ExternalIDs.IMDb != "" {
		item.ExternalIDs.IMDb = result.ExternalIDs.IMDb
	}
	if result.ExternalIDs.TMDb != "" {
		item.ExternalIDs.TMDb = result.ExternalIDs.TMDb
	}
	if result.ExternalIDs.TVDb != "" {
		item.ExternalIDs.TVDb = result.ExternalIDs.TVDb
	}
	if result.Metadata.Summary != "" {
		item.Metadata.Summary = result.Metadata.Summary
	}
	if result.Metadata.Tagline != "" {
		item.Metadata.Tagline = result.Metadata.Tagline
	}
	if len(result.Metadata.Genres) > 0 {
		item.Metadata.Genres = result.Metadata.Genres
	}
	if len(result.Metadata.Cast) > 0 {
		item.Metadata.Cast = result.Metadata.Cast
	}
	if len(result.Metadata.Directors) > 0 {
		item.Metadata.Directors = result.Metadata.Directors
	}
	if result.Metadata.PosterURL != "" {
		item.Metadata.PosterURL = result.Metadata.PosterURL
	}
	if result.Metadata.BackdropURL != "" {
		item.Metadata.BackdropURL = result.Metadata.BackdropURL
	}
	if result.Metadata.LogoURL != "" {
		item.Metadata.LogoURL = result.Metadata.LogoURL
	}
	if result.Metadata.HeroAssetsVersion >= 2 {
		// Version 2 is an authoritative refresh of homepage video assets. Copying
		// empty values is intentional so a removed or no-longer-usable trailer
		// cannot remain pinned from an older catalog response.
		item.Metadata.TrailerYouTubeKey = result.Metadata.TrailerYouTubeKey
		item.Metadata.TrailerName = result.Metadata.TrailerName
		item.Metadata.TrailerCandidates = append([]domain.TrailerCandidate(nil), result.Metadata.TrailerCandidates...)
	} else {
		if result.Metadata.TrailerYouTubeKey != "" {
			item.Metadata.TrailerYouTubeKey = result.Metadata.TrailerYouTubeKey
			item.Metadata.TrailerName = result.Metadata.TrailerName
		}
		if len(result.Metadata.TrailerCandidates) > 0 {
			item.Metadata.TrailerCandidates = append([]domain.TrailerCandidate(nil), result.Metadata.TrailerCandidates...)
		}
	}
	if result.Metadata.HeroAssetsVersion > 0 {
		item.Metadata.HeroAssetsVersion = result.Metadata.HeroAssetsVersion
	}
	if result.Metadata.MetadataSource != "" {
		item.Metadata.MetadataSource = result.Metadata.MetadataSource
	}
	if result.Metadata.OfficialSeasonCount > 0 {
		item.Metadata.OfficialSeasonCount = result.Metadata.OfficialSeasonCount
	}
	if result.Metadata.OfficialEpisodeCount > 0 {
		item.Metadata.OfficialEpisodeCount = result.Metadata.OfficialEpisodeCount
	}
	if result.Metadata.LatestEpisode != nil {
		latest := *result.Metadata.LatestEpisode
		item.Metadata.LatestEpisode = &latest
	}
}

func episodeSortTitle(item domain.MediaItem) string {
	return normalizeSortTitle(fmt.Sprintf("%s %04d %04d %s", item.SeriesTitle, item.Season, item.Episode, item.Title))
}
