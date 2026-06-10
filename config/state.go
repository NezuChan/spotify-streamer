package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	librespot "github.com/devgianlu/go-librespot"
)

// FileStateStore implements librespot.StateStore interface
type FileStateStore struct {
	Path string
	mu   sync.RWMutex
}

// Load reads the state from disk
func (s *FileStateStore) Load() (*librespot.AppState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			// State file doesn't exist yet, return empty state
			return &librespot.AppState{}, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state librespot.AppState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &state, nil
}

// Save writes the state to disk
func (s *FileStateStore) Save(state *librespot.AppState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(s.Path, data, 0600); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}
