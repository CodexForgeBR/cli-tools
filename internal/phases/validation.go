package phases

import (
	"context"
	"os"

	"github.com/CodexForgeBR/cli-tools/internal/ai"
	"github.com/CodexForgeBR/cli-tools/internal/parser"
)

// ValidationConfig configures the validation phase.
type ValidationConfig struct {
	Runner     ai.AIRunner
	OutputPath string
	Prompt     string
}

// ValidationPhaseResult contains the result of validation with parsed data.
type ValidationPhaseResult struct {
	Verdict      string
	Feedback     string
	BlockedTasks []string
}

// RunValidationPhase executes the validation phase using the configured runner.
// It runs the AI with the validation prompt and writes output to the specified path.
func RunValidationPhase(ctx context.Context, cfg ValidationConfig) error {
	return cfg.Runner.Run(ctx, cfg.Prompt, cfg.OutputPath)
}

// RunValidationPhaseWithResult executes validation and parses the result.
func RunValidationPhaseWithResult(ctx context.Context, cfg ValidationConfig) (ValidationPhaseResult, error) {
	// Run validation phase
	err := RunValidationPhase(ctx, cfg)
	if err != nil {
		return ValidationPhaseResult{}, err
	}

	// Read and parse validation output
	output, err := os.ReadFile(cfg.OutputPath)
	if err != nil {
		return ValidationPhaseResult{}, err
	}

	// Parse validation JSON
	parsed, err := parser.ParseValidation(string(output))
	if err != nil {
		return ValidationPhaseResult{}, err
	}

	// Handle nil result (no validation block found)
	if parsed == nil {
		return ValidationPhaseResult{}, nil
	}

	// Convert to result format
	result := ValidationPhaseResult{
		Verdict:      parsed.Verdict,
		Feedback:     parsed.Feedback,
		BlockedTasks: parsed.BlockedTasks,
	}

	return result, nil
}
