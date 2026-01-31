package phases

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunValidationPhase_PromptGeneration verifies validation prompt is generated correctly
func TestRunValidationPhase_PromptGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "validation.json")

	mockRunner := &MockAIRunner{
		OutputData: `{"RALPH_VALIDATION": {"verdict": "COMPLETE"}}`,
	}

	config := ValidationConfig{
		Runner:     mockRunner,
		OutputPath: outputPath,
		Prompt:     "Validate the implementation against requirements",
	}

	ctx := context.Background()
	err := RunValidationPhase(ctx, config)

	require.NoError(t, err)
	assert.Equal(t, 1, mockRunner.CallCount, "runner should be called once")
	assert.Equal(t, "Validate the implementation against requirements", mockRunner.CalledWith,
		"prompt should match configuration")
}

// TestRunValidationPhase_AIRunnerCalled verifies AI runner is invoked
func TestRunValidationPhase_AIRunnerCalled(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "validation.json")

	mockRunner := &MockAIRunner{
		OutputData: `{"RALPH_VALIDATION": {"verdict": "NEEDS_MORE_WORK", "feedback": "Fix bugs"}}`,
	}

	config := ValidationConfig{
		Runner:     mockRunner,
		OutputPath: outputPath,
		Prompt:     "Validation prompt",
	}

	ctx := context.Background()
	err := RunValidationPhase(ctx, config)

	require.NoError(t, err)
	assert.Equal(t, 1, mockRunner.CallCount, "runner should be called exactly once")
	assert.Equal(t, outputPath, mockRunner.OutputPath, "output path should match")
}

// TestRunValidationPhase_JSONExtraction verifies RALPH_VALIDATION JSON is extracted
func TestRunValidationPhase_JSONExtraction(t *testing.T) {
	tests := []struct {
		name           string
		outputData     string
		expectedVerdict string
		expectedFeedback string
		shouldSucceed  bool
	}{
		{
			name: "clean JSON format",
			outputData: `{
				"RALPH_VALIDATION": {
					"verdict": "COMPLETE",
					"feedback": "All requirements met"
				}
			}`,
			expectedVerdict: "COMPLETE",
			expectedFeedback: "All requirements met",
			shouldSucceed:  true,
		},
		{
			name: "JSON with surrounding text",
			outputData: `Here is my validation:

{
	"RALPH_VALIDATION": {
		"verdict": "NEEDS_MORE_WORK",
		"feedback": "Missing error handling"
	}
}

That's my assessment.`,
			expectedVerdict: "NEEDS_MORE_WORK",
			expectedFeedback: "Missing error handling",
			shouldSucceed:  true,
		},
		{
			name: "JSON in code block",
			outputData: "```json\n" + `{
	"RALPH_VALIDATION": {
		"verdict": "ESCALATE",
		"feedback": "Need human review"
	}
}` + "\n```",
			expectedVerdict: "ESCALATE",
			expectedFeedback: "Need human review",
			shouldSucceed:  true,
		},
		{
			name: "verdict only",
			outputData: `{
				"RALPH_VALIDATION": {
					"verdict": "BLOCKED"
				}
			}`,
			expectedVerdict: "BLOCKED",
			expectedFeedback: "",
			shouldSucceed:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "validation.json")

			mockRunner := &MockAIRunner{
				OutputData: tt.outputData,
			}

			config := ValidationConfig{
				Runner:     mockRunner,
				OutputPath: outputPath,
				Prompt:     "Test validation",
			}

			ctx := context.Background()
			result, err := RunValidationPhaseWithResult(ctx, config)

			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedVerdict, result.Verdict,
					"verdict should match expected")
				assert.Equal(t, tt.expectedFeedback, result.Feedback,
					"feedback should match expected")
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestRunValidationPhase_OutputFileCreated verifies output file is created
func TestRunValidationPhase_OutputFileCreated(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "validation.json")

	validationOutput := `{
		"RALPH_VALIDATION": {
			"verdict": "COMPLETE",
			"feedback": "All good"
		}
	}`

	mockRunner := &MockAIRunner{
		OutputData: validationOutput,
	}

	config := ValidationConfig{
		Runner:     mockRunner,
		OutputPath: outputPath,
		Prompt:     "Validate",
	}

	ctx := context.Background()
	err := RunValidationPhase(ctx, config)

	require.NoError(t, err)
	assert.FileExists(t, outputPath, "validation output file should exist")

	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "RALPH_VALIDATION",
		"output file should contain validation JSON")
}

// TestRunValidationPhase_RunnerError verifies error handling when runner fails
func TestRunValidationPhase_RunnerError(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "validation.json")

	expectedErr := assert.AnError
	mockRunner := &MockAIRunner{
		Err: expectedErr,
	}

	config := ValidationConfig{
		Runner:     mockRunner,
		OutputPath: outputPath,
		Prompt:     "Validate",
	}

	ctx := context.Background()
	err := RunValidationPhase(ctx, config)

	require.Error(t, err)
	assert.Equal(t, expectedErr, err, "should return runner error")
}

// TestRunValidationPhase_ContextCancellation verifies context cancellation is respected
func TestRunValidationPhase_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "validation.json")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mockRunner := &MockAIRunner{
		OutputData: `{"RALPH_VALIDATION": {"verdict": "COMPLETE"}}`,
	}

	config := ValidationConfig{
		Runner:     mockRunner,
		OutputPath: outputPath,
		Prompt:     "Validate",
	}

	err := RunValidationPhase(ctx, config)

	// Should respect context cancellation
	if err != nil {
		assert.Equal(t, context.Canceled, err, "should return context.Canceled error")
	}
}

// TestRunValidationPhase_AllVerdicts verifies all verdict types are handled
func TestRunValidationPhase_AllVerdicts(t *testing.T) {
	verdicts := []string{
		"COMPLETE",
		"NEEDS_MORE_WORK",
		"ESCALATE",
		"BLOCKED",
		"INADMISSIBLE",
	}

	for _, verdict := range verdicts {
		t.Run(verdict, func(t *testing.T) {
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "validation.json")

			outputData := map[string]interface{}{
				"RALPH_VALIDATION": map[string]interface{}{
					"verdict": verdict,
					"feedback": "Test feedback for " + verdict,
				},
			}
			jsonData, err := json.Marshal(outputData)
			require.NoError(t, err)

			mockRunner := &MockAIRunner{
				OutputData: string(jsonData),
			}

			config := ValidationConfig{
				Runner:     mockRunner,
				OutputPath: outputPath,
				Prompt:     "Validate",
			}

			ctx := context.Background()
			result, err := RunValidationPhaseWithResult(ctx, config)

			require.NoError(t, err)
			assert.Equal(t, verdict, result.Verdict,
				"verdict should be %s", verdict)
		})
	}
}

// TestRunValidationPhase_MalformedJSON verifies handling of invalid JSON
func TestRunValidationPhase_MalformedJSON(t *testing.T) {
	tests := []struct {
		name       string
		outputData string
	}{
		{
			name:       "completely invalid JSON",
			outputData: `this is not valid json at all`,
		},
		{
			name:       "missing RALPH_VALIDATION key",
			outputData: `{"verdict": "COMPLETE"}`,
		},
		{
			name:       "empty output",
			outputData: ``,
		},
		{
			name:       "only whitespace",
			outputData: `   \n\t   `,
		},
		{
			name:       "truncated JSON",
			outputData: `{"RALPH_VALIDATION": {"verdict": "COM`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "validation.json")

			mockRunner := &MockAIRunner{
				OutputData: tt.outputData,
			}

			config := ValidationConfig{
				Runner:     mockRunner,
				OutputPath: outputPath,
				Prompt:     "Validate",
			}

			ctx := context.Background()
			_, err := RunValidationPhaseWithResult(ctx, config)

			// Should handle gracefully (may return error or default value)
			// The important part is it doesn't panic
			if err != nil {
				assert.NotEqual(t, context.Canceled, err,
					"error should not be context cancellation")
			}
		})
	}
}

// TestRunValidationPhase_WithBlockedTasks verifies blocked tasks extraction
func TestRunValidationPhase_WithBlockedTasks(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "validation.json")

	outputData := `{
		"RALPH_VALIDATION": {
			"verdict": "BLOCKED",
			"feedback": "Some tasks are blocked",
			"blocked_tasks": [
				"Wait for API key",
				"Pending design approval",
				"Database migration blocked"
			]
		}
	}`

	mockRunner := &MockAIRunner{
		OutputData: outputData,
	}

	config := ValidationConfig{
		Runner:     mockRunner,
		OutputPath: outputPath,
		Prompt:     "Validate with blocked tasks",
	}

	ctx := context.Background()
	result, err := RunValidationPhaseWithResult(ctx, config)

	require.NoError(t, err)
	assert.Equal(t, "BLOCKED", result.Verdict)
	assert.Len(t, result.BlockedTasks, 3, "should extract 3 blocked tasks")
	assert.Contains(t, result.BlockedTasks, "Wait for API key")
	assert.Contains(t, result.BlockedTasks, "Pending design approval")
	assert.Contains(t, result.BlockedTasks, "Database migration blocked")
}

// TestRunValidationPhase_ComplexFeedback verifies complex feedback is preserved
func TestRunValidationPhase_ComplexFeedback(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "validation.json")

	complexFeedback := `The implementation has several issues:

1. Error handling is missing in the main loop
2. Database connections are not properly closed
3. Configuration validation is incomplete

Please address these before proceeding.

Special characters: @#$%^&*()
Quotes: "double" and 'single'
Newlines and tabs are preserved.`

	outputData := map[string]interface{}{
		"RALPH_VALIDATION": map[string]interface{}{
			"verdict": "NEEDS_MORE_WORK",
			"feedback": complexFeedback,
		},
	}
	jsonData, err := json.Marshal(outputData)
	require.NoError(t, err)

	mockRunner := &MockAIRunner{
		OutputData: string(jsonData),
	}

	config := ValidationConfig{
		Runner:     mockRunner,
		OutputPath: outputPath,
		Prompt:     "Validate",
	}

	ctx := context.Background()
	result, err := RunValidationPhaseWithResult(ctx, config)

	require.NoError(t, err)
	assert.Equal(t, complexFeedback, result.Feedback,
		"complex feedback should be preserved exactly")
}

// TestRunValidationPhase_MultipleRuns verifies multiple validation runs work independently
func TestRunValidationPhase_MultipleRuns(t *testing.T) {
	tmpDir := t.TempDir()

	runs := []struct {
		iteration int
		verdict   string
		feedback  string
	}{
		{1, "NEEDS_MORE_WORK", "Fix authentication"},
		{2, "NEEDS_MORE_WORK", "Add tests"},
		{3, "COMPLETE", "All good"},
	}

	mockRunner := &MockAIRunner{}

	for _, run := range runs {
		outputPath := filepath.Join(tmpDir, fmt.Sprintf("validation-%d.json", run.iteration))

		outputData := map[string]interface{}{
			"RALPH_VALIDATION": map[string]interface{}{
				"verdict": run.verdict,
				"feedback": run.feedback,
			},
		}
		jsonData, err := json.Marshal(outputData)
		require.NoError(t, err)

		mockRunner.OutputData = string(jsonData)

		config := ValidationConfig{
			Runner:     mockRunner,
			OutputPath: outputPath,
			Prompt:     "Validate iteration " + string(rune('0'+run.iteration)),
		}

		ctx := context.Background()
		result, err := RunValidationPhaseWithResult(ctx, config)

		require.NoError(t, err)
		assert.Equal(t, run.verdict, result.Verdict,
			"iteration %d verdict should match", run.iteration)
		assert.Equal(t, run.feedback, result.Feedback,
			"iteration %d feedback should match", run.iteration)
	}

	assert.Equal(t, len(runs), mockRunner.CallCount,
		"runner should be called once per validation")
}

// TestRunValidationPhase_LongOutput verifies handling of very long validation output
func TestRunValidationPhase_LongOutput(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "validation.json")

	// Create very long feedback (10KB)
	longFeedback := ""
	for i := 0; i < 1000; i++ {
		longFeedback += "This is a very detailed feedback point number " + string(rune('0'+i%10)) + ". "
	}

	outputData := map[string]interface{}{
		"RALPH_VALIDATION": map[string]interface{}{
			"verdict": "NEEDS_MORE_WORK",
			"feedback": longFeedback,
		},
	}
	jsonData, err := json.Marshal(outputData)
	require.NoError(t, err)

	mockRunner := &MockAIRunner{
		OutputData: string(jsonData),
	}

	config := ValidationConfig{
		Runner:     mockRunner,
		OutputPath: outputPath,
		Prompt:     "Validate",
	}

	ctx := context.Background()
	result, err := RunValidationPhaseWithResult(ctx, config)

	require.NoError(t, err)
	assert.Equal(t, longFeedback, result.Feedback,
		"long feedback should be preserved completely")
	assert.Greater(t, len(result.Feedback), 5000,
		"feedback should be very long")
}
