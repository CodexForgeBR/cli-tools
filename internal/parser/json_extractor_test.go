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

// TestExtractJSON_NoJSONPresent tests when key exists but no braces found.
func TestExtractJSON_NoJSONPresent(t *testing.T) {
	text := "This text mentions the verdict word but has no braces at all"

	result, err := ExtractJSON(text, "verdict")
	assert.NoError(t, err, "Should return nil when key found but no braces")
	assert.Nil(t, result)
}

// TestExtractJSON_DeeplyNestedJSON tests deeply nested JSON structures.
func TestExtractJSON_DeeplyNestedJSON(t *testing.T) {
	text := `{
		"verdict": "pass",
		"level1": {
			"level2": {
				"level3": {
					"level4": {
						"level5": {
							"deep": "value"
						}
					}
				}
			}
		}
	}`

	result, err := ExtractJSON(text, "verdict")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "pass", result["verdict"])

	// Verify deep nesting is preserved
	l1, ok := result["level1"].(map[string]interface{})
	require.True(t, ok)
	l2, ok := l1["level2"].(map[string]interface{})
	require.True(t, ok)
	l3, ok := l2["level3"].(map[string]interface{})
	require.True(t, ok)
	l4, ok := l3["level4"].(map[string]interface{})
	require.True(t, ok)
	l5, ok := l4["level5"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "value", l5["deep"])
}

// TestExtractFromCodeBlock_NoClosingFence tests malformed code block without closing fence.
func TestExtractFromCodeBlock_NoClosingFence(t *testing.T) {
	text := "```json\n{\"verdict\": \"pass\"}\n"

	// ExtractJSON should handle this by falling back to bracket matching
	result, err := ExtractJSON(text, "verdict")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "pass", result["verdict"])
}

// TestExtractFromCodeBlock_EmptyCodeBlock tests empty code block.
func TestExtractFromCodeBlock_EmptyCodeBlock(t *testing.T) {
	text := "```json\n```\nLater: {\"verdict\": \"found\"}"

	// Should fall back to bracket matching
	result, err := ExtractJSON(text, "verdict")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "found", result["verdict"])
}

// TestMatchBraces_NoOpeningBrace tests matchBraces with invalid start.
func TestMatchBraces_NoOpeningBrace(t *testing.T) {
	s := "no brace at start"
	_, ok := matchBraces(s)
	assert.False(t, ok, "matchBraces should return false when first char is not '{'")
}

// TestMatchBraces_EmptyString tests matchBraces with empty string.
func TestMatchBraces_EmptyString(t *testing.T) {
	_, ok := matchBraces("")
	assert.False(t, ok, "matchBraces should return false for empty string")
}

// TestMatchBraces_ComplexNesting tests complex nesting of braces and brackets.
func TestMatchBraces_ComplexNesting(t *testing.T) {
	s := `{"a": [1, {"b": [2, 3]}, 4], "c": {"d": {"e": [5, {"f": 6}]}}}`
	end, ok := matchBraces(s)
	assert.True(t, ok)
	assert.Equal(t, len(s)-1, end, "Should find the closing brace at the end")
}

// TestMatchBraces_UnterminatedString tests unclosed string.
func TestMatchBraces_UnterminatedString(t *testing.T) {
	s := `{"key": "unterminated string`
	_, ok := matchBraces(s)
	assert.False(t, ok, "Should return false for unterminated string")
}

// TestExtractByBracketMatch_BackwardMatch tests backward brace matching.
func TestExtractByBracketMatch_BackwardMatch(t *testing.T) {
	text := `prefix {"verdict": "pass", "extra": "data"} suffix`

	result, err := ExtractJSON(text, "verdict")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "pass", result["verdict"])
	assert.Equal(t, "data", result["extra"])
}

// TestExtractByBracketMatch_InvalidJSONAfterMatch tests when bracket-matched text is invalid JSON.
func TestExtractByBracketMatch_InvalidJSONAfterMatch(t *testing.T) {
	text := `The verdict is in this broken json: {verdict: pass}`

	result, err := ExtractJSON(text, "verdict")
	assert.Error(t, err, "Should return error for invalid JSON")
	assert.Nil(t, result)
}

// TestExtractByBracketMatch_BackwardMatchInvalidJSON tests when backward match finds braces but invalid JSON.
func TestExtractByBracketMatch_BackwardMatchInvalidJSON(t *testing.T) {
	// Place invalid JSON before the key, and valid JSON after
	// The backward match should fail to parse, then fall through to forward match
	text := `{invalid json here} verdict after {"verdict": "pass"}`

	result, err := ExtractJSON(text, "verdict")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "pass", result["verdict"])
}

// TestExtractByBracketMatch_BackwardBraceUnmarshalFails tests the path where
// backward brace matching succeeds (braces match and the substring contains
// the key), but json.Unmarshal fails because the content is not valid JSON.
// This exercises extractByBracketMatch line 104 (err != nil from Unmarshal).
func TestExtractByBracketMatch_BackwardBraceUnmarshalFails(t *testing.T) {
	// {verdict: not-valid-json} has matching braces, contains "verdict",
	// but is not valid JSON (keys must be quoted). Unmarshal will fail.
	// The function should then fall through to forward search and find
	// the valid JSON object.
	text := `{verdict: not-valid-json} then {"verdict": "recovered"}`

	result, err := ExtractJSON(text, "verdict")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "recovered", result["verdict"])
}

// TestExtractByBracketMatch_KeyNotFoundInFunction documents that the keyIdx == -1
// guard in extractByBracketMatch (line 92-94) is unreachable dead code because
// ExtractJSON already checks strings.Contains(text, key) before calling it.
func TestExtractByBracketMatch_KeyNotFoundGuardIsDeadCode(t *testing.T) {
	// ExtractJSON at line 30-32 returns (nil, nil) when key is not in text,
	// so extractByBracketMatch's own keyIdx == -1 guard can never be reached.
	// This test documents that the 1 uncovered statement is acceptable.
	result, err := ExtractJSON("no key here", "verdict")
	assert.NoError(t, err)
	assert.Nil(t, result, "ExtractJSON returns nil before extractByBracketMatch is called")
}
