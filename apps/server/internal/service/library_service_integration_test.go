package service_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/adapters/catalog/jsoncatalog"
	"github.com/mihkelwow/kodumedia/internal/service"
)

func TestEnsureSystemLibrariesCreatesStableEmptyCollections(t *testing.T) {
	t.Parallel()

	store, err := jsoncatalog.Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	libraries := service.NewLibraryService(store)
	if err := libraries.EnsureSystemLibraries(context.Background()); err != nil {
		t.Fatalf("ensure system libraries: %v", err)
	}

	items, err := libraries.List(context.Background())
	if err != nil {
		t.Fatalf("list libraries: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected three system libraries, got %d", len(items))
	}
	for _, library := range items {
		if library.Sources == nil {
			t.Fatalf("library %q has a nil sources collection", library.ID)
		}
	}
}
