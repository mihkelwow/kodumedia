package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

const (
	musicArtworkLimit  = int64(2 << 30)
	musicMetadataLimit = int64(1 << 30)
	themeMusicLimit    = int64(2 << 30)
	subtitleCacheLimit = int64(1 << 30)
)

type optimizedCacheController interface {
	ApplyStoragePolicy(policy string, maximumBytes, reserveBytes int64, reservePercent int)
	OptimizedCacheUsage() int64
	ClearOptimizedCache() int64
}

type subtitleCacheMaintainer interface {
	MaintainCache(context.Context, ports.Catalog, int64) int64
}

type StorageService struct {
	dataDir    string
	catalog    ports.Catalog
	settings   func() domain.ServerSettings
	transcoder optimizedCacheController
	artwork    *MusicArtworkService
	metadata   *MusicMetadataProvider
	themes     *ThemeService
	subtitles  subtitleCacheMaintainer
	watch      *WatchService
}

func NewStorageService(
	dataDir string,
	catalog ports.Catalog,
	settings func() domain.ServerSettings,
	transcoder optimizedCacheController,
	artwork *MusicArtworkService,
	metadata *MusicMetadataProvider,
	themes *ThemeService,
	subtitles subtitleCacheMaintainer,
	watch *WatchService,
) *StorageService {
	return &StorageService{
		dataDir: dataDir, catalog: catalog, settings: settings, transcoder: transcoder,
		artwork: artwork, metadata: metadata, themes: themes, subtitles: subtitles, watch: watch,
	}
}

func (s *StorageService) Run(ctx context.Context) {
	_, _ = s.Cleanup(ctx)
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.Cleanup(ctx)
		}
	}
}

func (s *StorageService) Report(ctx context.Context) (domain.StorageReport, error) {
	if s == nil {
		return domain.StorageReport{}, domain.ErrUnavailable
	}
	if err := ctx.Err(); err != nil {
		return domain.StorageReport{}, err
	}
	settings := domain.QualitySettings{CacheMaxGB: 50, CacheFreeReserveGB: 10, CacheFreeReservePercent: 10}
	if s.settings != nil {
		settings = s.settings().Quality
	}
	free, total, _ := storageDiskSpace(s.dataDir)
	reserve := int64(settings.CacheFreeReserveGB) << 30
	percentReserve := int64(total / 100 * uint64(settings.CacheFreeReservePercent))
	if percentReserve > reserve {
		reserve = percentReserve
	}
	category := func(id, relative string, limit int64, clearable bool) domain.StorageCategory {
		size := maintenanceDirectorySize(filepath.Join(s.dataDir, relative))
		return domain.StorageCategory{ID: id, SizeBytes: size, LimitBytes: limit, Clearable: clearable}
	}
	optimized := int64(0)
	if s.transcoder != nil {
		optimized = s.transcoder.OptimizedCacheUsage()
	}
	optimizedLimit := int64(settings.CacheMaxGB) << 30
	categories := []domain.StorageCategory{
		{ID: "updates", SizeBytes: maintenanceDirectorySize(filepath.Join(s.dataDir, "updates")), RetentionCount: 2},
		{ID: "optimized-media", SizeBytes: optimized, LimitBytes: optimizedLimit, ReclaimableBytes: optimized, Clearable: true},
		category("transcodes", "transcodes", 0, false),
		category("music-artwork", "music-artwork", musicArtworkLimit, false),
		category("music-metadata", "music-metadata", musicMetadataLimit, false),
		category("themes", filepath.Join("themes", "series"), themeMusicLimit, false),
		category("subtitles-cache", filepath.Join("subtitles", "cache"), subtitleCacheLimit, false),
		category("subtitles-downloaded", filepath.Join("subtitles", "downloaded"), 0, false),
		{ID: "logs", SizeBytes: logStorageSize(s.dataDir), LimitBytes: 24 << 20},
	}
	for index := range categories {
		if categories[index].ID == "transcodes" {
			categories[index].ReclaimableBytes = categories[index].SizeBytes
		}
	}
	return domain.StorageReport{
		DataDirectory:        s.dataDir,
		DataSizeBytes:        maintenanceDirectorySize(s.dataDir),
		DiskFreeBytes:        int64(free),
		DiskTotalBytes:       int64(total),
		RequiredReserveBytes: reserve,
		Categories:           categories,
		GeneratedAt:          time.Now().UTC(),
	}, nil
}

func (s *StorageService) Cleanup(ctx context.Context) (domain.StorageCleanupResult, error) {
	before := maintenanceDirectorySize(s.dataDir)
	if s.watch != nil {
		_, _ = s.watch.PruneOrphans(ctx)
	}
	if s.artwork != nil {
		s.artwork.Maintain(ctx, musicArtworkLimit)
	}
	musicItems, err := s.listMusic(ctx)
	if err == nil && s.metadata != nil {
		s.metadata.MaintainItems(ctx, musicItems, musicMetadataLimit)
	}
	if s.themes != nil {
		s.themes.Maintain(ctx, themeMusicLimit)
	}
	if s.subtitles != nil {
		s.subtitles.MaintainCache(ctx, s.catalog, subtitleCacheLimit)
	}
	if s.transcoder != nil && s.settings != nil {
		quality := s.settings().Quality
		s.transcoder.ApplyStoragePolicy(
			quality.CachePolicy,
			int64(quality.CacheMaxGB)<<30,
			int64(quality.CacheFreeReserveGB)<<30,
			quality.CacheFreeReservePercent,
		)
	}
	rotateDataLogs(s.dataDir)
	after := maintenanceDirectorySize(s.dataDir)
	report, reportErr := s.Report(ctx)
	if reportErr != nil {
		return domain.StorageCleanupResult{}, reportErr
	}
	freed := before - after
	if freed < 0 {
		freed = 0
	}
	return domain.StorageCleanupResult{FreedBytes: freed, Report: report}, nil
}

func (s *StorageService) Clear(ctx context.Context, action string) (domain.StorageCleanupResult, error) {
	action = strings.ToLower(strings.TrimSpace(action))
	switch action {
	case "cleanup":
		return s.Cleanup(ctx)
	case "clear-optimized":
		freed := int64(0)
		if s.transcoder != nil {
			freed = s.transcoder.ClearOptimizedCache()
		}
		report, err := s.Report(ctx)
		if err != nil {
			return domain.StorageCleanupResult{}, err
		}
		return domain.StorageCleanupResult{FreedBytes: freed, Report: report}, nil
	default:
		return domain.StorageCleanupResult{}, fmt.Errorf("%w: tundmatu kettahoolduse tegevus", domain.ErrInvalidInput)
	}
}

func (s *StorageService) listMusic(ctx context.Context) ([]domain.MediaItem, error) {
	items := []domain.MediaItem{}
	for offset := 0; ; offset += 500 {
		page, err := s.catalog.ListMedia(ctx, domain.MediaQuery{Types: []domain.MediaType{domain.MediaTypeMusic}, Limit: 500, Offset: offset})
		if err != nil {
			return nil, err
		}
		items = append(items, page.Items...)
		if offset+len(page.Items) >= page.Total {
			return items, nil
		}
	}
}

func maintenanceDirectorySize(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if info, infoErr := entry.Info(); infoErr == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func logStorageSize(dataDir string) int64 {
	var total int64
	for _, path := range []string{
		filepath.Join(dataDir, "kodumedia.log"),
		filepath.Join(dataDir, "kodumedia.log.1"),
		filepath.Join(dataDir, "updates", "last-update.log"),
		filepath.Join(dataDir, "updates", "last-update.log.1"),
	} {
		if info, err := os.Stat(path); err == nil {
			total += info.Size()
		}
	}
	return total
}

func rotateDataLogs(dataDir string) {
	rotateLogFile(filepath.Join(dataDir, "kodumedia.log"), 10<<20)
	rotateLogFile(filepath.Join(dataDir, "updates", "last-update.log"), 2<<20)
}

func rotateLogFile(path string, maximum int64) {
	info, err := os.Stat(path)
	if err != nil || info.Size() <= maximum {
		return
	}
	_ = os.Remove(path + ".1")
	_ = os.Rename(path, path+".1")
}
