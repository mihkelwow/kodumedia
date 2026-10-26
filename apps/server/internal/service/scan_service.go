package service

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

var supportedExtensions = extensionSet(
	// Modern and legacy video containers and elementary streams handled by FFmpeg.
	".mkv", ".mk3d", ".mp4", ".m4v", ".mov", ".qt", ".avi", ".divx", ".webm", ".ogv",
	".ts", ".mts", ".m2ts", ".m2t", ".mpeg", ".mpg", ".mpe", ".mpeg2", ".m1v", ".m2v",
	".vob", ".vro", ".wmv", ".asf", ".flv", ".f4v", ".3gp", ".3g2", ".mxf", ".dv",
	".rm", ".rmvb", ".amv", ".wtv", ".dvr-ms", ".nut", ".y4m", ".h264", ".264", ".h265", ".265", ".hevc",
	// Audio-only formats. Container probing remains the source of truth after discovery.
	".mp3", ".mp2", ".mpa", ".flac", ".m4a", ".m4b", ".aac", ".adts", ".ogg", ".oga", ".opus",
	".wav", ".wma", ".aiff", ".aif", ".aifc", ".ape", ".mka", ".ac3", ".eac3", ".ec3", ".dts",
	".dtshd", ".truehd", ".thd", ".mlp", ".tta", ".tak", ".wv", ".amr", ".awb", ".au", ".snd", ".caf", ".ra", ".spx",
)

var audioExtensions = extensionSet(
	".mp3", ".mp2", ".mpa", ".flac", ".m4a", ".m4b", ".aac", ".adts", ".ogg", ".oga", ".opus",
	".wav", ".wma", ".aiff", ".aif", ".aifc", ".ape", ".mka", ".ac3", ".eac3", ".ec3", ".dts",
	".dtshd", ".truehd", ".thd", ".mlp", ".tta", ".tak", ".wv", ".amr", ".awb", ".au", ".snd", ".caf", ".ra", ".spx",
)

func extensionSet(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

type ScanOption func(*ScanService)

func WithScanMetadata(metadata *MetadataService) ScanOption {
	return func(service *ScanService) { service.metadata = metadata }
}

func WithScanMusicMetadata(metadata *MusicMetadataProvider) ScanOption {
	return func(service *ScanService) { service.musicMetadata = metadata }
}

type ScanService struct {
	catalog       ports.Catalog
	probe         ports.MediaProbe
	jobs          ports.JobRunner
	metadata      *MetadataService
	musicMetadata *MusicMetadataProvider
	now           func() time.Time
	workers       int
}

func NewScanService(catalog ports.Catalog, probe ports.MediaProbe, jobs ports.JobRunner, options ...ScanOption) *ScanService {
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	if workers > 4 {
		workers = 4
	}
	service := &ScanService{catalog: catalog, probe: probe, jobs: jobs, now: time.Now, workers: workers}
	for _, option := range options {
		option(service)
	}
	return service
}
func (s *ScanService) Start(ctx context.Context, libraryID string) (domain.Job, error) {
	if _, err := s.catalog.GetLibrary(ctx, libraryID); err != nil {
		return domain.Job{}, err
	}
	job := s.jobs.Start("library-scan", func(update ports.JobProgress) error { return s.scan(context.Background(), libraryID, update) })
	return job, nil
}
func (s *ScanService) scan(ctx context.Context, libraryID string, update ports.JobProgress) error {
	library, err := s.catalog.GetLibrary(ctx, libraryID)
	if err != nil {
		return err
	}
	if len(library.Sources) == 0 {
		return fmt.Errorf("%w: library has no source folders", domain.ErrInvalidInput)
	}
	update(2, "Failide otsimine")
	candidates, err := collectCandidates(library.Kind, library.Sources)
	if err != nil {
		return err
	}
	existing, err := s.existingByID(ctx, libraryID)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		return s.catalog.ReplaceLibraryMedia(ctx, libraryID, nil)
	}
	paths := make(chan string)
	results := make(chan domain.MediaItem, s.workers)
	workerCount := s.workers
	if workerCount > len(candidates) {
		workerCount = len(candidates)
	}
	for index := 0; index < workerCount; index++ {
		go func() {
			for path := range paths {
				results <- s.buildItem(ctx, library, path, existing)
			}
		}()
	}
	go func() {
		for _, path := range candidates {
			paths <- path
		}
		close(paths)
	}()
	items := make([]domain.MediaItem, 0, len(candidates))
	var completed int64
	for range candidates {
		items = append(items, <-results)
		current := atomic.AddInt64(&completed, 1)
		progress := 5 + int((current*64)/int64(len(candidates)))
		update(progress, fmt.Sprintf("Analüüsitud %d / %d", current, len(candidates)))
	}
	if library.Kind == domain.LibraryKindMusic {
		update(70, "Artistide ja albumite rühmitamine")
	} else {
		update(70, "Filmide ja seriaalide rühmitamine")
	}
	items = s.organizeSeries(items, existing, library)
	if library.Kind == domain.LibraryKindMusic && s.musicMetadata != nil && s.musicMetadata.Available() {
		s.enrichMusicMetadata(ctx, items, update)
	}
	if s.shouldRefreshMetadata() {
		s.enrichMetadata(ctx, items, update)
	}
	sort.Slice(items, func(left, right int) bool {
		if items[left].Type != items[right].Type {
			return items[left].Type < items[right].Type
		}
		return strings.ToLower(items[left].SortTitle) < strings.ToLower(items[right].SortTitle)
	})
	update(97, "Kataloogi salvestamine")
	return s.catalog.ReplaceLibraryMedia(ctx, libraryID, items)
}

func (s *ScanService) enrichMusicMetadata(ctx context.Context, items []domain.MediaItem, update ports.JobProgress) {
	eligible := 0
	for index := range items {
		item := items[index]
		if item.Type == domain.MediaTypeMusic && (item.MusicClassification == nil || len(item.MusicClassification.Genres) == 0 || len(item.MusicClassification.Styles) == 0) {
			eligible++
		}
	}
	if eligible == 0 {
		return
	}
	completed := 0
	for index := range items {
		item := items[index]
		if item.Type != domain.MediaTypeMusic || (item.MusicClassification != nil && len(item.MusicClassification.Genres) > 0 && len(item.MusicClassification.Styles) > 0) {
			continue
		}
		lookupCtx, cancel := context.WithTimeout(ctx, 18*time.Second)
		enriched, changed, err := s.musicMetadata.Enrich(lookupCtx, item)
		cancel()
		if err == nil && changed {
			items[index] = enriched
		}
		completed++
		update(72+int((int64(completed)*22)/int64(eligible)), fmt.Sprintf("Muusika metaandmed %d / %d", completed, eligible))
	}
}

// RepairSeriesGrouping normalizes legacy episode parent links without touching
// the media files or running FFprobe. Older releases included episode years and
// previously persisted parent IDs in the series key, which could split one show
// into several cards. The repair runs once at startup only when a difference is
// detected, and the normal scanner applies the same rules later.
func (s *ScanService) RepairSeriesGrouping(ctx context.Context) (int, error) {
	libraries, err := s.catalog.ListLibraries(ctx)
	if err != nil {
		return 0, err
	}
	repaired := 0
	for _, library := range libraries {
		changed, repairErr := s.repairLibrarySeriesGrouping(ctx, library)
		if repairErr != nil {
			return repaired, fmt.Errorf("repair series grouping for %s: %w", library.Name, repairErr)
		}
		if changed {
			repaired++
		}
	}
	return repaired, nil
}

func (s *ScanService) repairLibrarySeriesGrouping(ctx context.Context, library domain.Library) (bool, error) {
	items, err := s.listAllLibraryMedia(ctx, library.ID)
	if err != nil {
		return false, err
	}
	if len(items) == 0 {
		return false, nil
	}
	existing := make(map[string]domain.MediaItem, len(items))
	files := make([]domain.MediaItem, 0, len(items))
	existingParents := make(map[string]domain.MediaItem)
	desiredCounts := make(map[string]int)
	changed := false
	for _, item := range items {
		existing[item.ID] = item
		if item.Type == domain.MediaTypeSeries {
			existingParents[item.ID] = item
			continue
		}
		if item.Type == domain.MediaTypeEpisode {
			sourceTitle, sourceYear := classifiedSeriesIdentity(library.Kind, item)
			desiredID := stableSeriesID(library.ID, sourceTitle, sourceYear)
			desiredCounts[desiredID]++
			if item.SeriesID != desiredID {
				changed = true
			}
		}
		files = append(files, item)
	}
	if len(existingParents) != len(desiredCounts) {
		changed = true
	}
	for id, count := range desiredCounts {
		parent, found := existingParents[id]
		if !found || parent.EpisodeCount != count {
			changed = true
			break
		}
	}
	if !changed {
		return false, nil
	}
	rebuilt := s.organizeSeries(files, existing, library)
	sort.Slice(rebuilt, func(left, right int) bool {
		if rebuilt[left].Type != rebuilt[right].Type {
			return rebuilt[left].Type < rebuilt[right].Type
		}
		return strings.ToLower(rebuilt[left].SortTitle) < strings.ToLower(rebuilt[right].SortTitle)
	})
	if err := s.catalog.ReplaceLibraryMedia(ctx, library.ID, rebuilt); err != nil {
		return false, err
	}
	return true, nil
}

func (s *ScanService) listAllLibraryMedia(ctx context.Context, libraryID string) ([]domain.MediaItem, error) {
	const pageSize = 500
	items := make([]domain.MediaItem, 0, 256)
	for offset := 0; ; offset += pageSize {
		page, err := s.catalog.ListMedia(ctx, domain.MediaQuery{LibraryID: libraryID, Limit: pageSize, Offset: offset, Sort: domain.MediaSortTitle, Order: domain.SortAscending})
		if err != nil {
			return nil, err
		}
		items = append(items, page.Items...)
		if len(items) >= page.Total || len(page.Items) == 0 {
			break
		}
	}
	return items, nil
}

func (s *ScanService) existingByID(ctx context.Context, libraryID string) (map[string]domain.MediaItem, error) {
	result := map[string]domain.MediaItem{}
	offset := 0
	for {
		page, err := s.catalog.ListMedia(ctx, domain.MediaQuery{LibraryID: libraryID, Limit: 500, Offset: offset, Sort: domain.MediaSortTitle, Order: domain.SortAscending})
		if err != nil {
			return nil, err
		}
		for _, item := range page.Items {
			result[item.ID] = item
		}
		offset += len(page.Items)
		if offset >= page.Total || len(page.Items) == 0 {
			break
		}
	}
	return result, nil
}
func (s *ScanService) buildItem(ctx context.Context, library domain.Library, path string, existing map[string]domain.MediaItem) domain.MediaItem {
	info, _ := os.Stat(path)
	now := s.now().UTC()
	id := stableID("media", library.ID, filepath.Clean(path))
	old, hadOld := existing[id]
	identity := classifyMediaPath(library.Kind, path)
	item := domain.MediaItem{ID: id, LibraryID: library.ID, Type: identity.Type, Title: identity.Title, SortTitle: identity.SortTitle, Year: identity.Year, Season: identity.Season, Episode: identity.Episode, EpisodeTitle: identity.EpisodeTitle, SeriesTitle: identity.SeriesTitle, Artist: identity.Artist, Album: identity.Album, TrackNumber: identity.TrackNumber, File: domain.MediaFile{Path: path, Name: filepath.Base(path), Extension: strings.ToLower(filepath.Ext(path)), ModifiedAt: now}, AddedAt: now, UpdatedAt: now}
	if item.Type == domain.MediaTypeEpisode {
		item.SeriesID = stableSeriesID(library.ID, identity.SeriesTitle, identity.Year)
	}
	if info != nil {
		item.File.SizeBytes = info.Size()
		item.File.ModifiedAt = info.ModTime().UTC()
	}
	item.File.RelativePath = relativeToSources(path, library.Sources)
	if hadOld {
		item.AddedAt = old.AddedAt
		item.ExternalIDs = old.ExternalIDs
		item.Metadata = old.Metadata
		item.ReleaseDate = old.ReleaseDate
		if old.Type == domain.MediaTypeMusic {
			item.Title = old.Title
			item.SortTitle = old.SortTitle
			item.Year = old.Year
			item.Artist = old.Artist
			item.Album = old.Album
			item.TrackNumber = old.TrackNumber
			item.DiscNumber = old.DiscNumber
			item.MusicClassification = old.MusicClassification
		}
		if old.Type == item.Type && old.Metadata.MetadataSource != "" {
			item.Title = old.Title
			item.SortTitle = old.SortTitle
			item.Year = old.Year
			item.EpisodeTitle = old.EpisodeTitle
			if old.SeriesTitle != "" {
				item.SeriesTitle = old.SeriesTitle
			}
		}
		if old.File.SizeBytes == item.File.SizeBytes && old.File.ModifiedAt.Equal(item.File.ModifiedAt) {
			item.Technical = old.Technical
		}
	}
	if scanTechnicalDataNeedsProbe(item) && s.probe != nil {
		probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		result, err := s.probe.Probe(probeCtx, path)
		cancel()
		if err == nil {
			item.Technical = result.Technical
			if item.Metadata.MetadataSource == "" || item.Type == domain.MediaTypeMusic {
				applyTags(&item, result.Tags)
			}
		}
	}
	if item.Type == domain.MediaTypeMusic {
		applyMusicSidecar(&item, path)
	}
	if item.Technical.Container == "" {
		item.Technical.Container = containerFromExtension(item.File.Extension)
	}
	if item.SortTitle == "" {
		item.SortTitle = normalizeSortTitle(item.Title)
	}
	return item
}

func scanTechnicalDataNeedsProbe(item domain.MediaItem) bool {
	if item.Technical.ProbeVersion < 3 || item.Technical.Container == "" || item.Technical.DurationMS <= 0 || len(item.Technical.Tracks) == 0 {
		return true
	}
	if item.Type == domain.MediaTypeMusic {
		return item.Technical.AudioCodec == ""
	}
	return item.Technical.VideoCodec == "" && item.Technical.AudioCodec == ""
}

func (s *ScanService) organizeSeries(files []domain.MediaItem, existing map[string]domain.MediaItem, library domain.Library) []domain.MediaItem {
	groups := map[string][]int{}
	sourceTitles := map[string]string{}
	for index := range files {
		if files[index].Type != domain.MediaTypeEpisode {
			continue
		}
		sourceTitle, sourceYear := classifiedSeriesIdentity(library.Kind, files[index])
		if sourceTitle == "" {
			sourceTitle = files[index].SeriesTitle
		}
		seriesID := stableSeriesID(library.ID, sourceTitle, sourceYear)
		files[index].SeriesID = seriesID
		files[index].SeriesTitle = sourceTitle
		groups[seriesID] = append(groups[seriesID], index)
		if sourceTitles[seriesID] == "" {
			sourceTitles[seriesID] = sourceTitle
		}
	}
	result := make([]domain.MediaItem, 0, len(files)+len(groups))
	result = append(result, files...)
	for seriesID, indexes := range groups {
		if len(indexes) == 0 {
			continue
		}
		first := files[indexes[0]]
		sourceTitle := sourceTitles[seriesID]
		parent := domain.MediaItem{ID: seriesID, LibraryID: library.ID, Type: domain.MediaTypeSeries, Title: sourceTitle, SortTitle: normalizeSortTitle(sourceTitle), Year: first.Year, AddedAt: first.AddedAt, UpdatedAt: first.UpdatedAt, EpisodeCount: len(indexes), File: domain.MediaFile{ModifiedAt: first.File.ModifiedAt}}
		if old, found := bestExistingSeriesParent(existing, files, indexes, seriesID, sourceTitle); found {
			parent.Title = old.Title
			parent.SortTitle = old.SortTitle
			parent.Year = old.Year
			parent.ReleaseDate = old.ReleaseDate
			parent.ExternalIDs = old.ExternalIDs
			parent.Metadata = old.Metadata
		}
		seasons := map[int]struct{}{}
		for _, index := range indexes {
			episode := &result[index]
			episode.SeriesID = seriesID
			episode.SeriesTitle = parent.Title
			episode.SortTitle = episodeSortTitle(*episode)
			if episode.AddedAt.After(parent.AddedAt) {
				parent.AddedAt = episode.AddedAt
			}
			if episode.UpdatedAt.After(parent.UpdatedAt) {
				parent.UpdatedAt = episode.UpdatedAt
			}
			if episode.File.ModifiedAt.After(parent.File.ModifiedAt) {
				parent.File.ModifiedAt = episode.File.ModifiedAt
			}
			if parent.Year == 0 && episode.Year > 0 {
				parent.Year = episode.Year
			}
			seasons[episode.Season] = struct{}{}
		}
		parent.SeasonCount = len(seasons)
		result = append(result, parent)
	}
	return result
}

func classifiedSeriesIdentity(kind domain.LibraryKind, item domain.MediaItem) (string, int) {
	path := strings.TrimSpace(item.File.Path)
	if path == "" {
		path = strings.TrimSpace(item.File.RelativePath)
	}
	if path == "" {
		path = item.File.Name
	}
	identity := classifyMediaPath(kind, path)
	if identity.Type == domain.MediaTypeEpisode && strings.TrimSpace(identity.SeriesTitle) != "" {
		groupYear := 0
		if identity.SeriesYearExplicit {
			groupYear = identity.Year
		}
		return identity.SeriesTitle, groupYear
	}
	return strings.TrimSpace(item.SeriesTitle), 0
}

func classifiedSeriesTitle(kind domain.LibraryKind, item domain.MediaItem) string {
	title, _ := classifiedSeriesIdentity(kind, item)
	return title
}

func bestExistingSeriesParent(existing map[string]domain.MediaItem, files []domain.MediaItem, indexes []int, canonicalID, sourceTitle string) (domain.MediaItem, bool) {
	candidateIDs := map[string]struct{}{canonicalID: {}}
	for _, index := range indexes {
		if oldEpisode, found := existing[files[index].ID]; found && oldEpisode.SeriesID != "" {
			candidateIDs[oldEpisode.SeriesID] = struct{}{}
		}
	}
	best := domain.MediaItem{}
	bestScore := -1
	for candidateID := range candidateIDs {
		candidate, found := existing[candidateID]
		if !found || candidate.Type != domain.MediaTypeSeries || likelyBoundaryFuzzyMismatch(candidate.Title, sourceTitle) {
			continue
		}
		score := seriesParentMetadataScore(candidate)
		if score > bestScore || (score == bestScore && candidate.UpdatedAt.After(best.UpdatedAt)) {
			best = candidate
			bestScore = score
		}
	}
	return best, bestScore >= 0
}

func seriesParentMetadataScore(item domain.MediaItem) int {
	score := 0
	if item.Metadata.MetadataSource != "" {
		score += 20
	}
	if item.ExternalIDs.TMDb != "" {
		score += 30
	}
	if item.ExternalIDs.IMDb != "" {
		score += 20
	}
	if item.Metadata.Summary != "" {
		score += 12
	}
	if item.Metadata.PosterURL != "" {
		score += 10
	}
	if item.Metadata.BackdropURL != "" {
		score += 8
	}
	if item.Metadata.IMDbRating > 0 {
		score += 6
	}
	return score
}
func (s *ScanService) enrichMetadata(ctx context.Context, items []domain.MediaItem, update ports.JobProgress) {
	parents := map[string]*domain.MediaItem{}
	for index := range items {
		if items[index].Type == domain.MediaTypeSeries {
			parents[items[index].ID] = &items[index]
		}
	}
	eligible := 0
	for index := range items {
		if shouldEnrichDuringScan(items[index]) {
			eligible++
		}
	}
	if eligible == 0 {
		return
	}
	completed := 0
	for index := range items {
		item := &items[index]
		if item.Type != domain.MediaTypeSeries || !shouldEnrichDuringScan(*item) {
			continue
		}
		if enriched, ok := s.tryEnrich(ctx, *item, nil); ok {
			*item = enriched
		}
		completed++
		update(72+int((int64(completed)*22)/int64(eligible)), fmt.Sprintf("Metaandmed %d / %d", completed, eligible))
	}
	for index := range items {
		item := &items[index]
		if item.Type == domain.MediaTypeSeries || !shouldEnrichDuringScan(*item) {
			continue
		}
		var parent *domain.MediaItem
		if item.Type == domain.MediaTypeEpisode {
			parent = parents[item.SeriesID]
			if parent != nil {
				item.SeriesTitle = parent.Title
				item.SortTitle = episodeSortTitle(*item)
			}
		}
		if enriched, ok := s.tryEnrich(ctx, *item, parent); ok {
			*item = enriched
		}
		completed++
		update(72+int((int64(completed)*22)/int64(eligible)), fmt.Sprintf("Metaandmed %d / %d", completed, eligible))
	}
	for index := range items {
		if items[index].Type != domain.MediaTypeEpisode {
			continue
		}
		if parent := parents[items[index].SeriesID]; parent != nil {
			items[index].SeriesTitle = parent.Title
			items[index].SortTitle = episodeSortTitle(items[index])
		}
	}
}
func (s *ScanService) tryEnrich(ctx context.Context, item domain.MediaItem, parent *domain.MediaItem) (domain.MediaItem, bool) {
	if s.metadata == nil {
		return item, false
	}
	enrichCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	enriched, err := s.metadata.Enrich(enrichCtx, item, parent)
	if err != nil {
		return item, false
	}
	return enriched, true
}
func (s *ScanService) shouldRefreshMetadata() bool {
	return s.metadata != nil && s.metadata.Available()
}

func shouldEnrichDuringScan(item domain.MediaItem) bool {
	if needsEditorialMetadata(item) {
		return true
	}
	return item.ExternalIDs.IMDb != "" && (item.Metadata.IMDbRating == 0 || item.Metadata.IMDbVotes == 0)
}
func stableSeriesID(libraryID, title string, year int) string {
	key := canonicalSeriesTitleKey(title)
	if year > 0 {
		key += "|" + strconv.Itoa(year)
	}
	return stableID("series", libraryID, key)
}

func canonicalSeriesTitleKey(title string) string {
	value := strings.ToLower(strings.TrimSpace(title))
	value = strings.NewReplacer(
		"&", " and ",
		"’", "", "'", "", "`", "", "´", "",
	).Replace(value)
	words := strings.FieldsFunc(value, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if len(words) > 1 {
		switch words[0] {
		case "the", "a", "an":
			words = words[1:]
		}
	}
	return strings.Join(words, " ")
}
func collectCandidates(kind domain.LibraryKind, sources []domain.LibrarySource) ([]string, error) {
	seen := map[string]struct{}{}
	for _, source := range sources {
		info, err := os.Stat(source.Path)
		if err != nil {
			return nil, fmt.Errorf("source %q unavailable: %w", source.Path, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("source %q is not a directory", source.Path)
		}
		err = filepath.WalkDir(source.Path, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				name := strings.ToLower(entry.Name())
				if name == ".git" || name == "@eadir" || name == "$recycle.bin" || name == "system volume information" {
					return filepath.SkipDir
				}
				return nil
			}
			extension := strings.ToLower(filepath.Ext(entry.Name()))
			if kind == domain.LibraryKindMusic {
				if _, audio := audioExtensions[extension]; !audio {
					return nil
				}
			} else if _, supported := supportedExtensions[extension]; !supported {
				return nil
			}
			canonical, err := filepath.Abs(path)
			if err != nil {
				return nil
			}
			seen[filepath.Clean(canonical)] = struct{}{}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("scan source %q: %w", source.Path, err)
		}
	}
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}
func parseFilename(item *domain.MediaItem) {
	kind := domain.LibraryKindMixed
	switch item.Type {
	case domain.MediaTypeMovie:
		kind = domain.LibraryKindMovies
	case domain.MediaTypeEpisode, domain.MediaTypeSeries:
		kind = domain.LibraryKindSeries
	case domain.MediaTypeMusic:
		kind = domain.LibraryKindMusic
	}
	path := item.File.Path
	if path == "" {
		path = item.File.Name
	}
	identity := classifyMediaPath(kind, path)
	item.Type = identity.Type
	item.Title = identity.Title
	item.SortTitle = identity.SortTitle
	item.Year = identity.Year
	item.SeriesTitle = identity.SeriesTitle
	item.Season = identity.Season
	item.Episode = identity.Episode
	item.EpisodeTitle = identity.EpisodeTitle
	item.Artist = identity.Artist
	item.Album = identity.Album
	item.TrackNumber = identity.TrackNumber
}
func applyTags(item *domain.MediaItem, tags map[string]string) {
	if tags == nil {
		return
	}
	if title := strings.TrimSpace(tags["title"]); title != "" {
		item.Title = title
		if item.Type == domain.MediaTypeEpisode {
			item.EpisodeTitle = title
		}
	}
	if item.Type == domain.MediaTypeMusic {
		if artist := firstTag(tags, "album_artist", "artist"); artist != "" {
			item.Artist = artist
		}
		if album := strings.TrimSpace(tags["album"]); album != "" {
			item.Album = album
		}
		genres := collectMusicTagValues(tags, "genre")
		styles := collectMusicTagValues(tags, "style", "subgenre", "grouping", "contentgroup")
		moods := collectMusicTagValues(tags, "mood")
		if len(genres) > 0 || len(styles) > 0 || len(moods) > 0 {
			classification := domain.MusicClassification{Genres: genres, Styles: styles, Moods: moods, Confidence: "tagged"}
			if len(genres) > 0 {
				classification.GenreSource = "embedded"
			}
			if len(styles) > 0 {
				classification.StyleSource = "embedded"
			}
			if len(moods) > 0 {
				classification.MoodSource = "embedded"
			}
			classification.Source = classificationSummarySource(classification)
			item.MusicClassification = &classification
			item.Metadata.Genres = append([]string(nil), genres...)
		}
		if track := strings.Split(tags["track"], "/")[0]; track != "" {
			item.TrackNumber, _ = strconv.Atoi(track)
		}
		if disc := strings.Split(tags["disc"], "/")[0]; disc != "" {
			item.DiscNumber, _ = strconv.Atoi(disc)
		}
	}
	if item.Year == 0 {
		if date := tags["date"]; len(date) >= 4 {
			item.Year, _ = strconv.Atoi(date[:4])
		}
	}
	if item.Type == domain.MediaTypeEpisode {
		item.SortTitle = episodeSortTitle(*item)
	} else {
		item.SortTitle = normalizeSortTitle(strings.Join([]string{item.Artist, item.Album, item.Title}, " "))
	}
}
func collectMusicTagValues(tags map[string]string, keys ...string) []string {
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		if value := strings.TrimSpace(tags[key]); value != "" {
			values = append(values, value)
		}
	}
	return splitMusicGenres(strings.Join(values, ";"))
}

func splitMusicGenres(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == ';' || r == ',' || r == '/' })
	result := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		key := strings.ToLower(part)
		if part == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, part)
	}
	return result
}

func firstTag(tags map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(tags[key]); value != "" {
			return value
		}
	}
	return ""
}
func relativeToSources(path string, sources []domain.LibrarySource) string {
	for _, source := range sources {
		relative, err := filepath.Rel(source.Path, path)
		if err == nil && relative != "." && !strings.HasPrefix(relative, "..") {
			return relative
		}
	}
	return filepath.Base(path)
}
func containerFromExtension(extension string) string {
	switch strings.ToLower(extension) {
	case ".mp4", ".m4v", ".m4a", ".m4b", ".mov", ".qt", ".3gp", ".3g2", ".f4v":
		return "mp4"
	case ".mkv", ".mk3d", ".mka":
		return "mkv"
	case ".webm":
		return "webm"
	case ".mp3", ".mp2", ".mpa":
		return "mp3"
	case ".flac":
		return "flac"
	case ".ogg", ".oga", ".opus", ".ogv", ".spx":
		return "ogg"
	case ".wav", ".aiff", ".aif", ".aifc":
		return "wav"
	case ".ts", ".mts", ".m2ts", ".m2t":
		return "mpegts"
	case ".mpeg", ".mpg", ".mpe", ".mpeg2", ".m1v", ".m2v", ".vob", ".vro":
		return "mpeg"
	case ".avi", ".divx":
		return "avi"
	case ".wmv", ".asf", ".wtv", ".dvr-ms":
		return "asf"
	case ".flv":
		return "flv"
	case ".rm", ".rmvb", ".ra":
		return "realmedia"
	case ".ac3":
		return "ac3"
	case ".eac3", ".ec3":
		return "eac3"
	case ".dts", ".dtshd":
		return "dts"
	case ".truehd", ".thd", ".mlp":
		return "truehd"
	default:
		return strings.TrimPrefix(strings.ToLower(extension), ".")
	}
}
