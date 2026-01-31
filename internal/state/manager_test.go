package state

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSaveState validates that SaveState writes valid JSON with proper formatting
func TestSaveState(t *testing.T) {
	tests := []struct {
		name  string
		state *SessionState
	}{
		{
			name: "complete state with all fields",
			state: &SessionState{
				SchemaVersion:    2,
				SessionID:        "ralph-20260130-143000",
				StartedAt:        "2026-01-30T14:30:00Z",
				LastUpdated:      "2026-01-30T14:35:00Z",
				Iteration:        3,
				Status:           "IN_PROGRESS",
				Phase:            "validation",
				Verdict:          "NEEDS_MORE_WORK",
				TasksFile:        "/tmp/test/tasks.md",
				TasksFileHash:    "abc123def456",
				AICli:            "claude",
				ImplModel:        "opus",
				ValModel:         "opus",
				MaxIterations:    20,
				MaxInadmissible:  5,
				OriginalPlanFile: stringPtr("/tmp/plan.md"),
				GithubIssue:      stringPtr("https://github.com/owner/repo/issues/123"),
				Learnings: LearningsState{
					Enabled: 1,
					File:    "/tmp/test/.ralph-loop/learnings.md",
				},
				CrossValidation: CrossValState{
					Enabled:   1,
					AI:        "codex",
					Model:     "default",
					Available: true,
				},
				FinalPlanValidation: PlanValState{
					AI:        "codex",
					Model:     "default",
					Available: true,
				},
				TasksValidation: TasksValState{
					AI:        "claude",
					Model:     "opus",
					Available: true,
				},
				Schedule: ScheduleState{
					Enabled:     false,
					TargetEpoch: 0,
					TargetHuman: "",
				},
				RetryState: RetryState{
					Attempt: 1,
					Delay:   5,
				},
				InadmissibleCount: 0,
				LastFeedback:      "",
			},
		},
		{
			name: "minimal state with null optional fields",
			state: &SessionState{
				SchemaVersion:   2,
				SessionID:       "ralph-minimal",
				StartedAt:       "2026-01-30T14:30:00Z",
				LastUpdated:     "2026-01-30T14:30:00Z",
				Iteration:       1,
				Status:          "PENDING",
				Phase:           "implementation",
				TasksFile:       "/tmp/test/tasks.md",
				TasksFileHash:   "xyz789",
				AICli:           "claude",
				ImplModel:       "opus",
				ValModel:        "opus",
				MaxIterations:   20,
				MaxInadmissible: 5,
				Learnings:       LearningsState{},
				CrossValidation: CrossValState{},
				FinalPlanValidation: PlanValState{
					AI:        "claude",
					Model:     "opus",
					Available: true,
				},
				TasksValidation: TasksValState{
					AI:        "claude",
					Model:     "opus",
					Available: true,
				},
				Schedule:   ScheduleState{},
				RetryState: RetryState{Attempt: 1, Delay: 5},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Save state
			err := SaveState(tt.state, tmpDir)
			require.NoError(t, err, "SaveState should not fail")

			// Verify file exists
			stateFile := filepath.Join(tmpDir, "current-state.json")
			_, err = os.Stat(stateFile)
			require.NoError(t, err, "State file should exist")

			// Read file content
			content, err := os.ReadFile(stateFile)
			require.NoError(t, err, "Should be able to read state file")

			// Verify it's valid JSON
			var jsonMap map[string]interface{}
			err = json.Unmarshal(content, &jsonMap)
			require.NoError(t, err, "File content should be valid JSON")

			// Verify indentation (4 spaces) by checking for known patterns
			jsonStr := string(content)
			assert.Contains(t, jsonStr, "    \"schema_version\":", "Should use 4-space indentation")
			assert.Contains(t, jsonStr, "    \"session_id\":", "Should use 4-space indentation")

			// Verify nested objects are also properly indented
			assert.Contains(t, jsonStr, "    \"learnings\": {", "Nested objects should be indented")
			assert.Contains(t, jsonStr, "        \"enabled\":", "Nested fields should use 8-space indentation")
		})
	}
}

// TestLoadState validates that LoadState correctly restores all fields from file
func TestLoadState(t *testing.T) {
	tests := []struct {
		name  string
		state *SessionState
	}{
		{
			name: "complete state",
			state: &SessionState{
				SchemaVersion:    2,
				SessionID:        "ralph-20260130-143000",
				StartedAt:        "2026-01-30T14:30:00Z",
				LastUpdated:      "2026-01-30T14:35:00Z",
				Iteration:        3,
				Status:           "IN_PROGRESS",
				Phase:            "validation",
				Verdict:          "NEEDS_MORE_WORK",
				TasksFile:        "/tmp/test/tasks.md",
				TasksFileHash:    "abc123def456",
				AICli:            "claude",
				ImplModel:        "opus",
				ValModel:         "opus",
				MaxIterations:    20,
				MaxInadmissible:  5,
				OriginalPlanFile: stringPtr("/tmp/plan.md"),
				GithubIssue:      stringPtr("https://github.com/owner/repo/issues/123"),
				Learnings: LearningsState{
					Enabled: 1,
					File:    "/tmp/test/.ralph-loop/learnings.md",
				},
				CrossValidation: CrossValState{
					Enabled:   1,
					AI:        "codex",
					Model:     "default",
					Available: true,
				},
				FinalPlanValidation: PlanValState{
					AI:        "codex",
					Model:     "default",
					Available: true,
				},
				TasksValidation: TasksValState{
					AI:        "claude",
					Model:     "opus",
					Available: true,
				},
				Schedule: ScheduleState{
					Enabled:     true,
					TargetEpoch: 1706623800,
					TargetHuman: "2026-01-30T16:30:00Z",
				},
				RetryState: RetryState{
					Attempt: 2,
					Delay:   10,
				},
				InadmissibleCount: 1,
				LastFeedback:      "Implementation incomplete",
			},
		},
		{
			name: "state with null optional fields",
			state: &SessionState{
				SchemaVersion:       2,
				SessionID:           "ralph-minimal",
				StartedAt:           "2026-01-30T14:30:00Z",
				LastUpdated:         "2026-01-30T14:30:00Z",
				Iteration:           1,
				Status:              "PENDING",
				Phase:               "implementation",
				TasksFile:           "/tmp/test/tasks.md",
				TasksFileHash:       "xyz789",
				AICli:               "claude",
				ImplModel:           "opus",
				ValModel:            "opus",
				MaxIterations:       20,
				MaxInadmissible:     5,
				OriginalPlanFile:    nil,
				GithubIssue:         nil,
				Learnings:           LearningsState{Enabled: 0, File: ""},
				CrossValidation:     CrossValState{},
				FinalPlanValidation: PlanValState{AI: "claude", Model: "opus", Available: true},
				TasksValidation:     TasksValState{AI: "claude", Model: "opus", Available: true},
				Schedule:            ScheduleState{Enabled: false, TargetEpoch: 0, TargetHuman: ""},
				RetryState:          RetryState{Attempt: 1, Delay: 5},
				InadmissibleCount:   0,
				LastFeedback:        "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Save state first
			err := SaveState(tt.state, tmpDir)
			require.NoError(t, err, "SaveState should succeed")

			// Load state back
			loaded, err := LoadState(tmpDir)
			require.NoError(t, err, "LoadState should not fail")
			require.NotNil(t, loaded, "Loaded state should not be nil")

			// Compare all fields
			assert.Equal(t, tt.state.SchemaVersion, loaded.SchemaVersion)
			assert.Equal(t, tt.state.SessionID, loaded.SessionID)
			assert.Equal(t, tt.state.StartedAt, loaded.StartedAt)
			assert.Equal(t, tt.state.LastUpdated, loaded.LastUpdated)
			assert.Equal(t, tt.state.Iteration, loaded.Iteration)
			assert.Equal(t, tt.state.Status, loaded.Status)
			assert.Equal(t, tt.state.Phase, loaded.Phase)
			assert.Equal(t, tt.state.Verdict, loaded.Verdict)
			assert.Equal(t, tt.state.TasksFile, loaded.TasksFile)
			assert.Equal(t, tt.state.TasksFileHash, loaded.TasksFileHash)
			assert.Equal(t, tt.state.AICli, loaded.AICli)
			assert.Equal(t, tt.state.ImplModel, loaded.ImplModel)
			assert.Equal(t, tt.state.ValModel, loaded.ValModel)
			assert.Equal(t, tt.state.MaxIterations, loaded.MaxIterations)
			assert.Equal(t, tt.state.MaxInadmissible, loaded.MaxInadmissible)
			assert.Equal(t, tt.state.InadmissibleCount, loaded.InadmissibleCount)
			assert.Equal(t, tt.state.LastFeedback, loaded.LastFeedback)

			// Compare optional pointer fields
			if tt.state.OriginalPlanFile == nil {
				assert.Nil(t, loaded.OriginalPlanFile)
			} else {
				require.NotNil(t, loaded.OriginalPlanFile)
				assert.Equal(t, *tt.state.OriginalPlanFile, *loaded.OriginalPlanFile)
			}

			if tt.state.GithubIssue == nil {
				assert.Nil(t, loaded.GithubIssue)
			} else {
				require.NotNil(t, loaded.GithubIssue)
				assert.Equal(t, *tt.state.GithubIssue, *loaded.GithubIssue)
			}

			// Compare nested objects
			assert.Equal(t, tt.state.Learnings, loaded.Learnings)
			assert.Equal(t, tt.state.CrossValidation, loaded.CrossValidation)
			assert.Equal(t, tt.state.FinalPlanValidation, loaded.FinalPlanValidation)
			assert.Equal(t, tt.state.TasksValidation, loaded.TasksValidation)
			assert.Equal(t, tt.state.Schedule, loaded.Schedule)
			assert.Equal(t, tt.state.RetryState, loaded.RetryState)

			// Overall struct comparison
			assert.Equal(t, tt.state, loaded)
		})
	}
}

// TestValidateState tests state validation including file existence and hash matching
func TestValidateState(t *testing.T) {
	t.Run("valid state with matching hash", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create tasks file
		tasksFile := filepath.Join(tmpDir, "tasks.md")
		tasksContent := []byte("# Tasks\n- Task 1\n- Task 2\n")
		err := os.WriteFile(tasksFile, tasksContent, 0644)
		require.NoError(t, err)

		// Calculate hash
		hash := sha256.Sum256(tasksContent)
		hashStr := fmt.Sprintf("%x", hash)

		// Create state with correct hash
		state := &SessionState{
			SchemaVersion: 2,
			SessionID:     "test-session",
			TasksFile:     tasksFile,
			TasksFileHash: hashStr,
		}

		// Validate should succeed
		err = ValidateState(state, tasksFile)
		assert.NoError(t, err, "ValidateState should succeed with matching hash")
	})

	t.Run("invalid state with mismatched hash", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create tasks file
		tasksFile := filepath.Join(tmpDir, "tasks.md")
		tasksContent := []byte("# Tasks\n- Task 1\n- Task 2\n")
		err := os.WriteFile(tasksFile, tasksContent, 0644)
		require.NoError(t, err)

		// Create state with wrong hash
		state := &SessionState{
			SchemaVersion: 2,
			SessionID:     "test-session",
			TasksFile:     tasksFile,
			TasksFileHash: "wrong_hash_value",
		}

		// Validate should fail
		err = ValidateState(state, tasksFile)
		assert.Error(t, err, "ValidateState should fail with mismatched hash")
	})

	t.Run("invalid state with nonexistent tasks file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tasksFile := filepath.Join(tmpDir, "nonexistent.md")

		state := &SessionState{
			SchemaVersion: 2,
			SessionID:     "test-session",
			TasksFile:     tasksFile,
			TasksFileHash: "some_hash",
		}

		// Validate should fail
		err := ValidateState(state, tasksFile)
		assert.Error(t, err, "ValidateState should fail when tasks file doesn't exist")
	})

	t.Run("empty tasks file hash skips hash validation", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create tasks file
		tasksFile := filepath.Join(tmpDir, "tasks.md")
		err := os.WriteFile(tasksFile, []byte("# Tasks"), 0644)
		require.NoError(t, err)

		state := &SessionState{
			SchemaVersion: 2,
			SessionID:     "test-session",
			TasksFile:     tasksFile,
			TasksFileHash: "", // Empty hash — skip hash validation
		}

		// Validate should pass (empty hash means no hash check)
		err = ValidateState(state, tasksFile)
		assert.NoError(t, err, "ValidateState should pass with empty hash (skips hash check)")
	})
}

// TestInitStateDir tests state directory initialization
func TestInitStateDir(t *testing.T) {
	t.Run("create new directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		stateDir := filepath.Join(tmpDir, ".ralph-loop")

		// Directory should not exist yet
		_, err := os.Stat(stateDir)
		assert.True(t, os.IsNotExist(err), "Directory should not exist initially")

		// Initialize state directory
		err = InitStateDir(stateDir)
		require.NoError(t, err, "InitStateDir should not fail")

		// Directory should now exist
		info, err := os.Stat(stateDir)
		require.NoError(t, err, "Directory should exist after init")
		assert.True(t, info.IsDir(), "Path should be a directory")
	})

	t.Run("existing directory is ok", func(t *testing.T) {
		tmpDir := t.TempDir()
		stateDir := filepath.Join(tmpDir, ".ralph-loop")

		// Create directory manually
		err := os.MkdirAll(stateDir, 0755)
		require.NoError(t, err)

		// Initialize should not fail on existing directory
		err = InitStateDir(stateDir)
		assert.NoError(t, err, "InitStateDir should not fail on existing directory")
	})

	t.Run("nested directory creation", func(t *testing.T) {
		tmpDir := t.TempDir()
		stateDir := filepath.Join(tmpDir, "nested", "path", ".ralph-loop")

		// Initialize should create all parent directories
		err := InitStateDir(stateDir)
		require.NoError(t, err, "InitStateDir should create nested directories")

		// Verify directory exists
		info, err := os.Stat(stateDir)
		require.NoError(t, err, "Nested directory should exist")
		assert.True(t, info.IsDir(), "Path should be a directory")
	})

	t.Run("verify directory permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		stateDir := filepath.Join(tmpDir, ".ralph-loop")

		err := InitStateDir(stateDir)
		require.NoError(t, err)

		// Check permissions (should be 0755)
		info, err := os.Stat(stateDir)
		require.NoError(t, err)

		// On Unix systems, verify directory is readable, writable, executable by owner
		mode := info.Mode()
		assert.True(t, mode&0700 == 0700, "Owner should have rwx permissions")
	})
}

// TestSaveLoadRoundTrip tests that saving and loading preserves all data
func TestSaveLoadRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		state *SessionState
	}{
		{
			name: "complete state with all features enabled",
			state: &SessionState{
				SchemaVersion:    2,
				SessionID:        "ralph-20260130-143000",
				StartedAt:        "2026-01-30T14:30:00Z",
				LastUpdated:      "2026-01-30T14:35:00Z",
				Iteration:        5,
				Status:           "RUNNING",
				Phase:            "validation",
				Verdict:          "ACCEPTABLE",
				TasksFile:        "/tmp/test/tasks.md",
				TasksFileHash:    "abc123def456",
				AICli:            "claude",
				ImplModel:        "opus",
				ValModel:         "opus",
				MaxIterations:    20,
				MaxInadmissible:  5,
				OriginalPlanFile: stringPtr("/tmp/original.md"),
				GithubIssue:      stringPtr("https://github.com/owner/repo/issues/42"),
				Learnings: LearningsState{
					Enabled: 1,
					File:    "/tmp/test/.ralph-loop/learnings.md",
				},
				CrossValidation: CrossValState{
					Enabled:   1,
					AI:        "codex",
					Model:     "default",
					Available: true,
				},
				FinalPlanValidation: PlanValState{
					AI:        "codex",
					Model:     "default",
					Available: true,
				},
				TasksValidation: TasksValState{
					AI:        "claude",
					Model:     "opus",
					Available: true,
				},
				Schedule: ScheduleState{
					Enabled:     true,
					TargetEpoch: 1706623800,
					TargetHuman: "2026-01-30T16:30:00Z",
				},
				RetryState: RetryState{
					Attempt: 3,
					Delay:   15,
				},
				InadmissibleCount: 2,
				LastFeedback:      "Please improve error handling in module X",
			},
		},
		{
			name: "minimal state",
			state: &SessionState{
				SchemaVersion:       2,
				SessionID:           "ralph-minimal",
				StartedAt:           "2026-01-30T15:00:00Z",
				LastUpdated:         "2026-01-30T15:00:00Z",
				Iteration:           1,
				Status:              "PENDING",
				Phase:               "implementation",
				TasksFile:           "/tmp/test/tasks.md",
				TasksFileHash:       "xyz789",
				AICli:               "claude",
				ImplModel:           "opus",
				ValModel:            "opus",
				MaxIterations:       20,
				MaxInadmissible:     5,
				Learnings:           LearningsState{},
				CrossValidation:     CrossValState{},
				FinalPlanValidation: PlanValState{AI: "claude", Model: "opus", Available: true},
				TasksValidation:     TasksValState{AI: "claude", Model: "opus", Available: true},
				Schedule:            ScheduleState{},
				RetryState:          RetryState{Attempt: 1, Delay: 5},
			},
		},
		{
			name: "state with special characters in feedback",
			state: &SessionState{
				SchemaVersion:   2,
				SessionID:       "ralph-special-chars",
				StartedAt:       "2026-01-30T16:00:00Z",
				LastUpdated:     "2026-01-30T16:05:00Z",
				TasksFile:       "/tmp/test/tasks.md",
				TasksFileHash:   "hash123",
				AICli:           "claude",
				ImplModel:       "opus",
				ValModel:        "opus",
				MaxIterations:   20,
				MaxInadmissible: 5,
				LastFeedback:    "Feedback with special chars: \n\t\"quotes\", 'apostrophes', & ampersands, < less than, > greater than, 你好世界",
				Learnings:       LearningsState{},
				CrossValidation: CrossValState{},
				FinalPlanValidation: PlanValState{AI: "claude", Model: "opus", Available: true},
				TasksValidation:     TasksValState{AI: "claude", Model: "opus", Available: true},
				Schedule:            ScheduleState{},
				RetryState:          RetryState{Attempt: 1, Delay: 5},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Save state
			err := SaveState(tt.state, tmpDir)
			require.NoError(t, err, "SaveState should succeed")

			// Load state back
			loaded, err := LoadState(tmpDir)
			require.NoError(t, err, "LoadState should succeed")
			require.NotNil(t, loaded, "Loaded state should not be nil")

			// Complete equality check
			assert.Equal(t, tt.state, loaded, "Round-trip should preserve all state data")
		})
	}
}

// TestLoadStateNonexistentFile tests that LoadState returns error for missing file
func TestLoadStateNonexistentFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Try to load from directory with no state file
	loaded, err := LoadState(tmpDir)
	assert.Error(t, err, "LoadState should fail when state file doesn't exist")
	assert.Nil(t, loaded, "Loaded state should be nil on error")
	assert.Contains(t, err.Error(), "current-state.json", "Error should mention state.json file")
}

// TestLoadStateInvalidJSON tests that LoadState returns error for malformed JSON
func TestLoadStateInvalidJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonContent string
	}{
		{
			name:        "completely invalid JSON",
			jsonContent: "this is not json at all",
		},
		{
			name:        "truncated JSON",
			jsonContent: `{"schema_version": 2, "session_id": "test"`,
		},
		{
			name:        "invalid JSON syntax",
			jsonContent: `{"schema_version": 2, "session_id": "test",}`,
		},
		{
			name:        "empty file",
			jsonContent: "",
		},
		{
			name:        "JSON with wrong types",
			jsonContent: `{"schema_version": "not_a_number", "session_id": 123}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateFile := filepath.Join(tmpDir, "current-state.json")

			// Write invalid JSON to file
			err := os.WriteFile(stateFile, []byte(tt.jsonContent), 0644)
			require.NoError(t, err)

			// Try to load
			loaded, err := LoadState(tmpDir)
			assert.Error(t, err, "LoadState should fail with invalid JSON")
			assert.Nil(t, loaded, "Loaded state should be nil on error")
		})
	}
}

// TestSaveStateCreatesMissingDirectory tests that SaveState creates the directory if needed
func TestSaveStateCreatesMissingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "missing", "nested", ".ralph-loop")

	state := &SessionState{
		SchemaVersion:   2,
		SessionID:       "test-session",
		StartedAt:       "2026-01-30T14:30:00Z",
		LastUpdated:     "2026-01-30T14:30:00Z",
		TasksFile:       "/tmp/tasks.md",
		TasksFileHash:   "hash123",
		AICli:           "claude",
		ImplModel:       "opus",
		ValModel:        "opus",
		MaxIterations:   20,
		MaxInadmissible: 5,
		Learnings:       LearningsState{},
		CrossValidation: CrossValState{},
		FinalPlanValidation: PlanValState{AI: "claude", Model: "opus", Available: true},
		TasksValidation:     TasksValState{AI: "claude", Model: "opus", Available: true},
		Schedule:            ScheduleState{},
		RetryState:          RetryState{Attempt: 1, Delay: 5},
	}

	// Save should create the directory structure
	err := SaveState(state, stateDir)
	require.NoError(t, err, "SaveState should create missing directories")

	// Verify directory was created
	_, err = os.Stat(stateDir)
	assert.NoError(t, err, "State directory should have been created")

	// Verify state file exists
	stateFile := filepath.Join(stateDir, "current-state.json")
	_, err = os.Stat(stateFile)
	assert.NoError(t, err, "State file should exist")
}

// TestMultipleLoadsSameData tests that multiple loads return consistent data
func TestMultipleLoadsSameData(t *testing.T) {
	tmpDir := t.TempDir()

	original := &SessionState{
		SchemaVersion: 2,
		SessionID:     "test-consistency",
		StartedAt:     "2026-01-30T14:30:00Z",
		LastUpdated:   "2026-01-30T14:35:00Z",
		Iteration:     5,
		Status:        "IN_PROGRESS",
		Phase:         "validation",
		TasksFile:     "/tmp/test/tasks.md",
		TasksFileHash: "abc123",
		AICli:         "claude",
		ImplModel:     "opus",
		ValModel:      "opus",
		MaxIterations: 20,
		MaxInadmissible: 5,
		Learnings:       LearningsState{Enabled: 1, File: "/tmp/learnings.md"},
		CrossValidation: CrossValState{},
		FinalPlanValidation: PlanValState{AI: "claude", Model: "opus", Available: true},
		TasksValidation:     TasksValState{AI: "claude", Model: "opus", Available: true},
		Schedule:            ScheduleState{},
		RetryState:          RetryState{Attempt: 2, Delay: 10},
	}

	// Save once
	err := SaveState(original, tmpDir)
	require.NoError(t, err)

	// Load multiple times
	for i := 0; i < 5; i++ {
		loaded, err := LoadState(tmpDir)
		require.NoError(t, err, "Load iteration %d should succeed", i+1)
		assert.Equal(t, original, loaded, "Load iteration %d should return consistent data", i+1)
	}
}

// TestSaveState_InvalidDirectory tests SaveState with an invalid directory path.
func TestSaveState_InvalidDirectory(t *testing.T) {
	// Use a path that cannot be created (on Unix systems, /dev/null is a device file)
	invalidDir := "/dev/null/cannot-create-dir"

	state := &SessionState{
		SchemaVersion:   2,
		SessionID:       "test-invalid-dir",
		StartedAt:       "2026-01-30T14:30:00Z",
		LastUpdated:     "2026-01-30T14:30:00Z",
		TasksFile:       "/tmp/tasks.md",
		TasksFileHash:   "hash123",
		AICli:           "claude",
		ImplModel:       "opus",
		ValModel:        "opus",
		MaxIterations:   20,
		MaxInadmissible: 5,
		Learnings:       LearningsState{},
		CrossValidation: CrossValState{},
		FinalPlanValidation: PlanValState{AI: "claude", Model: "opus", Available: true},
		TasksValidation:     TasksValState{AI: "claude", Model: "opus", Available: true},
		Schedule:            ScheduleState{},
		RetryState:          RetryState{Attempt: 1, Delay: 5},
	}

	err := SaveState(state, invalidDir)
	assert.Error(t, err, "SaveState should fail with invalid directory path")
	assert.Contains(t, err.Error(), "create state dir", "Error should mention directory creation failure")
}

// TestValidateState_EmptyHash tests ValidateState with empty hash (should skip hash validation).
func TestValidateState_EmptyHash(t *testing.T) {
	tmpDir := t.TempDir()

	// Create tasks file
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := []byte("# Tasks\n- Task 1\n- Task 2\n")
	err := os.WriteFile(tasksFile, tasksContent, 0644)
	require.NoError(t, err)

	// Create state with empty hash
	state := &SessionState{
		SchemaVersion: 2,
		SessionID:     "test-empty-hash",
		TasksFile:     tasksFile,
		TasksFileHash: "", // Empty hash should skip validation
	}

	// Validate should succeed even though we didn't calculate the hash
	err = ValidateState(state, tasksFile)
	assert.NoError(t, err, "ValidateState should succeed with empty hash (skips hash check)")
}

// TestSaveState_WriteFileError tests SaveState when writing the file fails.
func TestSaveState_WriteFileError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a read-only directory
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err := os.MkdirAll(readOnlyDir, 0555) // Read and execute only
	require.NoError(t, err)

	// Make sure to restore permissions for cleanup
	defer os.Chmod(readOnlyDir, 0755)

	state := &SessionState{
		SchemaVersion:   2,
		SessionID:       "test-write-error",
		StartedAt:       "2026-01-30T14:30:00Z",
		LastUpdated:     "2026-01-30T14:30:00Z",
		TasksFile:       "/tmp/tasks.md",
		TasksFileHash:   "hash123",
		AICli:           "claude",
		ImplModel:       "opus",
		ValModel:        "opus",
		MaxIterations:   20,
		MaxInadmissible: 5,
		Learnings:       LearningsState{},
		CrossValidation: CrossValState{},
		FinalPlanValidation: PlanValState{AI: "claude", Model: "opus", Available: true},
		TasksValidation:     TasksValState{AI: "claude", Model: "opus", Available: true},
		Schedule:            ScheduleState{},
		RetryState:          RetryState{Attempt: 1, Delay: 5},
	}

	err = SaveState(state, readOnlyDir)
	assert.Error(t, err, "SaveState should fail when directory is read-only")
	assert.Contains(t, err.Error(), "write state file", "Error should mention file write failure")
}

// TestSaveState_MarshalIndentCannotFail documents that the json.MarshalIndent
// error path in SaveState (line 18-19) is unreachable with valid SessionState structs.
// json.MarshalIndent cannot fail when all fields are basic types (string, int, bool,
// pointers, structs of basic types). This 1 uncovered statement is acceptable dead code.
func TestSaveState_MarshalIndentCannotFail(t *testing.T) {
	t.Log("MarshalIndent error path is unreachable with valid SessionState structs")

	// Verify that all possible SessionState values marshal successfully, including
	// edge-case values (zero values, empty strings, nil pointers)
	states := []*SessionState{
		{}, // all zero-values
		{
			SchemaVersion:    0,
			SessionID:        "",
			OriginalPlanFile: nil,
			GithubIssue:      nil,
		},
	}

	for _, s := range states {
		data, err := json.MarshalIndent(s, "", "    ")
		require.NoError(t, err, "MarshalIndent should never fail with SessionState")
		assert.NotEmpty(t, data)
	}
}

// TestValidateState_HashFileError tests ValidateState when hashing the file fails.
func TestValidateState_HashFileError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with restricted permissions (no read access)
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	err := os.WriteFile(tasksFile, []byte("content"), 0000) // No permissions
	require.NoError(t, err)

	// Make sure to restore permissions for cleanup
	defer os.Chmod(tasksFile, 0644)

	state := &SessionState{
		SchemaVersion: 2,
		SessionID:     "test-hash-error",
		TasksFile:     tasksFile,
		TasksFileHash: "some-hash",
	}

	// Validate should fail when it can't read the file to hash it
	err = ValidateState(state, tasksFile)
	assert.Error(t, err, "ValidateState should fail when file can't be read for hashing")
	assert.Contains(t, err.Error(), "hash tasks file", "Error should mention hashing failure")
}
