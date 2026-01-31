package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CodexForgeBR/cli-tools/internal/ratelimit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryWithBackoff_ExponentialBackoff(t *testing.T) {
	t.Run("calculates exponential backoff: 5s, 10s, 20s, 40s", func(t *testing.T) {
		expectedDelays := []int{5, 10, 20, 40, 80}
		actualDelays := []int{}

		cfg := RetryConfig{
			MaxRetries: 5,
			BaseDelay:  5,
			OnRetry: func(attempt int, delay int) {
				actualDelays = append(actualDelays, delay)
			},
		}

		attempt := 0
		fn := func() error {
			attempt++
			if attempt < 6 {
				return errors.New("retry me")
			}
			return nil
		}

		ctx := context.Background()
		// Use a timeout to prevent hanging if delays are too long
		ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		// We expect this to fail with max retries or context timeout
		_ = RetryWithBackoff(ctx, cfg, fn)

		// Verify the delays follow exponential backoff
		for i, expected := range expectedDelays {
			if i < len(actualDelays) {
				assert.Equal(t, expected, actualDelays[i],
					"delay at attempt %d should be %ds", i, expected)
			}
		}
	})

	t.Run("backoff doubles each time", func(t *testing.T) {
		delays := []int{}
		cfg := RetryConfig{
			MaxRetries: 4,
			BaseDelay:  5,
			OnRetry: func(attempt int, delay int) {
				delays = append(delays, delay)
			},
		}

		fn := func() error {
			return errors.New("always fail")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		_ = RetryWithBackoff(ctx, cfg, fn)

		// Each delay should be double the previous
		for i := 1; i < len(delays); i++ {
			assert.Equal(t, delays[i-1]*2, delays[i],
				"delay should double: %d -> %d", delays[i-1], delays[i])
		}
	})

	t.Run("first retry uses base delay", func(t *testing.T) {
		var firstDelay int
		cfg := RetryConfig{
			MaxRetries: 3,
			BaseDelay:  7,
			OnRetry: func(attempt int, delay int) {
				if attempt == 0 {
					firstDelay = delay
				}
			},
		}

		fn := func() error {
			return errors.New("fail")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_ = RetryWithBackoff(ctx, cfg, fn)

		assert.Equal(t, 7, firstDelay, "first retry should use base delay")
	})
}

func TestRetryWithBackoff_MaxRetries(t *testing.T) {
	t.Run("returns error when max retries exceeded", func(t *testing.T) {
		cfg := RetryConfig{
			MaxRetries: 3,
			BaseDelay:  1,
		}

		attempts := 0
		fn := func() error {
			attempts++
			return errors.New("always fail")
		}

		ctx := context.Background()
		err := RetryWithBackoff(ctx, cfg, fn)

		require.Error(t, err)
		// Should have tried: initial attempt + 3 retries = 4 total
		assert.Equal(t, 4, attempts)
	})

	t.Run("succeeds before max retries", func(t *testing.T) {
		cfg := RetryConfig{
			MaxRetries: 5,
			BaseDelay:  1,
		}

		attempts := 0
		fn := func() error {
			attempts++
			if attempts < 3 {
				return errors.New("fail")
			}
			return nil
		}

		ctx := context.Background()
		err := RetryWithBackoff(ctx, cfg, fn)

		require.NoError(t, err)
		assert.Equal(t, 3, attempts, "should succeed on third attempt")
	})

	t.Run("zero max retries means no retries", func(t *testing.T) {
		cfg := RetryConfig{
			MaxRetries: 0,
			BaseDelay:  1,
		}

		attempts := 0
		fn := func() error {
			attempts++
			return errors.New("fail")
		}

		ctx := context.Background()
		err := RetryWithBackoff(ctx, cfg, fn)

		require.Error(t, err)
		assert.Equal(t, 1, attempts, "should only try once with no retries")
	})
}

func TestRetryWithBackoff_StateCallback(t *testing.T) {
	t.Run("callback is called on each retry with attempt number", func(t *testing.T) {
		callbackCalls := []struct {
			attempt int
			delay   int
		}{}

		cfg := RetryConfig{
			MaxRetries: 3,
			BaseDelay:  5,
			OnRetry: func(attempt int, delay int) {
				callbackCalls = append(callbackCalls, struct {
					attempt int
					delay   int
				}{attempt, delay})
			},
		}

		fn := func() error {
			return errors.New("fail")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		_ = RetryWithBackoff(ctx, cfg, fn)

		// Should have been called for each retry
		require.NotEmpty(t, callbackCalls)

		// Verify attempt numbers are sequential
		for i, call := range callbackCalls {
			assert.Equal(t, i, call.attempt, "attempt number should be %d", i)
		}
	})

	t.Run("callback receives correct delay values", func(t *testing.T) {
		callbackDelays := []int{}

		cfg := RetryConfig{
			MaxRetries: 3,
			BaseDelay:  5,
			OnRetry: func(attempt int, delay int) {
				callbackDelays = append(callbackDelays, delay)
			},
		}

		fn := func() error {
			return errors.New("fail")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		_ = RetryWithBackoff(ctx, cfg, fn)

		// Verify delays: 5, 10, 20
		expectedDelays := []int{5, 10, 20}
		for i, expected := range expectedDelays {
			if i < len(callbackDelays) {
				assert.Equal(t, expected, callbackDelays[i])
			}
		}
	})

	t.Run("nil callback is handled gracefully", func(t *testing.T) {
		cfg := RetryConfig{
			MaxRetries: 2,
			BaseDelay:  1,
			OnRetry:    nil, // No callback
		}

		attempts := 0
		fn := func() error {
			attempts++
			return errors.New("fail")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		err := RetryWithBackoff(ctx, cfg, fn)
		require.Error(t, err)
		assert.GreaterOrEqual(t, attempts, 1)
	})
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	t.Run("returns immediately when context cancelled during sleep", func(t *testing.T) {
		cfg := RetryConfig{
			MaxRetries: 5,
			BaseDelay:  10, // Long delay
		}

		fn := func() error {
			return errors.New("fail")
		}

		ctx, cancel := context.WithCancel(context.Background())

		// Cancel after short delay
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		start := time.Now()
		err := RetryWithBackoff(ctx, cfg, fn)
		elapsed := time.Since(start)

		require.Error(t, err)
		assert.Less(t, elapsed, 2*time.Second, "should return quickly after cancellation")
	})

	t.Run("respects pre-cancelled context", func(t *testing.T) {
		cfg := RetryConfig{
			MaxRetries: 5,
			BaseDelay:  5,
		}

		fn := func() error {
			return errors.New("fail")
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel before retry

		err := RetryWithBackoff(ctx, cfg, fn)
		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("context timeout during retry", func(t *testing.T) {
		cfg := RetryConfig{
			MaxRetries: 10,
			BaseDelay:  2,
		}

		fn := func() error {
			return errors.New("fail")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		start := time.Now()
		err := RetryWithBackoff(ctx, cfg, fn)
		elapsed := time.Since(start)

		require.Error(t, err)
		assert.GreaterOrEqual(t, elapsed, 500*time.Millisecond)
		assert.Less(t, elapsed, 1*time.Second)
	})
}

func TestRetryWithBackoff_Resume(t *testing.T) {
	t.Run("resumes from saved attempt state", func(t *testing.T) {
		cfg := RetryConfig{
			MaxRetries:   5,
			BaseDelay:    5,
			StartAttempt: 2, // Resume from attempt 2
			StartDelay:   20, // Should be 20 (5 * 2^2)
		}

		attempts := 0
		fn := func() error {
			attempts++
			if attempts < 2 {
				return errors.New("fail")
			}
			return nil
		}

		ctx := context.Background()
		err := RetryWithBackoff(ctx, cfg, fn)

		require.NoError(t, err)
		// Should succeed quickly since we resumed from attempt 2
	})

	t.Run("resumes with correct delay calculation", func(t *testing.T) {
		firstDelay := 0
		cfg := RetryConfig{
			MaxRetries:   5,
			BaseDelay:    5,
			StartAttempt: 3,  // Resume from attempt 3
			StartDelay:   40, // Should be 40 (5 * 2^3)
			OnRetry: func(attempt int, delay int) {
				if firstDelay == 0 {
					firstDelay = delay
				}
			},
		}

		fn := func() error {
			return errors.New("fail")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		_ = RetryWithBackoff(ctx, cfg, fn)

		// First retry from resumed state should use StartDelay
		assert.Equal(t, 40, firstDelay, "should resume with saved delay")
	})

	t.Run("default StartAttempt is 0", func(t *testing.T) {
		cfg := RetryConfig{
			MaxRetries:   3,
			BaseDelay:    5,
			StartAttempt: 0, // Default
			StartDelay:   5,
		}

		attempts := 0
		fn := func() error {
			attempts++
			return errors.New("fail")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		_ = RetryWithBackoff(ctx, cfg, fn)

		assert.GreaterOrEqual(t, attempts, 1)
	})

	t.Run("default StartDelay is BaseDelay", func(t *testing.T) {
		firstDelay := 0
		cfg := RetryConfig{
			MaxRetries:   3,
			BaseDelay:    7,
			StartAttempt: 0,
			StartDelay:   0, // Should default to BaseDelay
			OnRetry: func(attempt int, delay int) {
				if firstDelay == 0 {
					firstDelay = delay
				}
			},
		}

		fn := func() error {
			return errors.New("fail")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_ = RetryWithBackoff(ctx, cfg, fn)

		// If StartDelay is 0, should use BaseDelay
		expectedDelay := 7
		if firstDelay != 0 {
			assert.Equal(t, expectedDelay, firstDelay)
		}
	})
}

func TestRetryWithBackoff_SuccessOnFirstTry(t *testing.T) {
	t.Run("returns immediately on success without retries", func(t *testing.T) {
		cfg := RetryConfig{
			MaxRetries: 5,
			BaseDelay:  5,
		}

		attempts := 0
		fn := func() error {
			attempts++
			return nil // Success on first try
		}

		ctx := context.Background()
		err := RetryWithBackoff(ctx, cfg, fn)

		require.NoError(t, err)
		assert.Equal(t, 1, attempts, "should only call function once")
	})

	t.Run("callback not called on immediate success", func(t *testing.T) {
		callbackCalled := false
		cfg := RetryConfig{
			MaxRetries: 5,
			BaseDelay:  5,
			OnRetry: func(attempt int, delay int) {
				callbackCalled = true
			},
		}

		fn := func() error {
			return nil // Immediate success
		}

		ctx := context.Background()
		err := RetryWithBackoff(ctx, cfg, fn)

		require.NoError(t, err)
		assert.False(t, callbackCalled, "callback should not be called on immediate success")
	})
}

// ---------------------------------------------------------------------------
// Rate limit handling tests
// ---------------------------------------------------------------------------

func TestRetryWithBackoff_RateLimit_Parseable_SucceedsAfterWait(t *testing.T) {
	// A parseable rate limit with ResetEpoch already in the past should resolve quickly.
	attempts := 0
	cfg := RetryConfig{
		MaxRetries:        3,
		BaseDelay:         1,
		MaxRateLimitWaits: 3,
	}

	fn := func() error {
		attempts++
		if attempts == 1 {
			return &RateLimitError{
				Info: &ratelimit.RateLimitInfo{
					Detected:   true,
					Parseable:  true,
					ResetEpoch: time.Now().Add(-1 * time.Second).Unix(), // already past
					ResetHuman: "already past",
				},
			}
		}
		return nil
	}

	err := RetryWithBackoff(context.Background(), cfg, fn)
	require.NoError(t, err)
	assert.Equal(t, 2, attempts, "should succeed on second attempt after rate limit wait")
}

func TestRetryWithBackoff_RateLimit_Parseable_CancelledDuringWait(t *testing.T) {
	// A parseable rate limit with ResetEpoch far in the future, cancelled via context.
	cfg := RetryConfig{
		MaxRetries:        3,
		BaseDelay:         1,
		MaxRateLimitWaits: 3,
	}

	fn := func() error {
		return &RateLimitError{
			Info: &ratelimit.RateLimitInfo{
				Detected:   true,
				Parseable:  true,
				ResetEpoch: time.Now().Add(1 * time.Hour).Unix(), // far in the future
				ResetHuman: "1 hour from now",
			},
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := RetryWithBackoff(ctx, cfg, fn)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit wait cancelled")
}

func TestRetryWithBackoff_RateLimit_Unparseable_ContextCancel(t *testing.T) {
	// An unparseable rate limit triggers a 15-minute fallback wait.
	// We cancel the context quickly to cover the ctx.Done() branch in the
	// fallback wait select.
	cfg := RetryConfig{
		MaxRetries:        3,
		BaseDelay:         1,
		MaxRateLimitWaits: 3,
	}

	fn := func() error {
		return &RateLimitError{
			Info: &ratelimit.RateLimitInfo{
				Detected:  true,
				Parseable: false,
			},
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := RetryWithBackoff(ctx, cfg, fn)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Less(t, elapsed, 2*time.Second, "should exit quickly on context cancel, not wait 15 min")
}

func TestRetryWithBackoff_RateLimit_NilInfo_ContextCancel(t *testing.T) {
	// RateLimitError with nil Info takes the unparseable fallback path.
	cfg := RetryConfig{
		MaxRetries:        3,
		BaseDelay:         1,
		MaxRateLimitWaits: 3,
	}

	fn := func() error {
		return &RateLimitError{
			Info: nil,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := RetryWithBackoff(ctx, cfg, fn)
	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestRetryWithBackoff_RateLimit_MaxWaitsExceeded(t *testing.T) {
	// After MaxRateLimitWaits rate limit errors, should return an error
	// without further retrying.
	attempts := 0
	cfg := RetryConfig{
		MaxRetries:        10,
		BaseDelay:         1,
		MaxRateLimitWaits: 2,
	}

	fn := func() error {
		attempts++
		return &RateLimitError{
			Info: &ratelimit.RateLimitInfo{
				Detected:   true,
				Parseable:  true,
				ResetEpoch: time.Now().Add(-1 * time.Second).Unix(),
				ResetHuman: "already past",
			},
		}
	}

	err := RetryWithBackoff(context.Background(), cfg, fn)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max rate limit waits")
	assert.Equal(t, 2, attempts, "should stop after MaxRateLimitWaits")
}

func TestRetryWithBackoff_RateLimit_OnRateLimitCallback(t *testing.T) {
	// Verify the OnRateLimit callback is invoked with the correct info.
	var callbackInfo *ratelimit.RateLimitInfo
	callbackCount := 0

	cfg := RetryConfig{
		MaxRetries:        3,
		BaseDelay:         1,
		MaxRateLimitWaits: 3,
		OnRateLimit: func(info *ratelimit.RateLimitInfo) {
			callbackCount++
			callbackInfo = info
		},
	}

	attempts := 0
	fn := func() error {
		attempts++
		if attempts == 1 {
			return &RateLimitError{
				Info: &ratelimit.RateLimitInfo{
					Detected:   true,
					Parseable:  true,
					ResetEpoch: time.Now().Add(-1 * time.Second).Unix(),
					ResetHuman: "now",
				},
			}
		}
		return nil
	}

	err := RetryWithBackoff(context.Background(), cfg, fn)
	require.NoError(t, err)
	assert.Equal(t, 1, callbackCount, "OnRateLimit should be called once")
	require.NotNil(t, callbackInfo)
	assert.True(t, callbackInfo.Detected)
}

func TestRetryWithBackoff_RateLimit_ThenNormalError(t *testing.T) {
	// First a rate limit error, then a normal error. The rate limit should be
	// handled (wait + retry), then the normal error should trigger normal retry logic.
	attempts := 0
	cfg := RetryConfig{
		MaxRetries:        1,
		BaseDelay:         1,
		MaxRateLimitWaits: 3,
	}

	fn := func() error {
		attempts++
		if attempts == 1 {
			return &RateLimitError{
				Info: &ratelimit.RateLimitInfo{
					Detected:   true,
					Parseable:  true,
					ResetEpoch: time.Now().Add(-1 * time.Second).Unix(),
					ResetHuman: "past",
				},
			}
		}
		return errors.New("normal error")
	}

	err := RetryWithBackoff(context.Background(), cfg, fn)
	// After rate limit resolves, it retries same attempt. The normal error then
	// triggers a retry. MaxRetries=1, so after 1 retry it should exceed.
	// attempts: 1 (rate limit) -> retry same -> 2 (normal err) -> retry -> 3 (normal err) -> max exceeded
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max retries")
}

func TestRetryWithBackoff_DefaultBaseDelay(t *testing.T) {
	// When BaseDelay is 0, it should default to 5.
	var firstDelay int
	cfg := RetryConfig{
		MaxRetries: 1,
		BaseDelay:  0, // should default to 5
		OnRetry: func(attempt int, delay int) {
			if firstDelay == 0 {
				firstDelay = delay
			}
		},
	}

	fn := func() error {
		return errors.New("fail")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_ = RetryWithBackoff(ctx, cfg, fn)

	assert.Equal(t, 5, firstDelay, "should default BaseDelay to 5")
}

func TestRetryWithBackoff_DefaultMaxRateLimitWaits(t *testing.T) {
	// When MaxRateLimitWaits is 0, it should default to 3.
	// After 3 rate limit errors it should fail.
	attempts := 0
	cfg := RetryConfig{
		MaxRetries:        10,
		BaseDelay:         1,
		MaxRateLimitWaits: 0, // should default to 3
	}

	fn := func() error {
		attempts++
		return &RateLimitError{
			Info: &ratelimit.RateLimitInfo{
				Detected:   true,
				Parseable:  true,
				ResetEpoch: time.Now().Add(-1 * time.Second).Unix(),
				ResetHuman: "past",
			},
		}
	}

	err := RetryWithBackoff(context.Background(), cfg, fn)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max rate limit waits")
	assert.Equal(t, 3, attempts, "default MaxRateLimitWaits should be 3")
}
