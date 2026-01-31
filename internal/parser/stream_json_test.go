package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseStreamJSON_AssistantTextContent tests parsing type:assistant content blocks
// with text content. These are the primary output blocks from Claude containing
// natural language responses and RALPH protocol markers.
func TestParseStreamJSON_AssistantTextContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single text content block",
			input:    `{"type":"assistant","message":{"content":[{"type":"text","text":"Hello world"}]}}`,
			expected: "Hello world",
		},
		{
			name:     "multiple text blocks in single message",
			input:    `{"type":"assistant","message":{"content":[{"type":"text","text":"First part"},{"type":"text","text":"Second part"}]}}`,
			expected: "First partSecond part",
		},
		{
			name:     "text with RALPH_STATUS marker",
			input:    "{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"Task complete.\\n\\n```json\\n{\\\"RALPH_STATUS\\\":{\\\"completed_tasks\\\":[\\\"T001\\\"],\\\"blocked_tasks\\\":[],\\\"notes\\\":\\\"Done\\\"}}\\n```\"}]}}",
			expected: "Task complete.\n\n```json\n{\"RALPH_STATUS\":{\"completed_tasks\":[\"T001\"],\"blocked_tasks\":[],\"notes\":\"Done\"}}\n```",
		},
		{
			name:     "text with RALPH_LEARNINGS marker",
			input:    `{"type":"assistant","message":{"content":[{"type":"text","text":"RALPH_LEARNINGS:\n- Pattern: Use interfaces"}]}}`,
			expected: "RALPH_LEARNINGS:\n- Pattern: Use interfaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseStreamJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseStreamJSON_ToolUseContent tests parsing type:assistant content blocks
// with tool_use content. Tool calls should be skipped as they don't contribute
// to the text output.
func TestParseStreamJSON_ToolUseContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single tool use - should be ignored",
			input:    `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Write","input":{"file_path":"/tmp/test.go","content":"package main"}}]}}`,
			expected: "",
		},
		{
			name:     "text followed by tool use",
			input:    `{"type":"assistant","message":{"content":[{"type":"text","text":"I'll write the file now."},{"type":"tool_use","name":"Write","input":{"file_path":"/tmp/test.go","content":"package main"}}]}}`,
			expected: "I'll write the file now.",
		},
		{
			name:     "tool use followed by text",
			input:    `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/tmp/test.go"}},{"type":"text","text":"File read successfully."}]}}`,
			expected: "File read successfully.",
		},
		{
			name:     "multiple tool uses with text interspersed",
			input:    `{"type":"assistant","message":{"content":[{"type":"text","text":"Starting"},{"type":"tool_use","name":"Write","input":{}},{"type":"text","text":"Done"},{"type":"tool_use","name":"Read","input":{}}]}}`,
			expected: "StartingDone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseStreamJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseStreamJSON_ResultFallback tests parsing type:result entries.
// When no assistant content is available, the result field provides a fallback.
func TestParseStreamJSON_ResultFallback(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple result text",
			input:    `{"type":"result","result":"Implementation complete with 2 tasks done."}`,
			expected: "Implementation complete with 2 tasks done.",
		},
		{
			name:     "result with newlines",
			input:    `{"type":"result","result":"Line 1\nLine 2\nLine 3"}`,
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "result with escaped quotes",
			input:    `{"type":"result","result":"Said \"hello\" to the world"}`,
			expected: `Said "hello" to the world`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseStreamJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseStreamJSON_MalformedLines tests that invalid JSON lines are
// gracefully skipped without causing panics or errors.
func TestParseStreamJSON_MalformedLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "incomplete JSON object",
			input:    `{"type":"assistant","message":{"content":[{"type":"text"`,
			expected: "",
		},
		{
			name:     "not JSON at all",
			input:    `This is just plain text, not JSON`,
			expected: "",
		},
		{
			name:     "missing quotes",
			input:    `{type:assistant,message:{content:[]}}`,
			expected: "",
		},
		{
			name:     "empty object",
			input:    `{}`,
			expected: "",
		},
		{
			name:     "null type field",
			input:    `{"type":null}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseStreamJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseStreamJSON_EmptyInput tests that empty input returns empty output.
func TestParseStreamJSON_EmptyInput(t *testing.T) {
	result := ParseStreamJSON("")
	assert.Equal(t, "", result)
}

// TestParseStreamJSON_MultiLineInput tests parsing multi-line JSONL input
// with a mix of valid and invalid lines. This simulates real Claude API
// streaming output.
func TestParseStreamJSON_MultiLineInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "complete sample from testdata",
			input: "{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"I'll implement the changes now.\"},{\"type\":\"tool_use\",\"name\":\"Write\",\"input\":{\"file_path\":\"/tmp/test.go\",\"content\":\"package main\"}}]}}\n" +
				"{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"Implementation complete. All tasks done.\\n\\n```json\\n{\\\"RALPH_STATUS\\\":{\\\"completed_tasks\\\":[\\\"T001\\\",\\\"T002\\\"],\\\"blocked_tasks\\\":[],\\\"notes\\\":\\\"All tasks completed\\\"}}\\n```\\n\\nRALPH_LEARNINGS:\\n- Pattern: Use table-driven tests in Go\\n- Gotcha: Remember to handle nil maps\"}]}}\n" +
				"{\"type\":\"result\",\"result\":\"Implementation complete with 2 tasks done.\"}",
			expected: "I'll implement the changes now.Implementation complete. All tasks done.\n\n```json\n{\"RALPH_STATUS\":{\"completed_tasks\":[\"T001\",\"T002\"],\"blocked_tasks\":[],\"notes\":\"All tasks completed\"}}\n```\n\nRALPH_LEARNINGS:\n- Pattern: Use table-driven tests in Go\n- Gotcha: Remember to handle nil mapsImplementation complete with 2 tasks done.",
		},
		{
			name: "mixed valid and invalid lines",
			input: `{"type":"assistant","message":{"content":[{"type":"text","text":"Valid line 1"}]}}
invalid line here
{"type":"assistant","message":{"content":[{"type":"text","text":"Valid line 2"}]}}
{"broken json
{"type":"result","result":"Final result"}`,
			expected: "Valid line 1Valid line 2Final result",
		},
		{
			name: "empty lines interspersed",
			input: `{"type":"assistant","message":{"content":[{"type":"text","text":"First"}]}}

{"type":"assistant","message":{"content":[{"type":"text","text":"Second"}]}}

{"type":"result","result":"Third"}`,
			expected: "FirstSecondThird",
		},
		{
			name: "only result lines",
			input: `{"type":"result","result":"Result 1"}
{"type":"result","result":"Result 2"}
{"type":"result","result":"Result 3"}`,
			expected: "Result 1Result 2Result 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseStreamJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseStreamJSON_UnknownTypes tests that unknown type fields are
// ignored gracefully.
func TestParseStreamJSON_UnknownTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unknown type field",
			input:    `{"type":"unknown","data":"some data"}`,
			expected: "",
		},
		{
			name: "mixed known and unknown types",
			input: `{"type":"unknown","data":"ignored"}
{"type":"assistant","message":{"content":[{"type":"text","text":"Valid"}]}}
{"type":"metadata","info":"also ignored"}`,
			expected: "Valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseStreamJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseStreamJSON_MissingFields tests handling of JSON objects with
// missing required fields.
func TestParseStreamJSON_MissingFields(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "assistant without message field",
			input:    `{"type":"assistant"}`,
			expected: "",
		},
		{
			name:     "assistant with empty content array",
			input:    `{"type":"assistant","message":{"content":[]}}`,
			expected: "",
		},
		{
			name:     "result without result field",
			input:    `{"type":"result"}`,
			expected: "",
		},
		{
			name:     "text content without text field",
			input:    `{"type":"assistant","message":{"content":[{"type":"text"}]}}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseStreamJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseStreamJSON_UnicodeContent tests handling of Unicode characters
// in text content.
func TestParseStreamJSON_UnicodeContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "emoji in text",
			input:    `{"type":"assistant","message":{"content":[{"type":"text","text":"Task complete ✓"}]}}`,
			expected: "Task complete ✓",
		},
		{
			name:     "chinese characters",
			input:    `{"type":"assistant","message":{"content":[{"type":"text","text":"测试"}]}}`,
			expected: "测试",
		},
		{
			name:     "mixed unicode",
			input:    `{"type":"result","result":"Hello 世界 🌍"}`,
			expected: "Hello 世界 🌍",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseStreamJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseStreamJSON_WithTestdata tests parsing using actual testdata file.
func TestParseStreamJSON_WithTestdata(t *testing.T) {
	// This test reads the actual testdata file to ensure compatibility
	// with real Claude API output format.
	input := "{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"I'll implement the changes now.\"},{\"type\":\"tool_use\",\"name\":\"Write\",\"input\":{\"file_path\":\"/tmp/test.go\",\"content\":\"package main\"}}]}}\n" +
		"{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"Implementation complete. All tasks done.\\n\\n```json\\n{\\\"RALPH_STATUS\\\":{\\\"completed_tasks\\\":[\\\"T001\\\",\\\"T002\\\"],\\\"blocked_tasks\\\":[],\\\"notes\\\":\\\"All tasks completed\\\"}}\\n```\\n\\nRALPH_LEARNINGS:\\n- Pattern: Use table-driven tests in Go\\n- Gotcha: Remember to handle nil maps\"}]}}\n" +
		"{\"type\":\"result\",\"result\":\"Implementation complete with 2 tasks done.\"}"

	result := ParseStreamJSON(input)

	require.NotEmpty(t, result)
	assert.Contains(t, result, "I'll implement the changes now.")
	assert.Contains(t, result, "Implementation complete. All tasks done.")
	assert.Contains(t, result, "RALPH_STATUS")
	assert.Contains(t, result, "RALPH_LEARNINGS")
	assert.Contains(t, result, "Pattern: Use table-driven tests in Go")
	assert.Contains(t, result, "Implementation complete with 2 tasks done.")
}

// TestExtractAssistantText_MessageNotMap tests when message field is not a map.
func TestExtractAssistantText_MessageNotMap(t *testing.T) {
	input := `{"type":"assistant","message":"not a map"}`

	result := ParseStreamJSON(input)
	assert.Equal(t, "", result, "Should return empty when message is not a map")
}

// TestExtractAssistantText_ContentNotArray tests when content field is not an array.
func TestExtractAssistantText_ContentNotArray(t *testing.T) {
	input := `{"type":"assistant","message":{"content":"not an array"}}`

	result := ParseStreamJSON(input)
	assert.Equal(t, "", result, "Should return empty when content is not an array")
}

// TestExtractAssistantText_ContentItemNotMap tests when content item is not a map.
func TestExtractAssistantText_ContentItemNotMap(t *testing.T) {
	input := `{"type":"assistant","message":{"content":["not a map", {"type":"text","text":"valid"}]}}`

	result := ParseStreamJSON(input)
	assert.Equal(t, "valid", result, "Should skip non-map items and extract valid ones")
}

// TestExtractAssistantText_EmptyTextContent tests when text field is empty.
func TestExtractAssistantText_EmptyTextContent(t *testing.T) {
	input := `{"type":"assistant","message":{"content":[{"type":"text","text":""}]}}`

	result := ParseStreamJSON(input)
	assert.Equal(t, "", result, "Should return empty when text content is empty")
}

// TestExtractResultText_EmptyResult tests when result field is empty.
func TestExtractResultText_EmptyResult(t *testing.T) {
	input := `{"type":"result","result":""}`

	result := ParseStreamJSON(input)
	assert.Equal(t, "", result, "Should return empty when result is empty")
}
