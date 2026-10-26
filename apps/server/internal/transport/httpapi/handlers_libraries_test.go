package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/mihkelwow/kodumedia/internal/adapters/catalog/jsoncatalog"
	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/service"
)

func TestListLibrariesEncodesEmptySourcesAsJSONArray(t *testing.T) {
	t.Parallel()

	store, err := jsoncatalog.Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	libraries := service.NewLibraryService(store)
	if err := libraries.EnsureSystemLibraries(context.Background()); err != nil {
		t.Fatalf("ensure system libraries: %v", err)
	}

	handler := NewRouter(Dependencies{Libraries: libraries})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/libraries", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}

	var payload struct {
		Items []struct {
			ID      string                 `json:"id"`
			Sources []domain.LibrarySource `json:"sources"`
		} `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Items) != 3 {
		t.Fatalf("expected three libraries, got %d", len(payload.Items))
	}
	for _, library := range payload.Items {
		if library.Sources == nil {
			t.Fatalf("library %q encoded sources as null or omitted", library.ID)
		}
		if len(library.Sources) != 0 {
			t.Fatalf("library %q unexpectedly has %d sources", library.ID, len(library.Sources))
		}
	}
}
