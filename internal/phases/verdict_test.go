package phases

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CodexForgeBR/cli-tools/internal/exitcode"
)

// TestProcessVerdict_AllTransitions uses table-driven tests to verify all verdict state transitions
func TestProcessVerdict_AllTransitions(t *testing.T) {
	tests := []struct {
		name                 string
		input                VerdictInput
		expectedAction       string
		expectedExitCode     int
		expectedFeedback     string
		expectedInadmissible int
		description          string
	}{
		// COMPLETE verdict transitions
		{
			name: "COMPLETE with zero unchecked tasks exits success",
			input: VerdictInput{
				Verdict:           "COMPLETE",
				Feedback:          "All tasks done",
				Remaining:         0,
				BlockedCount:      0,
				BlockedTasks:      []string{},
				InadmissibleCount: 0,
				MaxInadmissible:   5,
			},
			expectedAction:       "exit",
			expectedExitCode:     exitcode.Success,
			expectedFeedback:     "",
			expectedInadmissible: 0,
			description:          "COMPLETE + 0 unchecked → exit 0",
		},
		{
			name: "COMPLETE with doable unchecked tasks overrides to NEEDS_MORE_WORK",
			input: VerdictInput{
				Verdict:           "COMPLETE",
				Feedback:          "All done but wait...",
				Remaining:         5,
				BlockedCount:      2,
				BlockedTasks:      []string{"Task A", "Task B"},
				InadmissibleCount: 0,
				MaxInadmissible:   5,
			},
			expectedAction:       "continue",
			expectedExitCode:     0,
			expectedFeedback:     "Validation marked complete but 5 tasks remain unchecked. Continuing implementation.",
			expectedInadmissible: 0,
			description:          "COMPLETE + doable unchecked (unchecked > 0 AND blocked < unchecked) → override to NEEDS_MORE_WORK",
		},
		{
			name: "COMPLETE with all tasks blocked exits blocked",
			input: VerdictInput{
				Verdict:           "COMPLETE",
				Feedback:          "Complete but everything blocked",
				Remaining:         3,
				BlockedCount:      3,
				BlockedTasks:      []string{"Task X", "Task Y", "Task Z"},
				InadmissibleCount: 0,
				MaxInadmissible:   5,
			},
			expectedAction:       "exit",
			expectedExitCode:     exitcode.Blocked,
			expectedFeedback:     "",
			expectedInadmissible: 0,
			description:          "COMPLETE + all blocked (blocked >= unchecked) → exit 4 (Blocked)",
		},
		{
			name: "COMPLETE with more blocked than unchecked exits blocked",
			input: VerdictInput{
				Verdict:           "COMPLETE",
				Feedback:          "Done",
				Remaining:         2,
				BlockedCount:      5,
				BlockedTasks:      []string{"T1", "T2", "T3", "T4", "T5"},
				InadmissibleCount: 0,
				MaxInadmissible:   5,
			},
			expectedAction:       "exit",
			expectedExitCode:     exitcode.Blocked,
			expectedFeedback:     "",
			expectedInadmissible: 0,
			description:          "COMPLETE + blocked > unchecked → exit 4 (Blocked)",
		},

		// NEEDS_MORE_WORK verdict
		{
			name: "NEEDS_MORE_WORK returns feedback and continues",
			input: VerdictInput{
				Verdict:           "NEEDS_MORE_WORK",
				Feedback:          "Fix the authentication logic",
				Remaining:         8,
				BlockedCount:      0,
				BlockedTasks:      []string{},
				InadmissibleCount: 2,
				MaxInadmissible:   5,
			},
			expectedAction:       "continue",
			expectedExitCode:     0,
			expectedFeedback:     "Fix the authentication logic",
			expectedInadmissible: 2,
			description:          "NEEDS_MORE_WORK → returns feedback + continue signal",
		},

		// ESCALATE verdict
		{
			name: "ESCALATE exits with escalate code",
			input: VerdictInput{
				Verdict:           "ESCALATE",
				Feedback:          "Need human review for security concerns",
				Remaining:         5,
				BlockedCount:      0,
				BlockedTasks:      []string{},
				InadmissibleCount: 1,
				MaxInadmissible:   5,
			},
			expectedAction:       "exit",
			expectedExitCode:     exitcode.Escalate,
			expectedFeedback:     "",
			expectedInadmissible: 1,
			description:          "ESCALATE → exit 3",
		},

		// INADMISSIBLE verdict under threshold
		{
			name: "INADMISSIBLE under threshold increments count and continues",
			input: VerdictInput{
				Verdict:           "INADMISSIBLE",
				Feedback:          "Output format is incorrect",
				Remaining:         10,
				BlockedCount:      0,
				BlockedTasks:      []string{},
				InadmissibleCount: 2,
				MaxInadmissible:   5,
			},
			expectedAction:       "continue",
			expectedExitCode:     0,
			expectedFeedback:     "Output format is incorrect",
			expectedInadmissible: 3,
			description:          "INADMISSIBLE under threshold → increment count + continue",
		},
		{
			name: "INADMISSIBLE at threshold minus one increments and continues",
			input: VerdictInput{
				Verdict:           "INADMISSIBLE",
				Feedback:          "Still wrong format",
				Remaining:         10,
				BlockedCount:      0,
				BlockedTasks:      []string{},
				InadmissibleCount: 4,
				MaxInadmissible:   5,
			},
			expectedAction:       "continue",
			expectedExitCode:     0,
			expectedFeedback:     "Still wrong format",
			expectedInadmissible: 5,
			description:          "INADMISSIBLE at threshold-1 → increment count + continue",
		},

		// INADMISSIBLE verdict over threshold
		{
			name: "INADMISSIBLE at threshold exits inadmissible",
			input: VerdictInput{
				Verdict:           "INADMISSIBLE",
				Feedback:          "Exceeded max violations",
				Remaining:         10,
				BlockedCount:      0,
				BlockedTasks:      []string{},
				InadmissibleCount: 5,
				MaxInadmissible:   5,
			},
			expectedAction:       "exit",
			expectedExitCode:     exitcode.Inadmissible,
			expectedFeedback:     "",
			expectedInadmissible: 6,
			description:          "INADMISSIBLE at threshold → exit 6",
		},
		{
			name: "INADMISSIBLE over threshold exits inadmissible",
			input: VerdictInput{
				Verdict:           "INADMISSIBLE",
				Feedback:          "Too many violations",
				Remaining:         10,
				BlockedCount:      0,
				BlockedTasks:      []string{},
				InadmissibleCount: 10,
				MaxInadmissible:   5,
			},
			expectedAction:       "exit",
			expectedExitCode:     exitcode.Inadmissible,
			expectedFeedback:     "",
			expectedInadmissible: 11,
			description:          "INADMISSIBLE over threshold → exit 6",
		},

		// BLOCKED verdict with partial blocking
		{
			name: "BLOCKED with some doable tasks continues with doable",
			input: VerdictInput{
				Verdict:           "BLOCKED",
				Feedback:          "Some tasks blocked",
				Remaining:         10,
				BlockedCount:      3,
				BlockedTasks:      []string{"API key needed", "Design pending", "Review required"},
				InadmissibleCount: 0,
				MaxInadmissible:   5,
			},
			expectedAction:       "continue",
			expectedExitCode:     0,
			expectedFeedback:     "Some tasks blocked",
			expectedInadmissible: 0,
			description:          "BLOCKED partial (some doable) → continue with doable",
		},
		{
			name: "BLOCKED with exactly one doable task continues",
			input: VerdictInput{
				Verdict:           "BLOCKED",
				Feedback:          "Nearly all blocked",
				Remaining:         5,
				BlockedCount:      4,
				BlockedTasks:      []string{"T1", "T2", "T3", "T4"},
				InadmissibleCount: 0,
				MaxInadmissible:   5,
			},
			expectedAction:       "continue",
			expectedExitCode:     0,
			expectedFeedback:     "Nearly all blocked",
			expectedInadmissible: 0,
			description:          "BLOCKED with one doable → continue",
		},

		// BLOCKED verdict with full blocking
		{
			name: "BLOCKED with all tasks blocked exits blocked",
			input: VerdictInput{
				Verdict:           "BLOCKED",
				Feedback:          "Everything is blocked",
				Remaining:         5,
				BlockedCount:      5,
				BlockedTasks:      []string{"B1", "B2", "B3", "B4", "B5"},
				InadmissibleCount: 0,
				MaxInadmissible:   5,
			},
			expectedAction:       "exit",
			expectedExitCode:     exitcode.Blocked,
			expectedFeedback:     "",
			expectedInadmissible: 0,
			description:          "BLOCKED full (all blocked) → exit 4",
		},
		{
			name: "BLOCKED with more blocked than unchecked exits blocked",
			input: VerdictInput{
				Verdict:           "BLOCKED",
				Feedback:          "Overblocked",
				Remaining:         3,
				BlockedCount:      8,
				BlockedTasks:      []string{"X1", "X2", "X3", "X4", "X5", "X6", "X7", "X8"},
				InadmissibleCount: 0,
				MaxInadmissible:   5,
			},
			expectedAction:       "exit",
			expectedExitCode:     exitcode.Blocked,
			expectedFeedback:     "",
			expectedInadmissible: 0,
			description:          "BLOCKED with blocked > unchecked → exit 4",
		},

		// Unknown verdict
		{
			name: "Unknown verdict falls back to error",
			input: VerdictInput{
				Verdict:           "UNKNOWN_STATE",
				Feedback:          "Something went wrong",
				Remaining:         5,
				BlockedCount:      0,
				BlockedTasks:      []string{},
				InadmissibleCount: 0,
				MaxInadmissible:   5,
			},
			expectedAction:       "exit",
			expectedExitCode:     exitcode.Error,
			expectedFeedback:     "",
			expectedInadmissible: 0,
			description:          "Unknown verdict → fallback to error",
		},
		{
			name: "Empty verdict string falls back to error",
			input: VerdictInput{
				Verdict:           "",
				Feedback:          "Empty verdict",
				Remaining:         5,
				BlockedCount:      0,
				BlockedTasks:      []string{},
				InadmissibleCount: 0,
				MaxInadmissible:   5,
			},
			expectedAction:       "exit",
			expectedExitCode:     exitcode.Error,
			expectedFeedback:     "",
			expectedInadmissible: 0,
			description:          "Empty verdict → fallback to error",
		},

		// Edge cases
		{
			name: "COMPLETE with zero unchecked and zero blocked exits success",
			input: VerdictInput{
				Verdict:           "COMPLETE",
				Feedback:          "Perfect completion",
				Remaining:         0,
				BlockedCount:      0,
				BlockedTasks:      []string{},
				InadmissibleCount: 0,
				MaxInadmissible:   5,
			},
			expectedAction:       "exit",
			expectedExitCode:     exitcode.Success,
			expectedFeedback:     "",
			expectedInadmissible: 0,
			description:          "Perfect completion scenario",
		},
		{
			name: "NEEDS_MORE_WORK with empty feedback continues with empty string",
			input: VerdictInput{
				Verdict:           "NEEDS_MORE_WORK",
				Feedback:          "",
				Remaining:         5,
				BlockedCount:      0,
				BlockedTasks:      []string{},
				InadmissibleCount: 0,
				MaxInadmissible:   5,
			},
			expectedAction:       "continue",
			expectedExitCode:     0,
			expectedFeedback:     "",
			expectedInadmissible: 0,
			description:          "NEEDS_MORE_WORK with no feedback",
		},
		{
			name: "INADMISSIBLE with count zero under threshold",
			input: VerdictInput{
				Verdict:           "INADMISSIBLE",
				Feedback:          "First violation",
				Remaining:         10,
				BlockedCount:      0,
				BlockedTasks:      []string{},
				InadmissibleCount: 0,
				MaxInadmissible:   5,
			},
			expectedAction:       "continue",
			expectedExitCode:     0,
			expectedFeedback:     "First violation",
			expectedInadmissible: 1,
			description:          "First inadmissible violation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessVerdict(tt.input)

			assert.Equal(t, tt.expectedAction, result.Action,
				"action mismatch: %s", tt.description)
			assert.Equal(t, tt.expectedExitCode, result.ExitCode,
				"exit code mismatch: %s", tt.description)
			assert.Equal(t, tt.expectedFeedback, result.Feedback,
				"feedback mismatch: %s", tt.description)
			assert.Equal(t, tt.expectedInadmissible, result.NewInadmissibleCount,
				"inadmissible count mismatch: %s", tt.description)
		})
	}
}

// TestProcessVerdict_InadmissibleCountProgression verifies inadmissible counter increments correctly
func TestProcessVerdict_InadmissibleCountProgression(t *testing.T) {
	// Simulate multiple INADMISSIBLE verdicts in sequence
	count := 0
	maxInadmissible := 3

	for i := 1; i <= 5; i++ {
		input := VerdictInput{
			Verdict:           "INADMISSIBLE",
			Feedback:          "Violation",
			Remaining:         10,
			BlockedCount:      0,
			BlockedTasks:      []string{},
			InadmissibleCount: count,
			MaxInadmissible:   maxInadmissible,
		}

		result := ProcessVerdict(input)

		if i <= maxInadmissible {
			// Should continue and increment
			assert.Equal(t, "continue", result.Action, "iteration %d should continue", i)
			assert.Equal(t, 0, result.ExitCode, "iteration %d exit code should be 0", i)
			assert.Equal(t, count+1, result.NewInadmissibleCount,
				"iteration %d should increment count from %d to %d", i, count, count+1)
		} else {
			// Should exit at threshold
			assert.Equal(t, "exit", result.Action, "iteration %d should exit", i)
			assert.Equal(t, exitcode.Inadmissible, result.ExitCode,
				"iteration %d should exit with inadmissible code", i)
		}

		count = result.NewInadmissibleCount
	}
}

// TestProcessVerdict_BlockedCountThresholds verifies blocked task threshold logic
func TestProcessVerdict_BlockedCountThresholds(t *testing.T) {
	tests := []struct {
		name             string
		remaining        int
		blockedCount     int
		expectedContinue bool
	}{
		{"no blocked tasks", 10, 0, true},
		{"few blocked tasks", 10, 3, true},
		{"half blocked", 10, 5, true},
		{"one doable", 10, 9, true},
		{"all blocked", 10, 10, false},
		{"more blocked than unchecked", 5, 8, false},
		{"edge: exactly all blocked", 1, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := VerdictInput{
				Verdict:           "BLOCKED",
				Feedback:          "Test",
				Remaining:         tt.remaining,
				BlockedCount:      tt.blockedCount,
				BlockedTasks:      make([]string, tt.blockedCount),
				InadmissibleCount: 0,
				MaxInadmissible:   5,
			}

			result := ProcessVerdict(input)

			if tt.expectedContinue {
				assert.Equal(t, "continue", result.Action,
					"with %d remaining and %d blocked should continue", tt.remaining, tt.blockedCount)
			} else {
				assert.Equal(t, "exit", result.Action,
					"with %d remaining and %d blocked should exit", tt.remaining, tt.blockedCount)
				assert.Equal(t, exitcode.Blocked, result.ExitCode,
					"should exit with blocked code")
			}
		})
	}
}

// TestProcessVerdict_FeedbackPreservation verifies feedback is preserved correctly
func TestProcessVerdict_FeedbackPreservation(t *testing.T) {
	testFeedback := "This is important feedback with special chars: @#$%^&*()"

	continueVerdicts := []string{"NEEDS_MORE_WORK", "INADMISSIBLE", "BLOCKED"}

	for _, verdict := range continueVerdicts {
		t.Run(verdict, func(t *testing.T) {
			input := VerdictInput{
				Verdict:           verdict,
				Feedback:          testFeedback,
				Remaining:         10,
				BlockedCount:      1,
				BlockedTasks:      []string{"Task"},
				InadmissibleCount: 0,
				MaxInadmissible:   5,
			}

			result := ProcessVerdict(input)

			if result.Action == "continue" {
				assert.Equal(t, testFeedback, result.Feedback,
					"%s verdict should preserve feedback", verdict)
			}
		})
	}
}
