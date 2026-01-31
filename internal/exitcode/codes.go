// Package exitcode defines named exit codes for the ralph-loop CLI.
//
// Each code maps a specific termination condition to a numeric value
// recognized by shell scripts and CI pipelines.
package exitcode

// Exit code constants matching the ralph-loop data model.
const (
	Success       = 0   // All tasks complete and validated
	Error         = 1   // Invalid args, file not found, misconfiguration
	MaxIterations = 2   // Iteration limit reached
	Escalate      = 3   // Validation requested escalation
	Blocked       = 4   // All tasks blocked on external dependencies
	TasksInvalid  = 5   // Tasks don't implement original plan
	Inadmissible  = 6   // Inadmissible violation threshold exceeded
	Interrupted   = 130 // SIGINT/SIGTERM received
)

// Name returns the human-readable name for the given exit code.
// Unknown codes return "unknown".
func Name(code int) string {
	switch code {
	case Success:
		return "Success"
	case Error:
		return "Error"
	case MaxIterations:
		return "MaxIterations"
	case Escalate:
		return "Escalate"
	case Blocked:
		return "Blocked"
	case TasksInvalid:
		return "TasksInvalid"
	case Inadmissible:
		return "Inadmissible"
	case Interrupted:
		return "Interrupted"
	default:
		return "unknown"
	}
}
