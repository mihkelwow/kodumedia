package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type MusicEnrichmentService struct {
	catalog  ports.Catalog
	metadata *MusicMetadataProvider
	artwork  *MusicArtworkService
	jobs     ports.JobRunner
}

func NewMusicEnrichmentService(catalog ports.Catalog, metadata *MusicMetadataProvider, artwork *MusicArtworkService, jobs ports.JobRunner) *MusicEnrichmentService {
	return &MusicEnrichmentService{catalog: catalog, metadata: metadata, artwork: artwork, jobs: jobs}
}

func (s *MusicEnrichmentService) Start(ctx context.Context, libraryID string) (domain.Job, error) {
	if s == nil || s.catalog == nil || s.metadata == nil || s.artwork == nil || s.jobs == nil {
		return domain.Job{}, domain.ErrUnavailable
	}
	libraryID = strings.TrimSpace(libraryID)
	if libraryID != "" {
		library, err := s.catalog.GetLibrary(ctx, libraryID)
		if err != nil {
			return domain.Job{}, err
		}
		if library.Kind != domain.LibraryKindMusic {
			return domain.Job{}, fmt.Errorf("%w: valitud teek ei ole muusikateek", domain.ErrInvalidInput)
		}
	}
	job := s.jobs.Start("music-enrichment", func(update ports.JobProgress) error {
		return s.run(context.Background(), libraryID, update)
	})
	return job, nil
}

func (s *MusicEnrichmentService) run(ctx context.Context, libraryID string, update ports.JobProgress) error {
	update(2, "Muusikateekide laadimine")
	tracks, err := s.listTracks(ctx, libraryID)
	if err != nil {
		return err
	}
	if len(tracks) == 0 {
		update(99, "Muusikafaile ei leitud")
		return nil
	}

	changedCount := 0
	analysisFailures := 0
	pending := make([]domain.MediaItem, 0, 24)
	flush := func() error {
		if len(pending) == 0 {
			return nil
		}
		if err := s.catalog.UpdateMediaBatch(ctx, pending); err != nil {
			return err
		}
		pending = pending[:0]
		return nil
	}

	for index, item := range tracks {
		itemCtx, cancel := context.WithTimeout(ctx, 28*time.Second)
		enriched, changed, enrichErr := s.metadata.EnrichDeep(itemCtx, item)
		cancel()
		if enrichErr != nil {
			analysisFailures++
		}
		if changed {
			tracks[index] = enriched
			pending = append(pending, enriched)
			changedCount++
			if len(pending) >= 24 {
				if err := flush(); err != nil {
					return err
				}
			}
		}
		progress := 4 + int((int64(index+1)*51)/int64(len(tracks)))
		update(progress, fmt.Sprintf("Žanr ja stiil %d / %d", index+1, len(tracks)))
	}
	if err := flush(); err != nil {
		return err
	}

	artworkCount := 0
	artworkFailures := 0
	for index, item := range tracks {
		artworkCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
		if _, resolveErr := s.artwork.Refresh(artworkCtx, item.ID); resolveErr == nil {
			artworkCount++
		} else {
			artworkFailures++
		}
		cancel()
		progress := 56 + int((int64(index+1)*43)/int64(len(tracks)))
		update(progress, fmt.Sprintf("Artwork %d / %d", index+1, len(tracks)))
	}
	update(99, fmt.Sprintf("Valmis: %d stiiliuuendust, %d artworki, %d hoiatust", changedCount, artworkCount, analysisFailures+artworkFailures))
	return nil
}

func (s *MusicEnrichmentService) listTracks(ctx context.Context, libraryID string) ([]domain.MediaItem, error) {
	libraries, err := s.catalog.ListLibraries(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]domain.MediaItem, 0)
	for _, library := range libraries {
		if library.Kind != domain.LibraryKindMusic || (libraryID != "" && library.ID != libraryID) {
			continue
		}
		for offset := 0; ; offset += 500 {
			page, pageErr := s.catalog.ListMedia(ctx, domain.MediaQuery{
				LibraryID: library.ID,
				Types:     []domain.MediaType{domain.MediaTypeMusic},
				Limit:     500,
				Offset:    offset,
			})
			if pageErr != nil {
				return nil, pageErr
			}
			result = append(result, page.Items...)
			if offset+len(page.Items) >= page.Total || len(page.Items) == 0 {
				break
			}
		}
	}
	return result, nil
}
