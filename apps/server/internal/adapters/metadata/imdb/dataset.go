package imdb

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/adapters/atomicfile"
	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

const (
	datasetFilename  = "title.ratings.tsv.gz"
	datasetStateFile = "title.ratings.cache.json"
	maxDatasetBytes  = int64(512 << 20)
)

type datasetCacheState struct {
	Source        string     `json:"source"`
	ETag          string     `json:"etag,omitempty"`
	LastModified  string     `json:"lastModified,omitempty"`
	LastCheckedAt *time.Time `json:"lastCheckedAt,omitempty"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
	SizeBytes     int64      `json:"sizeBytes,omitempty"`
}

type DatasetManager struct {
	source     func() string
	cachePath  string
	statePath  string
	client     *http.Client
	refreshMu  sync.Mutex
	statusMu   sync.RWMutex
	status     domain.ComponentStatus
	cacheState datasetCacheState
}

func NewDatasetManager(source func() string, cacheDirectory string) (*DatasetManager, error) {
	if err := os.MkdirAll(cacheDirectory, 0o755); err != nil {
		return nil, fmt.Errorf("create IMDb cache directory: %w", err)
	}
	manager := &DatasetManager{
		source:    source,
		cachePath: filepath.Join(cacheDirectory, datasetFilename),
		statePath: filepath.Join(cacheDirectory, datasetStateFile),
		client: &http.Client{
			Timeout: 5 * time.Minute,
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				MaxIdleConns:          4,
				MaxIdleConnsPerHost:   2,
				IdleConnTimeout:       60 * time.Second,
				ResponseHeaderTimeout: 20 * time.Second,
			},
		},
	}
	manager.loadState()
	manager.initializeStatus()
	return manager, nil
}

func (m *DatasetManager) currentSource() string {
	if m.source == nil {
		return ""
	}
	return strings.TrimSpace(m.source())
}

func (m *DatasetManager) LocalPath() string {
	source := m.currentSource()
	if source == "" {
		return ""
	}
	if isRemoteSource(source) {
		m.statusMu.RLock()
		matches := strings.EqualFold(strings.TrimSpace(m.cacheState.Source), source)
		m.statusMu.RUnlock()
		if !matches {
			return ""
		}
		if info, err := os.Stat(m.cachePath); err == nil && !info.IsDir() {
			return m.cachePath
		}
		return ""
	}
	return source
}

func (m *DatasetManager) Status() domain.ComponentStatus {
	m.statusMu.RLock()
	defer m.statusMu.RUnlock()
	return cloneStatus(m.status)
}

func (m *DatasetManager) Inspect(ctx context.Context, source string) domain.ComponentStatus {
	source = strings.TrimSpace(source)
	if source == "" {
		return domain.ComponentStatus{Message: "IMDb reitingute allikas pole seadistatud."}
	}
	if isRemoteSource(source) {
		if current := m.currentSource(); strings.EqualFold(current, source) {
			m.statusMu.RLock()
			status := cloneStatus(m.status)
			cacheMatches := strings.EqualFold(strings.TrimSpace(m.cacheState.Source), source)
			m.statusMu.RUnlock()
			if status.Refreshing || (status.Available && cacheMatches) {
				return status
			}
		}
		return m.inspectRemote(ctx, source)
	}
	return inspectDatasetFile(ctx, source)
}

func (m *DatasetManager) Refresh(ctx context.Context, progress ports.JobProgress) (err error) {
	m.refreshMu.Lock()
	defer m.refreshMu.Unlock()

	source := m.currentSource()
	if source == "" {
		return fmt.Errorf("%w: IMDb ratings source is empty", domain.ErrInvalidInput)
	}
	m.setRefreshing(true, "IMDb andmestikku v\u00e4rskendatakse.")
	defer func() {
		if err != nil {
			m.finishWithError(err)
		}
	}()

	if !isRemoteSource(source) {
		if progress != nil {
			progress(35, "Kontrollin kohalikku IMDb faili")
		}
		status := inspectDatasetFile(ctx, source)
		if !status.Available {
			return fmt.Errorf("%w: %s", domain.ErrUnavailable, status.Message)
		}
		now := time.Now().UTC()
		status.LastCheckedAt = &now
		status.UpdatedAt = fileModTime(source)
		status.Refreshing = false
		m.statusMu.Lock()
		m.cacheState = datasetCacheState{Source: source, LastCheckedAt: &now, UpdatedAt: status.UpdatedAt, SizeBytes: status.SizeBytes}
		m.status = status
		m.statusMu.Unlock()
		if progress != nil {
			progress(100, "Kohalik IMDb andmestik on valmis")
		}
		return nil
	}

	parsed, parseErr := url.Parse(source)
	if parseErr != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("%w: IMDb remote source must be an absolute HTTPS URL", domain.ErrInvalidInput)
	}
	if progress != nil {
		progress(5, "Kontrollin IMDb andmestiku uuendust")
	}
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if requestErr != nil {
		return fmt.Errorf("create IMDb request: %w", requestErr)
	}
	request.Header.Set("Accept", "application/octet-stream")
	request.Header.Set("Accept-Encoding", "identity")
	request.Header.Set("User-Agent", "KoduMedia/0.2")

	m.statusMu.RLock()
	cached := m.cacheState
	m.statusMu.RUnlock()
	if strings.EqualFold(cached.Source, source) {
		if cached.ETag != "" {
			request.Header.Set("If-None-Match", cached.ETag)
		}
		if cached.LastModified != "" {
			request.Header.Set("If-Modified-Since", cached.LastModified)
		}
	}

	response, requestErr := m.client.Do(request)
	if requestErr != nil {
		return fmt.Errorf("download IMDb ratings: %w", requestErr)
	}
	defer response.Body.Close()

	now := time.Now().UTC()
	if response.StatusCode == http.StatusNotModified {
		if _, statErr := os.Stat(m.cachePath); statErr != nil {
			return fmt.Errorf("IMDb server returned not modified but cache is missing: %w", statErr)
		}
		cached.LastCheckedAt = &now
		cached.Source = source
		if err := m.persistState(cached); err != nil {
			return err
		}
		status := inspectDatasetFile(ctx, m.cachePath)
		if !status.Available {
			return fmt.Errorf("IMDb server returned not modified but the local cache is invalid: %s", status.Message)
		}
		status.Message = "IMDb andmestik on ajakohane; kasutusel on kohalik vahem\u00e4lu."
		status.LastCheckedAt = &now
		status.UpdatedAt = cached.UpdatedAt
		status.Refreshing = false
		m.statusMu.Lock()
		m.cacheState = cached
		m.status = status
		m.statusMu.Unlock()
		if progress != nil {
			progress(100, "IMDb andmestik on ajakohane")
		}
		return nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("IMDb dataset returned HTTP %d", response.StatusCode)
	}
	if response.ContentLength > maxDatasetBytes {
		return fmt.Errorf("IMDb dataset is larger than the configured safety limit")
	}

	temporary := m.cachePath + ".download"
	_ = os.Remove(temporary)
	file, createErr := os.OpenFile(temporary, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if createErr != nil {
		return fmt.Errorf("create IMDb cache file: %w", createErr)
	}
	written, copyErr := copyDataset(ctx, file, response.Body, response.ContentLength, progress)
	if syncErr := file.Sync(); copyErr == nil && syncErr != nil {
		copyErr = syncErr
	}
	if closeErr := file.Close(); copyErr == nil && closeErr != nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("store IMDb dataset: %w", copyErr)
	}
	if written <= 0 || written > maxDatasetBytes {
		_ = os.Remove(temporary)
		return fmt.Errorf("downloaded IMDb dataset has an invalid size")
	}
	if progress != nil {
		progress(88, "Kontrollin allalaaditud IMDb andmestikku")
	}
	validation := inspectDatasetFile(ctx, temporary)
	if !validation.Available {
		_ = os.Remove(temporary)
		return fmt.Errorf("downloaded IMDb dataset is invalid: %s", validation.Message)
	}
	if current := m.currentSource(); !strings.EqualFold(current, source) {
		_ = os.Remove(temporary)
		return fmt.Errorf("IMDb source changed while the download was running")
	}
	if err := atomicfile.Replace(temporary, m.cachePath); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("replace IMDb cache: %w", err)
	}

	state := datasetCacheState{
		Source:        source,
		ETag:          strings.TrimSpace(response.Header.Get("ETag")),
		LastModified:  strings.TrimSpace(response.Header.Get("Last-Modified")),
		LastCheckedAt: &now,
		UpdatedAt:     &now,
		SizeBytes:     written,
	}
	if err := m.persistState(state); err != nil {
		return err
	}
	status := validation
	status.Message = "IMDb title.ratings andmestik laaditi alla ja on kasutusvalmis."
	status.LastCheckedAt = &now
	status.UpdatedAt = &now
	status.SizeBytes = written
	status.Refreshing = false
	m.statusMu.Lock()
	m.cacheState = state
	m.status = status
	m.statusMu.Unlock()
	if progress != nil {
		progress(100, "IMDb andmestik on valmis")
	}
	return nil
}

func (m *DatasetManager) inspectRemote(ctx context.Context, source string) domain.ComponentStatus {
	parsed, err := url.Parse(source)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return domain.ComponentStatus{Message: "IMDb allikas peab olema absoluutne HTTPS aadress."}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, source, nil)
	if err != nil {
		return domain.ComponentStatus{Message: "IMDb allika kontrollp\u00e4ringut ei saanud luua."}
	}
	request.Header.Set("User-Agent", "KoduMedia/0.2")
	response, err := m.client.Do(request)
	if err != nil {
		return domain.ComponentStatus{Message: fmt.Sprintf("IMDb allikaga ei saanud \u00fchendust: %v", err)}
	}
	response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		return domain.ComponentStatus{Message: fmt.Sprintf("IMDb allikas vastas HTTP %d.", response.StatusCode)}
	}
	return domain.ComponentStatus{Available: true, Message: "IMDb allika URL on k\u00e4ttesaadav; fail salvestatakse p\u00e4rast seadete salvestamist serveri vahem\u00e4llu."}
}

func (m *DatasetManager) initializeStatus() {
	source := m.currentSource()
	if source == "" {
		m.status = domain.ComponentStatus{Message: "IMDb reitingute allikas pole seadistatud."}
		return
	}
	path := source
	if isRemoteSource(source) {
		if !strings.EqualFold(strings.TrimSpace(m.cacheState.Source), source) {
			m.status = domain.ComponentStatus{Message: "IMDb andmestik laaditakse serveri k\u00e4ivitamisel taustal alla."}
			return
		}
		path = m.cachePath
	}
	status := inspectDatasetFile(context.Background(), path)
	if status.Available {
		status.Message = "IMDb title.ratings andmestik on kasutusvalmis."
		status.LastCheckedAt = m.cacheState.LastCheckedAt
		status.UpdatedAt = m.cacheState.UpdatedAt
	}
	m.status = status
}

func (m *DatasetManager) loadState() {
	payload, err := os.ReadFile(m.statePath)
	if err != nil {
		return
	}
	var state datasetCacheState
	if json.Unmarshal(payload, &state) == nil {
		m.cacheState = state
	}
}

func (m *DatasetManager) persistState(state datasetCacheState) error {
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode IMDb cache state: %w", err)
	}
	payload = append(payload, '\n')
	temporary := m.statePath + ".tmp"
	if err := os.WriteFile(temporary, payload, 0o600); err != nil {
		return fmt.Errorf("write IMDb cache state: %w", err)
	}
	if err := atomicfile.Replace(temporary, m.statePath); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("replace IMDb cache state: %w", err)
	}
	return nil
}

func (m *DatasetManager) setRefreshing(refreshing bool, message string) {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	m.status.Refreshing = refreshing
	if message != "" {
		m.status.Message = message
	}
}

func (m *DatasetManager) finishWithError(err error) {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	m.status.Refreshing = false
	if path := m.localPathLocked(); path != "" {
		if status := inspectDatasetFile(context.Background(), path); status.Available {
			status.Message = fmt.Sprintf("IMDb uuendamine eba\u00f5nnestus; kasutusel on eelmine vahem\u00e4lu: %v", err)
			status.LastCheckedAt = m.cacheState.LastCheckedAt
			status.UpdatedAt = m.cacheState.UpdatedAt
			m.status = status
			return
		}
	}
	m.status.Available = false
	m.status.Message = fmt.Sprintf("IMDb andmestiku uuendamine eba\u00f5nnestus: %v", err)
}

func (m *DatasetManager) localPathLocked() string {
	source := m.currentSource()
	if isRemoteSource(source) {
		if strings.EqualFold(m.cacheState.Source, source) {
			return m.cachePath
		}
		return ""
	}
	return source
}

func inspectDatasetFile(ctx context.Context, path string) domain.ComponentStatus {
	path = strings.TrimSpace(path)
	if path == "" {
		return domain.ComponentStatus{Message: "IMDb reitingute faili pole seadistatud."}
	}
	file, err := os.Open(path)
	if err != nil {
		return domain.ComponentStatus{Message: fmt.Sprintf("IMDb faili ei saanud avada: %v", err)}
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		return domain.ComponentStatus{Message: "IMDb allikas ei ole loetav fail."}
	}
	var reader io.Reader = file
	bufferedFile := bufio.NewReader(file)
	magic, _ := bufferedFile.Peek(2)
	if len(magic) == 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		compressed, gzipErr := gzip.NewReader(bufferedFile)
		if gzipErr != nil {
			return domain.ComponentStatus{Message: fmt.Sprintf("IMDb gzip-fail on vigane: %v", gzipErr)}
		}
		defer compressed.Close()
		reader = compressed
	} else {
		reader = bufferedFile
	}
	type result struct {
		line string
		err  error
	}
	channel := make(chan result, 1)
	go func() {
		line, readErr := bufio.NewReader(reader).ReadString('\n')
		channel <- result{line: strings.TrimSpace(line), err: readErr}
	}()
	select {
	case <-ctx.Done():
		return domain.ComponentStatus{Message: "IMDb faili kontroll katkestati."}
	case value := <-channel:
		if value.err != nil && !errors.Is(value.err, io.EOF) {
			return domain.ComponentStatus{Message: fmt.Sprintf("IMDb faili ei saanud lugeda: %v", value.err)}
		}
		if value.line != "tconst\taverageRating\tnumVotes" {
			return domain.ComponentStatus{Message: "Fail ei ole IMDb title.ratings.tsv andmestik."}
		}
		updated := info.ModTime().UTC()
		return domain.ComponentStatus{Available: true, Message: "IMDb title.ratings andmestik on kasutatav.", UpdatedAt: &updated, SizeBytes: info.Size()}
	}
}

func copyDataset(ctx context.Context, destination io.Writer, source io.Reader, total int64, progress ports.JobProgress) (int64, error) {
	buffer := make([]byte, 128*1024)
	var written int64
	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}
		count, readErr := source.Read(buffer)
		if count > 0 {
			written += int64(count)
			if written > maxDatasetBytes {
				return written, fmt.Errorf("IMDb dataset exceeded the safety limit")
			}
			if _, err := destination.Write(buffer[:count]); err != nil {
				return written, err
			}
			if progress != nil && total > 0 {
				value := 10 + int((written*72)/total)
				if value > 82 {
					value = 82
				}
				progress(value, "Laen IMDb reitinguandmestikku")
			}
		}
		if errors.Is(readErr, io.EOF) {
			return written, nil
		}
		if readErr != nil {
			return written, readErr
		}
	}
}

func isRemoteSource(source string) bool {
	parsed, err := url.Parse(strings.TrimSpace(source))
	return err == nil && (strings.EqualFold(parsed.Scheme, "https") || strings.EqualFold(parsed.Scheme, "http")) && parsed.Host != ""
}

func fileModTime(path string) *time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	value := info.ModTime().UTC()
	return &value
}

func cloneStatus(status domain.ComponentStatus) domain.ComponentStatus {
	if status.LastCheckedAt != nil {
		value := *status.LastCheckedAt
		status.LastCheckedAt = &value
	}
	if status.UpdatedAt != nil {
		value := *status.UpdatedAt
		status.UpdatedAt = &value
	}
	return status
}

var _ ports.RatingsDataset = (*DatasetManager)(nil)
