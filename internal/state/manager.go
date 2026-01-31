package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CodexForgeBR/cli-tools/internal/tasks"
)

const stateFileName = "current-state.json"

// SaveState persists the session state as indented JSON.
func SaveState(s *SessionState, dir string) error {
	// Marshal with 4-space indent
	data, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	path := filepath.Join(dir, stateFileName)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}

	return nil
}

// LoadState reads and parses the session state from the state directory.
func LoadState(dir string) (*SessionState, error) {
	path := filepath.Join(dir, stateFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}

	var s SessionState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}

	return &s, nil
}

// ValidateState checks that the state is consistent:
// - The tasks file exists
// - The tasks file hash matches (file hasn't changed)
func ValidateState(s *SessionState, tasksFile string) error {
	if _, err := os.Stat(tasksFile); err != nil {
		return fmt.Errorf("tasks file not found: %w", err)
	}

	currentHash, err := tasks.HashFile(tasksFile)
	if err != nil {
		return fmt.Errorf("hash tasks file: %w", err)
	}

	if s.TasksFileHash != "" && s.TasksFileHash != currentHash {
		return fmt.Errorf("tasks file changed: expected hash %s, got %s", s.TasksFileHash, currentHash)
	}

	return nil
}

// InitStateDir creates the state directory if it doesn't exist.
func InitStateDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}
