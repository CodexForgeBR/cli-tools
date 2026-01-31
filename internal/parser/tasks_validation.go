// Package parser provides text-parsing utilities for the ralph-loop CLI.
package parser

// TasksValidationResult holds the parsed fields from a RALPH_TASKS_VALIDATION JSON block.
// This structure represents validation feedback about whether tasks.md correctly
// implements the spec.md requirements.
type TasksValidationResult struct {
	// Verdict indicates the tasks validation outcome.
	// Valid values: VALID, INVALID
	Verdict string

	// Feedback provides detailed explanation of the verdict.
	Feedback string

	// MissingRequirements lists requirements from spec not covered in tasks
	MissingRequirements []string

	// OutOfScopeTasks lists tasks that add things not in spec
	OutOfScopeTasks []string

	// VagueTasks lists task IDs that need more clarity
	VagueTasks []string

	// QualityScore is a brief overall assessment
	QualityScore string
}

// ParseTasksValidation extracts RALPH_TASKS_VALIDATION fields from AI output text.
// Uses ExtractJSON to locate the JSON block, then maps fields to the result struct.
// Expects verdicts: VALID, INVALID
//
// Returns (nil, nil) if no RALPH_TASKS_VALIDATION block is found.
// Returns (nil, error) if the JSON is malformed.
// Returns (*TasksValidationResult, nil) if successfully parsed.
func ParseTasksValidation(text string) (*TasksValidationResult, error) {
	raw, err := ExtractJSON(text, "RALPH_TASKS_VALIDATION")
	if raw == nil || err != nil {
		return nil, err
	}

	// ExtractJSON returns the outer object containing RALPH_TASKS_VALIDATION.
	// Extract the nested RALPH_TASKS_VALIDATION object.
	tasksVal, ok := raw["RALPH_TASKS_VALIDATION"].(map[string]interface{})
	hasRalphTasksValidationKey := ok
	if !ok {
		// If RALPH_TASKS_VALIDATION is not a nested object, treat raw as the data
		tasksVal = raw
	}

	result := &TasksValidationResult{}

	// Track if we found any actual tasks validation fields
	hasTasksValidationFields := false

	// Extract verdict string (VALID or INVALID)
	if v, ok := tasksVal["verdict"].(string); ok {
		result.Verdict = v
		hasTasksValidationFields = true
	}

	// Extract feedback string
	if v, ok := tasksVal["feedback"].(string); ok {
		result.Feedback = v
		hasTasksValidationFields = true
	}

	// Extract missing_requirements array
	if v, ok := tasksVal["missing_requirements"].([]interface{}); ok {
		reqs := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				reqs = append(reqs, s)
			}
		}
		result.MissingRequirements = reqs
		hasTasksValidationFields = true
	}

	// Extract out_of_scope_tasks array
	if v, ok := tasksVal["out_of_scope_tasks"].([]interface{}); ok {
		tasks := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				tasks = append(tasks, s)
			}
		}
		result.OutOfScopeTasks = tasks
		hasTasksValidationFields = true
	}

	// Extract vague_tasks array
	if v, ok := tasksVal["vague_tasks"].([]interface{}); ok {
		tasks := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				tasks = append(tasks, s)
			}
		}
		result.VagueTasks = tasks
		hasTasksValidationFields = true
	}

	// Extract quality_score string
	if v, ok := tasksVal["quality_score"].(string); ok {
		result.QualityScore = v
		hasTasksValidationFields = true
	}

	// If no tasks validation fields were found AND there was no explicit RALPH_TASKS_VALIDATION key,
	// this was probably a false positive match (e.g., "RALPH_TASKS_VALIDATION" in text but not in JSON)
	if !hasTasksValidationFields && !hasRalphTasksValidationKey {
		return nil, nil
	}

	return result, nil
}
