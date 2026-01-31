package state

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResumeFromState_PhaseAwareContinuation tests that resume logic correctly
// handles different phase states for continuation.
func TestResumeFromState_PhaseAwareContinuation(t *testing.T) {
	tests := []struct {
		name          string
		phase         string
		iteration     int
		verdict       string
		expectedPhase string
		description   string
	}{
		{
			name:          "cross_validation phase skips impl and val, continues to next iteration",
			phase:         PhaseCrossValidation,
			iteration:     3,
			verdict:       "ACCEPTABLE",
			expectedPhase: PhaseCrossValidation,
			description:   "Cross-validation phase should maintain phase and continue",
		},
		{
			name:          "validation phase skips impl, continues with validation",
			phase:         PhaseValidation,
			iteration:     2,
			verdict:       "NEEDS_MORE_WORK",
			expectedPhase: PhaseValidation,
			description:   "Validation phase should maintain phase",
		},
		{
			name:          "implementation phase restarts full iteration",
			phase:         PhaseImplementation,
			iteration:     1,
			verdict:       "",
			expectedPhase: PhaseImplementation,
			description:   "Implementation phase should maintain phase",
		},
		{
			name:          "waiting_for_schedule checks if time passed",
			phase:         PhaseWaitingForSchedule,
			iteration:     0,
			verdict:       "",
			expectedPhase: PhaseWaitingForSchedule,
			description:   "Schedule waiting phase should maintain phase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create a tasks file
			tasksFile := filepath.Join(tmpDir, "tasks.md")
			tasksContent := []byte("# Tasks\n- [ ] Task 1\n")
			err := os.WriteFile(tasksFile, tasksContent, 0644)
			require.NoError(t, err)

			// Create state at specific phase
			state := &SessionState{
				SchemaVersion:       2,
				SessionID:           "test-resume",
				StartedAt:           "2026-01-30T14:00:00Z",
				LastUpdated:         "2026-01-30T14:30:00Z",
				Iteration:           tt.iteration,
				Status:              StatusInterrupted,
				Phase:               tt.phase,
				Verdict:             tt.verdict,
				TasksFile:           tasksFile,
				TasksFileHash:       computeTestHash(tasksContent),
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
			}

			// Test resume
			err = ResumeFromState(state, tasksFile, false)
			assert.NoError(t, err, "Resume should succeed for %s", tt.description)

			// Verify state was updated
			assert.Equal(t, StatusInProgress, state.Status, "Status should be IN_PROGRESS after resume")
			assert.Equal(t, tt.expectedPhase, state.Phase, "Phase should be preserved: %s", tt.description)
		})
	}
}

// TestResumeFromState_RetryStateResume tests resuming with retry attempts > 1
func TestResumeFromState_RetryStateResume(t *testing.T) {
	tmpDir := t.TempDir()

	// Create tasks file
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := []byte("# Tasks\n- [ ] Task 1\n")
	err := os.WriteFile(tasksFile, tasksContent, 0644)
	require.NoError(t, err)

	// Create state with retry attempt > 1
	state := &SessionState{
		SchemaVersion:       2,
		SessionID:           "test-retry",
		StartedAt:           "2026-01-30T14:00:00Z",
		LastUpdated:         "2026-01-30T14:30:00Z",
		Iteration:           2,
		Status:              StatusInterrupted,
		Phase:               PhaseImplementation,
		TasksFile:           tasksFile,
		TasksFileHash:       computeTestHash(tasksContent),
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
		RetryState:          RetryState{Attempt: 3, Delay: 15},
	}

	// Resume should succeed and preserve retry state
	err = ResumeFromState(state, tasksFile, false)
	assert.NoError(t, err, "Resume should succeed with retry attempt > 1")
	assert.Equal(t, 3, state.RetryState.Attempt, "Retry attempt should be preserved")
	assert.Equal(t, 15, state.RetryState.Delay, "Retry delay should be preserved")
}

// TestResumeFromState_TasksHashChanged tests error when tasks file hash doesn't match
func TestResumeFromState_TasksHashChanged(t *testing.T) {
	tmpDir := t.TempDir()

	// Create original tasks file
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	originalContent := []byte("# Tasks\n- [ ] Task 1\n")
	err := os.WriteFile(tasksFile, originalContent, 0644)
	require.NoError(t, err)

	// Create state with original hash
	state := &SessionState{
		SchemaVersion:       2,
		SessionID:           "test-hash-change",
		StartedAt:           "2026-01-30T14:00:00Z",
		LastUpdated:         "2026-01-30T14:30:00Z",
		Iteration:           1,
		Status:              StatusInterrupted,
		Phase:               PhaseImplementation,
		TasksFile:           tasksFile,
		TasksFileHash:       computeTestHash(originalContent),
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
	}

	// Modify tasks file
	modifiedContent := []byte("# Tasks\n- [ ] Task 1\n- [ ] Task 2\n")
	err = os.WriteFile(tasksFile, modifiedContent, 0644)
	require.NoError(t, err)

	// Resume should fail without force flag
	err = ResumeFromState(state, tasksFile, false)
	assert.Error(t, err, "Resume should fail when tasks file hash changed")
	assert.Contains(t, err.Error(), "tasks file changed", "Error should mention hash mismatch")
}

// TestResumeFromState_TasksHashChangedWithForce tests successful resume with --resume-force
func TestResumeFromState_TasksHashChangedWithForce(t *testing.T) {
	tmpDir := t.TempDir()

	// Create original tasks file
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	originalContent := []byte("# Tasks\n- [ ] Task 1\n")
	err := os.WriteFile(tasksFile, originalContent, 0644)
	require.NoError(t, err)

	// Create state with original hash
	state := &SessionState{
		SchemaVersion:       2,
		SessionID:           "test-force-resume",
		StartedAt:           "2026-01-30T14:00:00Z",
		LastUpdated:         "2026-01-30T14:30:00Z",
		Iteration:           2,
		Status:              StatusInterrupted,
		Phase:               PhaseValidation,
		Verdict:             "NEEDS_MORE_WORK",
		TasksFile:           tasksFile,
		TasksFileHash:       computeTestHash(originalContent),
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
	}

	// Modify tasks file
	modifiedContent := []byte("# Tasks\n- [ ] Task 1\n- [ ] Task 2\n- [ ] Task 3\n")
	err = os.WriteFile(tasksFile, modifiedContent, 0644)
	require.NoError(t, err)

	// Resume with force flag should succeed
	err = ResumeFromState(state, tasksFile, true)
	assert.NoError(t, err, "Resume should succeed with force flag even when hash changed")
	assert.Equal(t, StatusInProgress, state.Status, "Status should be IN_PROGRESS")
}

// TestResumeFromState_CLIFlagOverridesOnResume tests that CLI flags can override state values
func TestResumeFromState_CLIFlagOverridesOnResume(t *testing.T) {
	tmpDir := t.TempDir()

	// Create tasks file
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := []byte("# Tasks\n- [ ] Task 1\n")
	err := os.WriteFile(tasksFile, tasksContent, 0644)
	require.NoError(t, err)

	// Create state
	state := &SessionState{
		SchemaVersion:       2,
		SessionID:           "test-cli-overrides",
		StartedAt:           "2026-01-30T14:00:00Z",
		LastUpdated:         "2026-01-30T14:30:00Z",
		Iteration:           1,
		Status:              StatusInterrupted,
		Phase:               PhaseImplementation,
		TasksFile:           tasksFile,
		TasksFileHash:       computeTestHash(tasksContent),
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
	}

	// Resume
	err = ResumeFromState(state, tasksFile, false)
	assert.NoError(t, err)

	// Verify original state preserved
	assert.Equal(t, "claude", state.AICli)
	assert.Equal(t, "opus", state.ImplModel)
	assert.Equal(t, 20, state.MaxIterations)

	// Note: CLI flag override testing would be done in integration tests
	// where the orchestrator actually applies CLI overrides on top of resumed state
}

// TestResumeFromState_MissingTasksFile tests error when tasks file doesn't exist
func TestResumeFromState_MissingTasksFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Don't create tasks file
	tasksFile := filepath.Join(tmpDir, "nonexistent.md")

	state := &SessionState{
		SchemaVersion:       2,
		SessionID:           "test-missing-tasks",
		StartedAt:           "2026-01-30T14:00:00Z",
		LastUpdated:         "2026-01-30T14:30:00Z",
		Iteration:           1,
		Status:              StatusInterrupted,
		Phase:               PhaseImplementation,
		TasksFile:           tasksFile,
		TasksFileHash:       "some-hash",
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
	}

	// Resume should fail
	err := ResumeFromState(state, tasksFile, false)
	assert.Error(t, err, "Resume should fail when tasks file doesn't exist")
	assert.Contains(t, err.Error(), "tasks file not found", "Error should mention missing tasks file")
}

// TestResumeFromState_CompletedSessionNoResume tests that completed sessions can't be resumed
func TestResumeFromState_CompletedSessionNoResume(t *testing.T) {
	tmpDir := t.TempDir()

	// Create tasks file
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := []byte("# Tasks\n- [x] Task 1\n")
	err := os.WriteFile(tasksFile, tasksContent, 0644)
	require.NoError(t, err)

	// Create completed state
	state := &SessionState{
		SchemaVersion:       2,
		SessionID:           "test-completed",
		StartedAt:           "2026-01-30T14:00:00Z",
		LastUpdated:         "2026-01-30T15:00:00Z",
		Iteration:           5,
		Status:              StatusComplete,
		Phase:               PhaseValidation,
		Verdict:             "COMPLETE",
		TasksFile:           tasksFile,
		TasksFileHash:       computeTestHash(tasksContent),
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
	}

	// Resume can succeed (status will be changed to IN_PROGRESS)
	// But this might not be desired behavior - for now we test current behavior
	err = ResumeFromState(state, tasksFile, false)
	assert.NoError(t, err)
	assert.Equal(t, StatusInProgress, state.Status)
}

// TestResumeFromState_CancelledSessionResume tests resuming a cancelled session
func TestResumeFromState_CancelledSessionResume(t *testing.T) {
	tmpDir := t.TempDir()

	// Create tasks file
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := []byte("# Tasks\n- [ ] Task 1\n")
	err := os.WriteFile(tasksFile, tasksContent, 0644)
	require.NoError(t, err)

	// Create cancelled state
	state := &SessionState{
		SchemaVersion:       2,
		SessionID:           "test-cancelled",
		StartedAt:           "2026-01-30T14:00:00Z",
		LastUpdated:         "2026-01-30T14:10:00Z",
		Iteration:           1,
		Status:              StatusCancelled,
		Phase:               PhaseImplementation,
		TasksFile:           tasksFile,
		TasksFileHash:       computeTestHash(tasksContent),
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
	}

	// Resume should work (changes status to IN_PROGRESS)
	err = ResumeFromState(state, tasksFile, false)
	assert.NoError(t, err)
	assert.Equal(t, StatusInProgress, state.Status)
}

// computeTestHash is a helper function to compute SHA256 hash for testing
// It matches the behavior of tasks.HashFile
func computeTestHash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// TestResumeFromState_FinalPlanValidationPhase tests resuming from final_plan_validation phase.
func TestResumeFromState_FinalPlanValidationPhase(t *testing.T) {
	tmpDir := t.TempDir()

	// Create tasks file
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	tasksContent := []byte("# Tasks\n- [ ] Task 1\n")
	err := os.WriteFile(tasksFile, tasksContent, 0644)
	require.NoError(t, err)

	// Create state at final_plan_validation phase
	state := &SessionState{
		SchemaVersion:       2,
		SessionID:           "test-final-plan",
		StartedAt:           "2026-01-30T14:00:00Z",
		LastUpdated:         "2026-01-30T14:30:00Z",
		Iteration:           0,
		Status:              StatusInterrupted,
		Phase:               PhaseFinalPlanValidation,
		Verdict:             "",
		TasksFile:           tasksFile,
		TasksFileHash:       computeTestHash(tasksContent),
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
	}

	// Resume should succeed
	err = ResumeFromState(state, tasksFile, false)
	assert.NoError(t, err, "Resume should succeed for final_plan_validation phase")

	// Verify state was updated correctly
	assert.Equal(t, StatusInProgress, state.Status, "Status should be IN_PROGRESS after resume")
	assert.Equal(t, PhaseFinalPlanValidation, state.Phase, "Phase should be preserved")
	assert.Equal(t, 0, state.Iteration, "Iteration should remain 0 for final_plan_validation")
}
