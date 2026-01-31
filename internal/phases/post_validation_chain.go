package phases

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CodexForgeBR/cli-tools/internal/ai"
	"github.com/CodexForgeBR/cli-tools/internal/exitcode"
	"github.com/CodexForgeBR/cli-tools/internal/parser"
)

// PostValidationConfig configures the post-validation chain.
type PostValidationConfig struct {
	CrossValRunner    ai.AIRunner
	FinalPlanRunner   ai.AIRunner
	CrossValEnabled   bool
	FinalPlanEnabled  bool
	InadmissibleCount int
	MaxInadmissible   int
}

// PostValidationResult contains the outcome of the post-validation chain.
type PostValidationResult struct {
	Action   string // "success", "continue", "exit"
	ExitCode int
	Feedback string
}

// RunPostValidationChain orchestrates cross-val → final-plan → success/reject flow.
func RunPostValidationChain(ctx context.Context, cfg PostValidationConfig) PostValidationResult {
	// If both disabled, immediate success
	if !cfg.CrossValEnabled && !cfg.FinalPlanEnabled {
		return PostValidationResult{
			Action:   "success",
			ExitCode: exitcode.Success,
		}
	}

	// If only final-plan is enabled, run it directly
	if !cfg.CrossValEnabled && cfg.FinalPlanEnabled {
		return runFinalPlanValidation(ctx, cfg)
	}

	// Run cross-validation if enabled
	if cfg.CrossValEnabled {
		crossResult := runCrossValidation(ctx, cfg)
		if crossResult.Action != "success" {
			return crossResult
		}
	}

	// Cross-val passed or skipped - run final plan if enabled
	if cfg.FinalPlanEnabled {
		return runFinalPlanValidation(ctx, cfg)
	}

	// Everything passed
	return PostValidationResult{
		Action:   "success",
		ExitCode: exitcode.Success,
	}
}

func runCrossValidation(ctx context.Context, cfg PostValidationConfig) PostValidationResult {
	// Check for context cancellation
	if ctx.Err() != nil {
		return PostValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
		}
	}

	// Create temporary output file for cross-validation
	tmpDir := os.TempDir()
	outputPath := filepath.Join(tmpDir, "cross-validation-output.json")

	// Run cross-validation
	err := cfg.CrossValRunner.Run(ctx, "cross-validation-prompt", outputPath)
	if err != nil {
		return PostValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
		}
	}

	// Parse validation result
	output, err := os.ReadFile(outputPath)
	if err != nil {
		return PostValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
		}
	}

	parsed, err := parser.ParseValidation(string(output))
	if err != nil {
		return PostValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
		}
	}

	if parsed == nil {
		return PostValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
		}
	}

	// Process the verdict
	verdictInput := VerdictInput{
		Verdict:           parsed.Verdict,
		Feedback:          parsed.Feedback,
		Remaining:         parsed.Remaining,
		BlockedCount:      parsed.BlockedCount,
		BlockedTasks:      parsed.BlockedTasks,
		InadmissibleCount: cfg.InadmissibleCount,
		MaxInadmissible:   cfg.MaxInadmissible,
	}

	verdictResult := ProcessVerdict(verdictInput)

	// Map verdict result to post-validation result
	switch verdictResult.Action {
	case "exit":
		// If exiting with success, map to "success" to continue chain
		if verdictResult.ExitCode == exitcode.Success {
			return PostValidationResult{
				Action:   "success",
				ExitCode: exitcode.Success,
			}
		}
		// Otherwise exit with error code
		return PostValidationResult{
			Action:   "exit",
			ExitCode: verdictResult.ExitCode,
			Feedback: verdictResult.Feedback,
		}
	case "continue":
		return PostValidationResult{
			Action:   "continue",
			ExitCode: 0,
			Feedback: verdictResult.Feedback,
		}
	default:
		// Unreachable, but handle gracefully
		return PostValidationResult{
			Action:   "success",
			ExitCode: exitcode.Success,
		}
	}
}

func runFinalPlanValidation(ctx context.Context, cfg PostValidationConfig) PostValidationResult {
	// Check for context cancellation
	if ctx.Err() != nil {
		return PostValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
		}
	}

	// Create temporary output file for final-plan validation
	tmpDir := os.TempDir()
	outputPath := filepath.Join(tmpDir, "final-plan-validation-output.json")

	// Run final-plan validation
	err := cfg.FinalPlanRunner.Run(ctx, "final-plan-validation-prompt", outputPath)
	if err != nil {
		return PostValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
			Feedback: fmt.Sprintf("final-plan validation error: %v", err),
		}
	}

	// Parse validation result
	output, err := os.ReadFile(outputPath)
	if err != nil {
		return PostValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
		}
	}

	parsed, err := parser.ParseValidation(string(output))
	if err != nil {
		return PostValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
		}
	}

	if parsed == nil {
		return PostValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
		}
	}

	// Process the verdict
	verdictInput := VerdictInput{
		Verdict:           parsed.Verdict,
		Feedback:          parsed.Feedback,
		Remaining:         parsed.Remaining,
		BlockedCount:      parsed.BlockedCount,
		BlockedTasks:      parsed.BlockedTasks,
		InadmissibleCount: cfg.InadmissibleCount,
		MaxInadmissible:   cfg.MaxInadmissible,
	}

	verdictResult := ProcessVerdict(verdictInput)

	// Map verdict result to post-validation result
	switch verdictResult.Action {
	case "exit":
		// If exiting with success, map to "success" to allow completion
		if verdictResult.ExitCode == exitcode.Success {
			return PostValidationResult{
				Action:   "success",
				ExitCode: exitcode.Success,
			}
		}
		// Otherwise exit with error code
		return PostValidationResult{
			Action:   "exit",
			ExitCode: verdictResult.ExitCode,
			Feedback: verdictResult.Feedback,
		}
	case "continue":
		return PostValidationResult{
			Action:   "continue",
			ExitCode: 0,
			Feedback: verdictResult.Feedback,
		}
	default:
		// Unreachable, but handle gracefully
		return PostValidationResult{
			Action:   "success",
			ExitCode: exitcode.Success,
		}
	}
}
