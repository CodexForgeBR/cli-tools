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

// CrossValidationConfig configures the cross-validation phase.
type CrossValidationConfig struct {
	Runner            ai.AIRunner
	TasksFile         string
	ImplOutputFile    string // File path to implementation output
	ValOutputFile     string // File path to validation output
	InadmissibleCount int
	MaxInadmissible   int
}

// CrossValidationResult contains the outcome of cross-validation.
type CrossValidationResult struct {
	Action   string // "success", "continue", "exit"
	ExitCode int
	Feedback string
}

// RunCrossValidation executes the cross-validation phase.
// The cross-validator provides a second opinion on the validator's assessment.
func RunCrossValidation(ctx context.Context, cfg CrossValidationConfig) CrossValidationResult {
	// Check for context cancellation
	if ctx.Err() != nil {
		return CrossValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
		}
	}

	// Build the cross-validation prompt with file paths
	promptText := prompt.BuildCrossValidationPrompt(cfg.TasksFile, cfg.ValOutputFile, cfg.ImplOutputFile)

	// Create temporary output file for cross-validation
	tmpDir := os.TempDir()
	outputPath := filepath.Join(tmpDir, "cross-validation-output.txt")

	// Run cross-validation with the AI runner (pass prompt content, not file path)
	err := cfg.Runner.Run(ctx, promptText, outputPath)
	if err != nil {
		return CrossValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: fmt.Sprintf("cross-validation AI error: %v", err),
		}
	}

	// Parse cross-validation result
	output, err := os.ReadFile(outputPath)
	if err != nil {
		return CrossValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: fmt.Sprintf("failed to read cross-validation output: %v", err),
		}
	}

	parsed, err := parser.ParseCrossValidation(string(output))
	if err != nil {
		return CrossValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: fmt.Sprintf("failed to parse cross-validation: %v", err),
		}
	}

	if parsed == nil {
		return CrossValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: "no cross-validation verdict found",
		}
	}

	// Process the cross-validation verdict
	switch parsed.Verdict {
	case "CONFIRMED":
		// Cross-validator agrees - proceed to next phase
		return CrossValidationResult{
			Action:   "success",
			ExitCode: exitcode.Success,
		}
	case "REJECTED":
		// Cross-validator disagrees - send back to implementation
		return CrossValidationResult{
			Action:   "continue",
			ExitCode: 0,
			Feedback: parsed.Feedback,
		}
	default:
		return CrossValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: fmt.Sprintf("unknown cross-validation verdict: %s", parsed.Verdict),
		}
	}
}
