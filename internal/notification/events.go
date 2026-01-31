package notification

import "fmt"

// Event types matching the notification events.
const (
	EventCompleted     = "completed"
	EventMaxIterations = "max_iterations"
	EventEscalate      = "escalate"
	EventBlocked       = "blocked"
	EventTasksInvalid  = "tasks_invalid"
	EventInadmissible  = "inadmissible"
	EventInterrupted   = "interrupted"
	EventRateLimited   = "rate_limited"
)

// FormatEvent creates a notification message for the given event.
func FormatEvent(event string, projectName string, sessionID string, iteration int, exitCode int) string {
	switch event {
	case EventCompleted:
		return fmt.Sprintf("✅ %s [%s] completed successfully after %d iterations (exit %d)", projectName, sessionID, iteration, exitCode)
	case EventMaxIterations:
		return fmt.Sprintf("⚠️ %s [%s] reached max iterations (%d) (exit %d)", projectName, sessionID, iteration, exitCode)
	case EventEscalate:
		return fmt.Sprintf("🚨 %s [%s] ESCALATION required at iteration %d (exit %d)", projectName, sessionID, iteration, exitCode)
	case EventBlocked:
		return fmt.Sprintf("🔒 %s [%s] all tasks blocked at iteration %d (exit %d)", projectName, sessionID, iteration, exitCode)
	case EventTasksInvalid:
		return fmt.Sprintf("❌ %s [%s] tasks validation failed (exit %d)", projectName, sessionID, exitCode)
	case EventInadmissible:
		return fmt.Sprintf("🚫 %s [%s] inadmissible threshold exceeded at iteration %d (exit %d)", projectName, sessionID, iteration, exitCode)
	case EventInterrupted:
		return fmt.Sprintf("⏸️ %s [%s] interrupted at iteration %d. Use --resume (exit %d)", projectName, sessionID, iteration, exitCode)
	case EventRateLimited:
		return fmt.Sprintf("⏳ %s [%s] rate limit hit at iteration %d - waiting for reset", projectName, sessionID, iteration)
	default:
		return fmt.Sprintf("ℹ️ %s [%s] event: %s (exit %d)", projectName, sessionID, event, exitCode)
	}
}
