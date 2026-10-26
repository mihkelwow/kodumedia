package jsonwatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/mihkelwow/kodumedia/internal/adapters/atomicfile"
	"github.com/mihkelwow/kodumedia/internal/domain"
)

const schemaVersion = 1

type persistedState struct {
	Version int                          `json:"version"`
	Items   map[string]domain.WatchState `json:"items"`
}

type Store struct {
	path  string
	mu    sync.RWMutex
	state persistedState
}

func Open(path string) (*Store, error) {
	store := &Store{path: path, state: persistedState{Version: schemaVersion, Items: map[string]domain.WatchState{}}}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create watch-state directory: %w", err)
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read watch state: %w", err)
	}
	if len(data) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(data, &store.state); err != nil {
		return nil, fmt.Errorf("decode watch state: %w", err)
	}
	if store.state.Version != schemaVersion {
		return nil, fmt.Errorf("unsupported watch-state schema version %d", store.state.Version)
	}
	if store.state.Items == nil {
		store.state.Items = map[string]domain.WatchState{}
	}
	return store, nil
}

func (s *Store) Get(ctx context.Context, mediaID string) (domain.WatchState, error) {
	if err := ctx.Err(); err != nil {
		return domain.WatchState{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.state.Items[mediaID]
	if !ok {
		return domain.WatchState{}, domain.ErrNotFound
	}
	return state, nil
}

func (s *Store) List(ctx context.Context) ([]domain.WatchState, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.WatchState, 0, len(s.state.Items))
	for _, item := range s.state.Items {
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].LastWatchedAt.After(items[j].LastWatchedAt) })
	return items, nil
}

func (s *Store) Upsert(ctx context.Context, state domain.WatchState) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if state.MediaID == "" {
		return fmt.Errorf("%w: media id is required", domain.ErrInvalidInput)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Items[state.MediaID] = state
	return s.persistLocked()
}

func (s *Store) Delete(ctx context.Context, mediaID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.state.Items[mediaID]; !exists {
		return domain.ErrNotFound
	}
	delete(s.state.Items, mediaID)
	return s.persistLocked()
}

func (s *Store) persistLocked() error {
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode watch state: %w", err)
	}
	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("write temporary watch state: %w", err)
	}
	if err := atomicfile.Replace(tempPath, s.path); err != nil {
		return fmt.Errorf("replace watch state: %w", err)
	}
	return nil
}
