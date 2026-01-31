package schedule

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitUntil_PastTime(t *testing.T) {
	// Should return immediately for past times
	past := time.Now().Add(-1 * time.Hour)

	start := time.Now()
	err := WaitUntil(context.Background(), past)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, duration, 100*time.Millisecond, "should return immediately for past time")
}

func TestWaitUntil_FutureTime(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping wait test in short mode")
	}

	// Wait for a short duration (500ms)
	target := time.Now().Add(500 * time.Millisecond)

	start := time.Now()
	err := WaitUntil(context.Background(), target)
	duration := time.Since(start)

	require.NoError(t, err)

	// Should wait approximately the right amount of time (with some tolerance)
	assert.GreaterOrEqual(t, duration, 450*time.Millisecond, "should wait at least 450ms")
	assert.Less(t, duration, 700*time.Millisecond, "should not wait more than 700ms")
}

func TestWaitUntil_ContextCancellation(t *testing.T) {
	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Wait for a time in the future
	target := time.Now().Add(10 * time.Second)

	// Cancel after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := WaitUntil(ctx, target)
	duration := time.Since(start)

	// Should return with context cancelled error
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)

	// Should return quickly (not wait the full 10 seconds)
	assert.Less(t, duration, 1*time.Second, "should cancel quickly")
}

func TestWaitUntil_ContextTimeout(t *testing.T) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Wait for a time far in the future
	target := time.Now().Add(10 * time.Second)

	start := time.Now()
	err := WaitUntil(ctx, target)
	duration := time.Since(start)

	// Should return with deadline exceeded error
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)

	// Should timeout around 200ms
	assert.GreaterOrEqual(t, duration, 200*time.Millisecond, "should wait at least 200ms")
	assert.Less(t, duration, 500*time.Millisecond, "should timeout before 500ms")
}

func TestWaitUntil_CountdownUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping countdown test in short mode")
	}

	// This test verifies that the countdown ticker works
	// We wait for 15 seconds to ensure we get at least one ticker update
	target := time.Now().Add(15 * time.Second)

	// Use a context with timeout to avoid waiting the full 15 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	start := time.Now()
	err := WaitUntil(ctx, target)
	duration := time.Since(start)

	// Should timeout (we cancelled before target)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)

	// Should have waited long enough to see ticker updates (at least 10 seconds for one update)
	assert.GreaterOrEqual(t, duration, 10*time.Second, "should wait long enough to see ticker update")
}

func TestWaitUntil_ImmediateReturn(t *testing.T) {
	// Test that WaitUntil returns immediately for a time that's exactly now
	now := time.Now()

	start := time.Now()
	err := WaitUntil(context.Background(), now)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, duration, 100*time.Millisecond, "should return immediately")
}

func TestWaitUntil_VeryShortWait(t *testing.T) {
	// Test a very short wait (1 millisecond)
	target := time.Now().Add(1 * time.Millisecond)

	start := time.Now()
	err := WaitUntil(context.Background(), target)
	duration := time.Since(start)

	require.NoError(t, err)
	// Should complete quickly but not necessarily exactly 1ms due to scheduler
	assert.Less(t, duration, 100*time.Millisecond, "should complete quickly")
}

func TestWaitUntil_MultipleCallsInSequence(t *testing.T) {
	// Verify we can call WaitUntil multiple times
	for i := 0; i < 3; i++ {
		target := time.Now().Add(50 * time.Millisecond)
		err := WaitUntil(context.Background(), target)
		require.NoError(t, err)
	}
}

func TestWaitUntil_CancelledContext(t *testing.T) {
	// Test with an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	target := time.Now().Add(10 * time.Second)

	start := time.Now()
	err := WaitUntil(ctx, target)
	duration := time.Since(start)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Less(t, duration, 100*time.Millisecond, "should return immediately with cancelled context")
}

// ---------------------------------------------------------------------------
// WaitUntil interval clamping test
// ---------------------------------------------------------------------------

func TestWaitUntil_IntervalClamping(t *testing.T) {
	// Set target to 200ms from now. The adaptive interval (1s) is greater
	// than remaining, so the code should clamp: interval = remaining.
	target := time.Now().Add(200 * time.Millisecond)
	start := time.Now()
	err := WaitUntil(context.Background(), target)
	duration := time.Since(start)

	require.NoError(t, err)
	// Should complete in approximately 200ms, not 1s
	assert.Less(t, duration, 500*time.Millisecond, "should clamp interval to remaining time")
}

func TestWaitUntil_TargetExpiresBeforeFirstIteration(t *testing.T) {
	// Target is barely in the future (1 microsecond). By the time the
	// fmt.Printf runs and the for loop starts, time.Until(target) <= 0,
	// exercising the early-return path at lines 22-25.
	//
	// We try multiple times because the exact timing depends on scheduler.
	for i := 0; i < 20; i++ {
		target := time.Now().Add(1 * time.Microsecond)
		err := WaitUntil(context.Background(), target)
		require.NoError(t, err)
	}
}

// ---------------------------------------------------------------------------
// adaptiveInterval boundary tests
// ---------------------------------------------------------------------------

func TestAdaptiveInterval_OverOneHour(t *testing.T) {
	interval := adaptiveInterval(2 * time.Hour)
	assert.Equal(t, 60*time.Second, interval, ">1h should use 60s interval")
}

func TestAdaptiveInterval_ExactlyOneHour(t *testing.T) {
	// Exactly 1h is NOT > 1h, so falls to >10min bracket
	interval := adaptiveInterval(time.Hour)
	assert.Equal(t, 30*time.Second, interval, "=1h should use 30s interval")
}

func TestAdaptiveInterval_OverTenMinutes(t *testing.T) {
	interval := adaptiveInterval(30 * time.Minute)
	assert.Equal(t, 30*time.Second, interval, ">10min should use 30s interval")
}

func TestAdaptiveInterval_ExactlyTenMinutes(t *testing.T) {
	// Exactly 10min is NOT > 10min, so falls to >1min bracket
	interval := adaptiveInterval(10 * time.Minute)
	assert.Equal(t, 10*time.Second, interval, "=10min should use 10s interval")
}

func TestAdaptiveInterval_OverOneMinute(t *testing.T) {
	interval := adaptiveInterval(5 * time.Minute)
	assert.Equal(t, 10*time.Second, interval, ">1min should use 10s interval")
}

func TestAdaptiveInterval_ExactlyOneMinute(t *testing.T) {
	// Exactly 1min is NOT > 1min, so falls to <1min bracket
	interval := adaptiveInterval(time.Minute)
	assert.Equal(t, 1*time.Second, interval, "=1min should use 1s interval")
}

func TestAdaptiveInterval_UnderOneMinute(t *testing.T) {
	interval := adaptiveInterval(30 * time.Second)
	assert.Equal(t, 1*time.Second, interval, "<1min should use 1s interval")
}

func TestAdaptiveInterval_VerySmall(t *testing.T) {
	interval := adaptiveInterval(100 * time.Millisecond)
	assert.Equal(t, 1*time.Second, interval, "very small remaining should use 1s interval")
}
