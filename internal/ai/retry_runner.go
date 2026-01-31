package ai

import "context"

// RetryRunner wraps any AIRunner with RetryWithBackoff retry logic.
type RetryRunner struct {
	Inner    AIRunner
	RetryCfg RetryConfig
}

// Run delegates to the inner runner, retrying on failure using RetryWithBackoff.
func (r *RetryRunner) Run(ctx context.Context, prompt string, outputPath string) error {
	return RetryWithBackoff(ctx, r.RetryCfg, func() error {
		return r.Inner.Run(ctx, prompt, outputPath)
	})
}
