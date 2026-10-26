package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/adapters/catalog/jsoncatalog"
	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type maintenanceMetadataProvider struct{}

func (maintenanceMetadataProvider) Name() string    { return "test-metadata" }
func (maintenanceMetadataProvider) Available() bool { return true }
func (maintenanceMetadataProvider) Match(_ context.Context, query domain.MetadataMatchQuery) (domain.MetadataResult, error) {
	return domain.MetadataResult{
		Title: "Official Movie Title",
		Year:  2024,
		ExternalIDs: domain.ExternalIDs{
			TMDb: "42",
			IMDb: "tt0000042",
		},
		Metadata: domain.EditorialMetadata{
			Summary:        "Official description.",
			PosterURL:      "https://image.tmdb.org/t/p/w500/poster.jpg",
			BackdropURL:    "https://image.tmdb.org/t/p/w1280/backdrop.jpg",
			MetadataSource: "tmdb",
		},
	}, nil
}

func TestMetadataMaintenanceBackfillsCopiedCatalog(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, err := jsoncatalog.Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.July, 12, 8, 0, 0, 0, time.UTC)
	library := domain.Library{ID: "library-movies", Name: "Filmid", Kind: domain.LibraryKindMovies, Sources: []domain.LibrarySource{}, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateLibrary(ctx, library); err != nil {
		t.Fatal(err)
	}
	movie := domain.MediaItem{
		ID:        "copied-movie",
		LibraryID: library.ID,
		Type:      domain.MediaTypeMovie,
		Title:     "Release File Name",
		SortTitle: "Release File Name",
		Year:      2024,
		File:      domain.MediaFile{Name: "Release.File.Name.2024.mkv"},
		AddedAt:   now,
		UpdatedAt: now,
	}
	if err := store.ReplaceLibraryMedia(ctx, library.ID, []domain.MediaItem{movie}); err != nil {
		t.Fatal(err)
	}

	metadata := NewMetadataService(store, "et-EE", []ports.MetadataProvider{maintenanceMetadataProvider{}}, nil)
	jobs := NewJobManager()
	maintenance := NewMetadataMaintenanceService(ctx, store, metadata, nil, nil, jobs)
	job := maintenance.BackfillMissing(false)
	completed := waitForTestJob(t, jobs, job.ID)
	if completed.Status != domain.JobStatusSucceeded {
		t.Fatalf("backfill failed: %+v", completed)
	}

	updated, err := store.GetMedia(ctx, movie.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Official Movie Title" || updated.Metadata.Summary == "" || updated.Metadata.PosterURL == "" || updated.Metadata.BackdropURL == "" {
		t.Fatalf("copied catalog item was not repaired: %+v", updated)
	}
}

func waitForTestJob(t *testing.T, jobs *JobManager, id string) domain.Job {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		job, found := jobs.Get(id)
		if found && job.Status != domain.JobStatusQueued && job.Status != domain.JobStatusRunning {
			return job
		}
		time.Sleep(10 * time.Millisecond)
	}
	job, _ := jobs.Get(id)
	t.Fatalf("job did not finish: %+v", job)
	return domain.Job{}
}

type blockingRatingsDataset struct {
	started chan struct{}
	release chan struct{}
}

func (d *blockingRatingsDataset) LocalPath() string { return "" }
func (d *blockingRatingsDataset) Status() domain.ComponentStatus {
	return domain.ComponentStatus{Refreshing: true}
}
func (d *blockingRatingsDataset) Inspect(context.Context, string) domain.ComponentStatus {
	return domain.ComponentStatus{}
}
func (d *blockingRatingsDataset) Refresh(ctx context.Context, _ ports.JobProgress) error {
	select {
	case <-d.started:
	default:
		close(d.started)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-d.release:
		return nil
	}
}

func TestIMDbDownloadDoesNotBlockEditorialBackfill(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, err := jsoncatalog.Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.July, 12, 8, 0, 0, 0, time.UTC)
	library := domain.Library{ID: "library-movies", Name: "Filmid", Kind: domain.LibraryKindMovies, Sources: []domain.LibrarySource{}, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateLibrary(ctx, library); err != nil {
		t.Fatal(err)
	}
	movie := domain.MediaItem{ID: "movie", LibraryID: library.ID, Type: domain.MediaTypeMovie, Title: "Release Name", SortTitle: "Release Name", Year: 2024, AddedAt: now, UpdatedAt: now}
	if err := store.ReplaceLibraryMedia(ctx, library.ID, []domain.MediaItem{movie}); err != nil {
		t.Fatal(err)
	}

	dataset := &blockingRatingsDataset{started: make(chan struct{}), release: make(chan struct{})}
	metadata := NewMetadataService(store, "et-EE", []ports.MetadataProvider{maintenanceMetadataProvider{}}, nil)
	jobs := NewJobManager()
	maintenance := NewMetadataMaintenanceService(ctx, store, metadata, dataset, nil, jobs)
	ratingsJob := maintenance.RefreshIMDb()
	select {
	case <-dataset.started:
	case <-time.After(time.Second):
		t.Fatal("IMDb refresh did not start")
	}

	backfillJob := maintenance.BackfillMissing(false)
	completed := waitForTestJob(t, jobs, backfillJob.ID)
	if completed.Status != domain.JobStatusSucceeded {
		t.Fatalf("editorial backfill failed while IMDb was downloading: %+v", completed)
	}
	updated, err := store.GetMedia(ctx, movie.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Metadata.PosterURL == "" || updated.Metadata.Summary == "" {
		t.Fatalf("editorial metadata was not applied: %+v", updated)
	}

	close(dataset.release)
	if completedRatings := waitForTestJob(t, jobs, ratingsJob.ID); completedRatings.Status != domain.JobStatusSucceeded {
		t.Fatalf("IMDb refresh failed after release: %+v", completedRatings)
	}
}
