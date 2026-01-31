package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseValidation_CompleteVerdict tests extracting COMPLETE verdict.
// This verdict indicates all tasks have been successfully completed with
// no remaining work.
func TestParseValidation_CompleteVerdict(t *testing.T) {
	input := `I have reviewed all the implementation work thoroughly.

` + "```json\n" + `{
  "RALPH_VALIDATION": {
    "verdict": "COMPLETE",
    "feedback": "All tasks have been implemented correctly with proper test coverage.",
    "remaining": 0,
    "blocked_count": 0,
    "blocked_tasks": []
  }
}
` + "```"

	result, err := ParseValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "COMPLETE", result.Verdict)
	assert.Equal(t, "All tasks have been implemented correctly with proper test coverage.", result.Feedback)
	assert.Equal(t, 0, result.Remaining)
	assert.Equal(t, 0, result.BlockedCount)
	assert.Empty(t, result.BlockedTasks)
}

// TestParseValidation_NeedsMoreWorkVerdict tests extracting NEEDS_MORE_WORK verdict.
// This verdict indicates implementation is incomplete and requires additional work.
func TestParseValidation_NeedsMoreWorkVerdict(t *testing.T) {
	input := `After reviewing the implementation:

` + "```json\n" + `{
  "RALPH_VALIDATION": {
    "verdict": "NEEDS_MORE_WORK",
    "feedback": "T003 is incomplete - missing error handling in the parser function. T005 test coverage is below threshold.",
    "remaining": 3,
    "blocked_count": 0,
    "blocked_tasks": []
  }
}
` + "```"

	result, err := ParseValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "NEEDS_MORE_WORK", result.Verdict)
	assert.Contains(t, result.Feedback, "T003 is incomplete")
	assert.Contains(t, result.Feedback, "T005 test coverage is below threshold")
	assert.Equal(t, 3, result.Remaining)
	assert.Equal(t, 0, result.BlockedCount)
	assert.Empty(t, result.BlockedTasks)
}

// TestParseValidation_EscalateVerdict tests extracting ESCALATE verdict.
// This verdict indicates human intervention is required to proceed.
func TestParseValidation_EscalateVerdict(t *testing.T) {
	input := `This requires human intervention:

` + "```json\n" + `{
  "RALPH_VALIDATION": {
    "verdict": "ESCALATE",
    "feedback": "The API credentials are expired and cannot be refreshed programmatically. A human must regenerate the OAuth tokens.",
    "remaining": 5,
    "blocked_count": 0,
    "blocked_tasks": []
  }
}
` + "```"

	result, err := ParseValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "ESCALATE", result.Verdict)
	assert.Contains(t, result.Feedback, "API credentials are expired")
	assert.Contains(t, result.Feedback, "human must regenerate")
	assert.Equal(t, 5, result.Remaining)
	assert.Equal(t, 0, result.BlockedCount)
	assert.Empty(t, result.BlockedTasks)
}

// TestParseValidation_BlockedVerdict tests extracting BLOCKED verdict with blocked tasks.
// This verdict indicates tasks are blocked by external dependencies.
func TestParseValidation_BlockedVerdict(t *testing.T) {
	input := `Several tasks are blocked:

` + "```json\n" + `{
  "RALPH_VALIDATION": {
    "verdict": "BLOCKED",
    "feedback": "External service dependencies are unavailable.",
    "remaining": 4,
    "blocked_count": 3,
    "blocked_tasks": ["T010: Waiting for CI pipeline fix", "T011: Depends on T010", "T012: External API down"]
  }
}
` + "```"

	result, err := ParseValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "BLOCKED", result.Verdict)
	assert.Equal(t, "External service dependencies are unavailable.", result.Feedback)
	assert.Equal(t, 4, result.Remaining)
	assert.Equal(t, 3, result.BlockedCount)
	require.Len(t, result.BlockedTasks, 3)
	assert.Equal(t, "T010: Waiting for CI pipeline fix", result.BlockedTasks[0])
	assert.Equal(t, "T011: Depends on T010", result.BlockedTasks[1])
	assert.Equal(t, "T012: External API down", result.BlockedTasks[2])
}

// TestParseValidation_InadmissibleVerdict tests extracting INADMISSIBLE verdict.
// This verdict indicates the implementation violates quality standards or project rules.
func TestParseValidation_InadmissibleVerdict(t *testing.T) {
	input := `Inadmissible practices detected:

` + "```json\n" + `{
  "RALPH_VALIDATION": {
    "verdict": "INADMISSIBLE",
    "feedback": "Tests duplicate production logic instead of calling actual production code. Test helper re-implements the validation algorithm.",
    "remaining": 2,
    "blocked_count": 0,
    "blocked_tasks": []
  }
}
` + "```"

	result, err := ParseValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "INADMISSIBLE", result.Verdict)
	assert.Contains(t, result.Feedback, "Tests duplicate production logic")
	assert.Contains(t, result.Feedback, "re-implements the validation algorithm")
	assert.Equal(t, 2, result.Remaining)
	assert.Equal(t, 0, result.BlockedCount)
	assert.Empty(t, result.BlockedTasks)
}

// TestParseValidation_MissingFields tests graceful handling of missing fields.
// The parser should not panic and should return zero values for missing fields.
func TestParseValidation_MissingFields(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectNil     bool
		expectVerdict string
	}{
		{
			name: "missing feedback field",
			input: `{"RALPH_VALIDATION": {
				"verdict": "COMPLETE",
				"remaining": 0,
				"blocked_count": 0,
				"blocked_tasks": []
			}}`,
			expectNil:     false,
			expectVerdict: "COMPLETE",
		},
		{
			name: "missing remaining field",
			input: `{"RALPH_VALIDATION": {
				"verdict": "COMPLETE",
				"feedback": "All done",
				"blocked_count": 0,
				"blocked_tasks": []
			}}`,
			expectNil:     false,
			expectVerdict: "COMPLETE",
		},
		{
			name: "missing blocked_count field",
			input: `{"RALPH_VALIDATION": {
				"verdict": "COMPLETE",
				"feedback": "All done",
				"remaining": 0,
				"blocked_tasks": []
			}}`,
			expectNil:     false,
			expectVerdict: "COMPLETE",
		},
		{
			name: "missing blocked_tasks field",
			input: `{"RALPH_VALIDATION": {
				"verdict": "COMPLETE",
				"feedback": "All done",
				"remaining": 0,
				"blocked_count": 0
			}}`,
			expectNil:     false,
			expectVerdict: "COMPLETE",
		},
		{
			name: "only verdict field",
			input: `{"RALPH_VALIDATION": {
				"verdict": "COMPLETE"
			}}`,
			expectNil:     false,
			expectVerdict: "COMPLETE",
		},
		{
			name:      "empty RALPH_VALIDATION object",
			input:     `{"RALPH_VALIDATION": {}}`,
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseValidation(tt.input)
			require.NoError(t, err)

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				if tt.expectVerdict != "" {
					assert.Equal(t, tt.expectVerdict, result.Verdict)
				}
			}
		})
	}
}

// TestParseValidation_EmptyInput tests that empty input returns nil result.
func TestParseValidation_EmptyInput(t *testing.T) {
	result, err := ParseValidation("")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// TestParseValidation_NoRalphValidation tests input without RALPH_VALIDATION key.
func TestParseValidation_NoRalphValidation(t *testing.T) {
	input := `This is just some text without any RALPH_VALIDATION marker.

` + "```json\n" + `{
  "other_data": {
    "field": "value"
  }
}
` + "```"

	result, err := ParseValidation(input)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// TestParseValidation_MalformedJSON tests that malformed JSON returns an error.
func TestParseValidation_MalformedJSON(t *testing.T) {
	input := `Result:

` + "```json\n" + `{
  "RALPH_VALIDATION": {
    "verdict": "COMPLETE",
    "feedback": "All done"
    broken json here
  }
}
` + "```"

	result, err := ParseValidation(input)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestParseValidation_NestedInText tests extraction when RALPH_VALIDATION
// is embedded in surrounding text.
func TestParseValidation_NestedInText(t *testing.T) {
	input := `I have completed the validation review.

Here are my findings:

` + "```json\n" + `{
  "RALPH_VALIDATION": {
    "verdict": "NEEDS_MORE_WORK",
    "feedback": "Additional test coverage needed",
    "remaining": 2,
    "blocked_count": 0,
    "blocked_tasks": []
  }
}
` + "```\n\n" + `Please address the feedback above.`

	result, err := ParseValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "NEEDS_MORE_WORK", result.Verdict)
	assert.Equal(t, "Additional test coverage needed", result.Feedback)
	assert.Equal(t, 2, result.Remaining)
}

// TestParseValidation_MultipleJSONBlocks tests that the first RALPH_VALIDATION
// block is extracted when multiple JSON blocks exist.
func TestParseValidation_MultipleJSONBlocks(t *testing.T) {
	input := `First block:

` + "```json\n" + `{
  "RALPH_VALIDATION": {
    "verdict": "COMPLETE",
    "feedback": "First verdict",
    "remaining": 0,
    "blocked_count": 0,
    "blocked_tasks": []
  }
}
` + "```\n\n" + `Second block:

` + "```json\n" + `{
  "RALPH_VALIDATION": {
    "verdict": "NEEDS_MORE_WORK",
    "feedback": "Second verdict",
    "remaining": 1,
    "blocked_count": 0,
    "blocked_tasks": []
  }
}
` + "```"

	result, err := ParseValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should extract the first block
	assert.Equal(t, "COMPLETE", result.Verdict)
	assert.Equal(t, "First verdict", result.Feedback)
	assert.Equal(t, 0, result.Remaining)
}

// TestParseValidation_BlockedTasksArray tests proper extraction of
// blocked_tasks array with various formats.
func TestParseValidation_BlockedTasksArray(t *testing.T) {
	tests := []struct {
		name                 string
		input                string
		expectedBlockedTasks []string
	}{
		{
			name: "multiple blocked tasks",
			input: `{"RALPH_VALIDATION": {
				"verdict": "BLOCKED",
				"feedback": "Tasks blocked",
				"remaining": 5,
				"blocked_count": 3,
				"blocked_tasks": ["T001: Waiting", "T002: Dependency", "T003: API issue"]
			}}`,
			expectedBlockedTasks: []string{"T001: Waiting", "T002: Dependency", "T003: API issue"},
		},
		{
			name: "single blocked task",
			input: `{"RALPH_VALIDATION": {
				"verdict": "BLOCKED",
				"feedback": "Task blocked",
				"remaining": 1,
				"blocked_count": 1,
				"blocked_tasks": ["T001: Waiting for approval"]
			}}`,
			expectedBlockedTasks: []string{"T001: Waiting for approval"},
		},
		{
			name: "empty blocked tasks array",
			input: `{"RALPH_VALIDATION": {
				"verdict": "COMPLETE",
				"feedback": "Done",
				"remaining": 0,
				"blocked_count": 0,
				"blocked_tasks": []
			}}`,
			expectedBlockedTasks: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseValidation(tt.input)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectedBlockedTasks, result.BlockedTasks)
		})
	}
}

// TestParseValidation_NumericFieldTypes tests that numeric fields are
// properly extracted with correct types.
func TestParseValidation_NumericFieldTypes(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		expectedRemaining int
		expectedBlocked   int
	}{
		{
			name: "zero values",
			input: `{"RALPH_VALIDATION": {
				"verdict": "COMPLETE",
				"feedback": "Done",
				"remaining": 0,
				"blocked_count": 0,
				"blocked_tasks": []
			}}`,
			expectedRemaining: 0,
			expectedBlocked:   0,
		},
		{
			name: "positive values",
			input: `{"RALPH_VALIDATION": {
				"verdict": "NEEDS_MORE_WORK",
				"feedback": "More work needed",
				"remaining": 5,
				"blocked_count": 2,
				"blocked_tasks": ["T001", "T002"]
			}}`,
			expectedRemaining: 5,
			expectedBlocked:   2,
		},
		{
			name: "large values",
			input: `{"RALPH_VALIDATION": {
				"verdict": "NEEDS_MORE_WORK",
				"feedback": "Many tasks",
				"remaining": 100,
				"blocked_count": 50,
				"blocked_tasks": []
			}}`,
			expectedRemaining: 100,
			expectedBlocked:   50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseValidation(tt.input)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectedRemaining, result.Remaining)
			assert.Equal(t, tt.expectedBlocked, result.BlockedCount)
		})
	}
}

// TestParseValidation_SpecialCharactersInFeedback tests that feedback
// text with special characters is properly extracted.
func TestParseValidation_SpecialCharactersInFeedback(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedFeedback string
	}{
		{
			name: "newlines in feedback",
			input: `{"RALPH_VALIDATION": {
				"verdict": "NEEDS_MORE_WORK",
				"feedback": "Issues found:\n- Missing tests\n- Incomplete docs",
				"remaining": 2,
				"blocked_count": 0,
				"blocked_tasks": []
			}}`,
			expectedFeedback: "Issues found:\n- Missing tests\n- Incomplete docs",
		},
		{
			name: "escaped quotes in feedback",
			input: `{"RALPH_VALIDATION": {
				"verdict": "COMPLETE",
				"feedback": "Code says \"hello world\" correctly",
				"remaining": 0,
				"blocked_count": 0,
				"blocked_tasks": []
			}}`,
			expectedFeedback: `Code says "hello world" correctly`,
		},
		{
			name: "unicode characters in feedback",
			input: `{"RALPH_VALIDATION": {
				"verdict": "COMPLETE",
				"feedback": "Task completed ✓ 测试",
				"remaining": 0,
				"blocked_count": 0,
				"blocked_tasks": []
			}}`,
			expectedFeedback: "Task completed ✓ 测试",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseValidation(tt.input)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectedFeedback, result.Feedback)
		})
	}
}

// TestParseValidation_BracketMatchingFallback tests that bracket matching
// works when JSON is not in a fenced code block.
func TestParseValidation_BracketMatchingFallback(t *testing.T) {
	input := `Validation result: {"RALPH_VALIDATION": {"verdict": "COMPLETE", "feedback": "All done", "remaining": 0, "blocked_count": 0, "blocked_tasks": []}} and that's it.`

	result, err := ParseValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "COMPLETE", result.Verdict)
	assert.Equal(t, "All done", result.Feedback)
	assert.Equal(t, 0, result.Remaining)
}

// TestParseValidation_WithTestdata tests parsing using actual testdata files.
func TestParseValidation_WithTestdata(t *testing.T) {
	// Test COMPLETE verdict from testdata
	completeInput := `I have reviewed all the implementation work thoroughly.

` + "```json\n" + `{
  "RALPH_VALIDATION": {
    "verdict": "COMPLETE",
    "feedback": "All tasks have been implemented correctly with proper test coverage.",
    "remaining": 0,
    "blocked_count": 0,
    "blocked_tasks": []
  }
}
` + "```"

	result, err := ParseValidation(completeInput)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "COMPLETE", result.Verdict)
	assert.Equal(t, 0, result.Remaining)

	// Test BLOCKED verdict from testdata
	blockedInput := `Several tasks are blocked:

` + "```json\n" + `{
  "RALPH_VALIDATION": {
    "verdict": "BLOCKED",
    "feedback": "External service dependencies are unavailable.",
    "remaining": 4,
    "blocked_count": 3,
    "blocked_tasks": ["T010: Waiting for CI pipeline fix", "T011: Depends on T010", "T012: External API down"]
  }
}
` + "```"

	result, err = ParseValidation(blockedInput)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "BLOCKED", result.Verdict)
	assert.Equal(t, 4, result.Remaining)
	assert.Equal(t, 3, result.BlockedCount)
	assert.Len(t, result.BlockedTasks, 3)
}

// TestParseValidation_CaseInsensitiveKey tests that RALPH_VALIDATION key
// is matched case-sensitively (should NOT match ralph_validation).
func TestParseValidation_CaseInsensitiveKey(t *testing.T) {
	input := `{"ralph_validation": {"verdict": "COMPLETE", "feedback": "Done", "remaining": 0, "blocked_count": 0, "blocked_tasks": []}}`

	result, err := ParseValidation(input)
	assert.NoError(t, err)
	assert.Nil(t, result, "lowercase key should not match")
}

// TestParseValidation_AllVerdictTypes tests all five verdict types in a
// table-driven manner.
func TestParseValidation_AllVerdictTypes(t *testing.T) {
	tests := []struct {
		name            string
		verdict         string
		remaining       int
		blockedCount    int
		expectedVerdict string
	}{
		{
			name:            "COMPLETE",
			verdict:         "COMPLETE",
			remaining:       0,
			blockedCount:    0,
			expectedVerdict: "COMPLETE",
		},
		{
			name:            "NEEDS_MORE_WORK",
			verdict:         "NEEDS_MORE_WORK",
			remaining:       3,
			blockedCount:    0,
			expectedVerdict: "NEEDS_MORE_WORK",
		},
		{
			name:            "ESCALATE",
			verdict:         "ESCALATE",
			remaining:       5,
			blockedCount:    0,
			expectedVerdict: "ESCALATE",
		},
		{
			name:            "BLOCKED",
			verdict:         "BLOCKED",
			remaining:       4,
			blockedCount:    3,
			expectedVerdict: "BLOCKED",
		},
		{
			name:            "INADMISSIBLE",
			verdict:         "INADMISSIBLE",
			remaining:       2,
			blockedCount:    0,
			expectedVerdict: "INADMISSIBLE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `{"RALPH_VALIDATION": {"verdict": "` + tt.verdict + `", "feedback": "Test feedback", "remaining": ` +
				string(rune(tt.remaining+'0')) + `, "blocked_count": ` + string(rune(tt.blockedCount+'0')) + `, "blocked_tasks": []}}`

			result, err := ParseValidation(input)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectedVerdict, result.Verdict)
			assert.Equal(t, tt.remaining, result.Remaining)
			assert.Equal(t, tt.blockedCount, result.BlockedCount)
		})
	}
}
