package notification

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatEvent(t *testing.T) {
	tests := []struct {
		name        string
		event       string
		projectName string
		sessionID   string
		iteration   int
		exitCode    int
		wantContain []string
	}{
		{
			name:        "completed event",
			event:       EventCompleted,
			projectName: "my-project",
			sessionID:   "session-123",
			iteration:   5,
			exitCode:    0,
			wantContain: []string{"✅", "my-project", "[session-123]", "completed successfully", "5 iterations", "exit 0"},
		},
		{
			name:        "max iterations event",
			event:       EventMaxIterations,
			projectName: "my-project",
			sessionID:   "session-456",
			iteration:   10,
			exitCode:    1,
			wantContain: []string{"⚠️", "my-project", "[session-456]", "max iterations", "(10)", "exit 1"},
		},
		{
			name:        "escalate event",
			event:       EventEscalate,
			projectName: "critical-app",
			sessionID:   "session-789",
			iteration:   3,
			exitCode:    2,
			wantContain: []string{"🚨", "critical-app", "[session-789]", "ESCALATION", "iteration 3", "exit 2"},
		},
		{
			name:        "blocked event",
			event:       EventBlocked,
			projectName: "blocked-proj",
			sessionID:   "session-abc",
			iteration:   7,
			exitCode:    3,
			wantContain: []string{"🔒", "blocked-proj", "[session-abc]", "all tasks blocked", "iteration 7", "exit 3"},
		},
		{
			name:        "tasks invalid event",
			event:       EventTasksInvalid,
			projectName: "invalid-tasks",
			sessionID:   "session-def",
			iteration:   0,
			exitCode:    4,
			wantContain: []string{"❌", "invalid-tasks", "[session-def]", "tasks validation failed", "exit 4"},
		},
		{
			name:        "inadmissible event",
			event:       EventInadmissible,
			projectName: "threshold-proj",
			sessionID:   "session-ghi",
			iteration:   12,
			exitCode:    5,
			wantContain: []string{"🚫", "threshold-proj", "[session-ghi]", "inadmissible threshold", "iteration 12", "exit 5"},
		},
		{
			name:        "interrupted event",
			event:       EventInterrupted,
			projectName: "paused-proj",
			sessionID:   "session-jkl",
			iteration:   8,
			exitCode:    130,
			wantContain: []string{"⏸️", "paused-proj", "[session-jkl]", "interrupted", "iteration 8", "--resume", "exit 130"},
		},
		{
			name:        "unknown event",
			event:       "unknown_event",
			projectName: "test-proj",
			sessionID:   "session-xyz",
			iteration:   1,
			exitCode:    99,
			wantContain: []string{"ℹ️", "test-proj", "[session-xyz]", "event: unknown_event", "exit 99"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatEvent(tt.event, tt.projectName, tt.sessionID, tt.iteration, tt.exitCode)

			// Check that all expected substrings are present
			for _, want := range tt.wantContain {
				assert.Contains(t, result, want, "message should contain %q", want)
			}
		})
	}
}

func TestFormatEvent_RateLimitedEvent(t *testing.T) {
	result := FormatEvent(EventRateLimited, "my-project", "session-123", 8, 0)

	assert.Contains(t, result, "⏳")
	assert.Contains(t, result, "my-project")
	assert.Contains(t, result, "[session-123]")
	assert.Contains(t, result, "rate limit")
	assert.Contains(t, result, "iteration 8")
	assert.Contains(t, result, "waiting for reset")
	// Note: rate_limited event doesn't include exit code in the format
}

func TestFormatEvent_AllEventsIncludeRequiredFields(t *testing.T) {
	// Test that every event includes project name, session ID, and exit code
	events := []string{
		EventCompleted,
		EventMaxIterations,
		EventEscalate,
		EventBlocked,
		EventTasksInvalid,
		EventInadmissible,
		EventInterrupted,
	}

	projectName := "test-project"
	sessionID := "test-session-123"
	exitCode := 42

	for _, event := range events {
		t.Run(event, func(t *testing.T) {
			result := FormatEvent(event, projectName, sessionID, 5, exitCode)

			assert.Contains(t, result, projectName, "should include project name")
			assert.Contains(t, result, sessionID, "should include session ID")
			assert.Contains(t, result, "exit 42", "should include exit code")

			// Events that should include iteration (all except tasks_invalid)
			if event != EventTasksInvalid {
				assert.True(t,
					strings.Contains(result, "iteration") || strings.Contains(result, "iterations"),
					"should include iteration info for event %s", event)
			}
		})
	}
}

func TestEventConstants(t *testing.T) {
	// Verify event constant values match expected strings
	assert.Equal(t, "completed", EventCompleted)
	assert.Equal(t, "max_iterations", EventMaxIterations)
	assert.Equal(t, "escalate", EventEscalate)
	assert.Equal(t, "blocked", EventBlocked)
	assert.Equal(t, "tasks_invalid", EventTasksInvalid)
	assert.Equal(t, "inadmissible", EventInadmissible)
	assert.Equal(t, "interrupted", EventInterrupted)
}
