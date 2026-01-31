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

// TasksValidationConfig configures the tasks validation phase.
type TasksValidationConfig struct {
	Runner    ai.AIRunner
	SpecFile  string
	TasksFile string
}

// TasksValidationResult contains the outcome of tasks validation.
type TasksValidationResult struct {
	Action   string // "success", "exit"
	ExitCode int
	Feedback string
}

// RunTasksValidation executes the tasks validation phase.
// Validates that tasks.md correctly implements spec.md requirements.
func RunTasksValidation(ctx context.Context, cfg TasksValidationConfig) TasksValidationResult {
	// Check for context cancellation
	if ctx.Err() != nil {
		return TasksValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
		}
	}

	// Build the tasks validation prompt
	promptText := prompt.BuildTasksValidationPrompt(cfg.SpecFile, cfg.TasksFile)

	// Create temporary output file for tasks validation
	tmpDir := os.TempDir()
	outputPath := filepath.Join(tmpDir, "tasks-validation-output.txt")

	// Write prompt to a temporary file for the AI runner
	promptPath := filepath.Join(tmpDir, "tasks-validation-prompt.txt")
	if err := os.WriteFile(promptPath, []byte(promptText), 0644); err != nil {
		return TasksValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: fmt.Sprintf("failed to write prompt: %v", err),
		}
	}

	// Run tasks validation with the AI runner
	err := cfg.Runner.Run(ctx, promptPath, outputPath)
	if err != nil {
		return TasksValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: fmt.Sprintf("tasks validation AI error: %v", err),
		}
	}

	// Parse tasks validation result
	output, err := os.ReadFile(outputPath)
	if err != nil {
		return TasksValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: fmt.Sprintf("failed to read tasks validation output: %v", err),
		}
	}

	parsed, err := parser.ParseTasksValidation(string(output))
	if err != nil {
		return TasksValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: fmt.Sprintf("failed to parse tasks validation: %v", err),
		}
	}

	if parsed == nil {
		return TasksValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: "no tasks validation verdict found",
		}
	}

	// Process the tasks validation verdict
	switch parsed.Verdict {
	case "VALID":
		// Tasks correctly implement the spec - proceed
		return TasksValidationResult{
			Action:   "success",
			ExitCode: exitcode.Success,
		}
	case "INVALID":
		// Tasks don't match spec - exit with error
		return TasksValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: parsed.Feedback,
		}
	default:
		return TasksValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: fmt.Sprintf("unknown tasks validation verdict: %s", parsed.Verdict),
		}
	}
}
