package logging_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CodexForgeBR/cli-tools/internal/logging"
)

func init() {
	// Disable color output in tests so assertions match plain text.
	color.NoColor = true
}

// captureStderr captures stderr output produced by fn.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

// ---------------------------------------------------------------------------
// FormatDuration tests
// ---------------------------------------------------------------------------

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{0, "0s"},
		{45, "45s"},
		{90, "1m 30s"},
		{3661, "1h 1m 1s"},
		{7200, "2h 0m 0s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, logging.FormatDuration(tt.seconds))
		})
	}
}

// ---------------------------------------------------------------------------
// Log output tests
// ---------------------------------------------------------------------------

func TestInfoWritesToStderr(t *testing.T) {
	out := captureStderr(t, func() {
		logging.Info("test message")
	})
	assert.Contains(t, out, "[INFO]")
	assert.Contains(t, out, "test message")
}

func TestSuccessWritesToStderr(t *testing.T) {
	out := captureStderr(t, func() {
		logging.Success("done")
	})
	assert.Contains(t, out, "[SUCCESS]")
	assert.Contains(t, out, "done")
}

func TestWarnWritesToStderr(t *testing.T) {
	out := captureStderr(t, func() {
		logging.Warn("caution")
	})
	assert.Contains(t, out, "[WARN]")
	assert.Contains(t, out, "caution")
}

func TestErrorWritesToStderr(t *testing.T) {
	out := captureStderr(t, func() {
		logging.Error("failure")
	})
	assert.Contains(t, out, "[ERROR]")
	assert.Contains(t, out, "failure")
}

func TestPhaseWritesToStderr(t *testing.T) {
	out := captureStderr(t, func() {
		logging.Phase("implementation")
	})
	assert.Contains(t, out, "[PHASE]")
	assert.Contains(t, out, "implementation")
	// Phase output includes separator lines.
	assert.Contains(t, out, "━━━━")
}

func TestDebugSuppressedWhenNotVerbose(t *testing.T) {
	logging.SetVerbose(false)
	out := captureStderr(t, func() {
		logging.Debug("hidden")
	})
	assert.Empty(t, out)
}

func TestDebugShownWhenVerbose(t *testing.T) {
	logging.SetVerbose(true)
	defer logging.SetVerbose(false)

	out := captureStderr(t, func() {
		logging.Debug("visible")
	})
	assert.Contains(t, out, "[DEBUG]")
	assert.Contains(t, out, "visible")
}
