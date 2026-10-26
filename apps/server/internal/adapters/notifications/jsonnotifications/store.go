package jsonnotifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/adapters/atomicfile"
	"github.com/mihkelwow/kodumedia/internal/domain"
)

const schemaVersion = 1

type persistedState struct {
	Version int                   `json:"version"`
	Items   []domain.Notification `json:"items"`
}

type Store struct {
	path  string
	mu    sync.RWMutex
	state persistedState
}

func Open(path string) (*Store, error) {
	store := &Store{path: path, state: persistedState{Version: schemaVersion, Items: []domain.Notification{}}}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create notifications directory: %w", err)
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read notifications: %w", err)
	}
	if len(data) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(data, &store.state); err != nil {
		return nil, fmt.Errorf("decode notifications: %w", err)
	}
	if store.state.Version != schemaVersion {
		return nil, fmt.Errorf("unsupported notifications schema version %d", store.state.Version)
	}
	if store.state.Items == nil {
		store.state.Items = []domain.Notification{}
	}
	return store, nil
}

func (s *Store) List(ctx context.Context) ([]domain.Notification, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := append([]domain.Notification(nil), s.state.Items...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s *Store) Replace(ctx context.Context, items []domain.Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Items = append([]domain.Notification(nil), items...)
	return s.persistLocked()
}

func (s *Store) MarkAllRead(ctx context.Context, readAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := false
	for index := range s.state.Items {
		if s.state.Items[index].ReadAt == nil {
			value := readAt
			s.state.Items[index].ReadAt = &value
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return s.persistLocked()
}

func (s *Store) persistLocked() error {
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode notifications: %w", err)
	}
	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("write temporary notifications: %w", err)
	}
	if err := atomicfile.Replace(tempPath, s.path); err != nil {
		return fmt.Errorf("replace notifications: %w", err)
	}
	return nil
}
