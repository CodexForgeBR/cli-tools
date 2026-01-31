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

// retryMockRunner is a test double for AIRunner used in retry runner tests.
type retryMockRunner struct {
	calls   int
	results []error
}

func (m *retryMockRunner) Run(ctx context.Context, prompt string, outputPath string) error {
	idx := m.calls
	m.calls++
	if idx < len(m.results) {
		return m.results[idx]
	}
	return nil
}

// Compile-time interface check.
var _ AIRunner = (*RetryRunner)(nil)

func TestRetryRunner_DelegatesToInner(t *testing.T) {
	inner := &retryMockRunner{results: []error{nil}}
	runner := &RetryRunner{
		Inner:    inner,
		RetryCfg: RetryConfig{MaxRetries: 3, BaseDelay: 1},
	}

	err := runner.Run(context.Background(), "hello", "/tmp/out.txt")

	require.NoError(t, err)
	assert.Equal(t, 1, inner.calls, "should delegate exactly once on success")
}

func TestRetryRunner_RetriesOnError(t *testing.T) {
	inner := &retryMockRunner{
		results: []error{
			errors.New("fail1"),
			errors.New("fail2"),
			nil, // succeeds on third attempt
		},
	}
	runner := &RetryRunner{
		Inner: inner,
		RetryCfg: RetryConfig{
			MaxRetries: 5,
			BaseDelay:  1,
		},
	}

	// BaseDelay=1 means waits are 1s+2s=3s before third attempt; allow 5s
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := runner.Run(ctx, "prompt", "/tmp/out.txt")

	require.NoError(t, err)
	assert.Equal(t, 3, inner.calls, "should retry until success")
}

func TestRetryRunner_HandlesRateLimitError(t *testing.T) {
	rlErr := &RateLimitError{
		Info: &ratelimit.RateLimitInfo{
			Detected:   true,
			Parseable:  true,
			ResetEpoch: time.Now().Add(-1 * time.Second).Unix(), // already past
			ResetHuman: "now",
		},
	}
	inner := &retryMockRunner{
		results: []error{
			rlErr,
			nil, // succeeds after rate limit
		},
	}
	runner := &RetryRunner{
		Inner: inner,
		RetryCfg: RetryConfig{
			MaxRetries:        3,
			BaseDelay:         1,
			MaxRateLimitWaits: 3,
		},
	}

	err := runner.Run(context.Background(), "prompt", "/tmp/out.txt")

	require.NoError(t, err)
	assert.Equal(t, 2, inner.calls, "should retry after rate limit")
}

func TestRetryRunner_MaxRetriesExceeded(t *testing.T) {
	inner := &retryMockRunner{
		results: []error{
			errors.New("fail"),
			errors.New("fail"),
			errors.New("fail"),
			errors.New("fail"),
		},
	}
	runner := &RetryRunner{
		Inner: inner,
		RetryCfg: RetryConfig{
			MaxRetries: 2,
			BaseDelay:  1,
		},
	}

	// BaseDelay=1 means waits are 1s+2s=3s; allow 5s for all retries to exhaust
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := runner.Run(ctx, "prompt", "/tmp/out.txt")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "max retries")
}

func TestRetryRunner_ContextCancellation(t *testing.T) {
	inner := &retryMockRunner{
		results: []error{
			errors.New("fail"),
			errors.New("fail"),
			errors.New("fail"),
		},
	}
	runner := &RetryRunner{
		Inner: inner,
		RetryCfg: RetryConfig{
			MaxRetries: 10,
			BaseDelay:  60, // long delay
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel quickly
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := runner.Run(ctx, "prompt", "/tmp/out.txt")
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Less(t, elapsed, 2*time.Second, "should cancel quickly")
}

func TestRetryRunner_ImplementsAIRunner(t *testing.T) {
	// Compile-time check is above; this test verifies at runtime too
	var runner AIRunner = &RetryRunner{
		Inner:    &retryMockRunner{},
		RetryCfg: RetryConfig{},
	}
	assert.NotNil(t, runner)
}
