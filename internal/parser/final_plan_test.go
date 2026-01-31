package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseFinalPlan_ApproveVerdict tests extracting APPROVE verdict and mapping to CONFIRMED.
func TestParseFinalPlan_ApproveVerdict(t *testing.T) {
	input := `Final plan validation complete:

` + "```json\n" + `{
  "RALPH_FINAL_PLAN_VALIDATION": {
    "verdict": "APPROVE",
    "feedback": "Plan correctly interprets spec and is ready for implementation"
  }
}
` + "```"

	result, err := ParseFinalPlan(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "CONFIRMED", result.Verdict, "APPROVE should map to CONFIRMED")
	assert.Contains(t, result.Feedback, "ready for implementation")
}

// TestParseFinalPlan_RejectVerdict tests extracting REJECT verdict and mapping to NOT_IMPLEMENTED.
func TestParseFinalPlan_RejectVerdict(t *testing.T) {
	input := `Final plan validation found issues:

` + "```json\n" + `{
  "RALPH_FINAL_PLAN_VALIDATION": {
    "verdict": "REJECT",
    "feedback": "Plan includes out-of-scope features not mentioned in spec. Task T007 contradicts requirement 2.3."
  }
}
` + "```"

	result, err := ParseFinalPlan(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "NOT_IMPLEMENTED", result.Verdict, "REJECT should map to NOT_IMPLEMENTED")
	assert.Contains(t, result.Feedback, "out-of-scope")
	assert.Contains(t, result.Feedback, "contradicts requirement")
}

// TestParseFinalPlan_ConfirmedVerdict tests that CONFIRMED verdict is kept as-is.
func TestParseFinalPlan_ConfirmedVerdict(t *testing.T) {
	input := `{"RALPH_FINAL_PLAN_VALIDATION": {"verdict": "CONFIRMED", "feedback": "All good"}}`

	result, err := ParseFinalPlan(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "CONFIRMED", result.Verdict)
}

// TestParseFinalPlan_NotImplementedVerdict tests that NOT_IMPLEMENTED verdict is kept as-is.
func TestParseFinalPlan_NotImplementedVerdict(t *testing.T) {
	input := `{"RALPH_FINAL_PLAN_VALIDATION": {"verdict": "NOT_IMPLEMENTED", "feedback": "Issues found"}}`

	result, err := ParseFinalPlan(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "NOT_IMPLEMENTED", result.Verdict)
}

// TestParseFinalPlan_MissingFeedback tests graceful handling of missing feedback field.
func TestParseFinalPlan_MissingFeedback(t *testing.T) {
	input := `{"RALPH_FINAL_PLAN_VALIDATION": {"verdict": "APPROVE"}}`

	result, err := ParseFinalPlan(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "CONFIRMED", result.Verdict)
	assert.Empty(t, result.Feedback)
}

// TestParseFinalPlan_MissingVerdict tests graceful handling of missing verdict field.
func TestParseFinalPlan_MissingVerdict(t *testing.T) {
	input := `{"RALPH_FINAL_PLAN_VALIDATION": {"feedback": "Some feedback"}}`

	result, err := ParseFinalPlan(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Verdict)
	assert.Equal(t, "Some feedback", result.Feedback)
}

// TestParseFinalPlan_EmptyInput tests that empty input returns nil result.
func TestParseFinalPlan_EmptyInput(t *testing.T) {
	result, err := ParseFinalPlan("")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// TestParseFinalPlan_NoRalphFinalPlan tests input without RALPH_FINAL_PLAN_VALIDATION key.
func TestParseFinalPlan_NoRalphFinalPlan(t *testing.T) {
	input := `This is just some text without any RALPH_FINAL_PLAN_VALIDATION marker.

` + "```json\n" + `{
  "other_data": {
    "field": "value"
  }
}
` + "```"

	result, err := ParseFinalPlan(input)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// TestParseFinalPlan_MalformedJSON tests that malformed JSON returns an error.
func TestParseFinalPlan_MalformedJSON(t *testing.T) {
	input := `Result:

` + "```json\n" + `{
  "RALPH_FINAL_PLAN_VALIDATION": {
    "verdict": "APPROVE",
    "feedback": "All good"
    broken json here
  }
}
` + "```"

	result, err := ParseFinalPlan(input)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestParseFinalPlan_NestedInText tests extraction when RALPH_FINAL_PLAN_VALIDATION
// is embedded in surrounding text.
func TestParseFinalPlan_NestedInText(t *testing.T) {
	input := `I have completed the final plan validation review.

Here are my findings:

` + "```json\n" + `{
  "RALPH_FINAL_PLAN_VALIDATION": {
    "verdict": "REJECT",
    "feedback": "Plan misinterprets requirement 1.5 - should use HTTP polling, not WebSockets."
  }
}
` + "```\n\n" + `Please revise the plan accordingly.`

	result, err := ParseFinalPlan(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "NOT_IMPLEMENTED", result.Verdict)
	assert.Contains(t, result.Feedback, "misinterprets requirement")
}

// TestParseFinalPlan_BracketMatchingFallback tests that bracket matching
// works when JSON is not in a fenced code block.
func TestParseFinalPlan_BracketMatchingFallback(t *testing.T) {
	input := `Final plan validation result: {"RALPH_FINAL_PLAN_VALIDATION": {"verdict": "APPROVE", "feedback": "Looks good"}} and that's it.`

	result, err := ParseFinalPlan(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "CONFIRMED", result.Verdict)
	assert.Equal(t, "Looks good", result.Feedback)
}

// TestParseFinalPlan_SpecialCharactersInFeedback tests that feedback
// text with special characters is properly extracted.
func TestParseFinalPlan_SpecialCharactersInFeedback(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedFeedback string
	}{
		{
			name: "newlines in feedback",
			input: `{"RALPH_FINAL_PLAN_VALIDATION": {
				"verdict": "REJECT",
				"feedback": "Issues found:\n- Missing edge cases\n- Out of scope features"
			}}`,
			expectedFeedback: "Issues found:\n- Missing edge cases\n- Out of scope features",
		},
		{
			name: "escaped quotes in feedback",
			input: `{"RALPH_FINAL_PLAN_VALIDATION": {
				"verdict": "APPROVE",
				"feedback": "Plan correctly addresses \"user authentication\" requirement"
			}}`,
			expectedFeedback: `Plan correctly addresses "user authentication" requirement`,
		},
		{
			name: "unicode characters in feedback",
			input: `{"RALPH_FINAL_PLAN_VALIDATION": {
				"verdict": "APPROVE",
				"feedback": "Final plan validation complete ✓ 最终计划验证通过"
			}}`,
			expectedFeedback: "Final plan validation complete ✓ 最终计划验证通过",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseFinalPlan(tt.input)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectedFeedback, result.Feedback)
		})
	}
}

// TestParseFinalPlan_MultipleJSONBlocks tests that the first RALPH_FINAL_PLAN_VALIDATION
// block is extracted when multiple JSON blocks exist.
func TestParseFinalPlan_MultipleJSONBlocks(t *testing.T) {
	input := `First block:

` + "```json\n" + `{
  "RALPH_FINAL_PLAN_VALIDATION": {
    "verdict": "APPROVE",
    "feedback": "First verdict"
  }
}
` + "```\n\n" + `Second block:

` + "```json\n" + `{
  "RALPH_FINAL_PLAN_VALIDATION": {
    "verdict": "REJECT",
    "feedback": "Second verdict"
  }
}
` + "```"

	result, err := ParseFinalPlan(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should extract the first block
	assert.Equal(t, "CONFIRMED", result.Verdict)
	assert.Equal(t, "First verdict", result.Feedback)
}

// TestParseFinalPlan_AllVerdictTypes tests all verdict types in a
// table-driven manner, including mapping.
func TestParseFinalPlan_AllVerdictTypes(t *testing.T) {
	tests := []struct {
		name            string
		inputVerdict    string
		expectedVerdict string
	}{
		{
			name:            "APPROVE maps to CONFIRMED",
			inputVerdict:    "APPROVE",
			expectedVerdict: "CONFIRMED",
		},
		{
			name:            "REJECT maps to NOT_IMPLEMENTED",
			inputVerdict:    "REJECT",
			expectedVerdict: "NOT_IMPLEMENTED",
		},
		{
			name:            "CONFIRMED stays CONFIRMED",
			inputVerdict:    "CONFIRMED",
			expectedVerdict: "CONFIRMED",
		},
		{
			name:            "NOT_IMPLEMENTED stays NOT_IMPLEMENTED",
			inputVerdict:    "NOT_IMPLEMENTED",
			expectedVerdict: "NOT_IMPLEMENTED",
		},
		{
			name:            "unknown verdict kept as-is",
			inputVerdict:    "UNKNOWN",
			expectedVerdict: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `{"RALPH_FINAL_PLAN_VALIDATION": {"verdict": "` + tt.inputVerdict + `", "feedback": "Test feedback"}}`

			result, err := ParseFinalPlan(input)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectedVerdict, result.Verdict)
		})
	}
}

// TestParseFinalPlan_EmptyObject tests handling of empty RALPH_FINAL_PLAN_VALIDATION object.
func TestParseFinalPlan_EmptyObject(t *testing.T) {
	input := `{"RALPH_FINAL_PLAN_VALIDATION": {}}`

	result, err := ParseFinalPlan(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Verdict)
	assert.Empty(t, result.Feedback)
}

// TestParseFinalPlan_VerdictMapping tests the specific mapping behavior.
func TestParseFinalPlan_VerdictMapping(t *testing.T) {
	t.Run("APPROVE to CONFIRMED mapping", func(t *testing.T) {
		input := `{"RALPH_FINAL_PLAN_VALIDATION": {"verdict": "APPROVE"}}`
		result, err := ParseFinalPlan(input)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "CONFIRMED", result.Verdict, "APPROVE must map to CONFIRMED")
	})

	t.Run("REJECT to NOT_IMPLEMENTED mapping", func(t *testing.T) {
		input := `{"RALPH_FINAL_PLAN_VALIDATION": {"verdict": "REJECT"}}`
		result, err := ParseFinalPlan(input)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "NOT_IMPLEMENTED", result.Verdict, "REJECT must map to NOT_IMPLEMENTED")
	})
}
