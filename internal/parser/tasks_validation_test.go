package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseTasksValidation_ValidVerdict tests extracting VALID verdict.
func TestParseTasksValidation_ValidVerdict(t *testing.T) {
	input := `Tasks validation complete:

` + "```json\n" + `{
  "RALPH_TASKS_VALIDATION": {
    "verdict": "VALID",
    "feedback": "All requirements from spec.md are correctly captured in tasks.md"
  }
}
` + "```"

	result, err := ParseTasksValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "VALID", result.Verdict)
	assert.Contains(t, result.Feedback, "correctly captured")
}

// TestParseTasksValidation_InvalidVerdict tests extracting INVALID verdict.
func TestParseTasksValidation_InvalidVerdict(t *testing.T) {
	input := `Tasks validation found issues:

` + "```json\n" + `{
  "RALPH_TASKS_VALIDATION": {
    "verdict": "INVALID",
    "feedback": "Tasks are missing requirement 3.2 from spec. Task T005 is out of scope."
  }
}
` + "```"

	result, err := ParseTasksValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "INVALID", result.Verdict)
	assert.Contains(t, result.Feedback, "missing requirement")
	assert.Contains(t, result.Feedback, "out of scope")
}

// TestParseTasksValidation_WithNewFields tests extracting new fields from RALPH_TASKS_VALIDATION.
func TestParseTasksValidation_WithNewFields(t *testing.T) {
	input := `{"RALPH_TASKS_VALIDATION": {
		"verdict": "INVALID",
		"feedback": "Issues found",
		"missing_requirements": ["Requirement A", "Requirement B"],
		"out_of_scope_tasks": ["Task T001", "Task T002"],
		"vague_tasks": ["Task T003"],
		"quality_score": "Needs significant improvement"
	}}`

	result, err := ParseTasksValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "INVALID", result.Verdict)
	assert.Equal(t, "Issues found", result.Feedback)
	assert.Equal(t, []string{"Requirement A", "Requirement B"}, result.MissingRequirements)
	assert.Equal(t, []string{"Task T001", "Task T002"}, result.OutOfScopeTasks)
	assert.Equal(t, []string{"Task T003"}, result.VagueTasks)
	assert.Equal(t, "Needs significant improvement", result.QualityScore)
}

// TestParseTasksValidation_MissingFeedback tests graceful handling of missing feedback field.
func TestParseTasksValidation_MissingFeedback(t *testing.T) {
	input := `{"RALPH_TASKS_VALIDATION": {"verdict": "VALID"}}`

	result, err := ParseTasksValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "VALID", result.Verdict)
	assert.Empty(t, result.Feedback)
}

// TestParseTasksValidation_MissingVerdict tests graceful handling of missing verdict field.
func TestParseTasksValidation_MissingVerdict(t *testing.T) {
	input := `{"RALPH_TASKS_VALIDATION": {"feedback": "Some feedback"}}`

	result, err := ParseTasksValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Verdict)
	assert.Equal(t, "Some feedback", result.Feedback)
}

// TestParseTasksValidation_EmptyInput tests that empty input returns nil result.
func TestParseTasksValidation_EmptyInput(t *testing.T) {
	result, err := ParseTasksValidation("")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// TestParseTasksValidation_NoRalphTasksValidation tests input without RALPH_TASKS_VALIDATION key.
func TestParseTasksValidation_NoRalphTasksValidation(t *testing.T) {
	input := `This is just some text without any RALPH_TASKS_VALIDATION marker.

` + "```json\n" + `{
  "other_data": {
    "field": "value"
  }
}
` + "```"

	result, err := ParseTasksValidation(input)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// TestParseTasksValidation_MalformedJSON tests that malformed JSON returns an error.
func TestParseTasksValidation_MalformedJSON(t *testing.T) {
	input := `Result:

` + "```json\n" + `{
  "RALPH_TASKS_VALIDATION": {
    "verdict": "VALID",
    "feedback": "All good"
    broken json here
  }
}
` + "```"

	result, err := ParseTasksValidation(input)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestParseTasksValidation_NestedInText tests extraction when RALPH_TASKS_VALIDATION
// is embedded in surrounding text.
func TestParseTasksValidation_NestedInText(t *testing.T) {
	input := `I have completed the tasks validation review.

Here are my findings:

` + "```json\n" + `{
  "RALPH_TASKS_VALIDATION": {
    "verdict": "INVALID",
    "feedback": "Task T003 should be split into two separate tasks for clarity."
  }
}
` + "```\n\n" + `Please update the tasks file accordingly.`

	result, err := ParseTasksValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "INVALID", result.Verdict)
	assert.Contains(t, result.Feedback, "split into two")
}

// TestParseTasksValidation_BracketMatchingFallback tests that bracket matching
// works when JSON is not in a fenced code block.
func TestParseTasksValidation_BracketMatchingFallback(t *testing.T) {
	input := `Tasks validation result: {"RALPH_TASKS_VALIDATION": {"verdict": "VALID", "feedback": "Looks good"}} and that's it.`

	result, err := ParseTasksValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "VALID", result.Verdict)
	assert.Equal(t, "Looks good", result.Feedback)
}

// TestParseTasksValidation_SpecialCharactersInFeedback tests that feedback
// text with special characters is properly extracted.
func TestParseTasksValidation_SpecialCharactersInFeedback(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedVerdict  string
		expectedFeedback string
	}{
		{
			name: "newlines in feedback",
			input: `{"RALPH_TASKS_VALIDATION": {
				"verdict": "INVALID",
				"feedback": "Issues found:\n- Missing T001\n- T002 out of scope"
			}}`,
			expectedVerdict:  "INVALID",
			expectedFeedback: "Issues found:\n- Missing T001\n- T002 out of scope",
		},
		{
			name: "escaped quotes in feedback",
			input: `{"RALPH_TASKS_VALIDATION": {
				"verdict": "VALID",
				"feedback": "Tasks correctly capture \"user authentication\" requirement"
			}}`,
			expectedVerdict:  "VALID",
			expectedFeedback: `Tasks correctly capture "user authentication" requirement`,
		},
		{
			name: "unicode characters in feedback",
			input: `{"RALPH_TASKS_VALIDATION": {
				"verdict": "VALID",
				"feedback": "Tasks validation complete ✓ 任务验证通过"
			}}`,
			expectedVerdict:  "VALID",
			expectedFeedback: "Tasks validation complete ✓ 任务验证通过",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTasksValidation(tt.input)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectedVerdict, result.Verdict)
			assert.Equal(t, tt.expectedFeedback, result.Feedback)
		})
	}
}

// TestParseTasksValidation_MultipleJSONBlocks tests that the first RALPH_TASKS_VALIDATION
// block is extracted when multiple JSON blocks exist.
func TestParseTasksValidation_MultipleJSONBlocks(t *testing.T) {
	input := `First block:

` + "```json\n" + `{
  "RALPH_TASKS_VALIDATION": {
    "verdict": "VALID",
    "feedback": "First verdict"
  }
}
` + "```\n\n" + `Second block:

` + "```json\n" + `{
  "RALPH_TASKS_VALIDATION": {
    "verdict": "INVALID",
    "feedback": "Second verdict"
  }
}
` + "```"

	result, err := ParseTasksValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should extract the first block
	assert.Equal(t, "VALID", result.Verdict)
	assert.Equal(t, "First verdict", result.Feedback)
}

// TestParseTasksValidation_AllVerdictTypes tests all verdict types.
func TestParseTasksValidation_AllVerdictTypes(t *testing.T) {
	tests := []struct {
		name            string
		inputVerdict    string
		expectedVerdict string
	}{
		{
			name:            "VALID stays VALID",
			inputVerdict:    "VALID",
			expectedVerdict: "VALID",
		},
		{
			name:            "INVALID stays INVALID",
			inputVerdict:    "INVALID",
			expectedVerdict: "INVALID",
		},
		{
			name:            "unknown verdict kept as-is",
			inputVerdict:    "UNKNOWN",
			expectedVerdict: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `{"RALPH_TASKS_VALIDATION": {"verdict": "` + tt.inputVerdict + `", "feedback": "Test feedback"}}`

			result, err := ParseTasksValidation(input)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectedVerdict, result.Verdict)
		})
	}
}

// TestParseTasksValidation_EmptyObject tests handling of empty RALPH_TASKS_VALIDATION object.
func TestParseTasksValidation_EmptyObject(t *testing.T) {
	input := `{"RALPH_TASKS_VALIDATION": {}}`

	result, err := ParseTasksValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Verdict)
	assert.Empty(t, result.Feedback)
}

// TestParseTasksValidation_EmptyArrayFields tests handling of empty array fields.
func TestParseTasksValidation_EmptyArrayFields(t *testing.T) {
	input := `{"RALPH_TASKS_VALIDATION": {
		"verdict": "VALID",
		"feedback": "All good",
		"missing_requirements": [],
		"out_of_scope_tasks": [],
		"vague_tasks": []
	}}`

	result, err := ParseTasksValidation(input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "VALID", result.Verdict)
	assert.Empty(t, result.MissingRequirements)
	assert.Empty(t, result.OutOfScopeTasks)
	assert.Empty(t, result.VagueTasks)
}
