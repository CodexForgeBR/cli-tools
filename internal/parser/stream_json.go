// Package parser provides text-parsing utilities for the ralph-loop CLI.
package parser

import (
	"encoding/json"
	"strings"
)

// ParseStreamJSON parses Claude CLI stream-json output and extracts text content.
// Each line is a JSON object. Text is extracted from assistant content blocks
// and result fallbacks. Malformed lines are silently skipped.
//
// Supported event types:
//   - type:assistant → extracts text from message.content[] where type="text"
//   - type:result → extracts result field as fallback
//
// Tool use content items are skipped as they don't contribute to text output.
func ParseStreamJSON(input string) string {
	if input == "" {
		return ""
	}

	var result strings.Builder
	lines := strings.Split(input, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try to parse the line as JSON
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Skip malformed JSON lines
			continue
		}

		// Extract text based on event type
		eventType, ok := event["type"].(string)
		if !ok {
			continue
		}

		switch eventType {
		case "assistant":
			extractAssistantText(event, &result)
		case "result":
			extractResultText(event, &result)
		}
	}

	return result.String()
}

// extractAssistantText extracts text content from assistant message events.
// It iterates through the message.content array and extracts text from
// content items with type="text", skipping tool_use items.
func extractAssistantText(event map[string]interface{}, result *strings.Builder) {
	message, ok := event["message"].(map[string]interface{})
	if !ok {
		return
	}

	content, ok := message["content"].([]interface{})
	if !ok {
		return
	}

	for _, item := range content {
		contentItem, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		itemType, ok := contentItem["type"].(string)
		if !ok || itemType != "text" {
			// Skip non-text items (e.g., tool_use)
			continue
		}

		text, ok := contentItem["text"].(string)
		if ok && text != "" {
			result.WriteString(text)
		}
	}
}

// extractResultText extracts text from result events as a fallback
// when no assistant content is available.
func extractResultText(event map[string]interface{}, result *strings.Builder) {
	resultText, ok := event["result"].(string)
	if ok && resultText != "" {
		result.WriteString(resultText)
	}
}
