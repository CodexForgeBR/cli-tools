// Package parser provides text-parsing utilities for the ralph-loop CLI.
//
// ExtractJSON locates and parses a JSON object from free-form text,
// using a key as an anchor. It tries a fenced-code-block extraction
// first, then falls back to bracket matching.
package parser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ExtractJSON searches text for a JSON object associated with key.
//
// Strategy:
//  1. Look for a ```json fenced code block that contains key and parse it.
//  2. Fall back to bracket-matching: find the first '{' after key, then
//     walk forward counting nesting depth while respecting string literals
//     (including escaped quotes). Parse the resulting substring.
//
// If key is not found anywhere in text the function returns (nil, nil).
// Malformed JSON that is found but cannot be parsed returns a non-nil error.
func ExtractJSON(text string, key string) (map[string]interface{}, error) {
	if text == "" {
		return nil, nil
	}

	// Key must appear somewhere in the text.
	if !strings.Contains(text, key) {
		return nil, nil
	}

	// --- Strategy 1: fenced code block ---
	if result, err := extractFromCodeBlock(text, key); result != nil || err != nil {
		return result, err
	}

	// --- Strategy 2: bracket matching ---
	return extractByBracketMatch(text, key)
}

// extractFromCodeBlock looks for ```json ... ``` blocks that contain key
// and attempts to parse the JSON object from within.
func extractFromCodeBlock(text string, key string) (map[string]interface{}, error) {
	const fence = "```"
	remaining := text

	for {
		openIdx := strings.Index(remaining, fence+"json")
		if openIdx == -1 {
			break
		}

		// Move past the opening fence + "json" tag.
		blockStart := openIdx + len(fence+"json")
		// Skip optional newline right after the tag.
		if blockStart < len(remaining) && remaining[blockStart] == '\n' {
			blockStart++
		}

		closeIdx := strings.Index(remaining[blockStart:], fence)
		if closeIdx == -1 {
			break
		}

		block := remaining[blockStart : blockStart+closeIdx]

		if strings.Contains(block, key) {
			// Try to parse the trimmed block content.
			trimmed := strings.TrimSpace(block)
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
				return nil, fmt.Errorf("json in code block: %w", err)
			}
			return result, nil
		}

		// Advance past this block and continue searching.
		remaining = remaining[blockStart+closeIdx+len(fence):]
	}

	return nil, nil
}

// extractByBracketMatch locates the JSON object that contains or
// follows key. It first looks backward from key for a preceding '{',
// then forward. In each case it uses matchBraces to isolate the
// complete object.
func extractByBracketMatch(text string, key string) (map[string]interface{}, error) {
	keyIdx := strings.Index(text, key)
	if keyIdx == -1 {
		return nil, nil
	}

	// Try 1: look backward from the key for a '{' that encloses it.
	if braceStart := strings.LastIndex(text[:keyIdx], "{"); braceStart >= 0 {
		raw := text[braceStart:]
		if end, ok := matchBraces(raw); ok {
			jsonStr := raw[:end+1]
			// The matched object must contain the key.
			if strings.Contains(jsonStr, key) {
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
					return result, nil
				}
			}
		}
	}

	// Try 2: look forward from the key for a '{'.
	braceStart := strings.Index(text[keyIdx:], "{")
	if braceStart == -1 {
		return nil, nil
	}
	braceStart += keyIdx // absolute index

	raw := text[braceStart:]
	end, ok := matchBraces(raw)
	if !ok {
		return nil, fmt.Errorf("unmatched braces after key %q", key)
	}

	jsonStr := raw[:end+1]

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("bracket-matched json: %w", err)
	}
	return result, nil
}

// matchBraces returns the index of the closing '}' that matches the
// opening '{' at position 0, correctly handling string literals
// (including escaped quotes), nested objects, and arrays.
// Curly-brace depth and square-bracket depth are tracked independently
// so that arrays inside objects do not interfere with brace matching.
// Returns (index, true) on success or (0, false) if unmatched.
func matchBraces(s string) (int, bool) {
	if len(s) == 0 || s[0] != '{' {
		return 0, false
	}

	braceDepth := 0
	bracketDepth := 0
	inString := false
	i := 0

	for i < len(s) {
		ch := s[i]

		if inString {
			if ch == '\\' {
				// Skip the escaped character.
				i += 2
				continue
			}
			if ch == '"' {
				inString = false
			}
			i++
			continue
		}

		// Outside a string.
		switch ch {
		case '"':
			inString = true
		case '{':
			braceDepth++
		case '}':
			braceDepth--
			if braceDepth == 0 && bracketDepth == 0 {
				return i, true
			}
		case '[':
			bracketDepth++
		case ']':
			bracketDepth--
		}
		i++
	}

	return 0, false
}
