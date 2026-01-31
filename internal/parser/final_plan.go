// Package parser provides text-parsing utilities for the ralph-loop CLI.
package parser

// FinalPlanResult holds the parsed fields from a RALPH_FINAL_PLAN_VALIDATION JSON block.
// This structure represents validation feedback about whether the final implementation
// plan is ready for execution.
type FinalPlanResult struct {
	// Verdict indicates the final plan validation outcome.
	// Valid values: CONFIRMED, NOT_IMPLEMENTED (mapped from APPROVE/REJECT)
	Verdict string

	// Feedback provides detailed explanation of the verdict.
	Feedback string
}

// ParseFinalPlan extracts RALPH_FINAL_PLAN_VALIDATION fields from AI output text.
// Uses ExtractJSON to locate the JSON block, then maps fields to the result struct.
// Maps "APPROVE" → "CONFIRMED" and "REJECT" → "NOT_IMPLEMENTED".
//
// Returns (nil, nil) if no RALPH_FINAL_PLAN_VALIDATION block is found.
// Returns (nil, error) if the JSON is malformed.
// Returns (*FinalPlanResult, nil) if successfully parsed.
func ParseFinalPlan(text string) (*FinalPlanResult, error) {
	raw, err := ExtractJSON(text, "RALPH_FINAL_PLAN_VALIDATION")
	if raw == nil || err != nil {
		return nil, err
	}

	// ExtractJSON returns the outer object containing RALPH_FINAL_PLAN_VALIDATION.
	// Extract the nested RALPH_FINAL_PLAN_VALIDATION object.
	finalPlan, ok := raw["RALPH_FINAL_PLAN_VALIDATION"].(map[string]interface{})
	hasRalphFinalPlanKey := ok
	if !ok {
		// If RALPH_FINAL_PLAN_VALIDATION is not a nested object, treat raw as the data
		finalPlan = raw
	}

	result := &FinalPlanResult{}

	// Track if we found any actual final plan validation fields
	hasFinalPlanFields := false

	// Extract verdict string and map it
	if v, ok := finalPlan["verdict"].(string); ok {
		// Map APPROVE → CONFIRMED and REJECT → NOT_IMPLEMENTED
		switch v {
		case "APPROVE":
			result.Verdict = "CONFIRMED"
		case "REJECT":
			result.Verdict = "NOT_IMPLEMENTED"
		default:
			// Keep original verdict if it's already CONFIRMED/NOT_IMPLEMENTED or unknown
			result.Verdict = v
		}
		hasFinalPlanFields = true
	}

	// Extract feedback string
	if v, ok := finalPlan["feedback"].(string); ok {
		result.Feedback = v
		hasFinalPlanFields = true
	}

	// If no final plan fields were found AND there was no explicit RALPH_FINAL_PLAN_VALIDATION key,
	// this was probably a false positive match (e.g., "RALPH_FINAL_PLAN_VALIDATION" in text but not in JSON)
	if !hasFinalPlanFields && !hasRalphFinalPlanKey {
		return nil, nil
	}

	return result, nil
}
