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

type LibraryService struct {
	catalog ports.Catalog
	now     func() time.Time
}

type CreateLibraryInput struct {
	Name string
	Kind domain.LibraryKind
}

type UpdateLibraryInput struct {
	Name *string
	Kind *domain.LibraryKind
}

func NewLibraryService(catalog ports.Catalog) *LibraryService {
	return &LibraryService{catalog: catalog, now: time.Now}
}

func (s *LibraryService) EnsureSystemLibraries(ctx context.Context) error {
	existing, err := s.catalog.ListLibraries(ctx)
	if err != nil {
		return err
	}
	byID := make(map[string]struct{}, len(existing))
	for _, library := range existing {
		byID[library.ID] = struct{}{}
	}

	defaults := []domain.Library{
		{ID: "library-movies", Name: "Filmid", Kind: domain.LibraryKindMovies, System: true, Sources: []domain.LibrarySource{}},
		{ID: "library-series", Name: "Seriaalid", Kind: domain.LibraryKindSeries, System: true, Sources: []domain.LibrarySource{}},
		{ID: "library-music", Name: "Muusika", Kind: domain.LibraryKindMusic, System: true, Sources: []domain.LibrarySource{}},
	}
	for _, library := range defaults {
		if _, found := byID[library.ID]; found {
			continue
		}
		now := s.now().UTC()
		library.CreatedAt = now
		library.UpdatedAt = now
		if err := s.catalog.CreateLibrary(ctx, library); err != nil {
			return fmt.Errorf("create system library %s: %w", library.ID, err)
		}
	}
	return nil
}

func (s *LibraryService) List(ctx context.Context) ([]domain.Library, error) {
	return s.catalog.ListLibraries(ctx)
}

func (s *LibraryService) Get(ctx context.Context, id string) (domain.Library, error) {
	return s.catalog.GetLibrary(ctx, id)
}

func (s *LibraryService) Create(ctx context.Context, input CreateLibraryInput) (domain.Library, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || !input.Kind.IsValid() {
		return domain.Library{}, domain.ErrInvalidInput
	}
	now := s.now().UTC()
	library := domain.Library{
		ID:        randomID("library"),
		Name:      name,
		Kind:      input.Kind,
		System:    false,
		Sources:   []domain.LibrarySource{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.catalog.CreateLibrary(ctx, library); err != nil {
		return domain.Library{}, err
	}
	return library, nil
}

func (s *LibraryService) Update(ctx context.Context, id string, input UpdateLibraryInput) (domain.Library, error) {
	library, err := s.catalog.GetLibrary(ctx, id)
	if err != nil {
		return domain.Library{}, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return domain.Library{}, domain.ErrInvalidInput
		}
		library.Name = name
	}
	if input.Kind != nil {
		if !input.Kind.IsValid() {
			return domain.Library{}, domain.ErrInvalidInput
		}
		if library.System && *input.Kind != library.Kind {
			return domain.Library{}, domain.ErrConflict
		}
		library.Kind = *input.Kind
	}
	library.UpdatedAt = s.now().UTC()
	if err := s.catalog.UpdateLibrary(ctx, library); err != nil {
		return domain.Library{}, err
	}
	return library, nil
}

func (s *LibraryService) Delete(ctx context.Context, id string) error {
	return s.catalog.DeleteLibrary(ctx, id)
}

func (s *LibraryService) AddSource(ctx context.Context, libraryID, sourcePath string) (domain.Library, error) {
	library, err := s.catalog.GetLibrary(ctx, libraryID)
	if err != nil {
		return domain.Library{}, err
	}
	canonical, err := canonicalDirectory(sourcePath)
	if err != nil {
		return domain.Library{}, fmt.Errorf("%w: %v", domain.ErrInvalidInput, err)
	}
	for _, source := range library.Sources {
		if samePath(source.Path, canonical) {
			return library, nil
		}
	}
	now := s.now().UTC()
	library.Sources = append(library.Sources, domain.LibrarySource{
		ID:        randomID("source"),
		Path:      canonical,
		CreatedAt: now,
	})
	library.UpdatedAt = now
	if err := s.catalog.UpdateLibrary(ctx, library); err != nil {
		return domain.Library{}, err
	}
	return library, nil
}

func (s *LibraryService) RemoveSource(ctx context.Context, libraryID, sourceID string) (domain.Library, error) {
	library, err := s.catalog.GetLibrary(ctx, libraryID)
	if err != nil {
		return domain.Library{}, err
	}
	filtered := make([]domain.LibrarySource, 0, len(library.Sources))
	found := false
	for _, source := range library.Sources {
		if source.ID == sourceID {
			found = true
			continue
		}
		filtered = append(filtered, source)
	}
	if !found {
		return domain.Library{}, domain.ErrNotFound
	}
	library.Sources = filtered
	library.UpdatedAt = s.now().UTC()
	if err := s.catalog.UpdateLibrary(ctx, library); err != nil {
		return domain.Library{}, err
	}
	return library, nil
}

func canonicalDirectory(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err == nil {
		absolute = resolved
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory")
	}
	return filepath.Clean(absolute), nil
}

func samePath(left, right string) bool {
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}
