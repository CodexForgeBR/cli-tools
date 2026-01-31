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

// captureStdout captures stdout output during function execution
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	// Save original stdout
	old := os.Stdout
	defer func() { os.Stdout = old }()

	// Create pipe
	r, w, err := os.Pipe()
	require.NoError(t, err)

	// Replace stdout
	os.Stdout = w

	// Create channel for output
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	// Execute function
	fn()

	// Close writer and restore stdout
	w.Close()
	os.Stdout = old

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
			output := captureStdout(t, func() {
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
	output := captureStdout(t, func() {
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
			output := captureStdout(t, func() {
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
			output := captureStdout(t, func() {
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
	output := captureStdout(t, func() {
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
			output := captureStdout(t, func() {
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
	output := captureStdout(t, func() {
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
	output := captureStdout(t, func() {
		PrintBlockedBanner(nil)
	})

	// Should still print banner with nil list
	assert.NotEmpty(t, output, "blocked banner should not be empty even with nil tasks")
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(t, tt.fn)
			assert.NotEmpty(t, output, "%s should produce output", tt.name)
			assert.Greater(t, len(output), 10, "%s should produce substantial output", tt.name)
		})
	}
}
