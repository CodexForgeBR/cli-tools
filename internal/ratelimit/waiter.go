package ratelimit

import (
	"context"
	"fmt"
	"time"
)

// WaitForReset waits until the rate limit reset time with adaptive countdown.
// Respects context cancellation for graceful shutdown.
func WaitForReset(ctx context.Context, info *RateLimitInfo) error {
	if info == nil || !info.Parseable {
		return fmt.Errorf("cannot wait: rate limit info is nil or not parseable")
	}

	now := time.Now().Unix()
	waitSeconds := info.ResetEpoch - now

	if waitSeconds <= 0 {
		// Already past reset time
		return nil
	}

	resetTime := time.Unix(info.ResetEpoch, 0)

	// Adaptive countdown loop
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			now := time.Now()
			remaining := resetTime.Sub(now)

			if remaining <= 0 {
				// Reset time reached
				return nil
			}

			// Adaptive sleep interval based on remaining time
			var sleepInterval time.Duration
			if remaining < 60*time.Second {
				// Last minute: every 5 seconds
				sleepInterval = 5 * time.Second
			} else if remaining < 5*time.Minute {
				// Last 5 minutes: every 30 seconds
				sleepInterval = 30 * time.Second
			} else {
				// Default: every 60 seconds
				sleepInterval = 60 * time.Second
			}

			// Adjust ticker if needed
			ticker.Reset(sleepInterval)
		}
	}
}

// FormatDuration formats seconds into human-readable duration.
// Examples: "2h 15m", "45m 30s", "30s"
func FormatDuration(seconds int64) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	var parts []string
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if secs > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", secs))
	}

	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " "
		}
		result += part
	}

	return result
}
