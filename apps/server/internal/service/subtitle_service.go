package service

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type SubtitleService struct {
	catalog ports.Catalog
	store   ports.SubtitleStore
	online  ports.OnlineSubtitleProvider
}

func NewSubtitleService(catalog ports.Catalog, store ports.SubtitleStore, online ports.OnlineSubtitleProvider) *SubtitleService {
	return &SubtitleService{catalog: catalog, store: store, online: online}
}

func (s *SubtitleService) Discover(ctx context.Context, media domain.MediaItem) ([]domain.SubtitleTrack, error) {
	if s == nil || s.store == nil {
		return []domain.SubtitleTrack{}, nil
	}
	tracks, err := s.store.Discover(ctx, media)
	if err != nil {
		return nil, err
	}
	if tracks == nil {
		tracks = []domain.SubtitleTrack{}
	}
	return tracks, nil
}

func (s *SubtitleService) Resolve(ctx context.Context, media domain.MediaItem, id string) (domain.SubtitleTrack, error) {
	if s == nil || s.store == nil {
		return domain.SubtitleTrack{}, domain.ErrUnavailable
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.SubtitleTrack{}, domain.ErrInvalidInput
	}
	return s.store.Resolve(ctx, media, id)
}

func (s *SubtitleService) WebVTT(ctx context.Context, mediaID, subtitleID string) (string, error) {
	if s == nil || s.catalog == nil || s.store == nil {
		return "", domain.ErrUnavailable
	}
	media, err := s.catalog.GetMedia(ctx, mediaID)
	if err != nil {
		return "", err
	}
	track, err := s.store.Resolve(ctx, media, subtitleID)
	if err != nil {
		return "", err
	}
	return s.store.WebVTT(ctx, media, track)
}

func (s *SubtitleService) Search(ctx context.Context, mediaID string, languages []string) ([]domain.SubtitleSearchResult, error) {
	if s == nil || s.catalog == nil || s.online == nil || !s.online.Available() {
		return nil, fmt.Errorf("%w: OpenSubtitles pole seadistatud", domain.ErrUnavailable)
	}
	media, err := s.catalog.GetMedia(ctx, mediaID)
	if err != nil {
		return nil, err
	}
	return s.online.Search(ctx, media, languages)
}

func (s *SubtitleService) Download(ctx context.Context, mediaID string, request domain.SubtitleDownloadRequest) (domain.SubtitleTrack, error) {
	if s == nil || s.catalog == nil || s.store == nil || s.online == nil || !s.online.Available() {
		return domain.SubtitleTrack{}, fmt.Errorf("%w: OpenSubtitles pole seadistatud", domain.ErrUnavailable)
	}
	if request.FileID <= 0 {
		return domain.SubtitleTrack{}, domain.ErrInvalidInput
	}
	media, err := s.catalog.GetMedia(ctx, mediaID)
	if err != nil {
		return domain.SubtitleTrack{}, err
	}
	reader, suggestedName, err := s.online.Download(ctx, request.FileID)
	if err != nil {
		return domain.SubtitleTrack{}, err
	}
	defer reader.Close()
	name := strings.TrimSpace(request.FileName)
	if name == "" {
		name = suggestedName
	}
	track, err := s.store.SaveDownloaded(ctx, media, name, io.LimitReader(reader, 20<<20+1))
	if err != nil {
		return domain.SubtitleTrack{}, err
	}
	if track.Language == "" {
		track.Language = strings.ToLower(strings.TrimSpace(request.Language))
	}
	return track, nil
}
