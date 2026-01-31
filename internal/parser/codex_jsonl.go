// Package parser provides text-parsing utilities for the ralph-loop CLI.
package parser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseCodexJSONL parses Codex CLI JSONL output and extracts text content.
// Each line is a JSON object. Text is extracted from item.completed events:
//   - item.type=agent_message → extract item.text
//   - item.type=assistant_message → extract item.text
//   - item.type=function_call → format as "Called: name(args)"
//
// Non-item.completed events are skipped.
// Output lines are separated by newlines.
func ParseCodexJSONL(input string) string {
	if input == "" {
		return ""
	}

	var result []string
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

		// Only process item.completed events
		eventType, ok := event["type"].(string)
		if !ok || eventType != "item.completed" {
			continue
		}

		// Extract the item object
		item, ok := event["item"].(map[string]interface{})
		if !ok {
			continue
		}

		// Extract text based on item type
		text := extractItemText(item)
		if text != "" {
			result = append(result, text)
		}
	}

	return strings.Join(result, "\n")
}

// extractItemText extracts text from an item object based on its type.
// Returns empty string if the item type is unknown or required fields are missing.
func extractItemText(item map[string]interface{}) string {
	itemType, ok := item["type"].(string)
	if !ok {
		return ""
	}

	switch itemType {
	case "agent_message", "assistant_message":
		text, ok := item["text"].(string)
		if ok {
			return text
		}

	case "function_call":
		name, nameOk := item["name"].(string)
		args, argsOk := item["arguments"].(string)
		if nameOk && argsOk {
			return fmt.Sprintf("Called: %s(%s)", name, args)
		}
	}

	return ""
}
