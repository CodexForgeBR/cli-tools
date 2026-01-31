package phases

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/CodexForgeBR/cli-tools/internal/exitcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCrossValidationRunner is a mock AI runner for testing cross-validation.
type mockCrossValidationRunner struct {
	output string
	err    error
}

func (m *mockCrossValidationRunner) Run(ctx context.Context, promptPath string, outputPath string) error {
	if m.err != nil {
		return m.err
	}
	return os.WriteFile(outputPath, []byte(m.output), 0644)
}

// TestRunCrossValidation_ConfirmedVerdict tests that CONFIRMED verdict leads to success.
func TestRunCrossValidation_ConfirmedVerdict(t *testing.T) {
	tmpDir := t.TempDir()
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))

	implOutputFile := filepath.Join(tmpDir, "impl-output.txt")
	valOutputFile := filepath.Join(tmpDir, "val-output.txt")
	require.NoError(t, os.WriteFile(implOutputFile, []byte("Implementation output"), 0644))
	require.NoError(t, os.WriteFile(valOutputFile, []byte("Validation output"), 0644))

	runner := &mockCrossValidationRunner{
		output: `Cross-validation complete:

` + "```json\n" + `{
  "RALPH_CROSS_VALIDATION": {
    "verdict": "CONFIRMED",
    "feedback": "Implementation correctly addresses all requirements"
  }
}
` + "```",
	}

	cfg := CrossValidationConfig{
		Runner:            runner,
		TasksFile:         tasksFile,
		ImplOutputFile:    implOutputFile,
		ValOutputFile:     valOutputFile,
		InadmissibleCount: 0,
		MaxInadmissible:   3,
	}

	result := RunCrossValidation(context.Background(), cfg)

	assert.Equal(t, "success", result.Action)
	assert.Equal(t, exitcode.Success, result.ExitCode)
	assert.Empty(t, result.Feedback)
}

// TestRunCrossValidation_RejectedVerdict tests that REJECTED verdict leads to continuation.
func TestRunCrossValidation_RejectedVerdict(t *testing.T) {
	tmpDir := t.TempDir()
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))

	implOutputFile := filepath.Join(tmpDir, "impl-output.txt")
	valOutputFile := filepath.Join(tmpDir, "val-output.txt")
	require.NoError(t, os.WriteFile(implOutputFile, []byte("Implementation output"), 0644))
	require.NoError(t, os.WriteFile(valOutputFile, []byte("Validation output"), 0644))

	runner := &mockCrossValidationRunner{
		output: `Cross-validation found issues:

` + "```json\n" + `{
  "RALPH_CROSS_VALIDATION": {
    "verdict": "REJECTED",
    "feedback": "Missing edge case handling for empty input"
  }
}
` + "```",
	}

	cfg := CrossValidationConfig{
		Runner:            runner,
		TasksFile:         tasksFile,
		ImplOutputFile:    implOutputFile,
		ValOutputFile:     valOutputFile,
		InadmissibleCount: 0,
		MaxInadmissible:   3,
	}

	result := RunCrossValidation(context.Background(), cfg)

	assert.Equal(t, "continue", result.Action)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Feedback, "edge case handling")
}

// TestRunCrossValidation_ContextCancelled tests handling of cancelled context.
func TestRunCrossValidation_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cfg := CrossValidationConfig{
		Runner:         &mockCrossValidationRunner{},
		TasksFile:      "/path/to/tasks.md",
		ImplOutputFile: "/path/to/impl.txt",
		ValOutputFile:  "/path/to/val.txt",
	}

	result := RunCrossValidation(ctx, cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
}

// TestRunCrossValidation_UnknownVerdict tests handling of unknown verdict.
func TestRunCrossValidation_UnknownVerdict(t *testing.T) {
	tmpDir := t.TempDir()
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))

	implOutputFile := filepath.Join(tmpDir, "impl-output.txt")
	valOutputFile := filepath.Join(tmpDir, "val-output.txt")
	require.NoError(t, os.WriteFile(implOutputFile, []byte("Implementation"), 0644))
	require.NoError(t, os.WriteFile(valOutputFile, []byte("Validation"), 0644))

	runner := &mockCrossValidationRunner{
		output: `{"RALPH_CROSS_VALIDATION": {"verdict": "UNKNOWN", "feedback": "Unknown state"}}`,
	}

	cfg := CrossValidationConfig{
		Runner:         runner,
		TasksFile:      tasksFile,
		ImplOutputFile: implOutputFile,
		ValOutputFile:  valOutputFile,
	}

	result := RunCrossValidation(context.Background(), cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Contains(t, result.Feedback, "unknown cross-validation verdict")
}

// TestRunCrossValidation_NoVerdictFound tests handling when no verdict is parsed.
func TestRunCrossValidation_NoVerdictFound(t *testing.T) {
	tmpDir := t.TempDir()
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))

	implOutputFile := filepath.Join(tmpDir, "impl-output.txt")
	valOutputFile := filepath.Join(tmpDir, "val-output.txt")
	require.NoError(t, os.WriteFile(implOutputFile, []byte("Implementation"), 0644))
	require.NoError(t, os.WriteFile(valOutputFile, []byte("Validation"), 0644))

	runner := &mockCrossValidationRunner{
		output: "Just some text without any verdict",
	}

	cfg := CrossValidationConfig{
		Runner:         runner,
		TasksFile:      tasksFile,
		ImplOutputFile: implOutputFile,
		ValOutputFile:  valOutputFile,
	}

	result := RunCrossValidation(context.Background(), cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Contains(t, result.Feedback, "no cross-validation verdict found")
}
