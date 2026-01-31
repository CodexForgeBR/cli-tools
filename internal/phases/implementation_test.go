package phases

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAIRunner is a test implementation of the AIRunner interface
type MockAIRunner struct {
	CalledWith   string
	OutputData   string
	OutputPath   string
	Err          error
	CallCount    int
	PromptLog    []string
	OutputPaths  []string
}

func (m *MockAIRunner) Run(ctx context.Context, prompt string, outputPath string) error {
	m.CallCount++
	m.CalledWith = prompt
	m.OutputPath = outputPath
	m.PromptLog = append(m.PromptLog, prompt)
	m.OutputPaths = append(m.OutputPaths, outputPath)

	if m.OutputData != "" {
		err := os.WriteFile(outputPath, []byte(m.OutputData), 0644)
		if err != nil {
			return err
		}
	}

	return m.Err
}

// TestRunImplementationPhase_FirstIteration verifies first iteration uses first prompt
func TestRunImplementationPhase_FirstIteration(t *testing.T) {
	tmpDir := t.TempDir()
	iterationDir := filepath.Join(tmpDir, "iteration-1")
	require.NoError(t, os.MkdirAll(iterationDir, 0755))

	outputPath := filepath.Join(iterationDir, "implementation.md")

	mockRunner := &MockAIRunner{
		OutputData: "Implementation output for iteration 1",
	}

	config := ImplementationConfig{
		Runner:        mockRunner,
		Iteration:     1,
		OutputPath:    outputPath,
		FirstPrompt:   "This is the first iteration prompt",
		ContinuePrompt: "This is the continue prompt",
	}

	ctx := context.Background()
	err := RunImplementationPhase(ctx, config)

	require.NoError(t, err)
	assert.Equal(t, 1, mockRunner.CallCount, "runner should be called once")
	assert.Equal(t, "This is the first iteration prompt", mockRunner.CalledWith,
		"first iteration should use first prompt")
	assert.Equal(t, outputPath, mockRunner.OutputPath, "output path should match")

	// Verify output file was created
	assert.FileExists(t, outputPath)
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Equal(t, "Implementation output for iteration 1", string(content))
}

// TestRunImplementationPhase_SubsequentIteration verifies subsequent iterations use continue prompt
func TestRunImplementationPhase_SubsequentIteration(t *testing.T) {
	tmpDir := t.TempDir()
	iterationDir := filepath.Join(tmpDir, "iteration-5")
	require.NoError(t, os.MkdirAll(iterationDir, 0755))

	outputPath := filepath.Join(iterationDir, "implementation.md")

	mockRunner := &MockAIRunner{
		OutputData: "Implementation output for iteration 5",
	}

	config := ImplementationConfig{
		Runner:        mockRunner,
		Iteration:     5,
		OutputPath:    outputPath,
		FirstPrompt:   "This is the first iteration prompt",
		ContinuePrompt: "This is the continue prompt",
	}

	ctx := context.Background()
	err := RunImplementationPhase(ctx, config)

	require.NoError(t, err)
	assert.Equal(t, 1, mockRunner.CallCount, "runner should be called once")
	assert.Equal(t, "This is the continue prompt", mockRunner.CalledWith,
		"subsequent iteration should use continue prompt")
	assert.Equal(t, outputPath, mockRunner.OutputPath, "output path should match")

	// Verify output file was created
	assert.FileExists(t, outputPath)
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Equal(t, "Implementation output for iteration 5", string(content))
}

// TestRunImplementationPhase_IterationProgression verifies correct prompt selection across iterations
func TestRunImplementationPhase_IterationProgression(t *testing.T) {
	tests := []struct {
		iteration      int
		expectedPrompt string
	}{
		{1, "FIRST"},
		{2, "CONTINUE"},
		{3, "CONTINUE"},
		{10, "CONTINUE"},
		{20, "CONTINUE"},
	}

	for _, tt := range tests {
		t.Run(string(rune('0'+tt.iteration)), func(t *testing.T) {
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "output.md")

			mockRunner := &MockAIRunner{
				OutputData: "test output",
			}

			config := ImplementationConfig{
				Runner:        mockRunner,
				Iteration:     tt.iteration,
				OutputPath:    outputPath,
				FirstPrompt:   "FIRST",
				ContinuePrompt: "CONTINUE",
			}

			ctx := context.Background()
			err := RunImplementationPhase(ctx, config)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedPrompt, mockRunner.CalledWith,
				"iteration %d should use %s prompt", tt.iteration, tt.expectedPrompt)
		})
	}
}

// TestRunImplementationPhase_RunnerError verifies error handling when runner fails
func TestRunImplementationPhase_RunnerError(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.md")

	expectedErr := assert.AnError
	mockRunner := &MockAIRunner{
		Err: expectedErr,
	}

	config := ImplementationConfig{
		Runner:        mockRunner,
		Iteration:     1,
		OutputPath:    outputPath,
		FirstPrompt:   "First prompt",
		ContinuePrompt: "Continue prompt",
	}

	ctx := context.Background()
	err := RunImplementationPhase(ctx, config)

	require.Error(t, err)
	assert.Equal(t, expectedErr, err, "should return runner error")
	assert.Equal(t, 1, mockRunner.CallCount, "runner should have been called")
}

// TestRunImplementationPhase_ContextCancellation verifies context cancellation is respected
func TestRunImplementationPhase_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.md")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mockRunner := &MockAIRunner{
		OutputData: "should not be written",
	}

	config := ImplementationConfig{
		Runner:        mockRunner,
		Iteration:     1,
		OutputPath:    outputPath,
		FirstPrompt:   "First prompt",
		ContinuePrompt: "Continue prompt",
	}

	err := RunImplementationPhase(ctx, config)

	// Should respect context cancellation
	if err != nil {
		assert.Equal(t, context.Canceled, err, "should return context.Canceled error")
	}
}

// TestRunImplementationPhase_OutputPathCreation verifies output file is created correctly
func TestRunImplementationPhase_OutputPathCreation(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.md")

	expectedContent := "Test implementation output"
	mockRunner := &MockAIRunner{
		OutputData: expectedContent,
	}

	config := ImplementationConfig{
		Runner:        mockRunner,
		Iteration:     1,
		OutputPath:    outputPath,
		FirstPrompt:   "Test prompt",
		ContinuePrompt: "Continue",
	}

	ctx := context.Background()
	err := RunImplementationPhase(ctx, config)

	require.NoError(t, err)
	assert.FileExists(t, outputPath, "output file should be created")

	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Equal(t, expectedContent, string(content), "output content should match")
}

// TestRunImplementationPhase_LearningsExtraction verifies learnings are extracted from output
func TestRunImplementationPhase_LearningsExtraction(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.md")

	outputWithLearnings := `# Implementation

Some implementation details here.

## Learnings

- Important insight about the codebase
- Performance optimization opportunity discovered
- Architecture pattern that worked well
`

	mockRunner := &MockAIRunner{
		OutputData: outputWithLearnings,
	}

	config := ImplementationConfig{
		Runner:           mockRunner,
		Iteration:        1,
		OutputPath:       outputPath,
		FirstPrompt:      "Test prompt",
		ContinuePrompt:   "Continue",
		ExtractLearnings: true,
	}

	ctx := context.Background()
	result, err := RunImplementationPhaseWithLearnings(ctx, config)

	require.NoError(t, err)
	assert.NotEmpty(t, result.Learnings, "learnings should be extracted")
	assert.Contains(t, result.Learnings, "Important insight about the codebase")
	assert.Contains(t, result.Learnings, "Performance optimization opportunity")
	assert.Contains(t, result.Learnings, "Architecture pattern")
}

// TestRunImplementationPhase_NoLearnings verifies handling when no learnings section exists
func TestRunImplementationPhase_NoLearnings(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.md")

	outputWithoutLearnings := `# Implementation

Just implementation details, no learnings section.
`

	mockRunner := &MockAIRunner{
		OutputData: outputWithoutLearnings,
	}

	config := ImplementationConfig{
		Runner:           mockRunner,
		Iteration:        1,
		OutputPath:       outputPath,
		FirstPrompt:      "Test prompt",
		ContinuePrompt:   "Continue",
		ExtractLearnings: true,
	}

	ctx := context.Background()
	result, err := RunImplementationPhaseWithLearnings(ctx, config)

	require.NoError(t, err)
	assert.Empty(t, result.Learnings, "learnings should be empty when not present")
}

// TestRunImplementationPhase_MultipleIterations verifies multiple iterations work correctly
func TestRunImplementationPhase_MultipleIterations(t *testing.T) {
	tmpDir := t.TempDir()
	mockRunner := &MockAIRunner{}

	iterations := []int{1, 2, 3, 4, 5}
	for _, iter := range iterations {
		iterationDir := filepath.Join(tmpDir, "iteration-%d", string(rune('0'+iter)))
		require.NoError(t, os.MkdirAll(iterationDir, 0755))

		outputPath := filepath.Join(iterationDir, "implementation.md")
		mockRunner.OutputData = "Output for iteration " + string(rune('0'+iter))

		config := ImplementationConfig{
			Runner:        mockRunner,
			Iteration:     iter,
			OutputPath:    outputPath,
			FirstPrompt:   "FIRST",
			ContinuePrompt: "CONTINUE",
		}

		ctx := context.Background()
		err := RunImplementationPhase(ctx, config)
		require.NoError(t, err)
	}

	assert.Equal(t, len(iterations), mockRunner.CallCount,
		"runner should be called once per iteration")
	assert.Len(t, mockRunner.PromptLog, len(iterations),
		"should have prompt log for all iterations")

	// Verify first iteration used first prompt
	assert.Equal(t, "FIRST", mockRunner.PromptLog[0])

	// Verify subsequent iterations used continue prompt
	for i := 1; i < len(iterations); i++ {
		assert.Equal(t, "CONTINUE", mockRunner.PromptLog[i],
			"iteration %d should use continue prompt", iterations[i])
	}
}

// TestRunImplementationPhase_EmptyPrompts verifies handling of empty prompts
func TestRunImplementationPhase_EmptyPrompts(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.md")

	mockRunner := &MockAIRunner{
		OutputData: "output",
	}

	config := ImplementationConfig{
		Runner:        mockRunner,
		Iteration:     1,
		OutputPath:    outputPath,
		FirstPrompt:   "",
		ContinuePrompt: "",
	}

	ctx := context.Background()
	err := RunImplementationPhase(ctx, config)

	// Should still work with empty prompts (runner receives empty string)
	require.NoError(t, err)
	assert.Equal(t, "", mockRunner.CalledWith, "empty prompt should be passed to runner")
}

// TestRunImplementationPhase_LongPrompts verifies handling of very long prompts
func TestRunImplementationPhase_LongPrompts(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.md")

	// Create a very long prompt (10KB)
	longPrompt := ""
	for i := 0; i < 1000; i++ {
		longPrompt += "This is a very long prompt that tests handling of large prompt strings. "
	}

	mockRunner := &MockAIRunner{
		OutputData: "output",
	}

	config := ImplementationConfig{
		Runner:        mockRunner,
		Iteration:     1,
		OutputPath:    outputPath,
		FirstPrompt:   longPrompt,
		ContinuePrompt: "short",
	}

	ctx := context.Background()
	err := RunImplementationPhase(ctx, config)

	require.NoError(t, err)
	assert.Equal(t, longPrompt, mockRunner.CalledWith,
		"long prompt should be passed completely to runner")
	assert.Greater(t, len(mockRunner.CalledWith), 5000,
		"prompt should be very long")
}

// TestRunImplementationPhase_SpecialCharactersInPrompts verifies special characters are handled
func TestRunImplementationPhase_SpecialCharactersInPrompts(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.md")

	specialPrompt := "Prompt with special chars: @#$%^&*(){}[]|\\\"'<>?/~`"

	mockRunner := &MockAIRunner{
		OutputData: "output",
	}

	config := ImplementationConfig{
		Runner:        mockRunner,
		Iteration:     1,
		OutputPath:    outputPath,
		FirstPrompt:   specialPrompt,
		ContinuePrompt: "continue",
	}

	ctx := context.Background()
	err := RunImplementationPhase(ctx, config)

	require.NoError(t, err)
	assert.Equal(t, specialPrompt, mockRunner.CalledWith,
		"special characters should be preserved in prompt")
}
