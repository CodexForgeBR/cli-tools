// Package parser provides text-parsing utilities for the ralph-loop CLI.
package parser

// CrossValidationResult holds the parsed fields from a RALPH_CROSS_VALIDATION JSON block.
// This structure represents cross-validation feedback from an independent AI agent
// reviewing the validator's assessment of the implementer's work.
type CrossValidationResult struct {
	// Verdict indicates the cross-validation outcome.
	// Valid values: CONFIRMED, REJECTED
	Verdict string

	// TasksVerified is the number of tasks independently verified
	TasksVerified int

	// DiscrepanciesFound is the number of discrepancies found
	DiscrepanciesFound int

	// FilesActuallyRead lists production files independently verified
	FilesActuallyRead []string

	// CodeQuotes contains file/imports/production_calls for verified functionality
	CodeQuotes []map[string]interface{}

	// Discrepancies lists task_id/claimed/actual mismatches
	Discrepancies []map[string]interface{}

	// Feedback provides detailed explanation for REJECTED verdict
	Feedback string
}

// ParseCrossValidation extracts RALPH_CROSS_VALIDATION fields from AI output text.
// Uses ExtractJSON to locate the JSON block, then maps fields to the result struct.
//
// Returns (nil, nil) if no RALPH_CROSS_VALIDATION block is found.
// Returns (nil, error) if the JSON is malformed.
// Returns (*CrossValidationResult, nil) if successfully parsed.
func ParseCrossValidation(text string) (*CrossValidationResult, error) {
	raw, err := ExtractJSON(text, "RALPH_CROSS_VALIDATION")
	if raw == nil || err != nil {
		return nil, err
	}

	// ExtractJSON returns the outer object containing RALPH_CROSS_VALIDATION.
	// Extract the nested RALPH_CROSS_VALIDATION object.
	crossVal, ok := raw["RALPH_CROSS_VALIDATION"].(map[string]interface{})
	hasRalphCrossValidationKey := ok
	if !ok {
		// If RALPH_CROSS_VALIDATION is not a nested object, treat raw as the data
		crossVal = raw
	}

	result := &CrossValidationResult{}

	// Track if we found any actual cross-validation fields
	hasCrossValidationFields := false

	// Extract verdict string
	if v, ok := crossVal["verdict"].(string); ok {
		result.Verdict = v
		hasCrossValidationFields = true
	}

	// Extract tasks_verified number
	if v, ok := crossVal["tasks_verified"].(float64); ok {
		result.TasksVerified = int(v)
		hasCrossValidationFields = true
	}

	// Extract discrepancies_found number
	if v, ok := crossVal["discrepancies_found"].(float64); ok {
		result.DiscrepanciesFound = int(v)
		hasCrossValidationFields = true
	}

	// Extract files_actually_read array
	if v, ok := crossVal["files_actually_read"].([]interface{}); ok {
		files := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				files = append(files, s)
			}
		}
		result.FilesActuallyRead = files
		hasCrossValidationFields = true
	}

	// Extract code_quotes array
	if v, ok := crossVal["code_quotes"].([]interface{}); ok {
		quotes := make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				quotes = append(quotes, m)
			}
		}
		result.CodeQuotes = quotes
		hasCrossValidationFields = true
	}

	// Extract discrepancies array
	if v, ok := crossVal["discrepancies"].([]interface{}); ok {
		discrepancies := make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				discrepancies = append(discrepancies, m)
			}
		}
		result.Discrepancies = discrepancies
		hasCrossValidationFields = true
	}

	// Extract feedback string
	if v, ok := crossVal["feedback"].(string); ok {
		result.Feedback = v
		hasCrossValidationFields = true
	}

	// If no cross-validation fields were found AND there was no explicit RALPH_CROSS_VALIDATION key,
	// this was probably a false positive match (e.g., "RALPH_CROSS_VALIDATION" in text but not in JSON)
	if !hasCrossValidationFields && !hasRalphCrossValidationKey {
		return nil, nil
	}

	return result, nil
}
