// Package learnings provides functionality for extracting and managing
// learnings from ralph-loop implementation iterations.
package learnings

import (
	"strings"
)

// ExtractLearnings extracts content from RALPH_LEARNINGS blocks in AI output.
// It looks for the RALPH_LEARNINGS: marker and returns all content after it
// until a blank line, closing code fence (```), or end of string.
//
// Returns empty string if:
//   - No RALPH_LEARNINGS block is found
//   - The block is immediately followed by a blank line (empty learnings)
//   - The block contains only whitespace or bare dashes ("- ")
func ExtractLearnings(output string) string {
	lines := strings.Split(output, "\n")
	startIdx := -1

	// Find the RALPH_LEARNINGS marker
	for i, line := range lines {
		if strings.Contains(line, "RALPH_LEARNINGS:") {
			// Check if there's content on the same line
			idx := strings.Index(line, "RALPH_LEARNINGS:")
			afterMarker := strings.TrimSpace(line[idx+len("RALPH_LEARNINGS:"):])
			if afterMarker != "" {
				// Content on same line as marker
				return afterMarker
			}
			startIdx = i + 1
			break
		}
	}

	if startIdx == -1 {
		return ""
	}

	// Collect lines until we hit a code fence, blank line, or EOF
	var learningLines []string

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Stop at code fence
		if strings.HasPrefix(trimmed, "```") {
			break
		}

		if trimmed == "" {
			// If we haven't collected any content yet and hit a blank line,
			// this is an empty learnings block
			if len(learningLines) == 0 {
				break
			}
			// If we've collected content, blank line marks end of learnings
			break
		}

		learningLines = append(learningLines, line)
	}

	if len(learningLines) == 0 {
		return ""
	}

	result := strings.Join(learningLines, "\n")
	result = strings.TrimSpace(result)

	// Check if only contains bare dashes
	if result == "-" {
		return ""
	}

	// Check if all lines are just bare dashes with no content
	resultLines := strings.Split(result, "\n")
	hasContent := false
	for _, line := range resultLines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && trimmed != "-" {
			hasContent = true
			break
		}
	}

	if !hasContent {
		return ""
	}

	return result
}
