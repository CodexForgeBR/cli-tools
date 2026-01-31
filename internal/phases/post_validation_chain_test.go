package phases

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/CodexForgeBR/cli-tools/internal/exitcode"
	"github.com/stretchr/testify/assert"
)

// TestRunPostValidationChain_SuccessFlow verifies complete success path
func TestRunPostValidationChain_SuccessFlow(t *testing.T) {
	// Setup: cross-val confirms, final-plan confirms
	crossValRunner := &MockAIRunner{
		OutputData: makeValidationJSON("COMPLETE", "Cross validation passed"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeValidationJSON("COMPLETE", "Final plan validated"),
	}

	config := PostValidationConfig{
		CrossValRunner:  crossValRunner,
		FinalPlanRunner: finalPlanRunner,
		CrossValEnabled: true,
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
		OutputData: makeValidationJSON("NEEDS_MORE_WORK", "Cross validation found issues"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeValidationJSON("COMPLETE", "Should not reach this"),
	}

	config := PostValidationConfig{
		CrossValRunner:  crossValRunner,
		FinalPlanRunner: finalPlanRunner,
		CrossValEnabled: true,
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
		OutputData: makeValidationJSON("COMPLETE", "Cross validation passed"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeValidationJSON("NEEDS_MORE_WORK", "Implementation doesn't match original plan"),
	}

	config := PostValidationConfig{
		CrossValRunner:  crossValRunner,
		FinalPlanRunner: finalPlanRunner,
		CrossValEnabled: true,
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
		OutputData: makeValidationJSON("COMPLETE", "Should not be called"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeValidationJSON("COMPLETE", "Final plan validated"),
	}

	config := PostValidationConfig{
		CrossValRunner:  crossValRunner,
		FinalPlanRunner: finalPlanRunner,
		CrossValEnabled: false,
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
		OutputData: makeValidationJSON("COMPLETE", "Cross validation passed"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeValidationJSON("COMPLETE", "Should not be called"),
	}

	config := PostValidationConfig{
		CrossValRunner:  crossValRunner,
		FinalPlanRunner: finalPlanRunner,
		CrossValEnabled: true,
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
		OutputData: makeValidationJSON("COMPLETE", "Should not be called"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeValidationJSON("COMPLETE", "Should not be called"),
	}

	config := PostValidationConfig{
		CrossValRunner:  crossValRunner,
		FinalPlanRunner: finalPlanRunner,
		CrossValEnabled: false,
		FinalPlanEnabled: false,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "success", result.Action, "should succeed immediately when both disabled")
	assert.Equal(t, exitcode.Success, result.ExitCode)
	assert.Equal(t, 0, crossValRunner.CallCount, "cross-val should NOT be called")
	assert.Equal(t, 0, finalPlanRunner.CallCount, "final-plan should NOT be called")
}

// TestRunPostValidationChain_CrossValEscalate verifies escalation from cross-val
func TestRunPostValidationChain_CrossValEscalate(t *testing.T) {
	crossValRunner := &MockAIRunner{
		OutputData: makeValidationJSON("ESCALATE", "Need human review for security"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeValidationJSON("COMPLETE", "Should not reach this"),
	}

	config := PostValidationConfig{
		CrossValRunner:  crossValRunner,
		FinalPlanRunner: finalPlanRunner,
		CrossValEnabled: true,
		FinalPlanEnabled: true,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "exit", result.Action, "escalate should exit")
	assert.Equal(t, exitcode.Escalate, result.ExitCode)
	assert.Equal(t, 0, finalPlanRunner.CallCount, "final-plan should NOT be called after escalate")
}

// TestRunPostValidationChain_FinalPlanEscalate verifies escalation from final-plan
func TestRunPostValidationChain_FinalPlanEscalate(t *testing.T) {
	crossValRunner := &MockAIRunner{
		OutputData: makeValidationJSON("COMPLETE", "Cross validation passed"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeValidationJSON("ESCALATE", "Implementation deviates from original plan"),
	}

	config := PostValidationConfig{
		CrossValRunner:  crossValRunner,
		FinalPlanRunner: finalPlanRunner,
		CrossValEnabled: true,
		FinalPlanEnabled: true,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "exit", result.Action, "escalate should exit")
	assert.Equal(t, exitcode.Escalate, result.ExitCode)
	assert.Equal(t, 1, crossValRunner.CallCount, "cross-val should be called first")
	assert.Equal(t, 1, finalPlanRunner.CallCount, "final-plan should be called after cross-val")
}

// TestRunPostValidationChain_CrossValBlocked verifies blocked from cross-val
func TestRunPostValidationChain_CrossValBlocked(t *testing.T) {
	crossValRunner := &MockAIRunner{
		OutputData: makeValidationJSONWithBlocked("BLOCKED", "All tasks blocked", []string{"Task A", "Task B"}),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeValidationJSON("COMPLETE", "Should not reach this"),
	}

	config := PostValidationConfig{
		CrossValRunner:  crossValRunner,
		FinalPlanRunner: finalPlanRunner,
		CrossValEnabled: true,
		FinalPlanEnabled: true,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "exit", result.Action, "fully blocked should exit")
	assert.Equal(t, exitcode.Blocked, result.ExitCode)
	assert.Equal(t, 0, finalPlanRunner.CallCount, "final-plan should NOT be called after blocked")
}

// TestRunPostValidationChain_FinalPlanBlocked verifies blocked from final-plan
func TestRunPostValidationChain_FinalPlanBlocked(t *testing.T) {
	crossValRunner := &MockAIRunner{
		OutputData: makeValidationJSON("COMPLETE", "Cross validation passed"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeValidationJSONWithBlocked("BLOCKED", "Cannot proceed", []string{"Blocker"}),
	}

	config := PostValidationConfig{
		CrossValRunner:  crossValRunner,
		FinalPlanRunner: finalPlanRunner,
		CrossValEnabled: true,
		FinalPlanEnabled: true,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "exit", result.Action, "blocked should exit")
	assert.Equal(t, exitcode.Blocked, result.ExitCode)
}

// TestRunPostValidationChain_CrossValInadmissible verifies inadmissible from cross-val
func TestRunPostValidationChain_CrossValInadmissible(t *testing.T) {
	crossValRunner := &MockAIRunner{
		OutputData: makeValidationJSON("INADMISSIBLE", "Invalid output format"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeValidationJSON("COMPLETE", "Should not reach this"),
	}

	config := PostValidationConfig{
		CrossValRunner:     crossValRunner,
		FinalPlanRunner:    finalPlanRunner,
		CrossValEnabled:    true,
		FinalPlanEnabled:   true,
		InadmissibleCount:  0,
		MaxInadmissible:    5,
	}

	ctx := context.Background()
	result := RunPostValidationChain(ctx, config)

	assert.Equal(t, "continue", result.Action, "inadmissible under threshold should continue")
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, 0, finalPlanRunner.CallCount, "final-plan should NOT be called after inadmissible")
}

// TestRunPostValidationChain_ContextCancellation verifies context cancellation handling
func TestRunPostValidationChain_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	crossValRunner := &MockAIRunner{
		OutputData: makeValidationJSON("COMPLETE", "Should not complete"),
	}

	finalPlanRunner := &MockAIRunner{
		OutputData: makeValidationJSON("COMPLETE", "Should not complete"),
	}

	config := PostValidationConfig{
		CrossValRunner:  crossValRunner,
		FinalPlanRunner: finalPlanRunner,
		CrossValEnabled: true,
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
			OutputData: makeValidationJSON("COMPLETE", "Should not reach"),
		}

		config := PostValidationConfig{
			CrossValRunner:  crossValRunner,
			FinalPlanRunner: finalPlanRunner,
			CrossValEnabled: true,
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
			OutputData: makeValidationJSON("COMPLETE", "Cross validation passed"),
		}

		finalPlanRunner := &MockAIRunner{
			Err: assert.AnError,
		}

		config := PostValidationConfig{
			CrossValRunner:  crossValRunner,
			FinalPlanRunner: finalPlanRunner,
			CrossValEnabled: true,
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
		name                string
		crossValVerdict     string
		crossValFeedback    string
		finalPlanVerdict    string
		finalPlanFeedback   string
		crossValEnabled     bool
		finalPlanEnabled    bool
		expectedAction      string
		expectedExitCode    int
		expectedFeedback    string
		crossValCallCount   int
		finalPlanCallCount  int
	}{
		{
			name:                "both complete",
			crossValVerdict:     "COMPLETE",
			crossValFeedback:    "",
			finalPlanVerdict:    "COMPLETE",
			finalPlanFeedback:   "",
			crossValEnabled:     true,
			finalPlanEnabled:    true,
			expectedAction:      "success",
			expectedExitCode:    exitcode.Success,
			expectedFeedback:    "",
			crossValCallCount:   1,
			finalPlanCallCount:  1,
		},
		{
			name:                "cross-val needs work",
			crossValVerdict:     "NEEDS_MORE_WORK",
			crossValFeedback:    "Fix bugs",
			finalPlanVerdict:    "COMPLETE",
			finalPlanFeedback:   "",
			crossValEnabled:     true,
			finalPlanEnabled:    true,
			expectedAction:      "continue",
			expectedExitCode:    0,
			expectedFeedback:    "Fix bugs",
			crossValCallCount:   1,
			finalPlanCallCount:  0,
		},
		{
			name:                "final-plan needs work",
			crossValVerdict:     "COMPLETE",
			crossValFeedback:    "",
			finalPlanVerdict:    "NEEDS_MORE_WORK",
			finalPlanFeedback:   "Align with plan",
			crossValEnabled:     true,
			finalPlanEnabled:    true,
			expectedAction:      "continue",
			expectedExitCode:    0,
			expectedFeedback:    "Align with plan",
			crossValCallCount:   1,
			finalPlanCallCount:  1,
		},
		{
			name:                "only cross-val enabled and complete",
			crossValVerdict:     "COMPLETE",
			crossValFeedback:    "",
			finalPlanVerdict:    "COMPLETE",
			finalPlanFeedback:   "",
			crossValEnabled:     true,
			finalPlanEnabled:    false,
			expectedAction:      "success",
			expectedExitCode:    exitcode.Success,
			expectedFeedback:    "",
			crossValCallCount:   1,
			finalPlanCallCount:  0,
		},
		{
			name:                "only final-plan enabled and complete",
			crossValVerdict:     "COMPLETE",
			crossValFeedback:    "",
			finalPlanVerdict:    "COMPLETE",
			finalPlanFeedback:   "",
			crossValEnabled:     false,
			finalPlanEnabled:    true,
			expectedAction:      "success",
			expectedExitCode:    exitcode.Success,
			expectedFeedback:    "",
			crossValCallCount:   0,
			finalPlanCallCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crossValRunner := &MockAIRunner{
				OutputData: makeValidationJSON(tt.crossValVerdict, tt.crossValFeedback),
			}

			finalPlanRunner := &MockAIRunner{
				OutputData: makeValidationJSON(tt.finalPlanVerdict, tt.finalPlanFeedback),
			}

			config := PostValidationConfig{
				CrossValRunner:  crossValRunner,
				FinalPlanRunner: finalPlanRunner,
				CrossValEnabled: tt.crossValEnabled,
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

func makeValidationJSON(verdict string, feedback string) string {
	data := map[string]interface{}{
		"RALPH_VALIDATION": map[string]interface{}{
			"verdict": verdict,
			"feedback": feedback,
		},
	}
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func makeValidationJSONWithBlocked(verdict string, feedback string, blockedTasks []string) string {
	data := map[string]interface{}{
		"RALPH_VALIDATION": map[string]interface{}{
			"verdict": verdict,
			"feedback": feedback,
			"blocked_tasks": blockedTasks,
		},
	}
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}
