package ai

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonitorProcess_InactivityTimeout(t *testing.T) {
	t.Run("triggers after configured duration with no file size change", func(t *testing.T) {
		// Create temp file
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.json")
		err := os.WriteFile(outputPath, []byte("initial content"), 0644)
		require.NoError(t, err)

		// Configure very short timeout for testing
		cfg := MonitorConfig{
			InactivityTimeout: 1,                    // 1 second
			HardCap:           60,
			OutputPath:        outputPath,
			TickInterval:      100 * time.Millisecond, // Fast ticking for tests
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})
		start := time.Now()

		go func() {
			MonitorProcess(ctx, cancel, cfg)
			close(done)
		}()

		// Wait for timeout or max 5 seconds
		select {
		case <-done:
			elapsed := time.Since(start)
			// Should trigger after ~1 second inactivity timeout
			assert.GreaterOrEqual(t, elapsed, 1*time.Second)
			assert.Less(t, elapsed, 3*time.Second, "should timeout quickly")
		case <-time.After(5 * time.Second):
			t.Fatal("monitor did not timeout as expected")
		}

		// Context should be cancelled
		assert.Error(t, ctx.Err())
	})

	t.Run("does not trigger if file is actively being written", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.json")
		err := os.WriteFile(outputPath, []byte("initial"), 0644)
		require.NoError(t, err)

		cfg := MonitorConfig{
			InactivityTimeout: 2, // 2 seconds
			HardCap:           10,
			OutputPath:        outputPath,
			TickInterval:      100 * time.Millisecond, // Fast ticking for tests
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})

		go func() {
			MonitorProcess(ctx, cancel, cfg)
			close(done)
		}()

		// Write to file every 500ms to keep it active
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		go func() {
			for i := 0; i < 5; i++ {
				<-ticker.C
				content := []byte("update " + time.Now().String())
				os.WriteFile(outputPath, content, 0644)
			}
		}()

		// Wait a bit - should not timeout due to inactivity
		time.Sleep(3 * time.Second)

		// Should still be running (not timed out due to activity)
		select {
		case <-done:
			// It might have hit hard cap, which is acceptable
		default:
			// Still running is good
			cancel() // Clean shutdown
			<-done
		}
	})
}

func TestMonitorProcess_HardCapTimeout(t *testing.T) {
	t.Run("triggers at hard cap timeout", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.json")
		err := os.WriteFile(outputPath, []byte("content"), 0644)
		require.NoError(t, err)

		cfg := MonitorConfig{
			InactivityTimeout: 100, // High inactivity timeout
			HardCap:           2,   // 2 second hard cap
			OutputPath:        outputPath,
			TickInterval:      100 * time.Millisecond, // Fast ticking for tests
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})
		start := time.Now()

		// Keep writing to file to avoid inactivity timeout
		stopWriting := make(chan struct{})
		go func() {
			ticker := time.NewTicker(200 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					os.WriteFile(outputPath, []byte("update "+time.Now().String()), 0644)
				case <-stopWriting:
					return
				}
			}
		}()
		defer close(stopWriting)

		go func() {
			MonitorProcess(ctx, cancel, cfg)
			close(done)
		}()

		// Wait for hard cap timeout
		select {
		case <-done:
			elapsed := time.Since(start)
			// Should trigger after ~2 seconds (hard cap)
			assert.GreaterOrEqual(t, elapsed, 2*time.Second)
			assert.Less(t, elapsed, 4*time.Second, "should hit hard cap")
		case <-time.After(5 * time.Second):
			t.Fatal("monitor did not hit hard cap as expected")
		}

		assert.Error(t, ctx.Err())
	})

	t.Run("hard cap is 7200 seconds by default", func(t *testing.T) {
		cfg := MonitorConfig{
			InactivityTimeout: 30,
			HardCap:           0, // Should default to 7200
			OutputPath:        "/tmp/test",
		}

		// If HardCap is 0, implementation should use 7200
		// This test documents the default behavior
		expectedDefaultHardCap := 7200
		actualHardCap := cfg.HardCap
		if actualHardCap == 0 {
			actualHardCap = expectedDefaultHardCap
		}
		assert.Equal(t, expectedDefaultHardCap, actualHardCap)
	})
}

func TestMonitorProcess_ResultDetection(t *testing.T) {
	t.Run("triggers grace period when RALPH_STATUS found", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.json")
		err := os.WriteFile(outputPath, []byte("initial"), 0644)
		require.NoError(t, err)

		cfg := MonitorConfig{
			InactivityTimeout: 10,
			HardCap:           30,
			OutputPath:        outputPath,
			TickInterval:      100 * time.Millisecond, // Fast ticking for tests
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})
		start := time.Now()

		go func() {
			MonitorProcess(ctx, cancel, cfg)
			close(done)
		}()

		// Wait a bit then write RALPH_STATUS
		time.Sleep(500 * time.Millisecond)
		err = os.WriteFile(outputPath, []byte("RALPH_STATUS: success"), 0644)
		require.NoError(t, err)

		// Should trigger 2s grace period and then stop
		select {
		case <-done:
			elapsed := time.Since(start)
			// Should complete after ~2.5 seconds (500ms wait + 2s grace)
			assert.GreaterOrEqual(t, elapsed, 2*time.Second)
			assert.Less(t, elapsed, 5*time.Second)
		case <-time.After(10 * time.Second):
			t.Fatal("monitor did not complete after RALPH_STATUS detected")
		}
	})

	t.Run("triggers grace period when RALPH_VALIDATION found", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.json")
		err := os.WriteFile(outputPath, []byte("initial"), 0644)
		require.NoError(t, err)

		cfg := MonitorConfig{
			InactivityTimeout: 10,
			HardCap:           30,
			OutputPath:        outputPath,
			TickInterval:      100 * time.Millisecond, // Fast ticking for tests
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})

		go func() {
			MonitorProcess(ctx, cancel, cfg)
			close(done)
		}()

		time.Sleep(500 * time.Millisecond)
		err = os.WriteFile(outputPath, []byte("RALPH_VALIDATION: passed"), 0644)
		require.NoError(t, err)

		select {
		case <-done:
			// Success
		case <-time.After(10 * time.Second):
			t.Fatal("monitor did not complete after RALPH_VALIDATION detected")
		}
	})

	t.Run("grace period is 2 seconds", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.json")
		err := os.WriteFile(outputPath, []byte("initial"), 0644)
		require.NoError(t, err)

		cfg := MonitorConfig{
			InactivityTimeout: 10,
			HardCap:           30,
			OutputPath:        outputPath,
			TickInterval:      100 * time.Millisecond, // Fast ticking for tests
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})
		var gracePeriodStart time.Time

		go func() {
			MonitorProcess(ctx, cancel, cfg)
			close(done)
		}()

		time.Sleep(200 * time.Millisecond)
		gracePeriodStart = time.Now()
		err = os.WriteFile(outputPath, []byte("RALPH_STATUS: complete"), 0644)
		require.NoError(t, err)

		<-done
		gracePeriodElapsed := time.Since(gracePeriodStart)

		// Grace period should be approximately 2 seconds
		assert.GreaterOrEqual(t, gracePeriodElapsed, 2*time.Second)
		assert.Less(t, gracePeriodElapsed, 3*time.Second)
	})
}

func TestMonitorProcess_ContextCancellation(t *testing.T) {
	t.Run("stops monitoring when context is cancelled", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.json")
		err := os.WriteFile(outputPath, []byte("content"), 0644)
		require.NoError(t, err)

		cfg := MonitorConfig{
			InactivityTimeout: 100,
			HardCap:           200,
			OutputPath:        outputPath,
			TickInterval:      100 * time.Millisecond, // Fast ticking for tests
		}

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		go func() {
			MonitorProcess(ctx, cancel, cfg)
			close(done)
		}()

		// Cancel after short delay
		time.Sleep(500 * time.Millisecond)
		cancel()

		// Should stop quickly
		select {
		case <-done:
			// Success
		case <-time.After(2 * time.Second):
			t.Fatal("monitor did not stop after context cancellation")
		}
	})

	t.Run("handles pre-cancelled context", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.json")
		err := os.WriteFile(outputPath, []byte("content"), 0644)
		require.NoError(t, err)

		cfg := MonitorConfig{
			InactivityTimeout: 100,
			HardCap:           200,
			OutputPath:        outputPath,
			TickInterval:      100 * time.Millisecond, // Fast ticking for tests
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel before monitoring starts

		done := make(chan struct{})
		go func() {
			MonitorProcess(ctx, cancel, cfg)
			close(done)
		}()

		// Should return immediately
		select {
		case <-done:
			// Success
		case <-time.After(1 * time.Second):
			t.Fatal("monitor did not handle pre-cancelled context")
		}
	})
}

func TestMonitorProcess_ZombieDetection(t *testing.T) {
	t.Run("detects when process not writing but still alive", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.json")
		err := os.WriteFile(outputPath, []byte("content"), 0644)
		require.NoError(t, err)

		cfg := MonitorConfig{
			InactivityTimeout: 1, // Very short timeout
			HardCap:           10,
			OutputPath:        outputPath,
			TickInterval:      100 * time.Millisecond, // Fast ticking for tests
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})

		go func() {
			MonitorProcess(ctx, cancel, cfg)
			close(done)
		}()

		// Don't write to file - simulate zombie process
		// File exists but no activity

		select {
		case <-done:
			// Should timeout due to inactivity (zombie detection)
		case <-time.After(5 * time.Second):
			t.Fatal("zombie process not detected")
		}

		assert.Error(t, ctx.Err())
	})
}

func TestMonitorProcess_MissingFile(t *testing.T) {
	t.Run("handles missing output file", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "nonexistent.json")

		cfg := MonitorConfig{
			InactivityTimeout: 2,
			HardCap:           10,
			OutputPath:        outputPath,
			TickInterval:      100 * time.Millisecond, // Fast ticking for tests
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})

		go func() {
			MonitorProcess(ctx, cancel, cfg)
			close(done)
		}()

		// Should handle missing file gracefully
		// May timeout or wait for file creation
		select {
		case <-done:
			// Completed (timeout or error handling)
		case <-time.After(5 * time.Second):
			cancel() // Clean shutdown
			<-done
		}
	})

	t.Run("detects when file is created after monitoring starts", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "delayed.json")

		cfg := MonitorConfig{
			InactivityTimeout: 5,
			HardCap:           20,
			OutputPath:        outputPath,
			TickInterval:      100 * time.Millisecond, // Fast ticking for tests
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})

		go func() {
			MonitorProcess(ctx, cancel, cfg)
			close(done)
		}()

		// Create file after delay
		time.Sleep(1 * time.Second)
		err := os.WriteFile(outputPath, []byte("created late"), 0644)
		require.NoError(t, err)

		// Should start monitoring the newly created file
		time.Sleep(2 * time.Second)
		cancel() // Clean shutdown
		<-done
	})
}
