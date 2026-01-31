package prompt

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildImplFirstPrompt_IncludesInadmissibleRules verifies that the first
// implementation prompt includes the inadmissible practices section.
func TestBuildImplFirstPrompt_IncludesInadmissibleRules(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	learnings := ""

	result := BuildImplFirstPrompt(tasksFile, learnings)

	assert.Contains(t, result, "INADMISSIBLE PRACTICES", "prompt should include inadmissible practices section")
	assert.Contains(t, result, "PRODUCTION CODE DUPLICATION IN TESTS", "prompt should include specific inadmissible rule")
	assert.Contains(t, result, "MOCK THE SUBJECT UNDER TEST", "prompt should include mock rule")
	assert.Contains(t, result, "TRIVIAL/EMPTY TESTS", "prompt should include trivial tests rule")
	assert.Contains(t, result, "TESTS FOR NON-EXISTENT FUNCTIONALITY", "prompt should include non-existent functionality rule")
}

// TestBuildImplFirstPrompt_IncludesEvidenceRules verifies that the first
// implementation prompt includes evidence capture instructions.
func TestBuildImplFirstPrompt_IncludesEvidenceRules(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	learnings := ""

	result := BuildImplFirstPrompt(tasksFile, learnings)

	assert.Contains(t, result, "EVIDENCE CAPTURE FOR NON-FILE TASKS", "prompt should include evidence capture section")
	assert.Contains(t, result, "Deploy X", "prompt should include deploy evidence example")
	assert.Contains(t, result, "Run tests", "prompt should include test evidence example")
	assert.Contains(t, result, "Build X", "prompt should include build evidence example")
	assert.Contains(t, result, "Playwright MCP", "prompt should mention Playwright MCP in evidence section")
}

// TestBuildImplFirstPrompt_IncludesPlaywrightRules verifies that the first
// implementation prompt includes Playwright MCP validation rules.
func TestBuildImplFirstPrompt_IncludesPlaywrightRules(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	learnings := ""

	result := BuildImplFirstPrompt(tasksFile, learnings)

	assert.Contains(t, result, "PLAYWRIGHT MCP VALIDATION", "prompt should include Playwright MCP section header")
	assert.Contains(t, result, "APP NOT RUNNING", "prompt should mention app not running rule")
	assert.Contains(t, result, "START IT", "prompt should include start app instruction")
	assert.Contains(t, result, "FORBIDDEN EXCUSES", "prompt should include forbidden excuses section")
}

// TestBuildImplFirstPrompt_IncludesTasksFile verifies that the tasks file path
// is correctly included in the prompt.
func TestBuildImplFirstPrompt_IncludesTasksFile(t *testing.T) {
	tasksFile := "/custom/path/to/tasks.md"
	learnings := ""

	result := BuildImplFirstPrompt(tasksFile, learnings)

	assert.Contains(t, result, tasksFile, "prompt should include the tasks file path")
	assert.Contains(t, result, "TASKS FILE:", "prompt should have tasks file label")
}

// TestBuildImplFirstPrompt_IncludesLearnings verifies that learnings are
// included when provided.
func TestBuildImplFirstPrompt_IncludesLearnings(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	learnings := "Pattern: Always use strict null checks\nGotcha: API returns null on empty"

	result := BuildImplFirstPrompt(tasksFile, learnings)

	assert.Contains(t, result, "LEARNINGS FROM PREVIOUS ITERATIONS", "prompt should include learnings header")
	assert.Contains(t, result, learnings, "prompt should include the actual learnings content")
	assert.Contains(t, result, "Codebase Patterns", "prompt should mention codebase patterns section")
}

// TestBuildImplFirstPrompt_OmitsLearnings verifies that the learnings section
// is omitted when no learnings are provided.
func TestBuildImplFirstPrompt_OmitsLearnings(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	learnings := ""

	result := BuildImplFirstPrompt(tasksFile, learnings)

	assert.NotContains(t, result, "LEARNINGS FROM PREVIOUS ITERATIONS", "prompt should not include learnings header when empty")
}

// TestBuildImplFirstPrompt_IncludesRalphStatus verifies that the prompt
// includes instructions for RALPH_STATUS output.
func TestBuildImplFirstPrompt_IncludesRalphStatus(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	learnings := ""

	result := BuildImplFirstPrompt(tasksFile, learnings)

	assert.Contains(t, result, "RALPH_STATUS", "prompt should mention RALPH_STATUS")
	assert.Contains(t, result, "completed_tasks", "prompt should mention completed_tasks field")
	assert.Contains(t, result, "blocked_tasks", "prompt should mention blocked_tasks field")
}

// TestBuildImplFirstPrompt_IncludesLearningsOutput verifies that the prompt
// includes instructions for outputting new learnings.
func TestBuildImplFirstPrompt_IncludesLearningsOutput(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	learnings := ""

	result := BuildImplFirstPrompt(tasksFile, learnings)

	assert.Contains(t, result, "RALPH_LEARNINGS", "prompt should mention RALPH_LEARNINGS")
	assert.Contains(t, result, "LEARNINGS OUTPUT", "prompt should include learnings output section")
	assert.Contains(t, result, "Pattern:", "prompt should show learnings format with Pattern")
	assert.Contains(t, result, "Gotcha:", "prompt should show learnings format with Gotcha")
	assert.Contains(t, result, "Context:", "prompt should show learnings format with Context")
}

// TestBuildImplContinuePrompt_IncludesFeedback verifies that the continuation
// prompt includes the validator's feedback.
func TestBuildImplContinuePrompt_IncludesFeedback(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	feedback := "Task T001: You said you removed X but it's still in the code."
	learnings := ""

	result := BuildImplContinuePrompt(tasksFile, feedback, learnings)

	assert.Contains(t, result, "VALIDATION CAUGHT YOUR LIES", "prompt should include feedback header")
	assert.Contains(t, result, feedback, "prompt should include the actual feedback text")
	assert.Contains(t, result, "FIX YOUR LIES NOW", "prompt should include fix instruction")
}

// TestBuildImplContinuePrompt_IncludesEvidenceRules verifies that the
// continuation prompt includes evidence capture rules.
func TestBuildImplContinuePrompt_IncludesEvidenceRules(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	feedback := "Fix task T001"
	learnings := ""

	result := BuildImplContinuePrompt(tasksFile, feedback, learnings)

	assert.Contains(t, result, "EVIDENCE CAPTURE FOR NON-FILE TASKS", "prompt should include evidence capture section")
}

// TestBuildImplContinuePrompt_IncludesPlaywrightRules verifies that the
// continuation prompt includes Playwright MCP rules.
func TestBuildImplContinuePrompt_IncludesPlaywrightRules(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	feedback := "Fix task T001"
	learnings := ""

	result := BuildImplContinuePrompt(tasksFile, feedback, learnings)

	assert.Contains(t, result, "PLAYWRIGHT MCP VALIDATION", "prompt should include Playwright section")
	assert.Contains(t, result, "APP NOT RUNNING", "prompt should mention app not running rule")
}

// TestBuildImplContinuePrompt_IncludesRalphStatus verifies that the
// continuation prompt includes RALPH_STATUS output instructions.
func TestBuildImplContinuePrompt_IncludesRalphStatus(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	feedback := "Fix task T001"
	learnings := ""

	result := BuildImplContinuePrompt(tasksFile, feedback, learnings)

	assert.Contains(t, result, "RALPH_STATUS", "prompt should mention RALPH_STATUS")
	assert.Contains(t, result, "completed_tasks", "prompt should mention completed_tasks field")
}

// TestBuildImplContinuePrompt_IncludesLearnings verifies that learnings are
// included in continuation prompts when provided.
func TestBuildImplContinuePrompt_IncludesLearnings(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	feedback := "Fix task T001"
	learnings := "Pattern: Database connections must be pooled\nGotcha: Timeout is in milliseconds"

	result := BuildImplContinuePrompt(tasksFile, feedback, learnings)

	assert.Contains(t, result, "LEARNINGS FROM PREVIOUS ITERATIONS", "prompt should include learnings header")
	assert.Contains(t, result, learnings, "prompt should include the actual learnings content")
}

// TestBuildImplContinuePrompt_OmitsLearnings verifies that the learnings
// section is omitted when no learnings are provided.
func TestBuildImplContinuePrompt_OmitsLearnings(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	feedback := "Fix task T001"
	learnings := ""

	result := BuildImplContinuePrompt(tasksFile, feedback, learnings)

	assert.NotContains(t, result, "LEARNINGS FROM PREVIOUS ITERATIONS", "prompt should not include learnings header when empty")
}

// TestBuildImplContinuePrompt_WarnsCriticalRules verifies that the continuation
// prompt emphasizes critical rules about not writing tests for non-existent functionality.
func TestBuildImplContinuePrompt_WarnsCriticalRules(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	feedback := "Fix task T001"
	learnings := ""

	result := BuildImplContinuePrompt(tasksFile, feedback, learnings)

	assert.Contains(t, result, "CRITICAL", "prompt should emphasize critical rules")
	assert.Contains(t, result, "DO NOT WRITE TESTS FOR NON-EXISTENT FUNCTIONALITY", "prompt should warn about non-existent functionality")
	assert.Contains(t, result, "Implementation FIRST, then tests", "prompt should emphasize implementation order")
}

// TestBuildValidationPrompt_IncludesImplOutput verifies that the validation
// prompt includes the implementation output file path.
func TestBuildValidationPrompt_IncludesImplOutput(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	implOutputFile := "/path/to/impl-output.txt"

	result := BuildValidationPrompt(tasksFile, implOutputFile)

	assert.Contains(t, result, "IMPLEMENTATION OUTPUT FILE", "prompt should include impl output file header")
	assert.Contains(t, result, implOutputFile, "prompt should include the implementation output file path")
}

// TestBuildValidationPrompt_IncludesTasksFile verifies that the validation
// prompt includes the tasks file reference.
func TestBuildValidationPrompt_IncludesTasksFile(t *testing.T) {
	tasksFile := "/custom/path/to/tasks.md"
	implOutput := "Work completed"

	result := BuildValidationPrompt(tasksFile, implOutput)

	assert.Contains(t, result, tasksFile, "prompt should include the tasks file path")
	assert.Contains(t, result, "TASKS FILE TO CHECK AGAINST", "prompt should have tasks file label")
}

// TestBuildValidationPrompt_IncludesValidatorRole verifies that the validation
// prompt establishes the validator role and rules.
func TestBuildValidationPrompt_IncludesValidatorRole(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	implOutput := "Work completed"

	result := BuildValidationPrompt(tasksFile, implOutput)

	assert.Contains(t, result, "VALIDATOR", "prompt should mention validator role")
	assert.Contains(t, result, "THE IMPLEMENTER IS A LIAR", "prompt should establish adversarial stance")
	assert.Contains(t, result, "VALIDATION RULES", "prompt should include validation rules section")
}

// TestBuildValidationPrompt_IncludesCommonLies verifies that the validation
// prompt includes common lies to catch.
func TestBuildValidationPrompt_IncludesCommonLies(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	implOutput := "Work completed"

	result := BuildValidationPrompt(tasksFile, implOutput)

	assert.Contains(t, result, "COMMON LIES TO CATCH", "prompt should include common lies section")
	assert.Contains(t, result, "I removed X", "prompt should list 'removed' lie example")
	assert.Contains(t, result, "I created Y", "prompt should list 'created' lie example")
	assert.Contains(t, result, "Task is N/A", "prompt should list N/A lie example")
}

// TestBuildValidationPrompt_IncludesRalphValidation verifies that the
// validation prompt includes RALPH_VALIDATION output format.
func TestBuildValidationPrompt_IncludesRalphValidation(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	implOutput := "Work completed"

	result := BuildValidationPrompt(tasksFile, implOutput)

	assert.Contains(t, result, "RALPH_VALIDATION", "prompt should mention RALPH_VALIDATION")
	assert.Contains(t, result, "verdict", "prompt should mention verdict field")
	assert.Contains(t, result, "feedback", "prompt should mention feedback field")
	assert.Contains(t, result, "completed_tasks", "prompt should mention completed_tasks field")
	assert.Contains(t, result, "incomplete_tasks", "prompt should mention incomplete_tasks field")
	assert.Contains(t, result, "inadmissible_practices", "prompt should mention inadmissible_practices field")
}

// TestBuildValidationPrompt_IncludesVerdictOptions verifies that the validation
// prompt lists all possible verdict options.
func TestBuildValidationPrompt_IncludesVerdictOptions(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	implOutput := "Work completed"

	result := BuildValidationPrompt(tasksFile, implOutput)

	assert.Contains(t, result, "COMPLETE", "prompt should list COMPLETE verdict")
	assert.Contains(t, result, "NEEDS_MORE_WORK", "prompt should list NEEDS_MORE_WORK verdict")
	assert.Contains(t, result, "INADMISSIBLE", "prompt should list INADMISSIBLE verdict")
	assert.Contains(t, result, "ESCALATE", "prompt should list ESCALATE verdict")
	assert.Contains(t, result, "BLOCKED", "prompt should list BLOCKED verdict")
}

// TestBuildValidationPrompt_IncludesInadmissibleChecks verifies that the
// validation prompt includes detailed inadmissible practice checks.
func TestBuildValidationPrompt_IncludesInadmissibleChecks(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	implOutput := "Work completed"

	result := BuildValidationPrompt(tasksFile, implOutput)

	assert.Contains(t, result, "INADMISSIBLE PRACTICES", "prompt should include inadmissible section")
	assert.Contains(t, result, "PRODUCTION CODE DUPLICATION IN TESTS", "prompt should check for duplication")
	assert.Contains(t, result, "MOCKING THE SUBJECT UNDER TEST", "prompt should check for mocking")
	assert.Contains(t, result, "TRIVIAL/EMPTY TESTS", "prompt should check for trivial tests")
	assert.Contains(t, result, "TESTS FOR NON-EXISTENT FUNCTIONALITY", "prompt should check for non-existent functionality")
	assert.Contains(t, result, "DETECTION PROCESS", "prompt should include detection process")
}

// TestBuildValidationPrompt_NonEmpty verifies that all builder functions
// return non-empty prompts.
func TestBuildValidationPrompt_NonEmpty(t *testing.T) {
	tests := []struct {
		name   string
		result string
	}{
		{
			name:   "BuildImplFirstPrompt",
			result: BuildImplFirstPrompt("/path/to/tasks.md", ""),
		},
		{
			name:   "BuildImplContinuePrompt",
			result: BuildImplContinuePrompt("/path/to/tasks.md", "feedback", ""),
		},
		{
			name:   "BuildValidationPrompt",
			result: BuildValidationPrompt("/path/to/tasks.md", "output"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotEmpty(t, tt.result, "prompt should not be empty")
			assert.Greater(t, len(tt.result), 100, "prompt should be substantial")
		})
	}
}

// TestBuildImplFirstPrompt_NoPlaceholders verifies that all placeholders are
// replaced in the first implementation prompt.
func TestBuildImplFirstPrompt_NoPlaceholders(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	learnings := "Some learnings"

	result := BuildImplFirstPrompt(tasksFile, learnings)

	// Should not contain unreplaced template markers
	assert.NotContains(t, result, "{{TASKS_FILE}}", "should not contain tasks file placeholder")
	assert.NotContains(t, result, "{{INADMISSIBLE_RULES}}", "should not contain inadmissible rules placeholder")
	assert.NotContains(t, result, "{{EVIDENCE_RULES}}", "should not contain evidence rules placeholder")
	assert.NotContains(t, result, "{{PLAYWRIGHT_RULES}}", "should not contain playwright rules placeholder")
	assert.NotContains(t, result, "{{LEARNINGS}}", "should not contain learnings placeholder")
	assert.NotContains(t, result, "{{LEARNINGS_OUTPUT}}", "should not contain learnings output placeholder")
}

// TestBuildImplContinuePrompt_NoPlaceholders verifies that all placeholders are
// replaced in the continuation prompt.
func TestBuildImplContinuePrompt_NoPlaceholders(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	feedback := "Fix these issues"
	learnings := "Some learnings"

	result := BuildImplContinuePrompt(tasksFile, feedback, learnings)

	// Should not contain unreplaced template markers
	assert.NotContains(t, result, "{{TASKS_FILE}}", "should not contain tasks file placeholder")
	assert.NotContains(t, result, "{{FEEDBACK}}", "should not contain feedback placeholder")
	assert.NotContains(t, result, "{{EVIDENCE_RULES}}", "should not contain evidence rules placeholder")
	assert.NotContains(t, result, "{{PLAYWRIGHT_RULES}}", "should not contain playwright rules placeholder")
	assert.NotContains(t, result, "{{LEARNINGS}}", "should not contain learnings placeholder")
	assert.NotContains(t, result, "{{LEARNINGS_OUTPUT}}", "should not contain learnings output placeholder")
}

// TestBuildValidationPrompt_NoPlaceholders verifies that all placeholders are
// replaced in the validation prompt.
func TestBuildValidationPrompt_NoPlaceholders(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	implOutput := "Implementation output"

	result := BuildValidationPrompt(tasksFile, implOutput)

	// Should not contain unreplaced template markers
	assert.NotContains(t, result, "{{TASKS_FILE}}", "should not contain tasks file placeholder")
	assert.NotContains(t, result, "{{IMPL_OUTPUT}}", "should not contain impl output placeholder")
}

// TestBuildImplFirstPrompt_WithLearningsHasNoDoubleMarkers verifies that when
// learnings are provided, the nested learnings section placeholder is also replaced.
func TestBuildImplFirstPrompt_WithLearningsHasNoDoubleMarkers(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	learnings := "Pattern: Use dependency injection"

	result := BuildImplFirstPrompt(tasksFile, learnings)

	// The LEARNINGS placeholder inside learnings-section.txt should be replaced
	assert.NotContains(t, result, "{{LEARNINGS}}", "should not contain nested learnings placeholder")
}

// TestPromptStructure verifies that prompts have expected structural elements.
func TestPromptStructure(t *testing.T) {
	t.Run("ImplFirst has clear workflow", func(t *testing.T) {
		result := BuildImplFirstPrompt("/path/to/tasks.md", "")
		assert.Contains(t, result, "WORKFLOW:", "should include workflow section")
		assert.Contains(t, result, "BEGIN.", "should have clear begin instruction")
	})

	t.Run("ImplContinue emphasizes fixing", func(t *testing.T) {
		result := BuildImplContinuePrompt("/path/to/tasks.md", "feedback", "")
		assert.Contains(t, result, "FIX YOUR MISTAKES", "should emphasize fixing")
		assert.Contains(t, result, "REMEMBER:", "should remind of rules")
	})

	t.Run("Validation establishes adversarial role", func(t *testing.T) {
		result := BuildValidationPrompt("/path/to/tasks.md", "output")
		assert.Contains(t, result, "BE RUTHLESS", "should encourage strict validation")
		assert.Contains(t, result, "CATCH THEIR LIES", "should emphasize catching errors")
	})
}

// TestPromptLength verifies that prompts are substantial and comprehensive.
func TestPromptLength(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		minLines int
	}{
		{
			name:     "ImplFirst",
			prompt:   BuildImplFirstPrompt("/path/to/tasks.md", ""),
			minLines: 50,
		},
		{
			name:     "ImplContinue",
			prompt:   BuildImplContinuePrompt("/path/to/tasks.md", "feedback", ""),
			minLines: 30,
		},
		{
			name:     "Validation",
			prompt:   BuildValidationPrompt("/path/to/tasks.md", "output"),
			minLines: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := strings.Split(tt.prompt, "\n")
			assert.GreaterOrEqual(t, len(lines), tt.minLines,
				"prompt should be comprehensive with at least %d lines", tt.minLines)
		})
	}
}

// TestBuildCrossValidationPrompt_IncludesTasksFile verifies that the tasks file path
// is correctly included in the cross-validation prompt.
func TestBuildCrossValidationPrompt_IncludesTasksFile(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	valOutputFile := "/path/to/val-output.txt"
	implOutputFile := "/path/to/impl-output.txt"

	result := BuildCrossValidationPrompt(tasksFile, valOutputFile, implOutputFile)

	assert.Contains(t, result, tasksFile, "prompt should include the tasks file path")
	assert.Contains(t, result, "TASKS FILE:", "prompt should have tasks file label")
}

// TestBuildCrossValidationPrompt_IncludesImplOutput verifies that the implementation
// output file path is included in the cross-validation prompt.
func TestBuildCrossValidationPrompt_IncludesImplOutput(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	valOutputFile := "/path/to/val-output.txt"
	implOutputFile := "/path/to/impl-output.txt"

	result := BuildCrossValidationPrompt(tasksFile, valOutputFile, implOutputFile)

	assert.Contains(t, result, "IMPLEMENTATION OUTPUT FILE", "prompt should include impl output file header")
	assert.Contains(t, result, implOutputFile, "prompt should include the implementation output file path")
}

// TestBuildCrossValidationPrompt_IncludesValOutput verifies that the first validator's
// output file path is included in the cross-validation prompt.
func TestBuildCrossValidationPrompt_IncludesValOutput(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	valOutputFile := "/path/to/val-output.txt"
	implOutputFile := "/path/to/impl-output.txt"

	result := BuildCrossValidationPrompt(tasksFile, valOutputFile, implOutputFile)

	assert.Contains(t, result, "FIRST VALIDATOR OUTPUT FILE", "prompt should include validator output file header")
	assert.Contains(t, result, valOutputFile, "prompt should include the validator output file path")
}

// TestBuildCrossValidationPrompt_IncludesCrossValidatorRole verifies that the
// cross-validation prompt establishes the cross-validator role.
func TestBuildCrossValidationPrompt_IncludesCrossValidatorRole(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	valOutputFile := "/path/to/val-output.txt"
	implOutputFile := "/path/to/impl-output.txt"

	result := BuildCrossValidationPrompt(tasksFile, valOutputFile, implOutputFile)

	assert.Contains(t, result, "CROSS-VALIDATOR", "prompt should mention cross-validator role")
	assert.Contains(t, result, "SECOND OPINION", "prompt should emphasize second opinion")
	assert.Contains(t, result, "DO NOT JUST RUBBER-STAMP", "prompt should warn against rubber-stamping")
}

// TestBuildCrossValidationPrompt_NoPlaceholders verifies that all placeholders are replaced.
func TestBuildCrossValidationPrompt_NoPlaceholders(t *testing.T) {
	tasksFile := "/path/to/tasks.md"
	valOutputFile := "/path/to/val-output.txt"
	implOutputFile := "/path/to/impl-output.txt"

	result := BuildCrossValidationPrompt(tasksFile, valOutputFile, implOutputFile)

	assert.NotContains(t, result, "{{TASKS_FILE}}", "should not contain tasks file placeholder")
	assert.NotContains(t, result, "{{IMPL_OUTPUT_FILE}}", "should not contain impl output file placeholder")
	assert.NotContains(t, result, "{{VAL_OUTPUT_FILE}}", "should not contain val output file placeholder")
}

// TestBuildTasksValidationPrompt_IncludesSpecFile verifies that the spec file path
// is correctly included in the tasks validation prompt.
func TestBuildTasksValidationPrompt_IncludesSpecFile(t *testing.T) {
	specFile := "/path/to/spec.md"
	tasksFile := "/path/to/tasks.md"

	result := BuildTasksValidationPrompt(specFile, tasksFile)

	assert.Contains(t, result, specFile, "prompt should include the spec file path")
	assert.Contains(t, result, "SPEC FILE TO VALIDATE AGAINST", "prompt should have spec file label")
}

// TestBuildTasksValidationPrompt_IncludesTasksFile verifies that the tasks file path
// is correctly included in the tasks validation prompt.
func TestBuildTasksValidationPrompt_IncludesTasksFile(t *testing.T) {
	specFile := "/path/to/spec.md"
	tasksFile := "/path/to/tasks.md"

	result := BuildTasksValidationPrompt(specFile, tasksFile)

	assert.Contains(t, result, tasksFile, "prompt should include the tasks file path")
	assert.Contains(t, result, "TASKS FILE TO VALIDATE", "prompt should have tasks file label")
}

// TestBuildTasksValidationPrompt_IncludesValidationCriteria verifies that the
// tasks validation prompt includes validation criteria.
func TestBuildTasksValidationPrompt_IncludesValidationCriteria(t *testing.T) {
	specFile := "/path/to/spec.md"
	tasksFile := "/path/to/tasks.md"

	result := BuildTasksValidationPrompt(specFile, tasksFile)

	assert.Contains(t, result, "COMPLETE", "prompt should mention completeness check")
	assert.Contains(t, result, "ACCURATE", "prompt should mention accuracy check")
	assert.Contains(t, result, "ACTIONABLE", "prompt should mention actionability check")
	assert.Contains(t, result, "IN SCOPE", "prompt should mention scope check")
}

// TestBuildTasksValidationPrompt_NoPlaceholders verifies that all placeholders are replaced.
func TestBuildTasksValidationPrompt_NoPlaceholders(t *testing.T) {
	specFile := "/path/to/spec.md"
	tasksFile := "/path/to/tasks.md"

	result := BuildTasksValidationPrompt(specFile, tasksFile)

	assert.NotContains(t, result, "{{SPEC_FILE}}", "should not contain spec file placeholder")
	assert.NotContains(t, result, "{{TASKS_FILE}}", "should not contain tasks file placeholder")
}

// TestBuildFinalPlanPrompt_IncludesSpecFile verifies that the spec file path
// is correctly included in the final plan prompt.
func TestBuildFinalPlanPrompt_IncludesSpecFile(t *testing.T) {
	specFile := "/path/to/spec.md"
	tasksFile := "/path/to/tasks.md"
	planFile := "/path/to/plan.md"

	result := BuildFinalPlanPrompt(specFile, tasksFile, planFile)

	assert.Contains(t, result, specFile, "prompt should include the spec file path")
	assert.Contains(t, result, "SPEC FILE:", "prompt should have spec file label")
}

// TestBuildFinalPlanPrompt_IncludesTasksFile verifies that the tasks file path
// is correctly included in the final plan prompt.
func TestBuildFinalPlanPrompt_IncludesTasksFile(t *testing.T) {
	specFile := "/path/to/spec.md"
	tasksFile := "/path/to/tasks.md"
	planFile := "/path/to/plan.md"

	result := BuildFinalPlanPrompt(specFile, tasksFile, planFile)

	assert.Contains(t, result, tasksFile, "prompt should include the tasks file path")
	assert.Contains(t, result, "TASKS FILE", "prompt should have tasks file label")
}

// TestBuildFinalPlanPrompt_IncludesPlanFile verifies that the plan file path
// is correctly included in the final plan prompt.
func TestBuildFinalPlanPrompt_IncludesPlanFile(t *testing.T) {
	specFile := "/path/to/spec.md"
	tasksFile := "/path/to/tasks.md"
	planFile := "/path/to/plan.md"

	result := BuildFinalPlanPrompt(specFile, tasksFile, planFile)

	assert.Contains(t, result, planFile, "prompt should include the plan file path")
	assert.Contains(t, result, "PLAN FILE:", "prompt should have plan file label")
}

// TestBuildFinalPlanPrompt_IncludesValidationRole verifies that the final plan
// prompt establishes the validator role and checkpoint message.
func TestBuildFinalPlanPrompt_IncludesValidationRole(t *testing.T) {
	specFile := "/path/to/spec.md"
	tasksFile := "/path/to/tasks.md"
	planFile := "/path/to/plan.md"

	result := BuildFinalPlanPrompt(specFile, tasksFile, planFile)

	assert.Contains(t, result, "LAST CHECKPOINT", "prompt should mention last checkpoint")
	assert.Contains(t, result, "before execution begins", "prompt should emphasize timing")
}

// TestBuildFinalPlanPrompt_NoPlaceholders verifies that all placeholders are replaced.
func TestBuildFinalPlanPrompt_NoPlaceholders(t *testing.T) {
	specFile := "/path/to/spec.md"
	tasksFile := "/path/to/tasks.md"
	planFile := "/path/to/plan.md"

	result := BuildFinalPlanPrompt(specFile, tasksFile, planFile)

	assert.NotContains(t, result, "{{SPEC_FILE}}", "should not contain spec file placeholder")
	assert.NotContains(t, result, "{{TASKS_FILE}}", "should not contain tasks file placeholder")
	assert.NotContains(t, result, "{{PLAN_FILE}}", "should not contain plan file placeholder")
	assert.NotContains(t, result, "{{ORIGINAL_PLAN}}", "should not contain original plan placeholder")
}

// TestNewPromptBuilders_NonEmpty verifies that all new builder functions
// return non-empty prompts.
func TestNewPromptBuilders_NonEmpty(t *testing.T) {
	tests := []struct {
		name   string
		result string
	}{
		{
			name:   "BuildCrossValidationPrompt",
			result: BuildCrossValidationPrompt("/tasks.md", "impl", "val"),
		},
		{
			name:   "BuildTasksValidationPrompt",
			result: BuildTasksValidationPrompt("/spec.md", "/tasks.md"),
		},
		{
			name:   "BuildFinalPlanPrompt",
			result: BuildFinalPlanPrompt("/spec.md", "/tasks.md", "/plan.md"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotEmpty(t, tt.result, "prompt should not be empty")
			assert.Greater(t, len(tt.result), 50, "prompt should be substantial")
		})
	}
}
