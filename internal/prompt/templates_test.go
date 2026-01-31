package prompt

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTemplatesLoad verifies that all template files are loaded via go:embed
// and are non-empty.
func TestTemplatesLoad(t *testing.T) {
	tests := []struct {
		name     string
		template string
	}{
		{"ImplFirstTemplate", ImplFirstTemplate},
		{"ImplContinueTemplate", ImplContinueTemplate},
		{"InadmissibleRules", InadmissibleRules},
		{"EvidenceRules", EvidenceRules},
		{"PlaywrightRules", PlaywrightRules},
		{"LearningsSection", LearningsSection},
		{"LearningsOutput", LearningsOutput},
		{"ValidationTemplate", ValidationTemplate},
		{"CrossValidationTemplate", CrossValidationTemplate},
		{"TasksValidationTemplate", TasksValidationTemplate},
		{"FinalPlanTemplate", FinalPlanTemplate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotEmpty(t, tt.template, "template %s should not be empty", tt.name)
			assert.Greater(t, len(tt.template), 10, "template %s should have substantial content", tt.name)
		})
	}
}

// TestImplFirstTemplate_ContainsKeyMarkers verifies that the impl-first template
// contains expected placeholder markers and key content.
func TestImplFirstTemplate_ContainsKeyMarkers(t *testing.T) {
	// Check for placeholder markers
	assert.Contains(t, ImplFirstTemplate, "{{TASKS_FILE}}", "should have tasks file marker")
	assert.Contains(t, ImplFirstTemplate, "{{INADMISSIBLE_RULES}}", "should have inadmissible rules marker")
	assert.Contains(t, ImplFirstTemplate, "{{EVIDENCE_RULES}}", "should have evidence rules marker")
	assert.Contains(t, ImplFirstTemplate, "{{PLAYWRIGHT_RULES}}", "should have playwright rules marker")
	assert.Contains(t, ImplFirstTemplate, "{{LEARNINGS_SECTION}}", "should have learnings section marker")
	assert.Contains(t, ImplFirstTemplate, "{{LEARNINGS_OUTPUT}}", "should have learnings output marker")

	// Check for key content
	assert.Contains(t, ImplFirstTemplate, "ABSOLUTE RULES", "should mention absolute rules")
	assert.Contains(t, ImplFirstTemplate, "VIOLATION MEANS FAILURE", "should emphasize rule violations")
	assert.Contains(t, ImplFirstTemplate, "WORKFLOW:", "should include workflow section")
	assert.Contains(t, ImplFirstTemplate, "RALPH_STATUS", "should mention RALPH_STATUS output")
	assert.Contains(t, ImplFirstTemplate, "completed_tasks", "should mention completed_tasks field")
	assert.Contains(t, ImplFirstTemplate, "blocked_tasks", "should mention blocked_tasks field")
}

// TestImplContinueTemplate_ContainsKeyMarkers verifies that the impl-continue
// template contains expected markers and continuation-specific content.
func TestImplContinueTemplate_ContainsKeyMarkers(t *testing.T) {
	// Check for placeholder markers
	assert.Contains(t, ImplContinueTemplate, "{{TASKS_FILE}}", "should have tasks file marker")
	assert.Contains(t, ImplContinueTemplate, "{{FEEDBACK}}", "should have feedback marker")
	assert.Contains(t, ImplContinueTemplate, "{{EVIDENCE_RULES}}", "should have evidence rules marker")
	assert.Contains(t, ImplContinueTemplate, "{{PLAYWRIGHT_RULES}}", "should have playwright rules marker")
	assert.Contains(t, ImplContinueTemplate, "{{LEARNINGS_SECTION}}", "should have learnings section marker")
	assert.Contains(t, ImplContinueTemplate, "{{LEARNINGS_OUTPUT}}", "should have learnings output marker")

	// Check for key content
	assert.Contains(t, ImplContinueTemplate, "VALIDATION CAUGHT YOUR LIES", "should have feedback header")
	assert.Contains(t, ImplContinueTemplate, "FIX YOUR LIES NOW", "should emphasize fixing")
	assert.Contains(t, ImplContinueTemplate, "REMEMBER:", "should have reminder section")
	assert.Contains(t, ImplContinueTemplate, "CRITICAL", "should emphasize critical rules")
	assert.Contains(t, ImplContinueTemplate, "DO NOT WRITE TESTS FOR NON-EXISTENT FUNCTIONALITY", "should warn about non-existent functionality")
	assert.Contains(t, ImplContinueTemplate, "FIX YOUR MISTAKES", "should have fixing instruction")
}

// TestInadmissibleRulesTemplate_ContainsKeyMarkers verifies that the
// inadmissible rules template contains all required inadmissible practices.
func TestInadmissibleRulesTemplate_ContainsKeyMarkers(t *testing.T) {
	assert.Contains(t, InadmissibleRules, "INADMISSIBLE PRACTICES", "should have header")
	assert.Contains(t, InadmissibleRules, "AUTOMATIC FAILURE", "should emphasize automatic failure")

	// Check for all four main inadmissible practices
	assert.Contains(t, InadmissibleRules, "PRODUCTION CODE DUPLICATION IN TESTS", "should list duplication practice")
	assert.Contains(t, InadmissibleRules, "MOCK THE SUBJECT UNDER TEST", "should list mocking practice")
	assert.Contains(t, InadmissibleRules, "TRIVIAL/EMPTY TESTS", "should list trivial tests practice")
	assert.Contains(t, InadmissibleRules, "TESTS FOR NON-EXISTENT FUNCTIONALITY", "should list non-existent functionality practice")

	// Check for examples
	assert.Contains(t, InadmissibleRules, "WRONG:", "should provide wrong examples")
	assert.Contains(t, InadmissibleRules, "RIGHT:", "should provide right examples")
	assert.Contains(t, InadmissibleRules, "EXAMPLES OF INADMISSIBLE TEST-WRITING", "should have examples section")

	// Check for specific examples
	assert.Contains(t, InadmissibleRules, "page.keyboard.press", "should have keyboard example")
	assert.Contains(t, InadmissibleRules, "validateEmail", "should have function example")
	assert.Contains(t, InadmissibleRules, "/api/delete-user", "should have API endpoint example")
	assert.Contains(t, InadmissibleRules, "primary-view", "should have UI element example")

	// Check for detection and resolution guidance
	assert.Contains(t, InadmissibleRules, "DETECTION", "should explain detection process")
	assert.Contains(t, InadmissibleRules, "WHY THIS IS INADMISSIBLE", "should explain why it matters")
	assert.Contains(t, InadmissibleRules, "Implementation first, then tests", "should emphasize correct order")
}

// TestEvidenceRulesTemplate_ContainsKeyMarkers verifies that the evidence rules
// template contains guidance for capturing evidence.
func TestEvidenceRulesTemplate_ContainsKeyMarkers(t *testing.T) {
	assert.Contains(t, EvidenceRules, "EVIDENCE CAPTURE FOR NON-FILE TASKS", "should have header")

	// Check for task types
	assert.Contains(t, EvidenceRules, "Deploy X", "should mention deploy tasks")
	assert.Contains(t, EvidenceRules, "Run tests", "should mention test tasks")
	assert.Contains(t, EvidenceRules, "Build X", "should mention build tasks")
	assert.Contains(t, EvidenceRules, "Verify X", "should mention verify tasks")
	assert.Contains(t, EvidenceRules, "Run/Execute X", "should mention execute tasks")
	assert.Contains(t, EvidenceRules, "Playwright MCP", "should mention Playwright MCP tasks")

	// Check for examples of what to record
	assert.Contains(t, EvidenceRules, "Version deployed", "should show deploy evidence example")
	assert.Contains(t, EvidenceRules, "passed", "should show test result example")
	assert.Contains(t, EvidenceRules, "Build succeeded", "should show build evidence example")
	assert.Contains(t, EvidenceRules, "Screenshot path", "should show Playwright evidence example")
}

// TestPlaywrightRulesTemplate_ContainsKeyMarkers verifies that the Playwright
// rules template contains mandatory execution requirements.
func TestPlaywrightRulesTemplate_ContainsKeyMarkers(t *testing.T) {
	assert.Contains(t, PlaywrightRules, "PLAYWRIGHT MCP VALIDATION", "should have header")
	assert.Contains(t, PlaywrightRules, "MANDATORY EXECUTION", "should emphasize mandatory nature")

	// Check for key rules
	assert.Contains(t, PlaywrightRules, "APP NOT RUNNING", "should address app not running scenario")
	assert.Contains(t, PlaywrightRules, "IS NOT A BLOCKER", "should emphasize it's not a blocker")
	assert.Contains(t, PlaywrightRules, "START IT YOURSELF", "should instruct to start app")

	// Check for execution sequence
	assert.Contains(t, PlaywrightRules, "EXECUTION SEQUENCE", "should have execution sequence")
	assert.Contains(t, PlaywrightRules, "Start the application", "should mention starting app")
	assert.Contains(t, PlaywrightRules, "Wait for HTTP response", "should mention waiting for response")
	assert.Contains(t, PlaywrightRules, "Use Playwright MCP", "should mention using Playwright MCP")

	// Check for forbidden excuses
	assert.Contains(t, PlaywrightRules, "FORBIDDEN EXCUSES", "should list forbidden excuses")
	assert.Contains(t, PlaywrightRules, "App not running", "should list app not running excuse")
	assert.Contains(t, PlaywrightRules, "Server not started", "should list server not started excuse")
	assert.Contains(t, PlaywrightRules, "INADMISSIBLE verdict", "should mention inadmissible consequence")
}

// TestLearningsSectionTemplate_ContainsKeyMarkers verifies that the learnings
// section template has the correct structure for including learnings.
func TestLearningsSectionTemplate_ContainsKeyMarkers(t *testing.T) {
	assert.Contains(t, LearningsSection, "LEARNINGS FROM PREVIOUS ITERATIONS", "should have header")
	assert.Contains(t, LearningsSection, "{{LEARNINGS}}", "should have learnings placeholder")
	assert.Contains(t, LearningsSection, "Read these FIRST", "should emphasize reading first")
	assert.Contains(t, LearningsSection, "Codebase Patterns", "should mention codebase patterns")
}

// TestLearningsOutputTemplate_ContainsKeyMarkers verifies that the learnings
// output template provides the correct format for outputting learnings.
func TestLearningsOutputTemplate_ContainsKeyMarkers(t *testing.T) {
	assert.Contains(t, LearningsOutput, "LEARNINGS OUTPUT", "should have header")
	assert.Contains(t, LearningsOutput, "RALPH_LEARNINGS", "should mention RALPH_LEARNINGS marker")

	// Check for format guidance
	assert.Contains(t, LearningsOutput, "Pattern:", "should show Pattern format")
	assert.Contains(t, LearningsOutput, "Gotcha:", "should show Gotcha format")
	assert.Contains(t, LearningsOutput, "Context:", "should show Context format")

	// Check for guidance
	assert.Contains(t, LearningsOutput, "GENERAL learnings", "should emphasize general learnings")
	assert.Contains(t, LearningsOutput, "Do NOT include task-specific details", "should warn against task-specific details")
}

// TestValidationTemplate_ContainsKeyMarkers verifies that the validation
// template contains all validation rules and checks.
func TestValidationTemplate_ContainsKeyMarkers(t *testing.T) {
	// Check for placeholder markers
	assert.Contains(t, ValidationTemplate, "{{TASKS_FILE}}", "should have tasks file marker")
	assert.Contains(t, ValidationTemplate, "{{IMPL_OUTPUT_FILE}}", "should have impl output file marker")

	// Check for role establishment
	assert.Contains(t, ValidationTemplate, "VALIDATOR", "should establish validator role")
	assert.Contains(t, ValidationTemplate, "THE IMPLEMENTER IS A LIAR", "should establish adversarial stance")
	assert.Contains(t, ValidationTemplate, "DO NOT TRUST THEM", "should emphasize distrust")

	// Check for validation rules
	assert.Contains(t, ValidationTemplate, "VALIDATION RULES", "should have validation rules section")
	assert.Contains(t, ValidationTemplate, "READ THE TASKS FILE YOURSELF", "should emphasize independent verification")
	assert.Contains(t, ValidationTemplate, "CHECK EACH TASK", "should mention checking tasks")

	// Check for inadmissible practices
	assert.Contains(t, ValidationTemplate, "INADMISSIBLE PRACTICES", "should have inadmissible section")
	assert.Contains(t, ValidationTemplate, "AUTO-FAIL", "should emphasize automatic failure")
	assert.Contains(t, ValidationTemplate, "PRODUCTION CODE DUPLICATION IN TESTS", "should check for duplication")
	assert.Contains(t, ValidationTemplate, "MOCKING THE SUBJECT UNDER TEST", "should check for mocking")
	assert.Contains(t, ValidationTemplate, "TRIVIAL/EMPTY TESTS", "should check for trivial tests")
	assert.Contains(t, ValidationTemplate, "TESTS FOR NON-EXISTENT FUNCTIONALITY", "should check for non-existent functionality")

	// Check for detection process
	assert.Contains(t, ValidationTemplate, "DETECTION PROCESS", "should have detection process")
	assert.Contains(t, ValidationTemplate, "Read ALL test files", "should mention reading test files")
	assert.Contains(t, ValidationTemplate, "search the PRODUCTION code", "should mention searching production code")

	// Check for common lies
	assert.Contains(t, ValidationTemplate, "COMMON LIES TO CATCH", "should have common lies section")
	assert.Contains(t, ValidationTemplate, "I removed X", "should list removal lie")
	assert.Contains(t, ValidationTemplate, "I created Y", "should list creation lie")
	assert.Contains(t, ValidationTemplate, "Task is N/A", "should list N/A lie")

	// Check for verdict options
	assert.Contains(t, ValidationTemplate, "VERDICT OPTIONS", "should have verdict options")
	assert.Contains(t, ValidationTemplate, "COMPLETE", "should list COMPLETE verdict")
	assert.Contains(t, ValidationTemplate, "NEEDS_MORE_WORK", "should list NEEDS_MORE_WORK verdict")
	assert.Contains(t, ValidationTemplate, "INADMISSIBLE", "should list INADMISSIBLE verdict")
	assert.Contains(t, ValidationTemplate, "ESCALATE", "should list ESCALATE verdict")
	assert.Contains(t, ValidationTemplate, "BLOCKED", "should list BLOCKED verdict")

	// Check for output format
	assert.Contains(t, ValidationTemplate, "RALPH_VALIDATION", "should mention RALPH_VALIDATION")
	assert.Contains(t, ValidationTemplate, "verdict", "should have verdict field")
	assert.Contains(t, ValidationTemplate, "feedback", "should have feedback field")
	assert.Contains(t, ValidationTemplate, "completed_tasks", "should have completed_tasks field")
	assert.Contains(t, ValidationTemplate, "incomplete_tasks", "should have incomplete_tasks field")
	assert.Contains(t, ValidationTemplate, "inadmissible_practices", "should have inadmissible_practices field")

	// Check for final instructions
	assert.Contains(t, ValidationTemplate, "NOW VALIDATE", "should have validate instruction")
	assert.Contains(t, ValidationTemplate, "BE RUTHLESS", "should encourage strict validation")
	assert.Contains(t, ValidationTemplate, "CATCH THEIR LIES", "should emphasize catching errors")
}

// TestCrossValidationTemplate_ContainsKeyMarkers verifies that the cross-validation
// template establishes the second opinion role.
func TestCrossValidationTemplate_ContainsKeyMarkers(t *testing.T) {
	assert.Contains(t, CrossValidationTemplate, "CROSS-VALIDATOR", "should establish cross-validator role")
	assert.Contains(t, CrossValidationTemplate, "SECOND OPINION", "should emphasize second opinion")
	assert.Contains(t, CrossValidationTemplate, "{{TASKS_FILE}}", "should have tasks file marker")
	assert.Contains(t, CrossValidationTemplate, "{{IMPL_OUTPUT_FILE}}", "should have impl output file marker")
	assert.Contains(t, CrossValidationTemplate, "{{VAL_OUTPUT_FILE}}", "should have val output file marker")
	assert.Contains(t, CrossValidationTemplate, "DO NOT JUST RUBBER-STAMP", "should warn against rubber-stamping")
	assert.Contains(t, CrossValidationTemplate, "NOW CROSS-VALIDATE", "should have final instruction")
}

// TestTasksValidationTemplate_ContainsKeyMarkers verifies that the tasks
// validation template checks tasks against spec.
func TestTasksValidationTemplate_ContainsKeyMarkers(t *testing.T) {
	assert.Contains(t, TasksValidationTemplate, "validating that a tasks.md file", "should explain validation purpose")
	assert.Contains(t, TasksValidationTemplate, "{{SPEC_FILE}}", "should have spec file marker")
	assert.Contains(t, TasksValidationTemplate, "{{TASKS_FILE}}", "should have tasks file marker")
	assert.Contains(t, TasksValidationTemplate, "COMPLETE", "should check completeness")
	assert.Contains(t, TasksValidationTemplate, "ACCURATE", "should check accuracy")
	assert.Contains(t, TasksValidationTemplate, "ACTIONABLE", "should check actionability")
	assert.Contains(t, TasksValidationTemplate, "IN SCOPE", "should check scope")
	assert.Contains(t, TasksValidationTemplate, "NOW VALIDATE", "should have final instruction")
}

// TestFinalPlanTemplate_ContainsKeyMarkers verifies that the final plan
// template establishes the checkpoint role.
func TestFinalPlanTemplate_ContainsKeyMarkers(t *testing.T) {
	assert.Contains(t, FinalPlanTemplate, "final implementation plan", "should explain validation purpose")
	assert.Contains(t, FinalPlanTemplate, "LAST CHECKPOINT", "should emphasize last checkpoint")
	assert.Contains(t, FinalPlanTemplate, "{{SPEC_FILE}}", "should have spec file marker")
	assert.Contains(t, FinalPlanTemplate, "{{TASKS_FILE}}", "should have tasks file marker")
	assert.Contains(t, FinalPlanTemplate, "{{PLAN_FILE}}", "should have plan file marker")
	assert.Contains(t, FinalPlanTemplate, "correctly interprets the spec", "should check spec interpretation")
	assert.Contains(t, FinalPlanTemplate, "complete and covers all requirements", "should check completeness")
	assert.Contains(t, FinalPlanTemplate, "NOW VALIDATE", "should have final instruction")
}

// TestTemplateMarkerConsistency verifies that templates use consistent
// marker naming conventions.
func TestTemplateMarkerConsistency(t *testing.T) {
	// All placeholders should use {{UPPERCASE_MARKER}} format
	templates := map[string]string{
		"ImplFirstTemplate":       ImplFirstTemplate,
		"ImplContinueTemplate":    ImplContinueTemplate,
		"ValidationTemplate":      ValidationTemplate,
		"LearningsSection":        LearningsSection,
		"CrossValidationTemplate": CrossValidationTemplate,
		"TasksValidationTemplate": TasksValidationTemplate,
		"FinalPlanTemplate":       FinalPlanTemplate,
	}

	for name, template := range templates {
		t.Run(name, func(t *testing.T) {
			// Find all markers in the template
			markers := findMarkers(template)
			for _, marker := range markers {
				// Check that marker is uppercase
				assert.Equal(t, strings.ToUpper(marker), marker,
					"marker %s should be uppercase", marker)
				// Check that marker doesn't contain spaces
				assert.NotContains(t, marker, " ",
					"marker %s should not contain spaces", marker)
			}
		})
	}
}

// TestTemplateNoTypos verifies that templates don't contain common typos
// in critical keywords.
func TestTemplateNoTypos(t *testing.T) {
	allTemplates := []struct {
		name     string
		template string
	}{
		{"ImplFirstTemplate", ImplFirstTemplate},
		{"ImplContinueTemplate", ImplContinueTemplate},
		{"ValidationTemplate", ValidationTemplate},
		{"InadmissibleRules", InadmissibleRules},
	}

	for _, tt := range allTemplates {
		t.Run(tt.name, func(t *testing.T) {
			// Check for common typos (add more as needed)
			assert.NotContains(t, tt.template, "INADMISSABLE", "should use INADMISSIBLE not INADMISSABLE")
			assert.NotContains(t, tt.template, "PLAYWRIGT", "should use PLAYWRIGHT not PLAYWRIGT")
			assert.NotContains(t, tt.template, "RALP_", "should use RALPH_ not RALP_")
		})
	}
}

// TestTemplateLineBreaks verifies that templates have reasonable line breaks
// and aren't all on one line.
func TestTemplateLineBreaks(t *testing.T) {
	templates := []struct {
		name     string
		template string
		minLines int
	}{
		{"ImplFirstTemplate", ImplFirstTemplate, 40},
		{"ImplContinueTemplate", ImplContinueTemplate, 25},
		{"ValidationTemplate", ValidationTemplate, 100},
		{"InadmissibleRules", InadmissibleRules, 60},
		{"EvidenceRules", EvidenceRules, 10},
		{"PlaywrightRules", PlaywrightRules, 20},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			lines := strings.Split(tt.template, "\n")
			assert.GreaterOrEqual(t, len(lines), tt.minLines,
				"template should have at least %d lines for readability", tt.minLines)
		})
	}
}

// TestInadmissibleExamplesCompleteness verifies that the inadmissible rules
// template provides both positive and negative examples for each practice.
func TestInadmissibleExamplesCompleteness(t *testing.T) {
	// Should have examples showing both wrong and right approaches
	wrongCount := strings.Count(InadmissibleRules, "❌")
	rightCount := strings.Count(InadmissibleRules, "✅")

	assert.Greater(t, wrongCount, 0, "should have wrong examples marked with ❌")
	assert.Greater(t, rightCount, 0, "should have right examples marked with ✅")

	// Should have at least 4 wrong examples (one for each inadmissible practice)
	assert.GreaterOrEqual(t, wrongCount, 4, "should have examples for all inadmissible practices")
}

// TestValidationDetectionProcess verifies that the validation template
// includes a detailed detection process for inadmissible practices.
func TestValidationDetectionProcess(t *testing.T) {
	// Should have numbered or lettered steps
	assert.Contains(t, ValidationTemplate, "a.", "detection process should have step a")
	assert.Contains(t, ValidationTemplate, "b.", "detection process should have step b")
	assert.Contains(t, ValidationTemplate, "c.", "detection process should have step c")
	assert.Contains(t, ValidationTemplate, "d.", "detection process should have step d")

	// Should mention specific things to check
	assert.Contains(t, ValidationTemplate, "Read ALL test files", "should mention reading test files")
	assert.Contains(t, ValidationTemplate, "identify what functionality", "should mention identifying functionality")
	assert.Contains(t, ValidationTemplate, "search the PRODUCTION code", "should mention searching production")
}

// TestTemplateEmphasizes verifies that templates use emphasis appropriately
// for critical instructions.
func TestTemplateEmphasizes(t *testing.T) {
	tests := []struct {
		name     string
		template string
		emphasis []string
	}{
		{
			name:     "ImplFirstTemplate",
			template: ImplFirstTemplate,
			emphasis: []string{"ABSOLUTE RULES", "VIOLATION MEANS FAILURE"},
		},
		{
			name:     "ImplContinueTemplate",
			template: ImplContinueTemplate,
			emphasis: []string{"CRITICAL", "FIX YOUR LIES"},
		},
		{
			name:     "ValidationTemplate",
			template: ValidationTemplate,
			emphasis: []string{"THE IMPLEMENTER IS A LIAR", "BE RUTHLESS"},
		},
		{
			name:     "InadmissibleRules",
			template: InadmissibleRules,
			emphasis: []string{"AUTOMATIC FAILURE", "INADMISSIBLE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, emphasized := range tt.emphasis {
				assert.Contains(t, tt.template, emphasized,
					"template should emphasize %q", emphasized)
			}
		})
	}
}

// findMarkers extracts all {{MARKER}} placeholders from a template.
func findMarkers(template string) []string {
	var markers []string
	parts := strings.Split(template, "{{")
	for _, part := range parts[1:] {
		if idx := strings.Index(part, "}}"); idx != -1 {
			markers = append(markers, part[:idx])
		}
	}
	return markers
}
