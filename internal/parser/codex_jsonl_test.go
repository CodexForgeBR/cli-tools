package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseCodexJSONL_AgentMessage tests parsing item.completed events
// with agent_message type. These contain natural language responses from
// the Codex agent.
func TestParseCodexJSONL_AgentMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple agent message",
			input:    `{"type":"item.completed","item":{"type":"agent_message","text":"Starting implementation of the requested tasks."}}`,
			expected: "Starting implementation of the requested tasks.",
		},
		{
			name:     "agent message with newlines",
			input:    `{"type":"item.completed","item":{"type":"agent_message","text":"Line 1\nLine 2\nLine 3"}}`,
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "agent message with RALPH markers",
			input:    `{"type":"item.completed","item":{"type":"agent_message","text":"RALPH_STATUS: All done"}}`,
			expected: "RALPH_STATUS: All done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCodexJSONL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseCodexJSONL_AssistantMessage tests parsing item.completed events
// with assistant_message type. These are similar to agent_message but may
// come from different phases of the Codex execution.
func TestParseCodexJSONL_AssistantMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple assistant message",
			input:    `{"type":"item.completed","item":{"type":"assistant_message","text":"All tasks completed successfully."}}`,
			expected: "All tasks completed successfully.",
		},
		{
			name: "assistant message with RALPH_STATUS",
			input: "{\"type\":\"item.completed\",\"item\":{\"type\":\"assistant_message\",\"text\":\"All tasks completed successfully.\\n\\n```json\\n{\\\"RALPH_STATUS\\\":{\\\"completed_tasks\\\":[\\\"T001\\\"],\\\"blocked_tasks\\\":[],\\\"notes\\\":\\\"Done\\\"}}\\n```\"}}",
			expected: "All tasks completed successfully.\n\n```json\n{\"RALPH_STATUS\":{\"completed_tasks\":[\"T001\"],\"blocked_tasks\":[],\"notes\":\"Done\"}}\n```",
		},
		{
			name: "assistant message with RALPH_LEARNINGS",
			input: `{"type":"item.completed","item":{"type":"assistant_message","text":"RALPH_LEARNINGS:\n- Pattern: Use interfaces for testability"}}`,
			expected: "RALPH_LEARNINGS:\n- Pattern: Use interfaces for testability",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCodexJSONL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseCodexJSONL_FunctionCall tests parsing item.completed events
// with function_call type. Function calls should be formatted as
// "Called: name(args)" for visibility in the output.
func TestParseCodexJSONL_FunctionCall(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "write_file function call",
			input:    `{"type":"item.completed","item":{"type":"function_call","name":"write_file","arguments":"{\"path\":\"/tmp/test.go\",\"content\":\"package main\"}"}}`,
			expected: `Called: write_file({"path":"/tmp/test.go","content":"package main"})`,
		},
		{
			name:     "read_file function call",
			input:    `{"type":"item.completed","item":{"type":"function_call","name":"read_file","arguments":"{\"path\":\"/tmp/test.go\"}"}}`,
			expected: `Called: read_file({"path":"/tmp/test.go"})`,
		},
		{
			name:     "function call with empty arguments",
			input:    `{"type":"item.completed","item":{"type":"function_call","name":"list_files","arguments":"{}"}}`,
			expected: `Called: list_files({})`,
		},
		{
			name:     "function call with complex arguments",
			input:    `{"type":"item.completed","item":{"type":"function_call","name":"execute","arguments":"{\"cmd\":\"go test\",\"env\":{\"GO111MODULE\":\"on\"},\"timeout\":30}"}}`,
			expected: `Called: execute({"cmd":"go test","env":{"GO111MODULE":"on"},"timeout":30})`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCodexJSONL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseCodexJSONL_NonItemCompleted tests that non-item.completed events
// are skipped gracefully.
func TestParseCodexJSONL_NonItemCompleted(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "item.created event",
			input:    `{"type":"item.created","item":{"id":"123"}}`,
			expected: "",
		},
		{
			name:     "session.started event",
			input:    `{"type":"session.started","session":{"id":"abc"}}`,
			expected: "",
		},
		{
			name:     "unknown event type",
			input:    `{"type":"unknown.event","data":"something"}`,
			expected: "",
		},
		{
			name:     "missing type field",
			input:    `{"item":{"type":"agent_message","text":"Hello"}}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCodexJSONL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseCodexJSONL_EmptyInput tests that empty input returns empty output.
func TestParseCodexJSONL_EmptyInput(t *testing.T) {
	result := ParseCodexJSONL("")
	assert.Equal(t, "", result)
}

// TestParseCodexJSONL_MultiLineInput tests parsing multi-line JSONL input
// with a mix of different event types. This simulates real Codex API output.
func TestParseCodexJSONL_MultiLineInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "complete sample from testdata",
			input: "{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"Starting implementation of the requested tasks.\"}}\n" +
				"{\"type\":\"item.completed\",\"item\":{\"type\":\"function_call\",\"name\":\"write_file\",\"arguments\":\"{\\\"path\\\":\\\"/tmp/test.go\\\",\\\"content\\\":\\\"package main\\\"}\"}}\n" +
				"{\"type\":\"item.completed\",\"item\":{\"type\":\"assistant_message\",\"text\":\"All tasks completed successfully.\\n\\n```json\\n{\\\"RALPH_STATUS\\\":{\\\"completed_tasks\\\":[\\\"T001\\\"],\\\"blocked_tasks\\\":[],\\\"notes\\\":\\\"Done\\\"}}\\n```\\n\\nRALPH_LEARNINGS:\\n- Pattern: Use interfaces for testability\"}}",
			expected: "Starting implementation of the requested tasks.\nCalled: write_file({\"path\":\"/tmp/test.go\",\"content\":\"package main\"})\nAll tasks completed successfully.\n\n```json\n{\"RALPH_STATUS\":{\"completed_tasks\":[\"T001\"],\"blocked_tasks\":[],\"notes\":\"Done\"}}\n```\n\nRALPH_LEARNINGS:\n- Pattern: Use interfaces for testability",
		},
		{
			name: "mixed with non-item.completed events",
			input: `{"type":"session.started","session":{"id":"abc"}}
{"type":"item.completed","item":{"type":"agent_message","text":"First message"}}
{"type":"item.created","item":{"id":"123"}}
{"type":"item.completed","item":{"type":"assistant_message","text":"Second message"}}`,
			expected: "First message\nSecond message",
		},
		{
			name: "empty lines interspersed",
			input: `{"type":"item.completed","item":{"type":"agent_message","text":"Line 1"}}

{"type":"item.completed","item":{"type":"agent_message","text":"Line 2"}}

{"type":"item.completed","item":{"type":"agent_message","text":"Line 3"}}`,
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name: "only function calls",
			input: `{"type":"item.completed","item":{"type":"function_call","name":"read","arguments":"{}"}}
{"type":"item.completed","item":{"type":"function_call","name":"write","arguments":"{}"}}
{"type":"item.completed","item":{"type":"function_call","name":"execute","arguments":"{}"}}`,
			expected: "Called: read({})\nCalled: write({})\nCalled: execute({})",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCodexJSONL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseCodexJSONL_MalformedLines tests that invalid JSON lines are
// gracefully skipped without causing panics.
func TestParseCodexJSONL_MalformedLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "incomplete JSON object",
			input:    `{"type":"item.completed","item":{"type":"agent_message"`,
			expected: "",
		},
		{
			name:     "not JSON at all",
			input:    `This is just plain text, not JSON`,
			expected: "",
		},
		{
			name:     "missing quotes",
			input:    `{type:item.completed,item:{type:agent_message,text:hello}}`,
			expected: "",
		},
		{
			name:     "empty object",
			input:    `{}`,
			expected: "",
		},
		{
			name: "mixed valid and malformed",
			input: `{"type":"item.completed","item":{"type":"agent_message","text":"Valid"}}
{broken json here
{"type":"item.completed","item":{"type":"agent_message","text":"Also valid"}}`,
			expected: "Valid\nAlso valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCodexJSONL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseCodexJSONL_MissingFields tests handling of JSON objects with
// missing required fields.
func TestParseCodexJSONL_MissingFields(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "item.completed without item field",
			input:    `{"type":"item.completed"}`,
			expected: "",
		},
		{
			name:     "item without type field",
			input:    `{"type":"item.completed","item":{"text":"Hello"}}`,
			expected: "",
		},
		{
			name:     "agent_message without text field",
			input:    `{"type":"item.completed","item":{"type":"agent_message"}}`,
			expected: "",
		},
		{
			name:     "function_call without name field",
			input:    `{"type":"item.completed","item":{"type":"function_call","arguments":"{}"}}`,
			expected: "",
		},
		{
			name:     "function_call without arguments field",
			input:    `{"type":"item.completed","item":{"type":"function_call","name":"test"}}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCodexJSONL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseCodexJSONL_UnknownItemTypes tests handling of unknown item types
// within item.completed events.
func TestParseCodexJSONL_UnknownItemTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unknown item type",
			input:    `{"type":"item.completed","item":{"type":"unknown_type","data":"something"}}`,
			expected: "",
		},
		{
			name: "mixed known and unknown item types",
			input: `{"type":"item.completed","item":{"type":"agent_message","text":"Valid"}}
{"type":"item.completed","item":{"type":"unknown_type","data":"ignored"}}
{"type":"item.completed","item":{"type":"assistant_message","text":"Also valid"}}`,
			expected: "Valid\nAlso valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCodexJSONL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseCodexJSONL_UnicodeContent tests handling of Unicode characters
// in text content.
func TestParseCodexJSONL_UnicodeContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "emoji in agent message",
			input:    `{"type":"item.completed","item":{"type":"agent_message","text":"Task complete ✓"}}`,
			expected: "Task complete ✓",
		},
		{
			name:     "chinese characters",
			input:    `{"type":"item.completed","item":{"type":"assistant_message","text":"测试"}}`,
			expected: "测试",
		},
		{
			name:     "mixed unicode",
			input:    `{"type":"item.completed","item":{"type":"agent_message","text":"Hello 世界 🌍"}}`,
			expected: "Hello 世界 🌍",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCodexJSONL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseCodexJSONL_EscapedCharacters tests handling of escaped characters
// in JSON strings.
func TestParseCodexJSONL_EscapedCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "escaped quotes in text",
			input:    `{"type":"item.completed","item":{"type":"agent_message","text":"Said \"hello\" to the world"}}`,
			expected: `Said "hello" to the world`,
		},
		{
			name:     "escaped backslashes",
			input:    `{"type":"item.completed","item":{"type":"agent_message","text":"Path: C:\\Users\\test"}}`,
			expected: `Path: C:\Users\test`,
		},
		{
			name:     "escaped newlines",
			input:    `{"type":"item.completed","item":{"type":"agent_message","text":"Line 1\nLine 2"}}`,
			expected: "Line 1\nLine 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCodexJSONL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseCodexJSONL_WithTestdata tests parsing using actual testdata file.
func TestParseCodexJSONL_WithTestdata(t *testing.T) {
	// This test uses the exact format from testdata/output/codex-jsonl/sample-complete.jsonl
	input := "{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"Starting implementation of the requested tasks.\"}}\n" +
		"{\"type\":\"item.completed\",\"item\":{\"type\":\"function_call\",\"name\":\"write_file\",\"arguments\":\"{\\\"path\\\":\\\"/tmp/test.go\\\",\\\"content\\\":\\\"package main\\\"}\"}}\n" +
		"{\"type\":\"item.completed\",\"item\":{\"type\":\"assistant_message\",\"text\":\"All tasks completed successfully.\\n\\n```json\\n{\\\"RALPH_STATUS\\\":{\\\"completed_tasks\\\":[\\\"T001\\\"],\\\"blocked_tasks\\\":[],\\\"notes\\\":\\\"Done\\\"}}\\n```\\n\\nRALPH_LEARNINGS:\\n- Pattern: Use interfaces for testability\"}}"

	result := ParseCodexJSONL(input)

	require.NotEmpty(t, result)
	assert.Contains(t, result, "Starting implementation of the requested tasks.")
	assert.Contains(t, result, "Called: write_file")
	assert.Contains(t, result, "/tmp/test.go")
	assert.Contains(t, result, "All tasks completed successfully.")
	assert.Contains(t, result, "RALPH_STATUS")
	assert.Contains(t, result, "RALPH_LEARNINGS")
	assert.Contains(t, result, "Pattern: Use interfaces for testability")
}
