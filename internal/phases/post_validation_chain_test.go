package phases

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CodexForgeBR/cli-tools/internal/exitcode"
)

// TestRunPostValidationChain_SuccessFlow verifies complete success path
func TestRunPostValidationChain_SuccessFlow(t *testing.T) {
	// Setup: cross-val confirms, final-plan approves
	crossValRunner := &MockAIRunner{
		OutputData: makeCrossValidationJSON("CONFIRMED", "Cross validation passed"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeFinalPlanValidationJSON("APPROVE", "Final plan validated"),
	}

	config := PostValidationConfig{
		CrossValRunner:   crossValRunner,
		FinalPlanRunner:  finalPlanRunner,
		CrossValEnabled:  true,
		FinalPlanEnabled: true,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "success", result.Action, "both confirmations should lead to success")
	assert.Equal(t, exitcode.Success, result.ExitCode)
	assert.Empty(t, result.Feedback, "no feedback on success")
}

// TestRunPostValidationChain_CrossValReject verifies cross-val rejection returns to impl
func TestRunPostValidationChain_CrossValReject(t *testing.T) {
	crossValRunner := &MockAIRunner{
		OutputData: makeCrossValidationJSON("REJECTED", "Cross validation found issues"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeFinalPlanValidationJSON("APPROVE", "Should not reach this"),
	}

	config := PostValidationConfig{
		CrossValRunner:   crossValRunner,
		FinalPlanRunner:  finalPlanRunner,
		CrossValEnabled:  true,
		FinalPlanEnabled: true,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "continue", result.Action, "cross-val reject should continue impl loop")
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "Cross validation found issues", result.Feedback)
	assert.Equal(t, 1, crossValRunner.CallCount, "cross-val should be called")
	assert.Equal(t, 0, finalPlanRunner.CallCount, "final-plan should NOT be called after cross-val reject")
}

// TestRunPostValidationChain_FinalPlanReject verifies final-plan rejection returns to impl
func TestRunPostValidationChain_FinalPlanReject(t *testing.T) {
	crossValRunner := &MockAIRunner{
		OutputData: makeCrossValidationJSON("CONFIRMED", "Cross validation passed"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeFinalPlanValidationJSON("REJECT", "Implementation doesn't match original plan"),
	}

	config := PostValidationConfig{
		CrossValRunner:   crossValRunner,
		FinalPlanRunner:  finalPlanRunner,
		CrossValEnabled:  true,
		FinalPlanEnabled: true,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "continue", result.Action, "final-plan reject should continue impl loop")
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "Implementation doesn't match original plan", result.Feedback)
	assert.Equal(t, 1, crossValRunner.CallCount, "cross-val should be called")
	assert.Equal(t, 1, finalPlanRunner.CallCount, "final-plan should be called after cross-val pass")
}

// TestRunPostValidationChain_CrossValDisabled verifies skipping cross-val when disabled
func TestRunPostValidationChain_CrossValDisabled(t *testing.T) {
	crossValRunner := &MockAIRunner{
		OutputData: makeCrossValidationJSON("CONFIRMED", "Should not be called"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeFinalPlanValidationJSON("APPROVE", "Final plan validated"),
	}

	config := PostValidationConfig{
		CrossValRunner:   crossValRunner,
		FinalPlanRunner:  finalPlanRunner,
		CrossValEnabled:  false,
		FinalPlanEnabled: true,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "success", result.Action, "should succeed with only final-plan")
	assert.Equal(t, exitcode.Success, result.ExitCode)
	assert.Equal(t, 0, crossValRunner.CallCount, "cross-val should NOT be called when disabled")
	assert.Equal(t, 1, finalPlanRunner.CallCount, "final-plan should be called")
}

// TestRunPostValidationChain_FinalPlanDisabled verifies skipping final-plan when disabled
func TestRunPostValidationChain_FinalPlanDisabled(t *testing.T) {
	crossValRunner := &MockAIRunner{
		OutputData: makeCrossValidationJSON("CONFIRMED", "Cross validation passed"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeFinalPlanValidationJSON("APPROVE", "Should not be called"),
	}

	config := PostValidationConfig{
		CrossValRunner:   crossValRunner,
		FinalPlanRunner:  finalPlanRunner,
		CrossValEnabled:  true,
		FinalPlanEnabled: false,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "success", result.Action, "should succeed with only cross-val")
	assert.Equal(t, exitcode.Success, result.ExitCode)
	assert.Equal(t, 1, crossValRunner.CallCount, "cross-val should be called")
	assert.Equal(t, 0, finalPlanRunner.CallCount, "final-plan should NOT be called when disabled")
}

// TestRunPostValidationChain_BothDisabled verifies immediate success when both disabled
func TestRunPostValidationChain_BothDisabled(t *testing.T) {
	crossValRunner := &MockAIRunner{
		OutputData: makeCrossValidationJSON("CONFIRMED", "Should not be called"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeFinalPlanValidationJSON("APPROVE", "Should not be called"),
	}

	config := PostValidationConfig{
		CrossValRunner:   crossValRunner,
		FinalPlanRunner:  finalPlanRunner,
		CrossValEnabled:  false,
		FinalPlanEnabled: false,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "success", result.Action, "should succeed immediately when both disabled")
	assert.Equal(t, exitcode.Success, result.ExitCode)
	assert.Equal(t, 0, crossValRunner.CallCount, "cross-val should NOT be called")
	assert.Equal(t, 0, finalPlanRunner.CallCount, "final-plan should NOT be called")
}

// TestRunPostValidationChain_CrossValUnknownVerdict verifies unknown verdict exits with error
func TestRunPostValidationChain_CrossValUnknownVerdict(t *testing.T) {
	crossValRunner := &MockAIRunner{
		OutputData: makeCrossValidationJSON("ESCALATE", "Need human review for security"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeFinalPlanValidationJSON("APPROVE", "Should not reach this"),
	}

	config := PostValidationConfig{
		CrossValRunner:   crossValRunner,
		FinalPlanRunner:  finalPlanRunner,
		CrossValEnabled:  true,
		FinalPlanEnabled: true,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "exit", result.Action, "unknown verdict should exit with error")
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Equal(t, 0, finalPlanRunner.CallCount, "final-plan should NOT be called after unknown verdict")
}

// TestRunPostValidationChain_FinalPlanUnknownVerdict verifies unknown verdict exits with error
func TestRunPostValidationChain_FinalPlanUnknownVerdict(t *testing.T) {
	crossValRunner := &MockAIRunner{
		OutputData: makeCrossValidationJSON("CONFIRMED", "Cross validation passed"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeFinalPlanValidationJSON("ESCALATE", "Implementation deviates from original plan"),
	}

	config := PostValidationConfig{
		CrossValRunner:   crossValRunner,
		FinalPlanRunner:  finalPlanRunner,
		CrossValEnabled:  true,
		FinalPlanEnabled: true,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "exit", result.Action, "unknown verdict should exit with error")
	assert.Equal(t, exitcode.Error, result.ExitCode)
	assert.Equal(t, 1, crossValRunner.CallCount, "cross-val should be called first")
	assert.Equal(t, 1, finalPlanRunner.CallCount, "final-plan should be called after cross-val")
}

// TestRunPostValidationChain_ContextCancellation verifies context cancellation handling
func TestRunPostValidationChain_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	crossValRunner := &MockAIRunner{
		OutputData: makeCrossValidationJSON("CONFIRMED", "Should not complete"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeFinalPlanValidationJSON("APPROVE", "Should not complete"),
	}

	config := PostValidationConfig{
		CrossValRunner:   crossValRunner,
		FinalPlanRunner:  finalPlanRunner,
		CrossValEnabled:  true,
		FinalPlanEnabled: true,
	}

	result := RunPostValidationChain(ctx, config)

	// Should handle context cancellation gracefully
	assert.Equal(t, "exit", result.Action, "cancelled context should exit")
}

// TestRunPostValidationChain_RunnerErrors verifies error handling
func TestRunPostValidationChain_RunnerErrors(t *testing.T) {
	t.Run("cross-val runner error", func(t *testing.T) {
		crossValRunner := &MockAIRunner{
			Err: assert.AnError,
		}

		finalPlanRunner := &MockAIRunner{
			OutputData: makeFinalPlanValidationJSON("APPROVE", "Should not reach"),
		}

		config := PostValidationConfig{
			CrossValRunner:   crossValRunner,
			FinalPlanRunner:  finalPlanRunner,
			CrossValEnabled:  true,
			FinalPlanEnabled: true,
		}

		ctx := context.Background()
		result := RunPostValidationChain(ctx, config)

		assert.Equal(t, "exit", result.Action, "runner error should cause exit")
		assert.NotEqual(t, exitcode.Success, result.ExitCode)
		assert.Equal(t, 0, finalPlanRunner.CallCount, "final-plan should not run after error")
	})

	t.Run("final-plan runner error", func(t *testing.T) {
		crossValRunner := &MockAIRunner{
			OutputData: makeCrossValidationJSON("CONFIRMED", "Cross validation passed"),
		}

		finalPlanRunner := &MockAIRunner{
			Err: assert.AnError,
		}

		config := PostValidationConfig{
			CrossValRunner:   crossValRunner,
			FinalPlanRunner:  finalPlanRunner,
			CrossValEnabled:  true,
			FinalPlanEnabled: true,
		}

		ctx := context.Background()
		result := RunPostValidationChain(ctx, config)

		assert.Equal(t, "exit", result.Action, "runner error should cause exit")
		assert.NotEqual(t, exitcode.Success, result.ExitCode)
	})
}

// TestRunPostValidationChain_ComplexSequence verifies complex decision sequences
func TestRunPostValidationChain_ComplexSequence(t *testing.T) {
	tests := []struct {
		name               string
		crossValVerdict    string
		crossValFeedback   string
		finalPlanVerdict   string
		finalPlanFeedback  string
		crossValEnabled    bool
		finalPlanEnabled   bool
		expectedAction     string
		expectedExitCode   int
		expectedFeedback   string
		crossValCallCount  int
		finalPlanCallCount int
	}{
		{
			name:               "both pass",
			crossValVerdict:    "CONFIRMED",
			crossValFeedback:   "",
			finalPlanVerdict:   "APPROVE",
			finalPlanFeedback:  "",
			crossValEnabled:    true,
			finalPlanEnabled:   true,
			expectedAction:     "success",
			expectedExitCode:   exitcode.Success,
			expectedFeedback:   "",
			crossValCallCount:  1,
			finalPlanCallCount: 1,
		},
		{
			name:               "cross-val rejected",
			crossValVerdict:    "REJECTED",
			crossValFeedback:   "Fix bugs",
			finalPlanVerdict:   "APPROVE",
			finalPlanFeedback:  "",
			crossValEnabled:    true,
			finalPlanEnabled:   true,
			expectedAction:     "continue",
			expectedExitCode:   0,
			expectedFeedback:   "Fix bugs",
			crossValCallCount:  1,
			finalPlanCallCount: 0,
		},
		{
			name:               "final-plan rejected",
			crossValVerdict:    "CONFIRMED",
			crossValFeedback:   "",
			finalPlanVerdict:   "REJECT",
			finalPlanFeedback:  "Align with plan",
			crossValEnabled:    true,
			finalPlanEnabled:   true,
			expectedAction:     "continue",
			expectedExitCode:   0,
			expectedFeedback:   "Align with plan",
			crossValCallCount:  1,
			finalPlanCallCount: 1,
		},
		{
			name:               "only cross-val enabled and confirmed",
			crossValVerdict:    "CONFIRMED",
			crossValFeedback:   "",
			finalPlanVerdict:   "APPROVE",
			finalPlanFeedback:  "",
			crossValEnabled:    true,
			finalPlanEnabled:   false,
			expectedAction:     "success",
			expectedExitCode:   exitcode.Success,
			expectedFeedback:   "",
			crossValCallCount:  1,
			finalPlanCallCount: 0,
		},
		{
			name:               "only final-plan enabled and approved",
			crossValVerdict:    "CONFIRMED",
			crossValFeedback:   "",
			finalPlanVerdict:   "APPROVE",
			finalPlanFeedback:  "",
			crossValEnabled:    false,
			finalPlanEnabled:   true,
			expectedAction:     "success",
			expectedExitCode:   exitcode.Success,
			expectedFeedback:   "",
			crossValCallCount:  0,
			finalPlanCallCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crossValRunner := &MockAIRunner{
				OutputData: makeCrossValidationJSON(tt.crossValVerdict, tt.crossValFeedback),
			}

			finalPlanRunner := &MockAIRunner{
				OutputData: makeFinalPlanValidationJSON(tt.finalPlanVerdict, tt.finalPlanFeedback),
			}

			config := PostValidationConfig{
				CrossValRunner:   crossValRunner,
				FinalPlanRunner:  finalPlanRunner,
				CrossValEnabled:  tt.crossValEnabled,
				FinalPlanEnabled: tt.finalPlanEnabled,
			}

			ctx := context.Background()
			result := RunPostValidationChain(ctx, config)

			assert.Equal(t, tt.expectedAction, result.Action, "action should match")
			assert.Equal(t, tt.expectedExitCode, result.ExitCode, "exit code should match")
			assert.Equal(t, tt.expectedFeedback, result.Feedback, "feedback should match")
			assert.Equal(t, tt.crossValCallCount, crossValRunner.CallCount,
				"cross-val call count should match")
			assert.Equal(t, tt.finalPlanCallCount, finalPlanRunner.CallCount,
				"final-plan call count should match")
		})
	}
}

// Helper functions

func makeCrossValidationJSON(verdict string, feedback string) string {
	data := map[string]interface{}{
		"RALPH_CROSS_VALIDATION": map[string]interface{}{
			"verdict":  verdict,
			"feedback": feedback,
		},
	}
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func makeFinalPlanValidationJSON(verdict string, feedback string) string {
	data := map[string]interface{}{
		"RALPH_FINAL_PLAN_VALIDATION": map[string]interface{}{
			"verdict":  verdict,
			"feedback": feedback,
		},
	}
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

// mockDeleteOutputRunner is a mock that succeeds but removes the output file to trigger ReadFile errors.
type mockDeleteOutputRunner struct {
	CallCount int
}

func (m *mockDeleteOutputRunner) Run(ctx context.Context, prompt string, outputPath string) error {
	m.CallCount++
	// Remove the output file if it exists to guarantee ReadFile fails
	os.Remove(outputPath)
	return nil
}

// TestRunPostValidationChain_CrossValReadFileError tests runCrossValidation when output file cannot be read.
func TestRunPostValidationChain_CrossValReadFileError(t *testing.T) {
	crossValRunner := &mockDeleteOutputRunner{}

	config := PostValidationConfig{
		CrossValRunner:   crossValRunner,
		CrossValEnabled:  true,
		FinalPlanEnabled: false,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "exit", result.Action, "should exit when output file cannot be read")
	assert.Equal(t, exitcode.Error, result.ExitCode)
}

// TestRunPostValidationChain_CrossValParseError tests runCrossValidation when output is unparseable JSON.
func TestRunPostValidationChain_CrossValParseError(t *testing.T) {
	crossValRunner := &MockAIRunner{
		OutputData: "this is not valid json at all {{{",
	}

	config := PostValidationConfig{
		CrossValRunner:   crossValRunner,
		CrossValEnabled:  true,
		FinalPlanEnabled: false,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	// ParseCrossValidation should return nil, nil for unrecognized text (no RALPH_CROSS_VALIDATION found)
	// which leads to the "parsed == nil" branch
	assert.Equal(t, "exit", result.Action, "should exit when output has no validation block")
	assert.Equal(t, exitcode.Error, result.ExitCode)
}

// TestRunPostValidationChain_CrossValNilParsed tests runCrossValidation when parser returns nil.
func TestRunPostValidationChain_CrossValNilParsed(t *testing.T) {
	// Output with no RALPH_CROSS_VALIDATION block → parser returns nil
	crossValRunner := &MockAIRunner{
		OutputData: "Some text output without any JSON validation block",
	}

	config := PostValidationConfig{
		CrossValRunner:   crossValRunner,
		CrossValEnabled:  true,
		FinalPlanEnabled: false,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "exit", result.Action, "should exit when no validation verdict found")
	assert.Equal(t, exitcode.Error, result.ExitCode)
}

// TestRunPostValidationChain_FinalPlanReadFileError tests runFinalPlanValidation when output file cannot be read.
func TestRunPostValidationChain_FinalPlanReadFileError(t *testing.T) {
	finalPlanRunner := &mockDeleteOutputRunner{}

	config := PostValidationConfig{
		FinalPlanRunner:  finalPlanRunner,
		CrossValEnabled:  false,
		FinalPlanEnabled: true,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "exit", result.Action, "should exit when output file cannot be read")
	assert.Equal(t, exitcode.Error, result.ExitCode)
}

// TestRunPostValidationChain_FinalPlanParseError tests runFinalPlanValidation when output is unparseable.
func TestRunPostValidationChain_FinalPlanParseError(t *testing.T) {
	finalPlanRunner := &MockAIRunner{
		OutputData: "not json {{{",
	}

	config := PostValidationConfig{
		FinalPlanRunner:  finalPlanRunner,
		CrossValEnabled:  false,
		FinalPlanEnabled: true,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "exit", result.Action, "should exit when output cannot be parsed")
	assert.Equal(t, exitcode.Error, result.ExitCode)
}

// TestRunPostValidationChain_FinalPlanNilParsed tests runFinalPlanValidation when parser returns nil.
func TestRunPostValidationChain_FinalPlanNilParsed(t *testing.T) {
	finalPlanRunner := &MockAIRunner{
		OutputData: "Some text output without any JSON validation block",
	}

	config := PostValidationConfig{
		FinalPlanRunner:  finalPlanRunner,
		CrossValEnabled:  false,
		FinalPlanEnabled: true,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "exit", result.Action, "should exit when no validation verdict found")
	assert.Equal(t, exitcode.Error, result.ExitCode)
}

// TestRunPostValidationChain_CrossValMalformedJSON tests runCrossValidation when output has malformed JSON with key.
func TestRunPostValidationChain_CrossValMalformedJSON(t *testing.T) {
	crossValRunner := &MockAIRunner{
		OutputData: `RALPH_CROSS_VALIDATION {broken json {{`,
	}

	config := PostValidationConfig{
		CrossValRunner:   crossValRunner,
		CrossValEnabled:  true,
		FinalPlanEnabled: false,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "exit", result.Action, "should exit on parse error")
	assert.Equal(t, exitcode.Error, result.ExitCode)
}

// TestRunPostValidationChain_FinalPlanMalformedJSON tests runFinalPlanValidation when output has malformed JSON.
func TestRunPostValidationChain_FinalPlanMalformedJSON(t *testing.T) {
	finalPlanRunner := &MockAIRunner{
		OutputData: `RALPH_FINAL_PLAN_VALIDATION {broken json {{`,
	}

	config := PostValidationConfig{
		FinalPlanRunner:  finalPlanRunner,
		CrossValEnabled:  false,
		FinalPlanEnabled: true,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "exit", result.Action, "should exit on parse error")
	assert.Equal(t, exitcode.Error, result.ExitCode)
}

// TestRunPostValidationChain_FinalPlanContextCancelled tests runFinalPlanValidation with cancelled context.
func TestRunPostValidationChain_FinalPlanContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	finalPlanRunner := &MockAIRunner{
		OutputData: makeFinalPlanValidationJSON("APPROVE", ""),
	}

	config := PostValidationConfig{
		FinalPlanRunner:  finalPlanRunner,
		CrossValEnabled:  false,
		FinalPlanEnabled: true,
	}

	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "exit", result.Action, "should exit on context cancellation")
	assert.Equal(t, exitcode.Error, result.ExitCode)
}
