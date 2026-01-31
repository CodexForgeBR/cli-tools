package phases

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CodexForgeBR/cli-tools/internal/exitcode"
)

// mockFinalPlanRunner is a mock AI runner for testing final plan validation.
type mockFinalPlanRunner struct {
	output string
	err    error
}

func (m *mockFinalPlanRunner) Run(ctx context.Context, promptPath string, outputPath string) error {
	if m.err != nil {
		return m.err
	}
	return os.WriteFile(outputPath, []byte(m.output), 0644)
}

// TestRunFinalPlanValidation_ConfirmedVerdict tests that CONFIRMED verdict leads to success.
func TestRunFinalPlanValidation_ConfirmedVerdict(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.md")
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	planFile := filepath.Join(tmpDir, "plan.md")
	require.NoError(t, os.WriteFile(specFile, []byte("# Spec"), 0644))
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))
	require.NoError(t, os.WriteFile(planFile, []byte("# Plan"), 0644))

	runner := &mockFinalPlanRunner{
		output: `Final plan validation complete:

` + "```json\n" + `{
  "RALPH_FINAL_PLAN_VALIDATION": {
    "verdict": "APPROVE",
    "feedback": "Plan correctly interprets spec and is ready for implementation"
  }
}
` + "```",
	}

	cfg := FinalPlanValidationConfig{
		Runner:    runner,
		SpecFile:  specFile,
		TasksFile: tasksFile,
		PlanFile:  planFile,
	}

	result := RunFinalPlanValidation(context.Background(), cfg)

	assert.Equal(t, "success", result.Action)
	assert.Equal(t, exitcode.Success, result.ExitCode)
	assert.Empty(t, result.Feedback)
}

// TestRunFinalPlanValidation_NotImplementedVerdict tests that NOT_IMPLEMENTED verdict leads to exit.
func TestRunFinalPlanValidation_NotImplementedVerdict(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.md")
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	planFile := filepath.Join(tmpDir, "plan.md")
	require.NoError(t, os.WriteFile(specFile, []byte("# Spec"), 0644))
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))
	require.NoError(t, os.WriteFile(planFile, []byte("# Plan"), 0644))

	runner := &mockFinalPlanRunner{
		output: `Final plan validation found issues:

` + "```json\n" + `{
  "RALPH_FINAL_PLAN_VALIDATION": {
    "verdict": "REJECT",
    "feedback": "Plan includes out-of-scope features not in spec"
  }
}
` + "```",
	}

	cfg := FinalPlanValidationConfig{
		Runner:    runner,
		SpecFile:  specFile,
		TasksFile: tasksFile,
		PlanFile:  planFile,
	}

	result := RunFinalPlanValidation(context.Background(), cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Contains(t, result.Feedback, "out-of-scope features")
}

// TestRunFinalPlanValidation_ContextCancelled tests handling of cancelled context.
func TestRunFinalPlanValidation_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cfg := FinalPlanValidationConfig{
		Runner:    &mockFinalPlanRunner{},
		SpecFile:  "/path/to/spec.md",
		TasksFile: "/path/to/tasks.md",
		PlanFile:  "/path/to/plan.md",
	}

	result := RunFinalPlanValidation(ctx, cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
}

// TestRunFinalPlanValidation_UnknownVerdict tests handling of unknown verdict.
func TestRunFinalPlanValidation_UnknownVerdict(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.md")
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	planFile := filepath.Join(tmpDir, "plan.md")
	require.NoError(t, os.WriteFile(specFile, []byte("# Spec"), 0644))
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))
	require.NoError(t, os.WriteFile(planFile, []byte("# Plan"), 0644))

	runner := &mockFinalPlanRunner{
		output: `{"RALPH_FINAL_PLAN_VALIDATION": {"verdict": "UNKNOWN", "feedback": "Unknown state"}}`,
	}

	cfg := FinalPlanValidationConfig{
		Runner:    runner,
		SpecFile:  specFile,
		TasksFile: tasksFile,
		PlanFile:  planFile,
	}

	result := RunFinalPlanValidation(context.Background(), cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Contains(t, result.Feedback, "unknown final plan validation verdict")
}

// TestRunFinalPlanValidation_NoVerdictFound tests handling when no verdict is parsed.
func TestRunFinalPlanValidation_NoVerdictFound(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.md")
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	planFile := filepath.Join(tmpDir, "plan.md")
	require.NoError(t, os.WriteFile(specFile, []byte("# Spec"), 0644))
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))
	require.NoError(t, os.WriteFile(planFile, []byte("# Plan"), 0644))

	runner := &mockFinalPlanRunner{
		output: "Just some text without any verdict",
	}

	cfg := FinalPlanValidationConfig{
		Runner:    runner,
		SpecFile:  specFile,
		TasksFile: tasksFile,
		PlanFile:  planFile,
	}

	result := RunFinalPlanValidation(context.Background(), cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Contains(t, result.Feedback, "no final plan validation verdict found")
}

// mockFinalPlanDeleteRunner succeeds but removes the output file.
type mockFinalPlanDeleteRunner struct{}

func (m *mockFinalPlanDeleteRunner) Run(ctx context.Context, promptPath string, outputPath string) error {
	os.Remove(outputPath)
	return nil
}

// TestRunFinalPlanValidation_ReadFileError tests handling when the output file cannot be read.
func TestRunFinalPlanValidation_ReadFileError(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.md")
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	planFile := filepath.Join(tmpDir, "plan.md")
	require.NoError(t, os.WriteFile(specFile, []byte("# Spec"), 0644))
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))
	require.NoError(t, os.WriteFile(planFile, []byte("# Plan"), 0644))

	runner := &mockFinalPlanDeleteRunner{}

	cfg := FinalPlanValidationConfig{
		Runner:    runner,
		SpecFile:  specFile,
		TasksFile: tasksFile,
		PlanFile:  planFile,
	}

	result := RunFinalPlanValidation(context.Background(), cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Contains(t, result.Feedback, "failed to read final plan validation output")
}

// TestRunFinalPlanValidation_ParseError tests handling when output contains malformed JSON with key.
func TestRunFinalPlanValidation_ParseError(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.md")
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	planFile := filepath.Join(tmpDir, "plan.md")
	require.NoError(t, os.WriteFile(specFile, []byte("# Spec"), 0644))
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))
	require.NoError(t, os.WriteFile(planFile, []byte("# Plan"), 0644))

	runner := &mockFinalPlanRunner{
		output: `RALPH_FINAL_PLAN_VALIDATION {broken json {{`,
	}

	cfg := FinalPlanValidationConfig{
		Runner:    runner,
		SpecFile:  specFile,
		TasksFile: tasksFile,
		PlanFile:  planFile,
	}

	result := RunFinalPlanValidation(context.Background(), cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Contains(t, result.Feedback, "failed to parse final plan validation")
}

// TestRunFinalPlanValidation_OutputWriteError tests handling when output file cannot be written.
func TestRunFinalPlanValidation_OutputWriteError(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.md")
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	planFile := filepath.Join(tmpDir, "plan.md")
	require.NoError(t, os.WriteFile(specFile, []byte("# Spec"), 0644))
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))
	require.NoError(t, os.WriteFile(planFile, []byte("# Plan"), 0644))

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

	runner := &mockFinalPlanRunner{
		output: "irrelevant",
	}

	cfg := FinalPlanValidationConfig{
		Runner:    runner,
		SpecFile:  specFile,
		TasksFile: tasksFile,
		PlanFile:  planFile,
	}

	result := RunFinalPlanValidation(context.Background(), cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Contains(t, result.Feedback, "final plan validation AI error")
}

// TestRunFinalPlanValidation_RunnerError tests handling when the AI runner returns an error.
func TestRunFinalPlanValidation_RunnerError(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.md")
	tasksFile := filepath.Join(tmpDir, "tasks.md")
	planFile := filepath.Join(tmpDir, "plan.md")
	require.NoError(t, os.WriteFile(specFile, []byte("# Spec"), 0644))
	require.NoError(t, os.WriteFile(tasksFile, []byte("# Tasks"), 0644))
	require.NoError(t, os.WriteFile(planFile, []byte("# Plan"), 0644))

	runner := &mockFinalPlanRunner{
		err: assert.AnError,
	}

	cfg := FinalPlanValidationConfig{
		Runner:    runner,
		SpecFile:  specFile,
		TasksFile: tasksFile,
		PlanFile:  planFile,
	}

	result := RunFinalPlanValidation(context.Background(), cfg)

	assert.Equal(t, "exit", result.Action)
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Contains(t, result.Feedback, "final plan validation AI error")
}
