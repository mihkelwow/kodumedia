package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type ratingsInvalidator interface {
	Invalidate()
}

// MetadataMaintenanceService owns background-only metadata work. Keeping this
// orchestration out of HTTP handlers and scanners prevents UI requests from
// parsing or downloading large datasets.
type MetadataMaintenanceService struct {
	ctx      context.Context
	catalog  ports.Catalog
	metadata *MetadataService
	dataset  ports.RatingsDataset
	ratings  ratingsInvalidator
	jobs     ports.JobRunner

	jobMu              sync.Mutex
	activeIMDbJob      string
	activeBackfillJob  string
	pendingIMDbRefresh bool
	workMu             sync.Mutex
}

func NewMetadataMaintenanceService(
	ctx context.Context,
	catalog ports.Catalog,
	metadata *MetadataService,
	dataset ports.RatingsDataset,
	ratings ratingsInvalidator,
	jobs ports.JobRunner,
) *MetadataMaintenanceService {
	return &MetadataMaintenanceService{ctx: ctx, catalog: catalog, metadata: metadata, dataset: dataset, ratings: ratings, jobs: jobs}
}

// Start schedules non-blocking startup maintenance. The server begins serving
// immediately; copied catalogs are enriched in the background.
func (s *MetadataMaintenanceService) Start() {
	if s == nil {
		return
	}
	// Repair titles and artwork first. The IMDb transfer may continue in
	// parallel, but its catalog update is serialized behind this backfill.
	// This keeps copied 0.2.0 catalogs visually useful as soon as possible.
	if s.metadata != nil && s.metadata.EditorialAvailable() {
		s.BackfillMissing(false)
	}
	if s.dataset != nil {
		s.RefreshIMDb()
	}
}

func (s *MetadataMaintenanceService) RunPeriodicRefresh() {
	if s == nil {
		return
	}
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.RefreshIMDb()
		}
	}
}

func (s *MetadataMaintenanceService) OnSettingsChanged(previous, current domain.ServerSettings) {
	if s == nil {
		return
	}
	previousSource := strings.TrimSpace(previous.Metadata.IMDbRatingsSource)
	currentSource := strings.TrimSpace(current.Metadata.IMDbRatingsSource)
	if !strings.EqualFold(previousSource, currentSource) {
		if s.ratings != nil {
			s.ratings.Invalidate()
		}
		s.refreshIMDbAfterActiveJob()
	}

	currentToken := strings.TrimSpace(current.Metadata.TMDbToken)
	languageChanged := !strings.EqualFold(strings.TrimSpace(previous.Metadata.Language), strings.TrimSpace(current.Metadata.Language))
	// Saving valid TMDb settings is also an explicit repair signal. This makes
	// copied catalogs recover automatically even when the token itself did not
	// change (for example after a temporary network failure).
	if currentToken != "" {
		s.BackfillMissing(languageChanged)
	}
}

func (s *MetadataMaintenanceService) refreshIMDbAfterActiveJob() {
	if s == nil || s.jobs == nil || s.dataset == nil {
		return
	}
	s.jobMu.Lock()
	job, active := s.activeJobLocked(s.activeIMDbJob)
	if !active {
		s.jobMu.Unlock()
		s.RefreshIMDb()
		return
	}
	if s.pendingIMDbRefresh {
		s.jobMu.Unlock()
		return
	}
	s.pendingIMDbRefresh = true
	s.jobMu.Unlock()

	go func(jobID string) {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				s.jobMu.Lock()
				s.pendingIMDbRefresh = false
				s.jobMu.Unlock()
				return
			case <-ticker.C:
				current, found := s.jobs.Get(jobID)
				if found && (current.Status == domain.JobStatusQueued || current.Status == domain.JobStatusRunning) {
					continue
				}
				s.jobMu.Lock()
				s.pendingIMDbRefresh = false
				s.jobMu.Unlock()
				s.RefreshIMDb()
				return
			}
		}
	}(job.ID)
}

func (s *MetadataMaintenanceService) RefreshIMDb() domain.Job {
	if s == nil || s.jobs == nil || s.dataset == nil {
		return domain.Job{Type: "imdb-ratings-refresh", Status: domain.JobStatusFailed, Error: "IMDb dataset manager is unavailable"}
	}
	s.jobMu.Lock()
	if job, found := s.activeJobLocked(s.activeIMDbJob); found {
		s.jobMu.Unlock()
		return job
	}
	job := s.jobs.Start("imdb-ratings-refresh", func(update ports.JobProgress) error {
		// Network I/O does not hold the catalog-maintenance lock. TMDb artwork
		// repair can therefore proceed while the dataset is downloading. Only
		// the final catalog mutation is serialized to avoid lost updates.
		if err := s.dataset.Refresh(s.ctx, update); err != nil {
			return err
		}
		if s.ratings != nil {
			s.ratings.Invalidate()
		}
		update(86, "Uuendan kataloogi IMDb skoore")
		s.workMu.Lock()
		defer s.workMu.Unlock()
		return s.backfill(s.ctx, false, true, update, 86, 99)
	})
	s.activeIMDbJob = job.ID
	s.jobMu.Unlock()
	return job
}

func (s *MetadataMaintenanceService) BackfillMissing(forceEditorial bool) domain.Job {
	if s == nil || s.jobs == nil || s.metadata == nil {
		return domain.Job{Type: "metadata-backfill", Status: domain.JobStatusFailed, Error: "metadata service is unavailable"}
	}
	s.jobMu.Lock()
	if job, found := s.activeJobLocked(s.activeBackfillJob); found {
		s.jobMu.Unlock()
		return job
	}
	job := s.jobs.Start("metadata-backfill", func(update ports.JobProgress) error {
		s.workMu.Lock()
		defer s.workMu.Unlock()
		return s.backfill(s.ctx, forceEditorial, false, update, 3, 99)
	})
	s.activeBackfillJob = job.ID
	s.jobMu.Unlock()
	return job
}

func (s *MetadataMaintenanceService) activeJobLocked(id string) (domain.Job, bool) {
	if id == "" {
		return domain.Job{}, false
	}
	job, found := s.jobs.Get(id)
	if !found || (job.Status != domain.JobStatusQueued && job.Status != domain.JobStatusRunning) {
		return domain.Job{}, false
	}
	return job, true
}

func (s *MetadataMaintenanceService) backfill(
	ctx context.Context,
	forceEditorial bool,
	ratingsOnly bool,
	update ports.JobProgress,
	startProgress int,
	endProgress int,
) error {
	items, err := s.listAllMedia(ctx)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		update(endProgress, "Kataloogis pole veel meediat")
		return nil
	}

	repairedSeries := make(map[string]bool)
	changed := make(map[string]domain.MediaItem)
	if !ratingsOnly {
		repairedSeries = repairSuspiciousSeriesMatches(items)
		for index := range items {
			item := items[index]
			if repairedSeries[item.ID] || (item.Type == domain.MediaTypeEpisode && repairedSeries[item.SeriesID]) {
				changed[item.ID] = item
			}
		}
	}

	parents := make(map[string]*domain.MediaItem)
	for index := range items {
		if items[index].Type == domain.MediaTypeSeries {
			parents[items[index].ID] = &items[index]
		}
	}
	order := make([]int, 0, len(items))
	for index := range items {
		if items[index].Type == domain.MediaTypeSeries {
			order = append(order, index)
		}
	}
	for index := range items {
		if items[index].Type != domain.MediaTypeSeries {
			order = append(order, index)
		}
	}

	processed := 0
	for _, index := range order {
		if err := ctx.Err(); err != nil {
			return err
		}
		item := items[index]
		if item.Type != domain.MediaTypeMovie && item.Type != domain.MediaTypeSeries && item.Type != domain.MediaTypeEpisode {
			processed++
			continue
		}
		var parent *domain.MediaItem
		if item.Type == domain.MediaTypeEpisode {
			parent = parents[item.SeriesID]
			if parent != nil {
				item.SeriesTitle = parent.Title
			}
		}

		updated := item
		itemChanged := false
		forceItemRefresh := forceEditorial || repairedSeries[item.ID] || (item.Type == domain.MediaTypeEpisode && repairedSeries[item.SeriesID])
		if !ratingsOnly && s.metadata.EditorialAvailable() && (forceItemRefresh || needsEditorialMetadata(item)) {
			enrichCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
			candidate, enrichErr := s.metadata.Enrich(enrichCtx, item, parent)
			cancel()
			if enrichErr == nil {
				updated = candidate
				itemChanged = true
			} else if !errors.Is(enrichErr, domain.ErrUnavailable) && !errors.Is(enrichErr, domain.ErrNotFound) && !errors.Is(enrichErr, domain.ErrUnsupported) {
				// A single ambiguous or temporarily unavailable title must not stop
				// the rest of a copied catalog from being repaired.
			}
		} else if s.metadata.RatingsAvailable() {
			candidate, ratingChanged, ratingErr := s.metadata.ApplyRatings(ctx, item)
			if ratingErr == nil && ratingChanged {
				updated = candidate
				itemChanged = true
			}
		}

		if itemChanged {
			items[index] = updated
			changed[updated.ID] = updated
			if updated.Type == domain.MediaTypeSeries {
				parents[updated.ID] = &items[index]
			}
		}
		processed++
		progress := startProgress + int((int64(processed)*int64(endProgress-startProgress))/int64(len(order)))
		update(progress, fmt.Sprintf("Metaandmed %d / %d", processed, len(order)))
	}

	// A changed official series title is propagated without another provider
	// request for every episode.
	for index := range items {
		item := &items[index]
		if item.Type != domain.MediaTypeEpisode {
			continue
		}
		parent := parents[item.SeriesID]
		if parent == nil || item.SeriesTitle == parent.Title {
			continue
		}
		item.SeriesTitle = parent.Title
		item.SortTitle = episodeSortTitle(*item)
		changed[item.ID] = *item
	}

	if len(changed) == 0 {
		update(endProgress, "Metaandmed on juba ajakohased")
		return nil
	}
	batch := make([]domain.MediaItem, 0, len(changed))
	for _, item := range changed {
		batch = append(batch, item)
	}
	if err := s.catalog.UpdateMediaBatch(ctx, batch); err != nil {
		return err
	}
	update(endProgress, fmt.Sprintf("Uuendatud %d kirjet", len(batch)))
	return nil
}

func (s *MetadataMaintenanceService) listAllMedia(ctx context.Context) ([]domain.MediaItem, error) {
	items := make([]domain.MediaItem, 0, 512)
	for offset := 0; ; {
		page, err := s.catalog.ListMedia(ctx, domain.MediaQuery{Limit: 500, Offset: offset, Sort: domain.MediaSortTitle, Order: domain.SortAscending})
		if err != nil {
			return nil, err
		}
		items = append(items, page.Items...)
		offset += len(page.Items)
		if offset >= page.Total || len(page.Items) == 0 {
			return items, nil
		}
	}
}

type seriesSourceHint struct {
	Title string
	Year  int
	Count int
}

// repairSuspiciousSeriesMatches invalidates a very narrow class of stale TMDb
// matches where a short filename title was accepted as a substring of a longer
// word (for example "Silo" -> "Love of Silom"). It deliberately does not
// replace legitimate localized titles that merely differ from the filename.
func repairSuspiciousSeriesMatches(items []domain.MediaItem) map[string]bool {
	hintsBySeries := make(map[string]map[string]seriesSourceHint)
	for _, item := range items {
		if item.Type != domain.MediaTypeEpisode || item.SeriesID == "" {
			continue
		}
		path := strings.TrimSpace(item.File.Path)
		if path == "" {
			path = strings.TrimSpace(item.File.Name)
		}
		if path == "" {
			continue
		}
		identity := classifyMediaPath(domain.LibraryKindSeries, path)
		title := strings.TrimSpace(identity.SeriesTitle)
		if title == "" || strings.EqualFold(title, "Tundmatu seriaal") {
			continue
		}
		key := normalizeSortTitle(title)
		if hintsBySeries[item.SeriesID] == nil {
			hintsBySeries[item.SeriesID] = make(map[string]seriesSourceHint)
		}
		hint := hintsBySeries[item.SeriesID][key]
		if hint.Title == "" {
			hint.Title = title
			hint.Year = identity.Year
		}
		hint.Count++
		hintsBySeries[item.SeriesID][key] = hint
	}

	bestHints := make(map[string]seriesSourceHint, len(hintsBySeries))
	for seriesID, candidates := range hintsBySeries {
		best := seriesSourceHint{}
		for _, candidate := range candidates {
			if candidate.Count > best.Count {
				best = candidate
			}
		}
		if best.Title != "" {
			bestHints[seriesID] = best
		}
	}

	repaired := make(map[string]bool)
	for index := range items {
		item := &items[index]
		if item.Type != domain.MediaTypeSeries {
			continue
		}
		hint, found := bestHints[item.ID]
		if !found || !likelyBoundaryFuzzyMismatch(item.Title, hint.Title) {
			continue
		}
		item.Title = hint.Title
		item.SortTitle = normalizeSortTitle(hint.Title)
		item.Year = hint.Year
		item.ReleaseDate = nil
		item.ExternalIDs = domain.ExternalIDs{}
		item.Metadata = domain.EditorialMetadata{}
		repaired[item.ID] = true
	}

	if len(repaired) == 0 {
		return repaired
	}
	for index := range items {
		item := &items[index]
		if item.Type != domain.MediaTypeEpisode || !repaired[item.SeriesID] {
			continue
		}
		path := strings.TrimSpace(item.File.Path)
		if path == "" {
			path = strings.TrimSpace(item.File.Name)
		}
		identity := classifyMediaPath(domain.LibraryKindSeries, path)
		if identity.SeriesTitle != "" {
			item.SeriesTitle = identity.SeriesTitle
		}
		if identity.Season > 0 {
			item.Season = identity.Season
		}
		if identity.Episode > 0 {
			item.Episode = identity.Episode
		}
		if identity.EpisodeTitle != "" {
			item.Title = identity.EpisodeTitle
			item.EpisodeTitle = identity.EpisodeTitle
		}
		item.Year = identity.Year
		item.ReleaseDate = nil
		item.ExternalIDs = domain.ExternalIDs{}
		item.Metadata = domain.EditorialMetadata{}
		item.SortTitle = episodeSortTitle(*item)
	}
	return repaired
}

func likelyBoundaryFuzzyMismatch(metadataTitle, sourceTitle string) bool {
	metadata := normalizeSortTitle(metadataTitle)
	source := normalizeSortTitle(sourceTitle)
	if metadata == "" || source == "" || metadata == source {
		return false
	}
	metadataTokens := strings.Fields(metadata)
	sourceTokens := strings.Fields(source)
	if len(sourceTokens) != 1 {
		return false
	}
	sourceToken := sourceTokens[0]
	if len([]rune(sourceToken)) < 3 || len([]rune(sourceToken)) > 12 {
		return false
	}
	for _, token := range metadataTokens {
		if token == sourceToken {
			return false
		}
	}
	for _, token := range metadataTokens {
		if strings.HasPrefix(token, sourceToken) || strings.HasPrefix(sourceToken, token) {
			difference := len([]rune(token)) - len([]rune(sourceToken))
			if difference < 0 {
				difference = -difference
			}
			return difference <= 3
		}
	}
	return false
}
