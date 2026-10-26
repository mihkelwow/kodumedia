package inspection

import (
	"context"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

type inspectorDatasetStub struct {
	source string
	status domain.ComponentStatus
}

func (s *inspectorDatasetStub) LocalPath() string              { return "" }
func (s *inspectorDatasetStub) Status() domain.ComponentStatus { return s.status }
func (s *inspectorDatasetStub) Inspect(_ context.Context, source string) domain.ComponentStatus {
	s.source = source
	return s.status
}
func (s *inspectorDatasetStub) Refresh(_ context.Context, _ ports.JobProgress) error { return nil }

func TestInspectorDelegatesIMDbSourceToDatasetManager(t *testing.T) {
	dataset := &inspectorDatasetStub{status: domain.ComponentStatus{Available: true, Message: "IMDb title.ratings andmestik on kasutusvalmis."}}
	inspector := New(dataset)
	settings := domain.ServerSettings{Metadata: domain.MetadataSettings{IMDbRatingsSource: domain.DefaultIMDbRatingsSource}}

	status := inspector.Inspect(context.Background(), settings)
	if !status.Metadata.IMDb.Available {
		t.Fatalf("expected available IMDb status, got %+v", status.Metadata.IMDb)
	}
	if dataset.source != domain.DefaultIMDbRatingsSource {
		t.Fatalf("expected source %q, got %q", domain.DefaultIMDbRatingsSource, dataset.source)
	}
}
