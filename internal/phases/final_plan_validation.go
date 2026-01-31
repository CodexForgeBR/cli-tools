package phases

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CodexForgeBR/cli-tools/internal/ai"
	"github.com/CodexForgeBR/cli-tools/internal/exitcode"
	"github.com/CodexForgeBR/cli-tools/internal/parser"
	"github.com/CodexForgeBR/cli-tools/internal/prompt"
)

// FinalPlanValidationConfig configures the final plan validation phase.
type FinalPlanValidationConfig struct {
	Runner    ai.AIRunner
	SpecFile  string
	TasksFile string
	PlanFile  string
}

// FinalPlanValidationResult contains the outcome of final plan validation.
type FinalPlanValidationResult struct {
	Action   string // "success", "exit"
	ExitCode int
	Feedback string
}

// RunFinalPlanValidation executes the final plan validation phase.
// This is the last checkpoint before implementation begins.
func RunFinalPlanValidation(ctx context.Context, cfg FinalPlanValidationConfig) FinalPlanValidationResult {
	// Check for context cancellation
	if ctx.Err() != nil {
		return FinalPlanValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
		}
	}

	// Build the final plan validation prompt
	promptText := prompt.BuildFinalPlanPrompt(cfg.SpecFile, cfg.TasksFile, cfg.PlanFile)

	// Create temporary output file for final plan validation
	tmpDir := os.TempDir()
	outputPath := filepath.Join(tmpDir, "final-plan-validation-output.txt")

	// Run final plan validation with the AI runner (pass prompt content, not file path)
	err := cfg.Runner.Run(ctx, promptText, outputPath)
	if err != nil {
		return FinalPlanValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: fmt.Sprintf("final plan validation AI error: %v", err),
		}
	}

	// Parse final plan validation result
	output, err := os.ReadFile(outputPath)
	if err != nil {
		return FinalPlanValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: fmt.Sprintf("failed to read final plan validation output: %v", err),
		}
	}

	parsed, err := parser.ParseFinalPlan(string(output))
	if err != nil {
		return FinalPlanValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: fmt.Sprintf("failed to parse final plan validation: %v", err),
		}
	}

	if parsed == nil {
		return FinalPlanValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: "no final plan validation verdict found",
		}
	}

	// Process the final plan validation verdict
	switch parsed.Verdict {
	case "CONFIRMED":
		// Plan is approved - proceed with implementation
		return FinalPlanValidationResult{
			Action:   "success",
			ExitCode: exitcode.Success,
		}
	case "NOT_IMPLEMENTED":
		// Plan is rejected - exit with error
		return FinalPlanValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: parsed.Feedback,
		}
	default:
		return FinalPlanValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: fmt.Sprintf("unknown final plan validation verdict: %s", parsed.Verdict),
		}
	}
}
