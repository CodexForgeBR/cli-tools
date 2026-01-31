package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractJSON_CodeBlock(t *testing.T) {
	text := "Here is the result:\n```json\n{\"verdict\": \"pass\", \"reason\": \"all good\"}\n```\nDone."

	result, err := ExtractJSON(text, "verdict")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "pass", result["verdict"])
	assert.Equal(t, "all good", result["reason"])
}

func TestExtractJSON_BracketMatching(t *testing.T) {
	text := `The verdict is here: {"verdict": "fail", "reason": "missing tests"} and that's it.`

	result, err := ExtractJSON(text, "verdict")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "fail", result["verdict"])
	assert.Equal(t, "missing tests", result["reason"])
}

func TestExtractJSON_NestedObjects(t *testing.T) {
	text := `Output:
{
  "verdict": "pass",
  "details": {
    "coverage": 95,
    "files": ["a.go", "b.go"]
  },
  "meta": {
    "nested": {
      "deep": true
    }
  }
}
End.`

	result, err := ExtractJSON(text, "verdict")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "pass", result["verdict"])

	details, ok := result["details"].(map[string]interface{})
	require.True(t, ok, "details should be a nested object")
	assert.Equal(t, float64(95), details["coverage"])

	meta, ok := result["meta"].(map[string]interface{})
	require.True(t, ok)
	nested, ok := meta["nested"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, nested["deep"])
}

func TestExtractJSON_EscapedQuotes(t *testing.T) {
	text := `Result: {"verdict": "pass", "message": "said \"hello\" to the world"}`

	result, err := ExtractJSON(text, "verdict")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "pass", result["verdict"])
	assert.Equal(t, `said "hello" to the world`, result["message"])
}

func TestExtractJSON_MissingKey(t *testing.T) {
	text := `{"status": "ok", "count": 42}`

	result, err := ExtractJSON(text, "verdict")
	assert.NoError(t, err)
	assert.Nil(t, result, "missing key should return nil result")
}

func TestExtractJSON_MalformedJSON(t *testing.T) {
	text := `The verdict is: {"verdict": "pass", broken`

	result, err := ExtractJSON(text, "verdict")
	assert.Error(t, err, "malformed JSON should return an error")
	assert.Nil(t, result)
}

func TestExtractJSON_EmptyInput(t *testing.T) {
	result, err := ExtractJSON("", "verdict")
	assert.NoError(t, err)
	assert.Nil(t, result, "empty input should return nil result")
}

func TestExtractJSON_CodeBlockPreferred(t *testing.T) {
	// When both a code block and bare JSON exist, the code block wins.
	text := `Here: {"verdict": "bare"}
` + "```json\n" + `{"verdict": "fenced"}` + "\n```"

	result, err := ExtractJSON(text, "verdict")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "fenced", result["verdict"])
}

func TestExtractJSON_CodeBlockWithoutKey(t *testing.T) {
	// Code block exists but does not contain the key; fall back to bracket matching.
	text := "```json\n{\"other\": 1}\n```\nAlso: {\"verdict\": \"found\"}"

	result, err := ExtractJSON(text, "verdict")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "found", result["verdict"])
}

func TestExtractJSON_NestedArrays(t *testing.T) {
	text := `{"verdict": "pass", "items": [1, [2, 3], {"a": 4}]}`

	result, err := ExtractJSON(text, "verdict")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "pass", result["verdict"])

	items, ok := result["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 3)
}

func TestExtractJSON_BracesInsideStrings(t *testing.T) {
	// Braces inside string values must not confuse bracket matching.
	text := `{"verdict": "pass", "note": "use {curly} and [square] brackets"}`

	result, err := ExtractJSON(text, "verdict")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "pass", result["verdict"])
	assert.Equal(t, "use {curly} and [square] brackets", result["note"])
}

func TestExtractJSON_KeyAfterNonJSONText(t *testing.T) {
	text := `Some prose about the verdict.
More text.
Finally: {"verdict": "ok"}`

	result, err := ExtractJSON(text, "verdict")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "ok", result["verdict"])
}
