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

// mockTasksValidationRunner is a mock AI runner for testing tasks validation.
type mockTasksValidationRunner struct {
	output string
	err    error
}

func (m *mockTasksValidationRunner) Run(ctx context.Context, promptPath string, outputPath string) error {
	if m.err != nil {
		return m.err
	}
	return os.WriteFile(outputPath, []byte(m.output), 0644)
}

// TestRunTasksValidation_ValidVerdict tests that VALID verdict leads to success.
func TestRunTasksValidation_ValidVerdict(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.md")
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	require.NoError(t, os.WriteFile(specFile, []byte("# Spec"), 0644))
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))

	runner := &mockTasksValidationRunner{
		output: `Tasks validation complete:

` + "```json\n" + `{
  "RALPH_TASKS_VALIDATION": {
    "verdict": "VALID",
    "feedback": "All requirements correctly captured"
  }
}
` + "```",
	}

	cfg := TasksValidationConfig{
		Runner:    runner,
		SpecFile:  specFile,
		TasksFile: tasksFile,
	}

	result := RunTasksValidation(context.Background(), cfg)

	assert.Equal(t, "success", result.Action)
	assert.Equal(t, exitcode.Success, result.ExitCode)
	assert.Empty(t, result.Feedback)
}

// TestRunTasksValidation_InvalidVerdict tests that INVALID verdict leads to exit.
func TestRunTasksValidation_InvalidVerdict(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.md")
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	require.NoError(t, os.WriteFile(specFile, []byte("# Spec"), 0644))
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))

	runner := &mockTasksValidationRunner{
		output: `Tasks validation found issues:

` + "```json\n" + `{
  "RALPH_TASKS_VALIDATION": {
    "verdict": "INVALID",
    "feedback": "Missing requirement 3.2 from spec"
  }
}
` + "```",
	}

	cfg := TasksValidationConfig{
		Runner:    runner,
		SpecFile:  specFile,
		TasksFile: tasksFile,
	}

	result := RunTasksValidation(context.Background(), cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Contains(t, result.Feedback, "Missing requirement 3.2")
}

// TestRunTasksValidation_ContextCancelled tests handling of cancelled context.
func TestRunTasksValidation_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cfg := TasksValidationConfig{
		Runner:    &mockTasksValidationRunner{},
		SpecFile:  "/path/to/spec.md",
		TasksFile: "/path/to/tasks.md",
	}

	result := RunTasksValidation(ctx, cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
}

// TestRunTasksValidation_UnknownVerdict tests handling of unknown verdict.
func TestRunTasksValidation_UnknownVerdict(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.md")
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	require.NoError(t, os.WriteFile(specFile, []byte("# Spec"), 0644))
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))

	runner := &mockTasksValidationRunner{
		output: `{"RALPH_TASKS_VALIDATION": {"verdict": "UNKNOWN", "feedback": "Unknown state"}}`,
	}

	cfg := TasksValidationConfig{
		Runner:    runner,
		SpecFile:  specFile,
		TasksFile: tasksFile,
	}

	result := RunTasksValidation(context.Background(), cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Contains(t, result.Feedback, "unknown tasks validation verdict")
}

// TestRunTasksValidation_NoVerdictFound tests handling when no verdict is parsed.
func TestRunTasksValidation_NoVerdictFound(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.md")
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	require.NoError(t, os.WriteFile(specFile, []byte("# Spec"), 0644))
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))

	runner := &mockTasksValidationRunner{
		output: "Just some text without any verdict",
	}

	cfg := TasksValidationConfig{
		Runner:    runner,
		SpecFile:  specFile,
		TasksFile: tasksFile,
	}

	result := RunTasksValidation(context.Background(), cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Contains(t, result.Feedback, "no tasks validation verdict found")
}

// mockTasksValidationDeleteRunner succeeds but removes the output file.
type mockTasksValidationDeleteRunner struct{}

func (m *mockTasksValidationDeleteRunner) Run(ctx context.Context, promptPath string, outputPath string) error {
	os.Remove(outputPath)
	return nil
}

// TestRunTasksValidation_ReadFileError tests handling when the output file cannot be read.
func TestRunTasksValidation_ReadFileError(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.md")
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	require.NoError(t, os.WriteFile(specFile, []byte("# Spec"), 0644))
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))

	runner := &mockTasksValidationDeleteRunner{}

	cfg := TasksValidationConfig{
		Runner:    runner,
		SpecFile:  specFile,
		TasksFile: tasksFile,
	}

	result := RunTasksValidation(context.Background(), cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Contains(t, result.Feedback, "failed to read tasks validation output")
}

// TestRunTasksValidation_ParseError tests handling when output contains malformed JSON with key.
func TestRunTasksValidation_ParseError(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.md")
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	require.NoError(t, os.WriteFile(specFile, []byte("# Spec"), 0644))
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))

	runner := &mockTasksValidationRunner{
		output: `RALPH_TASKS_VALIDATION {broken json {{`,
	}

	cfg := TasksValidationConfig{
		Runner:    runner,
		SpecFile:  specFile,
		TasksFile: tasksFile,
	}

	result := RunTasksValidation(context.Background(), cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Contains(t, result.Feedback, "failed to parse tasks validation")
}

// TestRunTasksValidation_PromptWriteError tests handling when prompt file cannot be written.
func TestRunTasksValidation_PromptWriteError(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.md")
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	require.NoError(t, os.WriteFile(specFile, []byte("# Spec"), 0644))
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))

	readOnlyDir := filepath.Join(tmpDir, "readonly")
	require.NoError(t, os.MkdirAll(readOnlyDir, 0555))
	origTmpDir := os.Getenv("TMPDIR")
	os.Setenv("TMPDIR", readOnlyDir)
	defer func() {
		if origTmpDir != "" {
			os.Setenv("TMPDIR", origTmpDir)
		} else {
			os.Unsetenv("TMPDIR")
		}
	}()

	runner := &mockTasksValidationRunner{
		output: "irrelevant",
	}

	cfg := TasksValidationConfig{
		Runner:    runner,
		SpecFile:  specFile,
		TasksFile: tasksFile,
	}

	result := RunTasksValidation(context.Background(), cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Contains(t, result.Feedback, "failed to write prompt")
}

// TestRunTasksValidation_RunnerError tests handling when the AI runner returns an error.
func TestRunTasksValidation_RunnerError(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.md")
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	require.NoError(t, os.WriteFile(specFile, []byte("# Spec"), 0644))
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))

	runner := &mockTasksValidationRunner{
		err: assert.AnError,
	}

	cfg := TasksValidationConfig{
		Runner:    runner,
		SpecFile:  specFile,
		TasksFile: tasksFile,
	}

	result := RunTasksValidation(context.Background(), cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Contains(t, result.Feedback, "tasks validation AI error")
}
