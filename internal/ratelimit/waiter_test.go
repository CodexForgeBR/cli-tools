package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// WaitForReset tests
// ---------------------------------------------------------------------------

func TestWaitForReset_NilInfo(t *testing.T) {
	err := WaitForReset(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil or not parseable")
}

func TestWaitForReset_NotParseable(t *testing.T) {
	info := &RateLimitInfo{
		Detected:  true,
		Parseable: false,
	}
	err := WaitForReset(context.Background(), info)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil or not parseable")
}

func TestWaitForReset_AlreadyPast(t *testing.T) {
	info := &RateLimitInfo{
		Detected:   true,
		Parseable:  true,
		ResetEpoch: time.Now().Add(-10 * time.Second).Unix(),
		ResetHuman: "past",
	}
	err := WaitForReset(context.Background(), info)
	require.NoError(t, err)
}

func TestWaitForReset_ContextCancelled(t *testing.T) {
	info := &RateLimitInfo{
		Detected:   true,
		Parseable:  true,
		ResetEpoch: time.Now().Add(1 * time.Hour).Unix(),
		ResetHuman: "future",
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := WaitForReset(ctx, info)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestWaitForReset_ShortWait(t *testing.T) {
	// Wait 2 seconds from now. Unix() truncates to seconds, so actual wait
	// depends on sub-second alignment plus the ticker interval (1s for <1min).
	info := &RateLimitInfo{
		Detected:   true,
		Parseable:  true,
		ResetEpoch: time.Now().Add(2 * time.Second).Unix(),
		ResetHuman: "soon",
	}

	start := time.Now()
	err := WaitForReset(context.Background(), info)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, elapsed, 500*time.Millisecond)
	assert.Less(t, elapsed, 10*time.Second)
}

// ---------------------------------------------------------------------------
// WaitForReset adaptive interval tests
// ---------------------------------------------------------------------------

func TestWaitForReset_MediumWait_ContextCancel(t *testing.T) {
	// Reset 3 minutes from now — remaining < 5min triggers 30s interval branch
	info := &RateLimitInfo{
		Detected:   true,
		Parseable:  true,
		ResetEpoch: time.Now().Add(3 * time.Minute).Unix(),
		ResetHuman: "future",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := WaitForReset(ctx, info)
	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestWaitForReset_LongWait_ContextCancel(t *testing.T) {
	// Reset 10 minutes from now — remaining >= 5min triggers 60s interval branch
	info := &RateLimitInfo{
		Detected:   true,
		Parseable:  true,
		ResetEpoch: time.Now().Add(10 * time.Minute).Unix(),
		ResetHuman: "future",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := WaitForReset(ctx, info)
	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestWaitForReset_VeryLongWait_ContextCancel(t *testing.T) {
	// Reset 2 hours from now — remaining >= 5min triggers 60s interval default branch
	info := &RateLimitInfo{
		Detected:   true,
		Parseable:  true,
		ResetEpoch: time.Now().Add(2 * time.Hour).Unix(),
		ResetHuman: "future",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := WaitForReset(ctx, info)
	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

// ---------------------------------------------------------------------------
// FormatDuration tests
// ---------------------------------------------------------------------------

func TestFormatDuration_SecondsOnly(t *testing.T) {
	assert.Equal(t, "30s", FormatDuration(30))
}

func TestFormatDuration_Minutes(t *testing.T) {
	assert.Equal(t, "5m 30s", FormatDuration(330))
}

func TestFormatDuration_Hours(t *testing.T) {
	assert.Equal(t, "2h 15m 30s", FormatDuration(8130))
}

func TestFormatDuration_Zero(t *testing.T) {
	assert.Equal(t, "0s", FormatDuration(0))
}

func TestFormatDuration_ExactHour(t *testing.T) {
	assert.Equal(t, "1h", FormatDuration(3600))
}

func TestFormatDuration_ExactMinute(t *testing.T) {
	assert.Equal(t, "1m", FormatDuration(60))
}

func TestFormatDuration_LargeValue(t *testing.T) {
	assert.Equal(t, "10h 30m 45s", FormatDuration(37845))
}
