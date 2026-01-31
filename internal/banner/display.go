// Package banner provides colored banner display functions for the ralph-loop CLI.
//
// All banner functions write formatted output to stdout with color-coded headers
// and separators. These are used to display session status, completion, errors,
// and other important state transitions during ralph-loop execution.
package banner

import (
	"fmt"
	"strings"

	"github.com/CodexForgeBR/cli-tools/internal/logging"
	"github.com/fatih/color"
)

var (
	headerColor  = color.New(color.FgCyan, color.Bold).SprintFunc()
	successColor = color.New(color.FgGreen, color.Bold).SprintFunc()
	errorColor   = color.New(color.FgRed, color.Bold).SprintFunc()
	warnColor    = color.New(color.FgYellow, color.Bold).SprintFunc()
)

// PrintStartupBanner displays the startup banner with session info.
//
// Parameters:
//   - sessionID: Unique identifier for the session
//   - ai: AI provider name (e.g., "claude", "openai")
//   - model: Model identifier (e.g., "claude-3-opus")
//   - tasksFile: Path to the tasks file being processed
//
// Example output:
//
//	═══════════════════════════════════════════════════
//	  ralph-loop - AI Implementation-Validation Loop
//	═══════════════════════════════════════════════════
//	  Session:    20260130-153045
//	  AI:         claude
//	  Model:      claude-3-opus
//	  Tasks:      tasks.md
//	═══════════════════════════════════════════════════
func PrintStartupBanner(sessionID string, ai string, model string, tasksFile string) {
	sep := headerColor("═══════════════════════════════════════════════════")
	fmt.Println(sep)
	fmt.Println(headerColor("  ralph-loop - AI Implementation-Validation Loop"))
	fmt.Println(sep)
	fmt.Printf("  Session:    %s\n", sessionID)
	fmt.Printf("  AI:         %s\n", ai)
	fmt.Printf("  Model:      %s\n", model)
	fmt.Printf("  Tasks:      %s\n", tasksFile)
	fmt.Println(sep)
}

// PrintCompletionBanner displays the completion banner with stats.
//
// Parameters:
//   - iterations: Total number of iterations completed
//   - durationSecs: Total duration in seconds
//
// Example output:
//
//	═══════════════════════════════════════════════════
//	  ✓ All tasks completed successfully!
//	  Iterations: 5
//	  Duration:   1h 23m 45s (5025s)
//	═══════════════════════════════════════════════════
func PrintCompletionBanner(iterations int, durationSecs int) {
	sep := successColor("═══════════════════════════════════════════════════")
	fmt.Println(sep)
	fmt.Println(successColor("  ✓ All tasks completed successfully!"))
	fmt.Printf("  Iterations: %d\n", iterations)
	fmt.Printf("  Duration:   %s (%ds)\n", logging.FormatDuration(durationSecs), durationSecs)
	fmt.Println(sep)
}

// PrintEscalationBanner displays the escalation banner.
//
// Parameters:
//   - feedback: Reason for escalation
//
// Example output:
//
//	═══════════════════════════════════════════════════
//	  ✗ ESCALATION REQUIRED
//	═══════════════════════════════════════════════════
//	  Reason:
//	  Critical architectural decision needed
//	═══════════════════════════════════════════════════
func PrintEscalationBanner(feedback string) {
	sep := errorColor("═══════════════════════════════════════════════════")
	fmt.Println(sep)
	fmt.Println(errorColor("  ✗ ESCALATION REQUIRED"))
	fmt.Println(sep)
	fmt.Println("  Reason:")
	fmt.Printf("  %s\n", feedback)
	fmt.Println(sep)
}

// PrintBlockedBanner displays the blocked banner with task list.
//
// Parameters:
//   - blockedTasks: List of task identifiers that are blocked
//
// Example output:
//
//	═══════════════════════════════════════════════════
//	  ⚠ ALL TASKS BLOCKED
//	═══════════════════════════════════════════════════
//	  Blocked tasks:
//	    - T001: Implement config loader
//	    - T002: Add validation logic
//	═══════════════════════════════════════════════════
func PrintBlockedBanner(blockedTasks []string) {
	sep := warnColor("═══════════════════════════════════════════════════")
	fmt.Println(sep)
	fmt.Println(warnColor("  ⚠ ALL TASKS BLOCKED"))
	fmt.Println(sep)
	if len(blockedTasks) > 0 {
		fmt.Println("  Blocked tasks:")
		for _, task := range blockedTasks {
			fmt.Printf("    - %s\n", task)
		}
	}
	fmt.Println(sep)
}

// PrintMaxIterationsBanner displays when iteration limit is reached.
//
// Parameters:
//   - iterations: Current iteration count
//   - maxIterations: Maximum allowed iterations
//
// Example output:
//
//	═══════════════════════════════════════════════════
//	  ⚠ Max iterations reached (100/100)
//	═══════════════════════════════════════════════════
func PrintMaxIterationsBanner(iterations int, maxIterations int) {
	sep := warnColor("═══════════════════════════════════════════════════")
	fmt.Println(sep)
	fmt.Printf(warnColor("  ⚠ Max iterations reached (%d/%d)\n"), iterations, maxIterations)
	fmt.Println(sep)
}

// PrintInadmissibleBanner displays when inadmissible threshold is exceeded.
//
// Parameters:
//   - count: Current inadmissible count
//   - max: Maximum allowed inadmissible count
//
// Example output:
//
//	═══════════════════════════════════════════════════
//	  ✗ INADMISSIBLE threshold exceeded (5/5)
//	═══════════════════════════════════════════════════
func PrintInadmissibleBanner(count int, max int) {
	sep := errorColor("═══════════════════════════════════════════════════")
	fmt.Println(sep)
	fmt.Printf(errorColor("  ✗ INADMISSIBLE threshold exceeded (%d/%d)\n"), count, max)
	fmt.Println(sep)
}

// PrintInterruptedBanner displays when session is interrupted.
//
// Parameters:
//   - iteration: Current iteration number
//   - phase: Current phase being executed
//
// Example output:
//
//	═══════════════════════════════════════════════════
//	  ⚠ Session interrupted
//	  Iteration: 3
//	  Phase:     validation
//	  Use --resume to continue from this point
//	═══════════════════════════════════════════════════
func PrintInterruptedBanner(iteration int, phase string) {
	sep := warnColor("═══════════════════════════════════════════════════")
	fmt.Println(sep)
	fmt.Println(warnColor("  ⚠ Session interrupted"))
	fmt.Printf("  Iteration: %d\n", iteration)
	fmt.Printf("  Phase:     %s\n", phase)
	fmt.Println("  Use --resume to continue from this point")
	fmt.Println(sep)
}

// StatusInfo contains all fields for displaying session status.
type StatusInfo struct {
	SessionID         string
	Status            string
	Phase             string
	Verdict           string
	Iteration         int
	MaxIterations     int
	InadmissibleCount int
	MaxInadmissible   int
	StartedAt         string
	LastUpdated       string
	AICli             string
	ImplModel         string
	ValModel          string
	CrossValEnabled   bool
	CrossAI           string
	CrossModel        string
	RetryAttempt      int
	RetryDelay        int
	LastFeedback      string
}

// PrintStatusBanner displays current session status with all available fields.
//
// Example output:
//
//	──────────────────────────────────────────────────
//	  Session:    20260130-153045
//	  Status:     IN_PROGRESS
//	  Iteration:  3/20
//	  Phase:      validation
//	  Verdict:    NEEDS_MORE_WORK
//	  AI:         claude (impl: opus, val: opus)
//	  Started:    2026-01-30T15:30:45Z
//	  Updated:    2026-01-30T15:45:00Z
//	──────────────────────────────────────────────────
func PrintStatusBanner(info StatusInfo) {
	sep := strings.Repeat("─", 50)
	fmt.Println(sep)
	fmt.Printf("  Session:    %s\n", info.SessionID)
	fmt.Printf("  Status:     %s\n", info.Status)
	if info.MaxIterations > 0 {
		fmt.Printf("  Iteration:  %d/%d\n", info.Iteration, info.MaxIterations)
	} else {
		fmt.Printf("  Iteration:  %d\n", info.Iteration)
	}
	fmt.Printf("  Phase:      %s\n", info.Phase)
	fmt.Printf("  Verdict:    %s\n", info.Verdict)
	if info.AICli != "" {
		fmt.Printf("  AI:         %s (impl: %s, val: %s)\n", info.AICli, info.ImplModel, info.ValModel)
	}
	if info.CrossValEnabled {
		fmt.Printf("  Cross-val:  %s / %s\n", info.CrossAI, info.CrossModel)
	}
	if info.InadmissibleCount > 0 || info.MaxInadmissible > 0 {
		fmt.Printf("  Inadmiss.:  %d/%d\n", info.InadmissibleCount, info.MaxInadmissible)
	}
	if info.StartedAt != "" {
		fmt.Printf("  Started:    %s\n", info.StartedAt)
	}
	if info.LastUpdated != "" {
		fmt.Printf("  Updated:    %s\n", info.LastUpdated)
	}
	if info.RetryAttempt > 0 {
		fmt.Printf("  Retry:      attempt %d (delay %ds)\n", info.RetryAttempt, info.RetryDelay)
	}
	if info.LastFeedback != "" {
		feedback := info.LastFeedback
		if len(feedback) > 80 {
			feedback = feedback[:80] + "..."
		}
		fmt.Printf("  Feedback:   %s\n", feedback)
	}
	fmt.Println(sep)
}
