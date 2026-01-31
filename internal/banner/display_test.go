package banner

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStderr captures stderr output during function execution
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	// Save original stderr
	old := os.Stderr
	defer func() { os.Stderr = old }()

	// Create pipe
	r, w, err := os.Pipe()
	require.NoError(t, err)

	// Replace stderr
	os.Stderr = w

	// Create channel for output
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	// Execute function
	fn()

	// Close writer and restore stderr
	w.Close()
	os.Stderr = old

	// Get captured output
	output := <-outC
	return output
}

// TestPrintStartupBanner verifies startup banner includes all required information
func TestPrintStartupBanner(t *testing.T) {
	tests := []struct {
		name         string
		sessionID    string
		ai           string
		model        string
		tasksFile    string
		expectedText []string
	}{
		{
			name:      "standard configuration",
			sessionID: "sess-12345",
			ai:        "claude",
			model:     "opus",
			tasksFile: "tasks.md",
			expectedText: []string{
				"ralph-loop",
				"sess-12345",
				"claude",
				"opus",
				"tasks.md",
			},
		},
		{
			name:      "openai configuration",
			sessionID: "sess-67890",
			ai:        "openai",
			model:     "gpt-4",
			tasksFile: ".ralph-loop/tasks.md",
			expectedText: []string{
				"ralph-loop",
				"sess-67890",
				"openai",
				"gpt-4",
				".ralph-loop/tasks.md",
			},
		},
		{
			name:      "long session ID",
			sessionID: "session-2024-01-30-very-long-identifier",
			ai:        "gemini",
			model:     "pro",
			tasksFile: "project-tasks.md",
			expectedText: []string{
				"ralph-loop",
				"session-2024-01-30-very-long-identifier",
				"gemini",
				"pro",
				"project-tasks.md",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(t, func() {
				PrintStartupBanner(tt.sessionID, tt.ai, tt.model, tt.tasksFile)
			})

			// Verify all expected text appears in output
			for _, expected := range tt.expectedText {
				assert.Contains(t, output, expected,
					"startup banner should contain %q", expected)
			}

			// Verify output is not empty
			assert.NotEmpty(t, output, "startup banner should not be empty")
		})
	}
}

// TestPrintStartupBanner_ProjectName verifies project name appears prominently
func TestPrintStartupBanner_ProjectName(t *testing.T) {
	output := captureStderr(t, func() {
		PrintStartupBanner("test-session", "claude", "opus", "tasks.md")
	})

	// Project name should appear (case-insensitive check)
	lowerOutput := strings.ToLower(output)
	assert.True(t,
		strings.Contains(lowerOutput, "ralph") || strings.Contains(lowerOutput, "loop"),
		"startup banner should contain project name ralph-loop")
}

// TestPrintCompletionBanner verifies completion banner includes iteration count and duration
func TestPrintCompletionBanner(t *testing.T) {
	tests := []struct {
		name         string
		iterations   int
		durationSecs int
		checkFunc    func(t *testing.T, output string)
	}{
		{
			name:         "single iteration",
			iterations:   1,
			durationSecs: 30,
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "1", "should show 1 iteration")
				assert.Contains(t, output, "30", "should show 30 seconds")
			},
		},
		{
			name:         "multiple iterations",
			iterations:   15,
			durationSecs: 450,
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "15", "should show 15 iterations")
				assert.Contains(t, output, "450", "should show 450 seconds")
			},
		},
		{
			name:         "max iterations",
			iterations:   20,
			durationSecs: 1800,
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "20", "should show 20 iterations")
				assert.Contains(t, output, "1800", "should show 1800 seconds")
			},
		},
		{
			name:         "short duration",
			iterations:   3,
			durationSecs: 5,
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "3", "should show 3 iterations")
				assert.Contains(t, output, "5", "should show 5 seconds")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(t, func() {
				PrintCompletionBanner(tt.iterations, tt.durationSecs)
			})

			assert.NotEmpty(t, output, "completion banner should not be empty")
			tt.checkFunc(t, output)

			// Should contain success/completion indicator
			lowerOutput := strings.ToLower(output)
			hasSuccessIndicator := strings.Contains(lowerOutput, "complete") ||
				strings.Contains(lowerOutput, "success") ||
				strings.Contains(lowerOutput, "done") ||
				strings.Contains(lowerOutput, "finish")
			assert.True(t, hasSuccessIndicator, "completion banner should indicate success")
		})
	}
}

// TestPrintEscalationBanner verifies escalation banner shows escalation message
func TestPrintEscalationBanner(t *testing.T) {
	tests := []struct {
		name     string
		feedback string
	}{
		{
			name:     "simple escalation",
			feedback: "Need human review for security concerns",
		},
		{
			name:     "detailed escalation",
			feedback: "The implementation requires architectural decision that is beyond my scope. Please review the proposed changes to the database schema.",
		},
		{
			name:     "short escalation",
			feedback: "Help needed",
		},
		{
			name: "multiline escalation",
			feedback: `This task requires:
1. Access to production credentials
2. Manual verification of external API
3. Human judgment on business logic`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(t, func() {
				PrintEscalationBanner(tt.feedback)
			})

			assert.NotEmpty(t, output, "escalation banner should not be empty")
			assert.Contains(t, output, tt.feedback, "escalation banner should contain feedback")

			// Should contain escalation indicator
			lowerOutput := strings.ToLower(output)
			hasEscalationIndicator := strings.Contains(lowerOutput, "escalat") ||
				strings.Contains(lowerOutput, "human") ||
				strings.Contains(lowerOutput, "review") ||
				strings.Contains(lowerOutput, "assistance")
			assert.True(t, hasEscalationIndicator, "escalation banner should indicate escalation")
		})
	}
}

// TestPrintEscalationBanner_EmptyFeedback verifies handling of empty feedback
func TestPrintEscalationBanner_EmptyFeedback(t *testing.T) {
	output := captureStderr(t, func() {
		PrintEscalationBanner("")
	})

	// Should still print banner even with empty feedback
	assert.NotEmpty(t, output, "escalation banner should not be empty even with empty feedback")

	lowerOutput := strings.ToLower(output)
	hasEscalationIndicator := strings.Contains(lowerOutput, "escalat")
	assert.True(t, hasEscalationIndicator, "should indicate escalation even without feedback")
}

// TestPrintBlockedBanner verifies blocked banner shows blocked tasks
func TestPrintBlockedBanner(t *testing.T) {
	tests := []struct {
		name         string
		blockedTasks []string
		checkFunc    func(t *testing.T, output string)
	}{
		{
			name:         "single blocked task",
			blockedTasks: []string{"Wait for API key from DevOps"},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "Wait for API key from DevOps")
			},
		},
		{
			name: "multiple blocked tasks",
			blockedTasks: []string{
				"Pending database migration approval",
				"Waiting for design mockups",
				"External API rate limit reached",
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "Pending database migration approval")
				assert.Contains(t, output, "Waiting for design mockups")
				assert.Contains(t, output, "External API rate limit reached")
			},
		},
		{
			name:         "many blocked tasks",
			blockedTasks: []string{"Task 1", "Task 2", "Task 3", "Task 4", "Task 5"},
			checkFunc: func(t *testing.T, output string) {
				for _, task := range []string{"Task 1", "Task 2", "Task 3", "Task 4", "Task 5"} {
					assert.Contains(t, output, task)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(t, func() {
				PrintBlockedBanner(tt.blockedTasks)
			})

			assert.NotEmpty(t, output, "blocked banner should not be empty")
			tt.checkFunc(t, output)

			// Should contain blocked indicator
			lowerOutput := strings.ToLower(output)
			hasBlockedIndicator := strings.Contains(lowerOutput, "block") ||
				strings.Contains(lowerOutput, "wait") ||
				strings.Contains(lowerOutput, "stuck")
			assert.True(t, hasBlockedIndicator, "blocked banner should indicate blocked state")
		})
	}
}

// TestPrintBlockedBanner_EmptyList verifies handling of empty blocked tasks list
func TestPrintBlockedBanner_EmptyList(t *testing.T) {
	output := captureStderr(t, func() {
		PrintBlockedBanner([]string{})
	})

	// Should still print banner even with no tasks
	assert.NotEmpty(t, output, "blocked banner should not be empty even with no tasks")

	lowerOutput := strings.ToLower(output)
	hasBlockedIndicator := strings.Contains(lowerOutput, "block")
	assert.True(t, hasBlockedIndicator, "should indicate blocked state even without tasks")
}

// TestPrintBlockedBanner_NilList verifies handling of nil blocked tasks list
func TestPrintBlockedBanner_NilList(t *testing.T) {
	output := captureStderr(t, func() {
		PrintBlockedBanner(nil)
	})

	// Should still print banner with nil list
	assert.NotEmpty(t, output, "blocked banner should not be empty even with nil tasks")
}

// TestPrintMaxIterationsBanner verifies max iterations banner shows counts
func TestPrintMaxIterationsBanner(t *testing.T) {
	tests := []struct {
		name          string
		iterations    int
		maxIterations int
		checkFunc     func(t *testing.T, output string)
	}{
		{
			name:          "reached exact limit",
			iterations:    100,
			maxIterations: 100,
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "100", "should show iteration count")
				assert.Contains(t, output, "100/100", "should show both counts")
			},
		},
		{
			name:          "small iteration limit",
			iterations:    5,
			maxIterations: 5,
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "5", "should show iteration count")
				assert.Contains(t, output, "5/5", "should show both counts")
			},
		},
		{
			name:          "large iteration limit",
			iterations:    1000,
			maxIterations: 1000,
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "1000", "should show iteration count")
				assert.Contains(t, output, "1000/1000", "should show both counts")
			},
		},
		{
			name:          "zero iterations",
			iterations:    0,
			maxIterations: 0,
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "0", "should show zero")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(t, func() {
				PrintMaxIterationsBanner(tt.iterations, tt.maxIterations)
			})

			assert.NotEmpty(t, output, "max iterations banner should not be empty")
			tt.checkFunc(t, output)

			// Should contain max/limit indicator
			lowerOutput := strings.ToLower(output)
			hasMaxIndicator := strings.Contains(lowerOutput, "max") ||
				strings.Contains(lowerOutput, "limit") ||
				strings.Contains(lowerOutput, "reached")
			assert.True(t, hasMaxIndicator, "max iterations banner should indicate limit reached")
		})
	}
}

// TestPrintInadmissibleBanner verifies inadmissible banner shows counts
func TestPrintInadmissibleBanner(t *testing.T) {
	tests := []struct {
		name      string
		count     int
		max       int
		checkFunc func(t *testing.T, output string)
	}{
		{
			name:  "threshold exceeded",
			count: 5,
			max:   5,
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "5", "should show count")
				assert.Contains(t, output, "5/5", "should show both counts")
			},
		},
		{
			name:  "small threshold",
			count: 3,
			max:   3,
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "3", "should show count")
				assert.Contains(t, output, "3/3", "should show both counts")
			},
		},
		{
			name:  "large threshold",
			count: 100,
			max:   100,
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "100", "should show count")
				assert.Contains(t, output, "100/100", "should show both counts")
			},
		},
		{
			name:  "zero threshold",
			count: 0,
			max:   0,
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "0", "should show zero")
			},
		},
		{
			name:  "exceeded by one",
			count: 6,
			max:   5,
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "6", "should show current count")
				assert.Contains(t, output, "5", "should show max")
				assert.Contains(t, output, "6/5", "should show exceeded ratio")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(t, func() {
				PrintInadmissibleBanner(tt.count, tt.max)
			})

			assert.NotEmpty(t, output, "inadmissible banner should not be empty")
			tt.checkFunc(t, output)

			// Should contain inadmissible/threshold indicator
			lowerOutput := strings.ToLower(output)
			hasThresholdIndicator := strings.Contains(lowerOutput, "inadmissible") ||
				strings.Contains(lowerOutput, "threshold") ||
				strings.Contains(lowerOutput, "exceed")
			assert.True(t, hasThresholdIndicator, "inadmissible banner should indicate threshold exceeded")
		})
	}
}

// TestPrintInterruptedBanner verifies interrupted banner shows iteration and phase
func TestPrintInterruptedBanner(t *testing.T) {
	tests := []struct {
		name      string
		iteration int
		phase     string
		checkFunc func(t *testing.T, output string)
	}{
		{
			name:      "validation phase",
			iteration: 3,
			phase:     "validation",
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "3", "should show iteration")
				assert.Contains(t, output, "validation", "should show phase")
			},
		},
		{
			name:      "implementation phase",
			iteration: 7,
			phase:     "implementation",
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "7", "should show iteration")
				assert.Contains(t, output, "implementation", "should show phase")
			},
		},
		{
			name:      "planning phase",
			iteration: 1,
			phase:     "planning",
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "1", "should show iteration")
				assert.Contains(t, output, "planning", "should show phase")
			},
		},
		{
			name:      "first iteration",
			iteration: 0,
			phase:     "startup",
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "0", "should show zero iteration")
				assert.Contains(t, output, "startup", "should show phase")
			},
		},
		{
			name:      "empty phase",
			iteration: 5,
			phase:     "",
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "5", "should show iteration even with empty phase")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(t, func() {
				PrintInterruptedBanner(tt.iteration, tt.phase)
			})

			assert.NotEmpty(t, output, "interrupted banner should not be empty")
			tt.checkFunc(t, output)

			// Should contain interrupt indicator
			lowerOutput := strings.ToLower(output)
			hasInterruptIndicator := strings.Contains(lowerOutput, "interrupt") ||
				strings.Contains(lowerOutput, "stopped") ||
				strings.Contains(lowerOutput, "paused")
			assert.True(t, hasInterruptIndicator, "interrupted banner should indicate interruption")

			// Should mention resume capability
			hasResumeInfo := strings.Contains(lowerOutput, "resume") ||
				strings.Contains(lowerOutput, "continue")
			assert.True(t, hasResumeInfo, "interrupted banner should mention resume capability")
		})
	}
}

// TestPrintStatusBanner verifies status banner displays all fields correctly
func TestPrintStatusBanner(t *testing.T) {
	tests := []struct {
		name      string
		info      StatusInfo
		checkFunc func(t *testing.T, output string)
	}{
		{
			name: "complete status info",
			info: StatusInfo{
				SessionID:         "sess-2026-01-30",
				Status:            "IN_PROGRESS",
				Phase:             "validation",
				Verdict:           "NEEDS_MORE_WORK",
				Iteration:         5,
				MaxIterations:     20,
				InadmissibleCount: 2,
				MaxInadmissible:   5,
				StartedAt:         "2026-01-30T10:00:00Z",
				LastUpdated:       "2026-01-30T10:30:00Z",
				AICli:             "claude",
				ImplModel:         "opus",
				ValModel:          "sonnet",
				CrossValEnabled:   true,
				CrossAI:           "openai",
				CrossModel:        "gpt-4",
				RetryAttempt:      2,
				RetryDelay:        10,
				LastFeedback:      "Tests are failing, please fix",
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "sess-2026-01-30", "should show session ID")
				assert.Contains(t, output, "IN_PROGRESS", "should show status")
				assert.Contains(t, output, "validation", "should show phase")
				assert.Contains(t, output, "NEEDS_MORE_WORK", "should show verdict")
				assert.Contains(t, output, "5/20", "should show iteration count")
				assert.Contains(t, output, "2/5", "should show inadmissible count")
				assert.Contains(t, output, "claude", "should show AI CLI")
				assert.Contains(t, output, "opus", "should show impl model")
				assert.Contains(t, output, "sonnet", "should show val model")
				assert.Contains(t, output, "openai", "should show cross-val AI")
				assert.Contains(t, output, "gpt-4", "should show cross-val model")
				assert.Contains(t, output, "2026-01-30T10:00:00Z", "should show started timestamp")
				assert.Contains(t, output, "2026-01-30T10:30:00Z", "should show updated timestamp")
				assert.Contains(t, output, "2", "should show retry attempt")
				assert.Contains(t, output, "10", "should show retry delay")
				assert.Contains(t, output, "Tests are failing", "should show feedback")
			},
		},
		{
			name: "minimal status info",
			info: StatusInfo{
				SessionID: "minimal-session",
				Status:    "RUNNING",
				Phase:     "impl",
				Verdict:   "PASS",
				Iteration: 1,
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "minimal-session", "should show session ID")
				assert.Contains(t, output, "RUNNING", "should show status")
				assert.Contains(t, output, "impl", "should show phase")
				assert.Contains(t, output, "PASS", "should show verdict")
				assert.Contains(t, output, "1", "should show iteration")
				// Should not show max iterations when not provided
				assert.NotContains(t, output, "0/0", "should not show zero max iterations")
			},
		},
		{
			name: "with max iterations but no inadmissible",
			info: StatusInfo{
				SessionID:     "test-session",
				Status:        "ACTIVE",
				Phase:         "planning",
				Verdict:       "ADMISSIBLE",
				Iteration:     3,
				MaxIterations: 10,
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "test-session", "should show session ID")
				assert.Contains(t, output, "3/10", "should show iteration with max")
			},
		},
		{
			name: "long feedback gets truncated",
			info: StatusInfo{
				SessionID:    "trunc-session",
				Status:       "IN_PROGRESS",
				Phase:        "validation",
				Verdict:      "NEEDS_WORK",
				Iteration:    1,
				LastFeedback: "This is a very long feedback message that exceeds the maximum allowed length and should be truncated to prevent cluttering the banner display output",
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "This is a very long feedback", "should show start of feedback")
				assert.Contains(t, output, "...", "should show truncation indicator")
				// Full feedback should not appear
				assert.NotContains(t, output, "cluttering the banner display output", "should truncate long feedback")
			},
		},
		{
			name: "cross-validation disabled",
			info: StatusInfo{
				SessionID:       "no-cross-val",
				Status:          "RUNNING",
				Phase:           "impl",
				Verdict:         "PASS",
				Iteration:       2,
				AICli:           "claude",
				ImplModel:       "opus",
				ValModel:        "opus",
				CrossValEnabled: false,
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "claude", "should show AI CLI")
				assert.Contains(t, output, "opus", "should show model")
				// Cross-val info should not appear when disabled
				lowerOutput := strings.ToLower(output)
				if strings.Contains(lowerOutput, "cross") {
					// If "cross" appears, it should not be followed by AI/model info
					assert.NotContains(t, output, "Cross-val:", "should not show cross-val section when disabled")
				}
			},
		},
		{
			name: "no retry information",
			info: StatusInfo{
				SessionID: "no-retry",
				Status:    "OK",
				Phase:     "done",
				Verdict:   "PASS",
				Iteration: 1,
			},
			checkFunc: func(t *testing.T, output string) {
				// Should not show retry section (looking for "Retry:" or "attempt")
				assert.NotContains(t, output, "Retry:", "should not show retry section when not retrying")
				lowerOutput := strings.ToLower(output)
				assert.NotContains(t, lowerOutput, "attempt", "should not show retry attempt when not retrying")
			},
		},
		{
			name: "with retry information",
			info: StatusInfo{
				SessionID:    "with-retry",
				Status:       "RETRYING",
				Phase:        "validation",
				Verdict:      "ERROR",
				Iteration:    3,
				RetryAttempt: 1,
				RetryDelay:   5,
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "1", "should show retry attempt")
				assert.Contains(t, output, "5", "should show retry delay")
				lowerOutput := strings.ToLower(output)
				assert.Contains(t, lowerOutput, "retry", "should mention retry")
			},
		},
		{
			name: "empty timestamps",
			info: StatusInfo{
				SessionID:   "no-timestamps",
				Status:      "NEW",
				Phase:       "init",
				Verdict:     "PENDING",
				Iteration:   0,
				StartedAt:   "",
				LastUpdated: "",
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "no-timestamps", "should show session ID")
				// Timestamps should not show when empty
				assert.NotContains(t, output, "Started:", "should not show empty started timestamp")
				assert.NotContains(t, output, "Updated:", "should not show empty updated timestamp")
			},
		},
		{
			name: "inadmissible count without max",
			info: StatusInfo{
				SessionID:         "inadm-no-max",
				Status:            "RUNNING",
				Phase:             "validation",
				Verdict:           "INADMISSIBLE",
				Iteration:         2,
				InadmissibleCount: 3,
				MaxInadmissible:   0,
			},
			checkFunc: func(t *testing.T, output string) {
				// Should show inadmissible info even when max is zero
				assert.Contains(t, output, "3", "should show inadmissible count")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(t, func() {
				PrintStatusBanner(tt.info)
			})

			assert.NotEmpty(t, output, "status banner should not be empty")
			tt.checkFunc(t, output)
		})
	}
}

// TestPrintStatusBanner_RequiredFields verifies required fields always appear
func TestPrintStatusBanner_RequiredFields(t *testing.T) {
	info := StatusInfo{
		SessionID: "required-test",
		Status:    "TEST",
		Phase:     "testing",
		Verdict:   "TEST_VERDICT",
		Iteration: 99,
	}

	output := captureStderr(t, func() {
		PrintStatusBanner(info)
	})

	// These fields should always appear
	assert.Contains(t, output, "required-test", "should always show session ID")
	assert.Contains(t, output, "TEST", "should always show status")
	assert.Contains(t, output, "testing", "should always show phase")
	assert.Contains(t, output, "TEST_VERDICT", "should always show verdict")
	assert.Contains(t, output, "99", "should always show iteration")
}

// TestBannerOutput_NoColorCodes verifies banners work without ANSI color codes in plain environments
func TestBannerOutput_NotEmpty(t *testing.T) {
	// All banner functions should produce non-empty output
	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "startup banner",
			fn: func() {
				PrintStartupBanner("test", "claude", "opus", "tasks.md")
			},
		},
		{
			name: "completion banner",
			fn: func() {
				PrintCompletionBanner(10, 300)
			},
		},
		{
			name: "escalation banner",
			fn: func() {
				PrintEscalationBanner("test feedback")
			},
		},
		{
			name: "blocked banner",
			fn: func() {
				PrintBlockedBanner([]string{"test task"})
			},
		},
		{
			name: "max iterations banner",
			fn: func() {
				PrintMaxIterationsBanner(100, 100)
			},
		},
		{
			name: "inadmissible banner",
			fn: func() {
				PrintInadmissibleBanner(5, 5)
			},
		},
		{
			name: "interrupted banner",
			fn: func() {
				PrintInterruptedBanner(3, "validation")
			},
		},
		{
			name: "status banner",
			fn: func() {
				PrintStatusBanner(StatusInfo{
					SessionID: "test",
					Status:    "RUNNING",
					Phase:     "impl",
					Verdict:   "PASS",
					Iteration: 1,
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(t, tt.fn)
			assert.NotEmpty(t, output, "%s should produce output", tt.name)
			assert.Greater(t, len(output), 10, "%s should produce substantial output", tt.name)
		})
	}
}
