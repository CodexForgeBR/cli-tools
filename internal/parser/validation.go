// Package parser provides text-parsing utilities for the ralph-loop CLI.
package parser

// ValidationResult holds the parsed fields from a RALPH_VALIDATION JSON block.
// This structure represents validation feedback from the AI agent about task
// completion status.
type ValidationResult struct {
	// Verdict indicates the validation outcome.
	// Valid values: COMPLETE, NEEDS_MORE_WORK, ESCALATE, BLOCKED, INADMISSIBLE
	Verdict string

	// Feedback provides detailed explanation of the verdict.
	Feedback string

	// Remaining is the count of tasks still pending completion.
	Remaining int

	// BlockedCount is the count of tasks that are blocked.
	BlockedCount int

	// BlockedTasks is a list of task identifiers that are blocked,
	// typically in the format "T###: description".
	BlockedTasks []string
}

// ParseValidation extracts RALPH_VALIDATION fields from AI output text.
// Uses ExtractJSON to locate the JSON block, then maps fields to the result struct.
//
// Returns (nil, nil) if no RALPH_VALIDATION block is found.
// Returns (nil, error) if the JSON is malformed.
// Returns (*ValidationResult, nil) if successfully parsed.
func ParseValidation(text string) (*ValidationResult, error) {
	raw, err := ExtractJSON(text, "RALPH_VALIDATION")
	if raw == nil || err != nil {
		return nil, err
	}

	// ExtractJSON returns the outer object containing RALPH_VALIDATION.
	// Extract the nested RALPH_VALIDATION object.
	validation, ok := raw["RALPH_VALIDATION"].(map[string]interface{})
	hasRalphValidationKey := ok
	if !ok {
		// If RALPH_VALIDATION is not a nested object, treat raw as the validation data
		validation = raw
	}

	result := &ValidationResult{
		// Initialize with empty slice instead of nil for blocked_tasks
		BlockedTasks: []string{},
	}

	// Track if we found any actual validation fields
	hasValidationFields := false

	// Extract verdict string
	if v, ok := validation["verdict"].(string); ok {
		result.Verdict = v
		hasValidationFields = true
	}

	// Extract feedback string
	if v, ok := validation["feedback"].(string); ok {
		result.Feedback = v
		hasValidationFields = true
	}

	// Extract remaining count (JSON numbers are float64)
	if v, ok := validation["remaining"].(float64); ok {
		result.Remaining = int(v)
		hasValidationFields = true
	}

	// Extract blocked_count (JSON numbers are float64)
	if v, ok := validation["blocked_count"].(float64); ok {
		result.BlockedCount = int(v)
		hasValidationFields = true
	}

	// Extract blocked_tasks array
	if v, ok := validation["blocked_tasks"]; ok {
		if arr, ok := v.([]interface{}); ok {
			hasValidationFields = true
			// Keep empty slice if array is empty, don't append anything
			if len(arr) > 0 {
				for _, item := range arr {
					if s, ok := item.(string); ok {
						result.BlockedTasks = append(result.BlockedTasks, s)
					}
				}
			}
		}
	}

	// If no validation fields were found AND there was no explicit RALPH_VALIDATION key,
	// this was probably a false positive match (e.g., "RALPH_VALIDATION" in text but not in JSON)
	if !hasValidationFields && !hasRalphValidationKey {
		return nil, nil
	}

	return result, nil
}
