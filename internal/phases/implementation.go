package phases

import (
	"context"
	"os"
	"strings"

	"github.com/CodexForgeBR/cli-tools/internal/ai"
)

// ImplementationConfig configures the implementation phase.
type ImplementationConfig struct {
	Runner           ai.AIRunner
	Iteration        int
	OutputPath       string
	FirstPrompt      string
	ContinuePrompt   string
	ExtractLearnings bool
}

// ImplementationResult contains the result of the implementation phase with learnings.
type ImplementationResult struct {
	Learnings string
}

// RunImplementationPhase executes the implementation phase using the configured runner.
// It selects the appropriate prompt based on the iteration number:
// - iteration == 1: uses FirstPrompt
// - iteration > 1: uses ContinuePrompt
func RunImplementationPhase(ctx context.Context, cfg ImplementationConfig) error {
	// Select prompt based on iteration
	var prompt string
	if cfg.Iteration == 1 {
		prompt = cfg.FirstPrompt
	} else {
		prompt = cfg.ContinuePrompt
	}

	// Run AI with selected prompt
	return cfg.Runner.Run(ctx, prompt, cfg.OutputPath)
}

// RunImplementationPhaseWithLearnings executes the implementation phase and extracts learnings.
func RunImplementationPhaseWithLearnings(ctx context.Context, cfg ImplementationConfig) (ImplementationResult, error) {
	// Run the implementation phase
	err := RunImplementationPhase(ctx, cfg)
	if err != nil {
		return ImplementationResult{}, err
	}

	result := ImplementationResult{}

	// Extract learnings if enabled
	if cfg.ExtractLearnings {
		output, readErr := os.ReadFile(cfg.OutputPath)
		if readErr == nil {
			result.Learnings = extractLearnings(string(output))
		}
	}

	return result, nil
}

// extractLearnings extracts the learnings section from implementation output.
// It looks for a "## Learnings" section and returns its content.
func extractLearnings(output string) string {
	lines := strings.Split(output, "\n")
	var learnings []string
	inLearnings := false

	for _, line := range lines {
		// Check for learnings section header
		if strings.Contains(strings.ToLower(line), "## learnings") {
			inLearnings = true
			continue
		}

		// If we're in the learnings section
		if inLearnings {
			// Stop at next ## header
			if strings.HasPrefix(strings.TrimSpace(line), "## ") {
				break
			}
			// Add non-empty lines
			if strings.TrimSpace(line) != "" {
				learnings = append(learnings, line)
			}
		}
	}

	return strings.Join(learnings, "\n")
}
