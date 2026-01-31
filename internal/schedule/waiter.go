package schedule

import (
	"context"
	"fmt"
	"time"
)

// WaitUntil waits until the target time, displaying a countdown.
// Returns immediately if target is in the past.
// Respects context cancellation.
func WaitUntil(ctx context.Context, target time.Time) error {
	remaining := time.Until(target)
	if remaining <= 0 {
		return nil
	}

	fmt.Printf("Waiting until %s (%s remaining)\n", target.Format("2006-01-02 15:04:05"), remaining.Round(time.Second))

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(remaining):
			return nil
		case <-ticker.C:
			remaining = time.Until(target)
			if remaining <= 0 {
				return nil
			}
			fmt.Printf("  ... %s remaining\n", remaining.Round(time.Second))
		}
	}
}
