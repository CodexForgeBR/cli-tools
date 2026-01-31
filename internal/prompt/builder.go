package prompt

import "strings"

// BuildImplFirstPrompt constructs the first implementation iteration prompt.
// It includes inadmissible rules, evidence capture rules, playwright rules,
// and optionally includes learnings from previous sessions.
func BuildImplFirstPrompt(tasksFile string, learnings string) string {
	prompt := ImplFirstTemplate

	// Replace task file reference
	prompt = strings.ReplaceAll(prompt, "{{TASKS_FILE}}", tasksFile)

	// Include inadmissible rules section
	prompt = strings.ReplaceAll(prompt, "{{INADMISSIBLE_RULES}}", InadmissibleRules)

	// Include evidence capture rules
	prompt = strings.ReplaceAll(prompt, "{{EVIDENCE_RULES}}", EvidenceRules)

	// Include playwright rules
	prompt = strings.ReplaceAll(prompt, "{{PLAYWRIGHT_RULES}}", PlaywrightRules)

	// Include learnings section if provided
	if learnings != "" {
		learningsContent := strings.ReplaceAll(LearningsSection, "{{LEARNINGS}}", learnings)
		prompt = strings.ReplaceAll(prompt, "{{LEARNINGS_SECTION}}", learningsContent)
	} else {
		prompt = strings.ReplaceAll(prompt, "{{LEARNINGS_SECTION}}", "")
	}

	// Include learnings output instructions
	prompt = strings.ReplaceAll(prompt, "{{LEARNINGS_OUTPUT}}", LearningsOutput)

	return prompt
}

// BuildImplContinuePrompt constructs the continuation implementation prompt.
// This is used after validation finds issues that need to be fixed.
// It includes the validator's feedback and reminds about evidence and playwright rules.
func BuildImplContinuePrompt(tasksFile string, feedback string, learnings string) string {
	prompt := ImplContinueTemplate

	// Replace task file reference
	prompt = strings.ReplaceAll(prompt, "{{TASKS_FILE}}", tasksFile)

	// Include validation feedback
	prompt = strings.ReplaceAll(prompt, "{{FEEDBACK}}", feedback)

	// Include evidence capture rules
	prompt = strings.ReplaceAll(prompt, "{{EVIDENCE_RULES}}", EvidenceRules)

	// Include playwright rules
	prompt = strings.ReplaceAll(prompt, "{{PLAYWRIGHT_RULES}}", PlaywrightRules)

	// Include learnings section if provided
	if learnings != "" {
		learningsContent := strings.ReplaceAll(LearningsSection, "{{LEARNINGS}}", learnings)
		prompt = strings.ReplaceAll(prompt, "{{LEARNINGS_SECTION}}", learningsContent)
	} else {
		prompt = strings.ReplaceAll(prompt, "{{LEARNINGS_SECTION}}", "")
	}

	// Include learnings output instructions
	prompt = strings.ReplaceAll(prompt, "{{LEARNINGS_OUTPUT}}", LearningsOutput)

	return prompt
}

// BuildValidationPrompt constructs the validation phase prompt.
// The validator checks the implementer's work against the tasks file.
func BuildValidationPrompt(tasksFile string, implOutputFile string) string {
	prompt := ValidationTemplate

	// Replace task file reference
	prompt = strings.ReplaceAll(prompt, "{{TASKS_FILE}}", tasksFile)

	// Include implementation output file path
	prompt = strings.ReplaceAll(prompt, "{{IMPL_OUTPUT_FILE}}", implOutputFile)

	return prompt
}

// BuildCrossValidationPrompt constructs the cross-validation phase prompt.
// The cross-validator provides a second opinion on the validator's assessment.
func BuildCrossValidationPrompt(tasksFile string, valOutputFile string, implOutputFile string) string {
	prompt := CrossValidationTemplate

	// Replace task file reference
	prompt = strings.ReplaceAll(prompt, "{{TASKS_FILE}}", tasksFile)

	// Include implementation output file path
	prompt = strings.ReplaceAll(prompt, "{{IMPL_OUTPUT_FILE}}", implOutputFile)

	// Include first validator's output file path
	prompt = strings.ReplaceAll(prompt, "{{VAL_OUTPUT_FILE}}", valOutputFile)

	return prompt
}

// BuildTasksValidationPrompt constructs the tasks validation phase prompt.
// The validator checks if tasks.md correctly implements spec.md requirements.
func BuildTasksValidationPrompt(specFile string, tasksFile string) string {
	prompt := TasksValidationTemplate

	// Replace spec file reference
	prompt = strings.ReplaceAll(prompt, "{{SPEC_FILE}}", specFile)

	// Replace tasks file reference
	prompt = strings.ReplaceAll(prompt, "{{TASKS_FILE}}", tasksFile)

	return prompt
}

// BuildFinalPlanPrompt constructs the final plan validation phase prompt.
// The validator checks if the implementation plan is ready for execution.
func BuildFinalPlanPrompt(specFile string, tasksFile string, planFile string) string {
	prompt := FinalPlanTemplate

	// Replace spec file reference
	prompt = strings.ReplaceAll(prompt, "{{SPEC_FILE}}", specFile)

	// Replace tasks file reference
	prompt = strings.ReplaceAll(prompt, "{{TASKS_FILE}}", tasksFile)

	// Replace plan file reference (also accepts ORIGINAL_PLAN as alias)
	prompt = strings.ReplaceAll(prompt, "{{PLAN_FILE}}", planFile)
	prompt = strings.ReplaceAll(prompt, "{{ORIGINAL_PLAN}}", planFile)

	return prompt
}
