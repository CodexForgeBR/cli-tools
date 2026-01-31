package signal

import (
	"context"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetupSignalHandler_SIGINTCallsCallback verifies that SIGINT triggers the onInterrupt callback
func TestSetupSignalHandler_SIGINTCallsCallback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	callbackCalled := false
	var mu sync.Mutex
	onInterrupt := func() {
		mu.Lock()
		callbackCalled = true
		mu.Unlock()
	}

	// Setup handler in goroutine
	go SetupSignalHandler(ctx, cancel, onInterrupt)

	// Give handler time to install signal channel
	time.Sleep(50 * time.Millisecond)

	// Send SIGINT to self
	err := syscall.Kill(os.Getpid(), syscall.SIGINT)
	require.NoError(t, err, "failed to send SIGINT")

	// Wait for callback to be called
	deadline := time.After(1 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mu.Lock()
			if callbackCalled {
				mu.Unlock()
				return // Test passes
			}
			mu.Unlock()
		case <-deadline:
			t.Fatal("onInterrupt callback was not called within timeout")
		}
	}
}

// TestSetupSignalHandler_ContextCancellation verifies that the handler responds to context cancellation
func TestSetupSignalHandler_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	callbackCalled := false
	var mu sync.Mutex
	onInterrupt := func() {
		mu.Lock()
		callbackCalled = true
		mu.Unlock()
	}

	done := make(chan struct{})
	go func() {
		SetupSignalHandler(ctx, cancel, onInterrupt)
		close(done)
	}()

	// Give handler time to start
	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for handler to exit
	select {
	case <-done:
		// Handler exited as expected
	case <-time.After(1 * time.Second):
		t.Fatal("handler did not exit after context cancellation")
	}

	// Callback should NOT have been called for context cancellation
	mu.Lock()
	assert.False(t, callbackCalled, "onInterrupt should not be called for context cancellation")
	mu.Unlock()
}

// TestSetupSignalHandler_SIGTERMCallsCallback verifies that SIGTERM triggers the onInterrupt callback
func TestSetupSignalHandler_SIGTERMCallsCallback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	callbackCalled := false
	var mu sync.Mutex
	onInterrupt := func() {
		mu.Lock()
		callbackCalled = true
		mu.Unlock()
	}

	// Setup handler in goroutine
	go SetupSignalHandler(ctx, cancel, onInterrupt)

	// Give handler time to install signal channel
	time.Sleep(50 * time.Millisecond)

	// Send SIGTERM to self
	err := syscall.Kill(os.Getpid(), syscall.SIGTERM)
	require.NoError(t, err, "failed to send SIGTERM")

	// Wait for callback to be called
	deadline := time.After(1 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mu.Lock()
			if callbackCalled {
				mu.Unlock()
				return // Test passes
			}
			mu.Unlock()
		case <-deadline:
			t.Fatal("onInterrupt callback was not called within timeout")
		}
	}
}

// TestSetupSignalHandler_CancelFunctionCalled verifies that cancel() is invoked on signal
func TestSetupSignalHandler_CancelFunctionCalled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	onInterrupt := func() {
		// No-op callback
	}

	// Setup handler in goroutine
	go SetupSignalHandler(ctx, cancel, onInterrupt)

	// Give handler time to install signal channel
	time.Sleep(50 * time.Millisecond)

	// Send SIGINT to self
	err := syscall.Kill(os.Getpid(), syscall.SIGINT)
	require.NoError(t, err, "failed to send SIGINT")

	// Wait for context to be cancelled
	select {
	case <-ctx.Done():
		// Context was cancelled as expected
		assert.Equal(t, context.Canceled, ctx.Err())
	case <-time.After(1 * time.Second):
		t.Fatal("context was not cancelled within timeout")
	}
}

// TestSetupSignalHandler_MultipleSignals verifies handler responds to multiple signals correctly
func TestSetupSignalHandler_MultipleSignals(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	callCount := 0
	var mu sync.Mutex
	onInterrupt := func() {
		mu.Lock()
		callCount++
		mu.Unlock()
	}

	// Setup handler in goroutine
	go SetupSignalHandler(ctx, cancel, onInterrupt)

	// Give handler time to install signal channel
	time.Sleep(50 * time.Millisecond)

	// The first signal should trigger the callback and cancel context
	err := syscall.Kill(os.Getpid(), syscall.SIGINT)
	require.NoError(t, err, "failed to send first SIGINT")

	// Wait for first callback
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	firstCount := callCount
	mu.Unlock()

	assert.Equal(t, 1, firstCount, "callback should have been called exactly once after first signal")

	// Context should now be cancelled, so handler should exit and not process more signals
	select {
	case <-ctx.Done():
		// Expected
	case <-time.After(500 * time.Millisecond):
		t.Fatal("context not cancelled after first signal")
	}
}

// TestSetupSignalHandler_NilCallback verifies handler works even with nil callback
func TestSetupSignalHandler_NilCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Setup handler with nil callback - should not panic
	go SetupSignalHandler(ctx, cancel, nil)

	// Give handler time to start
	time.Sleep(50 * time.Millisecond)

	// Send SIGINT to self
	err := syscall.Kill(os.Getpid(), syscall.SIGINT)
	require.NoError(t, err, "failed to send SIGINT")

	// Wait for context to be cancelled (handler should still work)
	select {
	case <-ctx.Done():
		// Context was cancelled as expected, even without callback
	case <-time.After(1 * time.Second):
		t.Fatal("context was not cancelled within timeout")
	}
}
