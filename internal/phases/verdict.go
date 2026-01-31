package phases

import (
	"fmt"

	"github.com/CodexForgeBR/cli-tools/internal/exitcode"
)

// VerdictInput contains the data needed to process a validation verdict.
type VerdictInput struct {
	Verdict           string
	Feedback          string
	Remaining         int // unchecked tasks
	BlockedCount      int
	BlockedTasks      []string
	InadmissibleCount int
	MaxInadmissible   int
}

// VerdictResult contains the outcome of verdict processing.
type VerdictResult struct {
	Action               string // "continue" or "exit"
	ExitCode             int
	Feedback             string
	NewInadmissibleCount int
}

// ProcessVerdict handles all 5 primary verdicts with override logic.
func ProcessVerdict(input VerdictInput) VerdictResult {
	switch input.Verdict {
	case "COMPLETE":
		return processComplete(input)
	case "NEEDS_MORE_WORK":
		return VerdictResult{
			Action:               "continue",
			ExitCode:             0,
			Feedback:             input.Feedback,
			NewInadmissibleCount: input.InadmissibleCount,
		}
	case "ESCALATE":
		return VerdictResult{
			Action:               "exit",
			ExitCode:             exitcode.Escalate,
			Feedback:             "",
			NewInadmissibleCount: input.InadmissibleCount,
		}
	case "INADMISSIBLE":
		return processInadmissible(input)
	case "BLOCKED":
		return processBlocked(input)
	default:
		return VerdictResult{
			Action:               "exit",
			ExitCode:             exitcode.Error,
			Feedback:             "",
			NewInadmissibleCount: input.InadmissibleCount,
		}
	}
}

func processComplete(input VerdictInput) VerdictResult {
	// Override: if unchecked doable tasks remain, treat as NEEDS_MORE_WORK
	doable := input.Remaining - input.BlockedCount
	if input.Remaining > 0 && doable > 0 {
		return VerdictResult{
			Action:   "continue",
			ExitCode: 0,
			Feedback: fmt.Sprintf("Validation marked complete but %d tasks remain unchecked. Continuing implementation.", input.Remaining),
			NewInadmissibleCount: input.InadmissibleCount,
		}
	}
	// All blocked
	if input.Remaining > 0 && input.BlockedCount >= input.Remaining {
		return VerdictResult{
			Action:               "exit",
			ExitCode:             exitcode.Blocked,
			Feedback:             "",
			NewInadmissibleCount: input.InadmissibleCount,
		}
	}
	// Truly complete
	return VerdictResult{
		Action:               "exit",
		ExitCode:             exitcode.Success,
		Feedback:             "",
		NewInadmissibleCount: input.InadmissibleCount,
	}
}

func processInadmissible(input VerdictInput) VerdictResult {
	newCount := input.InadmissibleCount + 1
	if newCount > input.MaxInadmissible {
		return VerdictResult{
			Action:               "exit",
			ExitCode:             exitcode.Inadmissible,
			Feedback:             "",
			NewInadmissibleCount: newCount,
		}
	}
	return VerdictResult{
		Action:               "continue",
		ExitCode:             0,
		Feedback:             input.Feedback,
		NewInadmissibleCount: newCount,
	}
}

func processBlocked(input VerdictInput) VerdictResult {
	// If some tasks are doable, continue
	doable := input.Remaining - input.BlockedCount
	if doable > 0 {
		return VerdictResult{
			Action:               "continue",
			ExitCode:             0,
			Feedback:             input.Feedback,
			NewInadmissibleCount: input.InadmissibleCount,
		}
	}
	// All blocked
	return VerdictResult{
		Action:               "exit",
		ExitCode:             exitcode.Blocked,
		Feedback:             "",
		NewInadmissibleCount: input.InadmissibleCount,
	}
}
