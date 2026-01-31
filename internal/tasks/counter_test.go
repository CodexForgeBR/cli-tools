package tasks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mixedContent = `# Project Tasks

Some introductory paragraph text.

## Phase 1

- [x] Completed task one
- [X] Completed task two (uppercase X)
- [ ] Unchecked task one
- [ ] Unchecked task two

## Phase 2

- [ ] Unchecked task three
- [x] Completed task three

Regular list item - not a task.
`

func TestCountUnchecked_MixedContent(t *testing.T) {
	path := writeTempFile(t, mixedContent)
	count, err := CountUnchecked(path)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestCountChecked_MixedContent(t *testing.T) {
	path := writeTempFile(t, mixedContent)
	count, err := CountChecked(path)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestCountUnchecked_OnlyUnchecked(t *testing.T) {
	content := `- [ ] one
- [ ] two
- [ ] three
`
	path := writeTempFile(t, content)
	count, err := CountUnchecked(path)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestCountChecked_OnlyChecked(t *testing.T) {
	content := `- [x] alpha
- [X] bravo
- [x] charlie
`
	path := writeTempFile(t, content)
	count, err := CountChecked(path)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestCountUnchecked_EmptyFile(t *testing.T) {
	path := writeTempFile(t, "")
	count, err := CountUnchecked(path)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestCountChecked_EmptyFile(t *testing.T) {
	path := writeTempFile(t, "")
	count, err := CountChecked(path)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestCountChecked_BothCases(t *testing.T) {
	content := `- [x] lowercase
- [X] uppercase
`
	path := writeTempFile(t, content)
	count, err := CountChecked(path)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestCountUnchecked_NonTaskContent(t *testing.T) {
	content := `# Header

This is a paragraph of text.

- Regular list item
- Another item
  - Nested item

> Blockquote
`
	path := writeTempFile(t, content)
	count, err := CountUnchecked(path)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestCountChecked_NonTaskContent(t *testing.T) {
	content := `# Header

This is a paragraph of text.

- Regular list item without checkbox
`
	path := writeTempFile(t, content)
	count, err := CountChecked(path)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestCountUnchecked_IndentedTasks(t *testing.T) {
	content := `- [ ] top level
  - [ ] indented two spaces
    - [ ] indented four spaces
`
	path := writeTempFile(t, content)
	count, err := CountUnchecked(path)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestCountChecked_NoCheckedAmongUnchecked(t *testing.T) {
	content := `- [ ] one
- [ ] two
`
	path := writeTempFile(t, content)
	count, err := CountChecked(path)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// writeTempFile creates a temp file with the given content and returns its path.
func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "tasks.md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}
