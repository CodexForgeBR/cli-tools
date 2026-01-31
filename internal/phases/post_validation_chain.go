package phases

import (
	"context"
	"os"
	"path/filepath"

	"github.com/CodexForgeBR/cli-tools/internal/ai"
	"github.com/CodexForgeBR/cli-tools/internal/exitcode"
	"github.com/CodexForgeBR/cli-tools/internal/parser"
	"github.com/CodexForgeBR/cli-tools/internal/prompt"
)

// PostValidationConfig configures the post-validation chain.
type PostValidationConfig struct {
	CrossValRunner   ai.AIRunner
	FinalPlanRunner  ai.AIRunner
	CrossValEnabled  bool
	FinalPlanEnabled bool
	// File paths for prompt building
	TasksFile      string
	ImplOutputFile string
	ValOutputFile  string
	SpecFile       string // For final-plan validation
	PlanFile       string // For final-plan validation
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

	// Build the cross-validation prompt using proper prompt builder
	crossValPrompt := prompt.BuildCrossValidationPrompt(cfg.TasksFile, cfg.ValOutputFile, cfg.ImplOutputFile)

	// Create temporary output file for cross-validation
	tmpDir := os.TempDir()
	outputPath := filepath.Join(tmpDir, "cross-validation-output.json")
	defer os.Remove(outputPath)

	// Run cross-validation
	err := cfg.CrossValRunner.Run(ctx, crossValPrompt, outputPath)
	if err != nil {
		return PostValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
		}
	}

	// Parse cross-validation result
	output, err := os.ReadFile(outputPath)
	if err != nil {
		return PostValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
		}
	}

	parsed, err := parser.ParseCrossValidation(string(output))
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

	// Handle cross-validation verdicts directly (CONFIRMED/REJECTED)
	switch parsed.Verdict {
	case "CONFIRMED":
		return PostValidationResult{
			Action:   "success",
			ExitCode: exitcode.Success,
		}
	case "REJECTED":
		return PostValidationResult{
			Action:   "continue",
			ExitCode: 0,
			Feedback: parsed.Feedback,
		}
	default:
		// Unknown verdict
		return PostValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
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

	// Build the final-plan prompt using proper prompt builder
	finalPlanPrompt := prompt.BuildFinalPlanPrompt(cfg.SpecFile, cfg.TasksFile, cfg.PlanFile)

	// Create temporary output file for final-plan validation
	tmpDir := os.TempDir()
	outputPath := filepath.Join(tmpDir, "final-plan-validation-output.json")
	defer os.Remove(outputPath)

	// Run final-plan validation
	err := cfg.FinalPlanRunner.Run(ctx, finalPlanPrompt, outputPath)
	if err != nil {
		return PostValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
		}
	}

	// Parse final-plan result
	output, err := os.ReadFile(outputPath)
	if err != nil {
		return PostValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
		}
	}

	parsed, err := parser.ParseFinalPlan(string(output))
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

	// Handle final-plan verdicts (parser maps APPROVE→CONFIRMED, REJECT→NOT_IMPLEMENTED)
	switch parsed.Verdict {
	case "CONFIRMED":
		return PostValidationResult{
			Action:   "success",
			ExitCode: exitcode.Success,
		}
	case "NOT_IMPLEMENTED":
		return PostValidationResult{
			Action:   "continue",
			ExitCode: 0,
			Feedback: parsed.Feedback,
		}
	default:
		// Unknown verdict
		return PostValidationResult{
			Action:   "exit",
			ExitCode: exitcode.Error,
		}
	}
}
