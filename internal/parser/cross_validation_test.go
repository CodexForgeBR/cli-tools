package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseCrossValidation_ConfirmedVerdict tests extracting CONFIRMED verdict.
// This verdict indicates the cross-validator agrees with the validator's assessment.
func TestParseCrossValidation_ConfirmedVerdict(t *testing.T) {
	input := `Cross-validation review complete:

` + "```json\n" + `{
  "RALPH_CROSS_VALIDATION": {
    "verdict": "CONFIRMED",
    "feedback": "Implementation correctly addresses all task requirements. Code quality is good."
  }
}
` + "```"

	result, err := ParseCrossValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "CONFIRMED", result.Verdict)
	assert.Equal(t, "Implementation correctly addresses all task requirements. Code quality is good.", result.Feedback)
}

// TestParseCrossValidation_RejectedVerdict tests extracting REJECTED verdict.
// This verdict indicates the cross-validator disagrees with the validator's assessment.
func TestParseCrossValidation_RejectedVerdict(t *testing.T) {
	input := `Cross-validation found issues:

` + "```json\n" + `{
  "RALPH_CROSS_VALIDATION": {
    "verdict": "REJECTED",
    "feedback": "The implementation misses edge case handling for empty input. Tests don't cover the nil pointer scenario."
  }
}
` + "```"

	result, err := ParseCrossValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "REJECTED", result.Verdict)
	assert.Contains(t, result.Feedback, "edge case handling")
	assert.Contains(t, result.Feedback, "nil pointer scenario")
}

// TestParseCrossValidation_MissingFeedback tests graceful handling of missing feedback field.
func TestParseCrossValidation_MissingFeedback(t *testing.T) {
	input := `{"RALPH_CROSS_VALIDATION": {"verdict": "CONFIRMED"}}`

	result, err := ParseCrossValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "CONFIRMED", result.Verdict)
	assert.Empty(t, result.Feedback)
}

// TestParseCrossValidation_MissingVerdict tests graceful handling of missing verdict field.
func TestParseCrossValidation_MissingVerdict(t *testing.T) {
	input := `{"RALPH_CROSS_VALIDATION": {"feedback": "All good"}}`

	result, err := ParseCrossValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Verdict)
	assert.Equal(t, "All good", result.Feedback)
}

// TestParseCrossValidation_EmptyInput tests that empty input returns nil result.
func TestParseCrossValidation_EmptyInput(t *testing.T) {
	result, err := ParseCrossValidation("")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// TestParseCrossValidation_NoRalphCrossValidation tests input without RALPH_CROSS_VALIDATION key.
func TestParseCrossValidation_NoRalphCrossValidation(t *testing.T) {
	input := `This is just some text without any RALPH_CROSS_VALIDATION marker.

` + "```json\n" + `{
  "other_data": {
    "field": "value"
  }
}
` + "```"

	result, err := ParseCrossValidation(input)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// TestParseCrossValidation_MalformedJSON tests that malformed JSON returns an error.
func TestParseCrossValidation_MalformedJSON(t *testing.T) {
	input := `Result:

` + "```json\n" + `{
  "RALPH_CROSS_VALIDATION": {
    "verdict": "CONFIRMED",
    "feedback": "All good"
    broken json here
  }
}
` + "```"

	result, err := ParseCrossValidation(input)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestParseCrossValidation_NestedInText tests extraction when RALPH_CROSS_VALIDATION
// is embedded in surrounding text.
func TestParseCrossValidation_NestedInText(t *testing.T) {
	input := `I have completed the cross-validation review.

Here are my findings:

` + "```json\n" + `{
  "RALPH_CROSS_VALIDATION": {
    "verdict": "REJECTED",
    "feedback": "Validator missed critical security vulnerability in authentication logic."
  }
}
` + "```\n\n" + `Please address the feedback above.`

	result, err := ParseCrossValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "REJECTED", result.Verdict)
	assert.Contains(t, result.Feedback, "security vulnerability")
}

// TestParseCrossValidation_BracketMatchingFallback tests that bracket matching
// works when JSON is not in a fenced code block.
func TestParseCrossValidation_BracketMatchingFallback(t *testing.T) {
	input := `Cross-validation result: {"RALPH_CROSS_VALIDATION": {"verdict": "CONFIRMED", "feedback": "Looks good"}} and that's it.`

	result, err := ParseCrossValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "CONFIRMED", result.Verdict)
	assert.Equal(t, "Looks good", result.Feedback)
}

// TestParseCrossValidation_SpecialCharactersInFeedback tests that feedback
// text with special characters is properly extracted.
func TestParseCrossValidation_SpecialCharactersInFeedback(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedFeedback string
	}{
		{
			name: "newlines in feedback",
			input: `{"RALPH_CROSS_VALIDATION": {
				"verdict": "REJECTED",
				"feedback": "Issues found:\n- Missing edge cases\n- Incomplete tests"
			}}`,
			expectedFeedback: "Issues found:\n- Missing edge cases\n- Incomplete tests",
		},
		{
			name: "escaped quotes in feedback",
			input: `{"RALPH_CROSS_VALIDATION": {
				"verdict": "CONFIRMED",
				"feedback": "Code correctly handles \"edge cases\" as specified"
			}}`,
			expectedFeedback: `Code correctly handles "edge cases" as specified`,
		},
		{
			name: "unicode characters in feedback",
			input: `{"RALPH_CROSS_VALIDATION": {
				"verdict": "CONFIRMED",
				"feedback": "Cross-validation complete ✓ 验证通过"
			}}`,
			expectedFeedback: "Cross-validation complete ✓ 验证通过",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCrossValidation(tt.input)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectedFeedback, result.Feedback)
		})
	}
}

// TestParseCrossValidation_MultipleJSONBlocks tests that the first RALPH_CROSS_VALIDATION
// block is extracted when multiple JSON blocks exist.
func TestParseCrossValidation_MultipleJSONBlocks(t *testing.T) {
	input := `First block:

` + "```json\n" + `{
  "RALPH_CROSS_VALIDATION": {
    "verdict": "CONFIRMED",
    "feedback": "First verdict"
  }
}
` + "```\n\n" + `Second block:

` + "```json\n" + `{
  "RALPH_CROSS_VALIDATION": {
    "verdict": "REJECTED",
    "feedback": "Second verdict"
  }
}
` + "```"

	result, err := ParseCrossValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should extract the first block
	assert.Equal(t, "CONFIRMED", result.Verdict)
	assert.Equal(t, "First verdict", result.Feedback)
}

// TestParseCrossValidation_BothVerdictTypes tests both verdict types in a
// table-driven manner.
func TestParseCrossValidation_BothVerdictTypes(t *testing.T) {
	tests := []struct {
		name            string
		verdict         string
		expectedVerdict string
	}{
		{
			name:            "CONFIRMED",
			verdict:         "CONFIRMED",
			expectedVerdict: "CONFIRMED",
		},
		{
			name:            "REJECTED",
			verdict:         "REJECTED",
			expectedVerdict: "REJECTED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `{"RALPH_CROSS_VALIDATION": {"verdict": "` + tt.verdict + `", "feedback": "Test feedback"}}`

			result, err := ParseCrossValidation(input)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectedVerdict, result.Verdict)
		})
	}
}

// TestParseCrossValidation_WithTestdata tests parsing using actual testdata files.
func TestParseCrossValidation_WithTestdata(t *testing.T) {
	// Test CONFIRMED verdict from testdata
	confirmedInput := `Cross-validation review complete:

` + "```json\n" + `{
  "RALPH_CROSS_VALIDATION": {
    "verdict": "CONFIRMED",
    "feedback": "Implementation correctly addresses all task requirements. Code quality is good."
  }
}
` + "```"

	result, err := ParseCrossValidation(confirmedInput)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "CONFIRMED", result.Verdict)
	assert.Contains(t, result.Feedback, "Code quality is good")

	// Test REJECTED verdict from testdata
	rejectedInput := `Cross-validation found issues:

` + "```json\n" + `{
  "RALPH_CROSS_VALIDATION": {
    "verdict": "REJECTED",
    "feedback": "The implementation misses edge case handling for empty input. Tests don't cover the nil pointer scenario."
  }
}
` + "```"

	result, err = ParseCrossValidation(rejectedInput)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "REJECTED", result.Verdict)
	assert.Contains(t, result.Feedback, "edge case handling")
}

// TestParseCrossValidation_EmptyObject tests handling of empty RALPH_CROSS_VALIDATION object.
func TestParseCrossValidation_EmptyObject(t *testing.T) {
	input := `{"RALPH_CROSS_VALIDATION": {}}`

	result, err := ParseCrossValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Verdict)
	assert.Empty(t, result.Feedback)
}

// TestParseCrossValidation_NilResultFromExtractJSON tests when ExtractJSON returns nil.
func TestParseCrossValidation_NilResultFromExtractJSON(t *testing.T) {
	input := "This text does not contain the RALPH_CROSS_VALIDATION marker at all"

	result, err := ParseCrossValidation(input)
	assert.NoError(t, err)
	assert.Nil(t, result, "Should return nil when marker not found")
}

// TestParseCrossValidation_CompleteFieldsParsing tests all fields are parsed correctly.
func TestParseCrossValidation_CompleteFieldsParsing(t *testing.T) {
	input := `{
		"RALPH_CROSS_VALIDATION": {
			"verdict": "REJECTED",
			"tasks_verified": 5,
			"discrepancies_found": 2,
			"files_actually_read": ["file1.go", "file2.go", "file3.go"],
			"code_quotes": [
				{"file": "main.go", "imports": ["fmt", "os"]},
				{"file": "util.go", "production_calls": ["DoSomething"]}
			],
			"discrepancies": [
				{"task_id": "T001", "claimed": "DONE", "actual": "INCOMPLETE"},
				{"task_id": "T002", "claimed": "TESTED", "actual": "NO_TESTS"}
			],
			"feedback": "Found issues in implementation"
		}
	}`

	result, err := ParseCrossValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "REJECTED", result.Verdict)
	assert.Equal(t, 5, result.TasksVerified)
	assert.Equal(t, 2, result.DiscrepanciesFound)
	assert.Equal(t, []string{"file1.go", "file2.go", "file3.go"}, result.FilesActuallyRead)
	assert.Len(t, result.CodeQuotes, 2)
	assert.Len(t, result.Discrepancies, 2)
	assert.Equal(t, "Found issues in implementation", result.Feedback)
}

// TestParseCrossValidation_ArraysWithInvalidTypes tests arrays with non-string/non-object items.
func TestParseCrossValidation_ArraysWithInvalidTypes(t *testing.T) {
	input := `{
		"RALPH_CROSS_VALIDATION": {
			"verdict": "CONFIRMED",
			"files_actually_read": ["valid.go", 123, "another.go", null, true],
			"code_quotes": [{"file": "ok.go"}, "invalid", 456],
			"discrepancies": [{"task_id": "T001"}, "bad", {"task_id": "T002"}]
		}
	}`

	result, err := ParseCrossValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "CONFIRMED", result.Verdict)
	// Only string items should be included
	assert.Equal(t, []string{"valid.go", "another.go"}, result.FilesActuallyRead)
	// Only map items should be included
	assert.Len(t, result.CodeQuotes, 1)
	assert.Len(t, result.Discrepancies, 2)
}

// TestParseCrossValidation_NoFieldsButHasKey tests when RALPH_CROSS_VALIDATION key exists but has no valid fields.
func TestParseCrossValidation_NoFieldsButHasKey(t *testing.T) {
	input := `{
		"RALPH_CROSS_VALIDATION": {
			"unknown_field": "value",
			"another_unknown": 123
		}
	}`

	result, err := ParseCrossValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result, "Should return non-nil result when key exists")

	// All fields should be empty/zero
	assert.Empty(t, result.Verdict)
	assert.Zero(t, result.TasksVerified)
	assert.Zero(t, result.DiscrepanciesFound)
	assert.Nil(t, result.FilesActuallyRead)
	assert.Nil(t, result.CodeQuotes)
	assert.Nil(t, result.Discrepancies)
	assert.Empty(t, result.Feedback)
}
