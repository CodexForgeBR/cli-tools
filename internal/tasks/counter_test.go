package tasks

import (
	"os"
	"path/filepath"
	"strings"
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

func TestCountUnchecked_WhitespaceOnlyFile(t *testing.T) {
	content := "   \n\n  \t  \n\n"
	path := writeTempFile(t, content)
	count, err := CountUnchecked(path)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestCountChecked_WhitespaceOnlyFile(t *testing.T) {
	content := "   \n\n  \t  \n\n"
	path := writeTempFile(t, content)
	count, err := CountChecked(path)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestCountUnchecked_DeeplyNestedTasks(t *testing.T) {
	content := `- [ ] level 1
  - [ ] level 2
    - [ ] level 3
      - [ ] level 4
        - [ ] level 5
`
	path := writeTempFile(t, content)
	count, err := CountUnchecked(path)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestCountChecked_DeeplyNestedTasks(t *testing.T) {
	content := `- [x] level 1
  - [X] level 2
    - [x] level 3
      - [X] level 4
        - [x] level 5
`
	path := writeTempFile(t, content)
	count, err := CountChecked(path)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestCountUnchecked_NonExistentFile(t *testing.T) {
	_, err := CountUnchecked(filepath.Join(t.TempDir(), "nonexistent.md"))
	require.Error(t, err)
}

func TestCountChecked_NonExistentFile(t *testing.T) {
	_, err := CountChecked(filepath.Join(t.TempDir(), "nonexistent.md"))
	require.Error(t, err)
}

// TestCountUnchecked_ScannerError triggers a scanner error by writing a
// single line longer than bufio.MaxScanTokenSize (64 KB).
func TestCountUnchecked_ScannerError(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "huge-line.md")

	hugeLine := strings.Repeat("a", 70*1024)
	require.NoError(t, os.WriteFile(path, []byte(hugeLine), 0644))

	_, err := CountUnchecked(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too long")
}

// TestCountChecked_ScannerError triggers a scanner error by writing a
// single line longer than bufio.MaxScanTokenSize (64 KB).
func TestCountChecked_ScannerError(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "huge-line.md")

	hugeLine := strings.Repeat("a", 70*1024)
	require.NoError(t, os.WriteFile(path, []byte(hugeLine), 0644))

	_, err := CountChecked(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too long")
}

// writeTempFile creates a temp file with the given content and returns its path.
func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "tasks.md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}
