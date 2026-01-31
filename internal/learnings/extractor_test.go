package learnings

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractLearnings_WithMultipleItems(t *testing.T) {
	output := `Some implementation output here...
Running tests...
All tests passed!

RALPH_LEARNINGS:
- Pattern: Use table-driven tests in Go
- Gotcha: Remember to handle nil maps
- Context: The config package uses whitelisted vars
`

	result := ExtractLearnings(output)

	expected := `- Pattern: Use table-driven tests in Go
- Gotcha: Remember to handle nil maps
- Context: The config package uses whitelisted vars`

	assert.Equal(t, expected, result)
}

func TestExtractLearnings_NoBlock(t *testing.T) {
	output := `Some implementation output here...
Running tests...
All tests passed!
No learnings in this iteration.
`

	result := ExtractLearnings(output)

	assert.Equal(t, "", result)
}

func TestExtractLearnings_EmptyBlock(t *testing.T) {
	output := `Some implementation output here...

RALPH_LEARNINGS:

More output after...
`

	result := ExtractLearnings(output)

	assert.Equal(t, "", result)
}

func TestExtractLearnings_BareDashOnly(t *testing.T) {
	output := `Implementation complete.

RALPH_LEARNINGS:
-
`

	result := ExtractLearnings(output)

	assert.Equal(t, "", result)
}

func TestExtractLearnings_MultipleBareDashes(t *testing.T) {
	output := `Implementation complete.

RALPH_LEARNINGS:
-
-
-
`

	result := ExtractLearnings(output)

	assert.Equal(t, "", result)
}

func TestExtractLearnings_WithCodeFenceTermination(t *testing.T) {
	output := `Some implementation output...

RALPH_LEARNINGS:
- Pattern: Use context for cancellation
- Gotcha: Channels must be closed by sender
` + "```" + `

More code here...
`

	result := ExtractLearnings(output)

	expected := `- Pattern: Use context for cancellation
- Gotcha: Channels must be closed by sender`

	assert.Equal(t, expected, result)
}

func TestExtractLearnings_WithPatternGotchaContext(t *testing.T) {
	output := `Implementation successful!

RALPH_LEARNINGS:
- Pattern: Use sync.WaitGroup for goroutine coordination
- Gotcha: Buffered channels can deadlock if full
- Context: This project uses cobra for CLI framework
- Pattern: Always defer mutex.Unlock() immediately after Lock()
`

	result := ExtractLearnings(output)

	require.NotEmpty(t, result)
	assert.Contains(t, result, "Pattern: Use sync.WaitGroup")
	assert.Contains(t, result, "Gotcha: Buffered channels")
	assert.Contains(t, result, "Context: This project uses cobra")
	assert.Contains(t, result, "Pattern: Always defer mutex.Unlock()")
}

func TestExtractLearnings_OnlyWhitespace(t *testing.T) {
	output := `RALPH_LEARNINGS:



`

	result := ExtractLearnings(output)

	assert.Equal(t, "", result)
}

func TestExtractLearnings_SingleItem(t *testing.T) {
	output := `RALPH_LEARNINGS:
- Pattern: Use testify for assertions in Go tests
`

	result := ExtractLearnings(output)

	assert.Equal(t, "- Pattern: Use testify for assertions in Go tests", result)
}

func TestExtractLearnings_WithExtraSpacing(t *testing.T) {
	output := `
RALPH_LEARNINGS:
   - Pattern: Use t.TempDir() for test file operations
   - Gotcha: Remember to check file existence before reading

`

	result := ExtractLearnings(output)

	expected := `- Pattern: Use t.TempDir() for test file operations
   - Gotcha: Remember to check file existence before reading`

	assert.Equal(t, expected, result)
}

func TestExtractLearnings_MixedContentAndBareDashes(t *testing.T) {
	output := `RALPH_LEARNINGS:
-
- Pattern: Actual learning here
-
`

	result := ExtractLearnings(output)

	// Should return content because there's at least one line with actual content
	require.NotEmpty(t, result)
	assert.Contains(t, result, "Pattern: Actual learning here")
}

func TestExtractLearnings_NoNewlineAfterMarker(t *testing.T) {
	output := `RALPH_LEARNINGS:- Pattern: Use go modules for dependency management`

	result := ExtractLearnings(output)

	assert.Equal(t, "- Pattern: Use go modules for dependency management", result)
}

func TestExtractLearnings_MultilineContent(t *testing.T) {
	output := `RALPH_LEARNINGS:
- Pattern: Use table-driven tests with subtests
  This allows better test organization and reporting
- Gotcha: Remember to handle errors from deferred Close()
  You can use a named return value or check in defer
`

	result := ExtractLearnings(output)

	require.NotEmpty(t, result)
	assert.Contains(t, result, "This allows better test organization")
	assert.Contains(t, result, "You can use a named return value")
}

func TestExtractLearnings_EndOfString(t *testing.T) {
	// No code fence, should extract until end of string
	output := `RALPH_LEARNINGS:
- Pattern: Use errgroup for error handling in concurrent code
- Context: The internal packages follow Go project layout`

	result := ExtractLearnings(output)

	expected := `- Pattern: Use errgroup for error handling in concurrent code
- Context: The internal packages follow Go project layout`

	assert.Equal(t, expected, result)
}
