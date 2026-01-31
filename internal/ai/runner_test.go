package ai

import (
	"context"
	"errors"
	"testing"

	"github.com/CodexForgeBR/cli-tools/internal/ratelimit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAIRunnerInterface verifies the AIRunner interface contract
func TestAIRunnerInterface(t *testing.T) {
	t.Run("interface can be satisfied by mock implementation", func(t *testing.T) {
		mock := &mockRunner{}
		var _ AIRunner = mock // Compile-time check
		assert.NotNil(t, mock)
	})

	t.Run("interface defines Run method with correct signature", func(t *testing.T) {
		mock := &mockRunner{
			runFunc: func(ctx context.Context, prompt string, outputPath string) error {
				return nil
			},
		}

		ctx := context.Background()
		err := mock.Run(ctx, "test prompt", "/tmp/output.json")
		require.NoError(t, err)
	})

	t.Run("mock implementation can return errors", func(t *testing.T) {
		expectedErr := errors.New("test error")
		mock := &mockRunner{
			runFunc: func(ctx context.Context, prompt string, outputPath string) error {
				return expectedErr
			},
		}

		ctx := context.Background()
		err := mock.Run(ctx, "test prompt", "/tmp/output.json")
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("mock implementation receives correct parameters", func(t *testing.T) {
		var capturedCtx context.Context
		var capturedPrompt string
		var capturedOutputPath string

		mock := &mockRunner{
			runFunc: func(ctx context.Context, prompt string, outputPath string) error {
				capturedCtx = ctx
				capturedPrompt = prompt
				capturedOutputPath = outputPath
				return nil
			},
		}

		ctx := context.WithValue(context.Background(), "testKey", "testValue")
		expectedPrompt := "test prompt with details"
		expectedOutputPath := "/tmp/test-output.json"

		err := mock.Run(ctx, expectedPrompt, expectedOutputPath)
		require.NoError(t, err)
		assert.Equal(t, ctx, capturedCtx)
		assert.Equal(t, expectedPrompt, capturedPrompt)
		assert.Equal(t, expectedOutputPath, capturedOutputPath)
	})

	t.Run("mock implementation respects context cancellation", func(t *testing.T) {
		mock := &mockRunner{
			runFunc: func(ctx context.Context, prompt string, outputPath string) error {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return nil
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := mock.Run(ctx, "test prompt", "/tmp/output.json")
		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

// TestAIRunnerInterfaceContract ensures the interface can be used polymorphically
func TestAIRunnerInterfaceContract(t *testing.T) {
	testCases := []struct {
		name     string
		runner   AIRunner
		prompt   string
		output   string
		expectOK bool
	}{
		{
			name: "successful runner",
			runner: &mockRunner{
				runFunc: func(ctx context.Context, prompt string, outputPath string) error {
					return nil
				},
			},
			prompt:   "success test",
			output:   "/tmp/success.json",
			expectOK: true,
		},
		{
			name: "failing runner",
			runner: &mockRunner{
				runFunc: func(ctx context.Context, prompt string, outputPath string) error {
					return errors.New("runner failed")
				},
			},
			prompt:   "failure test",
			output:   "/tmp/failure.json",
			expectOK: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			err := tc.runner.Run(ctx, tc.prompt, tc.output)

			if tc.expectOK {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RateLimitError tests
// ---------------------------------------------------------------------------

func TestRateLimitError_Error(t *testing.T) {
	t.Run("with parseable info shows reset time", func(t *testing.T) {
		err := &RateLimitError{
			Info: &ratelimit.RateLimitInfo{
				Detected:   true,
				Parseable:  true,
				ResetHuman: "2026-01-30 18:00:00 UTC",
			},
		}
		assert.Contains(t, err.Error(), "resets at")
		assert.Contains(t, err.Error(), "2026-01-30 18:00:00 UTC")
	})

	t.Run("with unparseable info shows unknown", func(t *testing.T) {
		err := &RateLimitError{
			Info: &ratelimit.RateLimitInfo{
				Detected:  true,
				Parseable: false,
			},
		}
		assert.Contains(t, err.Error(), "reset time unknown")
	})

	t.Run("with nil info shows unknown", func(t *testing.T) {
		err := &RateLimitError{
			Info: nil,
		}
		assert.Contains(t, err.Error(), "reset time unknown")
	})
}

func TestRateLimitError_Unwrap(t *testing.T) {
	t.Run("returns underlying error", func(t *testing.T) {
		underlying := errors.New("command failed")
		err := &RateLimitError{
			UnderlyingErr: underlying,
		}
		assert.Equal(t, underlying, err.Unwrap())
	})

	t.Run("returns nil when no underlying error", func(t *testing.T) {
		err := &RateLimitError{}
		assert.Nil(t, err.Unwrap())
	})

	t.Run("errors.As works with RateLimitError", func(t *testing.T) {
		underlying := errors.New("command failed")
		rlErr := &RateLimitError{
			Info: &ratelimit.RateLimitInfo{
				Detected: true,
			},
			UnderlyingErr: underlying,
		}

		var target *RateLimitError
		assert.True(t, errors.As(rlErr, &target))
		assert.Equal(t, rlErr, target)
	})
}

// mockRunner is a test implementation of AIRunner
type mockRunner struct {
	runFunc func(ctx context.Context, prompt string, outputPath string) error
}

func (m *mockRunner) Run(ctx context.Context, prompt string, outputPath string) error {
	if m.runFunc != nil {
		return m.runFunc(ctx, prompt, outputPath)
	}
	return nil
}
