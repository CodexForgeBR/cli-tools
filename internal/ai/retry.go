package ai

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CodexForgeBR/cli-tools/internal/ratelimit"
)

// RetryConfig configures exponential backoff retry behavior.
type RetryConfig struct {
	MaxRetries        int
	BaseDelay         int // seconds (default 5)
	StartAttempt      int // for resume (default 0)
	StartDelay        int // for resume (default 0, will use BaseDelay)
	MaxRateLimitWaits int // max consecutive rate limit waits (default 3)
	OnRetry           func(attempt int, delay int)
	OnRateLimit       func(info *ratelimit.RateLimitInfo)
}

// RetryWithBackoff retries fn with exponential backoff.
// Delays: BaseDelay, BaseDelay*2, BaseDelay*4, BaseDelay*8, ...
// Handles rate limit errors specially: waits for reset time and retries without incrementing attempt.
func RetryWithBackoff(ctx context.Context, cfg RetryConfig, fn func() error) error {
	if cfg.BaseDelay == 0 {
		cfg.BaseDelay = 5
	}
	if cfg.MaxRateLimitWaits == 0 {
		cfg.MaxRateLimitWaits = 3
	}

	attempt := cfg.StartAttempt
	delay := cfg.StartDelay
	if delay == 0 {
		delay = cfg.BaseDelay
	}

	rateLimitWaits := 0

	for {
		err := fn()
		if err == nil {
			return nil
		}

		// Check if this is a rate limit error
		var rateLimitErr *RateLimitError
		if errors.As(err, &rateLimitErr) {
			rateLimitWaits++
			if rateLimitWaits >= cfg.MaxRateLimitWaits {
				return fmt.Errorf("max rate limit waits (%d) exceeded: %w", cfg.MaxRateLimitWaits, err)
			}

			// Notify caller about rate limit
			if cfg.OnRateLimit != nil {
				cfg.OnRateLimit(rateLimitErr.Info)
			}

			// Wait for rate limit reset if parseable
			if rateLimitErr.Info != nil && rateLimitErr.Info.Parseable {
				waitErr := ratelimit.WaitForReset(ctx, rateLimitErr.Info)
				if waitErr != nil {
					return fmt.Errorf("rate limit wait cancelled: %w", waitErr)
				}
			} else {
				// Fallback: 15 minute wait if time unparseable
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(15 * time.Minute):
				}
			}

			// Retry same attempt without incrementing
			continue
		}

		// Not a rate limit error - normal retry logic
		if attempt >= cfg.MaxRetries {
			return fmt.Errorf("max retries (%d) exceeded: %w", cfg.MaxRetries, err)
		}

		if cfg.OnRetry != nil {
			cfg.OnRetry(attempt, delay)
		}

		// Sleep with context awareness
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(delay) * time.Second):
		}

		// Double the delay for next attempt
		delay *= 2
		attempt++
	}
}
