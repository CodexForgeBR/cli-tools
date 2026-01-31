package schedule

import (
	"context"
	"fmt"
	"time"
)

// WaitUntil waits until the target time, displaying a countdown.
// Returns immediately if target is in the past.
// Respects context cancellation.
// Uses adaptive intervals: >1h=60s, >10min=30s, >1min=10s, <1min=1s.
func WaitUntil(ctx context.Context, target time.Time) error {
	remaining := time.Until(target)
	if remaining <= 0 {
		return nil
	}

	fmt.Printf("Waiting until %s (%s remaining)\n", target.Format("2006-01-02 15:04:05"), remaining.Round(time.Second))

	for {
		remaining = time.Until(target)
		if remaining <= 0 {
			return nil
		}

		interval := adaptiveInterval(remaining)

		// Don't sleep longer than remaining time
		if interval > remaining {
			interval = remaining
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
			remaining = time.Until(target)
			if remaining <= 0 {
				return nil
			}
			fmt.Printf("  ... %s remaining\n", remaining.Round(time.Second))
		}
	}
}

// adaptiveInterval returns the countdown display interval based on remaining time.
func adaptiveInterval(remaining time.Duration) time.Duration {
	switch {
	case remaining > time.Hour:
		return 60 * time.Second
	case remaining > 10*time.Minute:
		return 30 * time.Second
	case remaining > time.Minute:
		return 10 * time.Second
	default:
		return 1 * time.Second
	}
}
