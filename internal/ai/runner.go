package ai

import (
	"context"
	"fmt"

	"github.com/CodexForgeBR/cli-tools/internal/ratelimit"
)

// AIRunner defines the interface for running AI CLI tools.
type AIRunner interface {
	Run(ctx context.Context, prompt string, outputPath string) error
}

// RateLimitError is returned when a rate limit is detected in AI output.
type RateLimitError struct {
	Info          *ratelimit.RateLimitInfo
	UnderlyingErr error
}

func (e *RateLimitError) Error() string {
	if e.Info != nil && e.Info.Parseable {
		return fmt.Sprintf("rate limit detected (resets at %s)", e.Info.ResetHuman)
	}
	return "rate limit detected (reset time unknown)"
}

func (e *RateLimitError) Unwrap() error {
	return e.UnderlyingErr
}
