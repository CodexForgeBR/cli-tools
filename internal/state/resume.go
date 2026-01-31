package state

import (
	"fmt"
)

// ResumeFromState prepares an existing session state for resumption.
//
// It validates that the tasks file still exists and matches the recorded hash
// (unless force is true), then updates the session status to IN_PROGRESS to
// allow the orchestrator to continue from where it left off.
//
// Phase-aware continuation logic:
//   - cross_validation: Resume at cross-validation phase (skips impl+val)
//   - validation: Resume at validation phase (skips impl)
//   - implementation: Resume at implementation phase (full iteration restart)
//   - waiting_for_schedule: Resume schedule wait (checks if time has passed)
//
// Retry state is preserved across resume - if attempt > 1, the orchestrator
// will continue with the existing retry count and delay.
//
// Returns an error if:
//   - The tasks file doesn't exist
//   - The tasks file hash has changed (when force=false)
//   - Hash computation fails
func ResumeFromState(existing *SessionState, tasksFile string, force bool) error {
	// Validate state consistency unless force flag is set
	if !force {
		if err := ValidateState(existing, tasksFile); err != nil {
			return fmt.Errorf("state validation failed: %w", err)
		}
	}

	// Phase-aware continuation: The phase is already set in the state,
	// so we just need to mark the session as IN_PROGRESS. The orchestrator
	// will use the Phase field to determine where to continue.

	// Update status to allow continuation
	existing.Status = StatusInProgress

	return nil
}
