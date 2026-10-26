package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mihkelwow/kodumedia/internal/adapters/catalog/jsoncatalog"
	filesystemadapter "github.com/mihkelwow/kodumedia/internal/adapters/filesystem/local"
	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/service"
)

type httpTestSettingsRepository struct{ value domain.ServerSettings }

func (r *httpTestSettingsRepository) Current() domain.ServerSettings { return r.value }
func (r *httpTestSettingsRepository) Update(_ context.Context, value domain.ServerSettings) error {
	r.value = value
	return nil
}

type httpTestInspector struct{}

func (httpTestInspector) Inspect(_ context.Context, settings domain.ServerSettings) domain.SystemStatus {
	return domain.SystemStatus{
		FFmpeg:  domain.ComponentStatus{Available: settings.FFmpegPath != ""},
		FFprobe: domain.ComponentStatus{Available: settings.FFprobePath != ""},
	}
}

func TestSystemSettingsNeverExposeTMDbToken(t *testing.T) {
	t.Parallel()
	repository := &httpTestSettingsRepository{value: domain.ServerSettings{
		FFmpegPath: "ffmpeg.exe", FFprobePath: "ffprobe.exe",
		Metadata: domain.MetadataSettings{TMDbToken: "super-secret-token", Language: "et-EE", IMDbRatingsSource: domain.DefaultIMDbRatingsSource},
	}}
	settings := service.NewSettingsService(repository, httpTestInspector{}, "0.2.1")
	handler := NewRouter(Dependencies{Settings: settings})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/system/settings", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "super-secret-token") {
		t.Fatal("settings response exposed the stored TMDb token")
	}
	var payload domain.SystemConfiguration
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode settings response: %v", err)
	}
	if !payload.Settings.Metadata.TMDbTokenConfigured || payload.Settings.Metadata.TMDbTokenHint != "••••oken" {
		t.Fatalf("unexpected public token state: %+v", payload.Settings.Metadata)
	}
}

func TestFilesystemBrowseEncodesCollectionsAsArrays(t *testing.T) {
	t.Parallel()
	handler := NewRouter(Dependencies{Filesystem: filesystemadapter.New()})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/filesystem/browse", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload domain.DirectoryListing
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode filesystem response: %v", err)
	}
	if payload.Roots == nil || payload.Entries == nil {
		t.Fatalf("filesystem collections must be arrays: %+v", payload)
	}
}

func TestSeriesEndpointReturnsSeasonHierarchy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := jsoncatalog.Open(filepath.Join(t.TempDir(), "catalog.json"))
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	library := domain.Library{ID: "series-library", Name: "Seriaalid", Kind: domain.LibraryKindSeries, Sources: []domain.LibrarySource{}, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateLibrary(ctx, library); err != nil {
		t.Fatalf("create library: %v", err)
	}
	items := []domain.MediaItem{
		{ID: "show", LibraryID: library.ID, Type: domain.MediaTypeSeries, Title: "Show", SortTitle: "Show", AddedAt: now, UpdatedAt: now},
		{ID: "episode", LibraryID: library.ID, SeriesID: "show", Type: domain.MediaTypeEpisode, Title: "Pilot", SortTitle: "Show 0001 0001", SeriesTitle: "Show", Season: 1, Episode: 1, File: domain.MediaFile{Name: "Show.S01E01.mkv"}, AddedAt: now, UpdatedAt: now},
	}
	if err := store.ReplaceLibraryMedia(ctx, library.ID, items); err != nil {
		t.Fatalf("replace media: %v", err)
	}
	handler := NewRouter(Dependencies{Media: service.NewMediaService(store)})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/series/show", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload domain.SeriesDetails
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode series response: %v", err)
	}
	if len(payload.Seasons) != 1 || len(payload.Seasons[0].Episodes) != 1 || payload.Seasons[0].Episodes[0].File.Name != "Show.S01E01.mkv" {
		t.Fatalf("unexpected hierarchy: %+v", payload)
	}
}

type httpTestPicker struct {
	path   string
	called bool
}

func (p *httpTestPicker) Pick(_ context.Context, request domain.FilePickerRequest) (string, bool, error) {
	p.called = true
	if request.Mode != domain.FilePickerModeDirectory {
		return "", false, domain.ErrInvalidInput
	}
	return p.path, false, nil
}

func TestNativePickerAcceptsOnlyLoopbackBrowser(t *testing.T) {
	picker := &httpTestPicker{path: `C:\Media\Filmid`}
	handler := NewRouter(Dependencies{Picker: picker})
	body := strings.NewReader(`{"mode":"directory","title":"Vali kaust"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/filesystem/pick", body)
	request.RemoteAddr = "127.0.0.1:32100"
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK || !picker.called || !strings.Contains(response.Body.String(), `C:\\Media\\Filmid`) {
		t.Fatalf("unexpected picker response: status=%d called=%v body=%s", response.Code, picker.called, response.Body.String())
	}

	picker.called = false
	remote := httptest.NewRequest(http.MethodPost, "/api/v1/filesystem/pick", strings.NewReader(`{"mode":"directory","title":"Vali kaust"}`))
	remote.RemoteAddr = "192.168.1.25:32100"
	remoteResponse := httptest.NewRecorder()
	handler.ServeHTTP(remoteResponse, remote)
	if remoteResponse.Code != http.StatusServiceUnavailable || picker.called {
		t.Fatalf("remote request must not open a server-side dialog: status=%d called=%v", remoteResponse.Code, picker.called)
	}
}

type httpTestUpdateInstaller struct {
	called  bool
	content string
	status  domain.UpdateStatus
}

func (i *httpTestUpdateInstaller) Status() domain.UpdateStatus { return i.status }
func (i *httpTestUpdateInstaller) Install(_ context.Context, reader io.Reader, _ string) (domain.UpdateStatus, error) {
	i.called = true
	content, err := io.ReadAll(reader)
	if err != nil {
		return domain.UpdateStatus{}, err
	}
	i.content = string(content)
	return i.status, nil
}

func TestUpdateInstallAcceptsOnlyTrustedLoopbackRequest(t *testing.T) {
	installer := &httpTestUpdateInstaller{status: domain.UpdateStatus{State: domain.UpdateStateInstalling, CurrentVersion: "0.2.1", TargetVersion: "0.2.2", LocalOnly: true}}
	handler := NewRouter(Dependencies{Updates: service.NewUpdateService(installer, nil)})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("package", "kodumedia-0.2.2-windows-x64.zip")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("zip payload"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/system/update", &body)
	request.RemoteAddr = "127.0.0.1:32100"
	request.Host = "localhost:8096"
	request.Header.Set("Origin", "http://localhost:8096")
	request.Header.Set("X-KoduMedia-Update", "install")
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || !installer.called || installer.content != "zip payload" {
		t.Fatalf("unexpected update response: status=%d called=%v content=%q body=%s", response.Code, installer.called, installer.content, response.Body.String())
	}

	installer.called = false
	var remoteBody bytes.Buffer
	remoteWriter := multipart.NewWriter(&remoteBody)
	remotePart, _ := remoteWriter.CreateFormFile("package", "update.zip")
	_, _ = remotePart.Write([]byte("zip payload"))
	_ = remoteWriter.Close()
	remote := httptest.NewRequest(http.MethodPost, "/api/v1/system/update", &remoteBody)
	remote.RemoteAddr = "192.168.1.20:32100"
	remote.Host = "localhost:8096"
	remote.Header.Set("X-KoduMedia-Update", "install")
	remote.Header.Set("Content-Type", remoteWriter.FormDataContentType())
	remoteResponse := httptest.NewRecorder()
	handler.ServeHTTP(remoteResponse, remote)
	if remoteResponse.Code != http.StatusForbidden || installer.called {
		t.Fatalf("remote update must be rejected: status=%d called=%v", remoteResponse.Code, installer.called)
	}
}

func TestUpdateStatusRequiresLoopbackHost(t *testing.T) {
	installer := &httpTestUpdateInstaller{status: domain.UpdateStatus{State: domain.UpdateStateIdle, CurrentVersion: "0.2.2", LocalOnly: true}}
	handler := NewRouter(Dependencies{Updates: service.NewUpdateService(installer, nil)})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/system/update", nil)
	request.RemoteAddr = "[::1]:32100"
	request.Host = "localhost:8096"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"currentVersion":"0.2.2"`) {
		t.Fatalf("unexpected status response: %d %s", response.Code, response.Body.String())
	}
}
