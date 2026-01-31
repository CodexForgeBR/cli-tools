package learnings

import (
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitLearnings_CreatesFileWithTemplate(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "learnings.md")

	err := InitLearnings(filePath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(filePath)
	require.NoError(t, err)

	// Verify content matches template
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "# Ralph Loop Learnings")
	assert.Contains(t, contentStr, "## Codebase Patterns")
	assert.Contains(t, contentStr, "## Iteration Log")
	assert.Contains(t, contentStr, "<!-- Add reusable patterns discovered during implementation -->")
}

func TestInitLearnings_CreatesParentDirectories(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "nested", "deep", "learnings.md")

	err := InitLearnings(filePath)
	require.NoError(t, err)

	// Verify file exists in nested directory
	_, err = os.Stat(filePath)
	require.NoError(t, err)

	// Verify parent directories were created
	parentDir := filepath.Dir(filePath)
	info, err := os.Stat(parentDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestInitLearnings_OverwritesExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "learnings.md")

	// Create existing file with different content
	err := os.WriteFile(filePath, []byte("Old content here"), 0644)
	require.NoError(t, err)

	// Initialize should overwrite
	err = InitLearnings(filePath)
	require.NoError(t, err)

	// Verify new content
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# Ralph Loop Learnings")
	assert.NotContains(t, string(content), "Old content here")
}

func TestInitLearnings_FailsWithInvalidPath(t *testing.T) {
	// Try to initialize in a path that cannot be created (e.g., as a child of a file)
	tempDir := t.TempDir()

	// Create a regular file
	blockingFile := filepath.Join(tempDir, "blocking-file")
	err := os.WriteFile(blockingFile, []byte("content"), 0644)
	require.NoError(t, err)

	// Try to create a learnings file inside the regular file (impossible)
	invalidPath := filepath.Join(blockingFile, "nested", "learnings.md")

	err = InitLearnings(invalidPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create parent directory")
}

func TestInitLearnings_FailsWhenWriteFileFails(t *testing.T) {
	tempDir := t.TempDir()

	// Create a directory with the same name as the target file
	// This will cause os.WriteFile to fail
	filePath := filepath.Join(tempDir, "learnings.md")
	err := os.Mkdir(filePath, 0755)
	require.NoError(t, err)

	// Try to initialize - should fail because we can't write a file over a directory
	err = InitLearnings(filePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write learnings file")
}

func TestAppendLearnings_AddsEntryWithIterationAndTimestamp(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "learnings.md")

	// Initialize file first
	err := InitLearnings(filePath)
	require.NoError(t, err)

	// Append learning
	learningContent := `- Pattern: Use table-driven tests in Go
- Gotcha: Remember to handle nil maps`

	beforeAppend := time.Now()
	err = AppendLearnings(filePath, 3, learningContent)
	require.NoError(t, err)
	afterAppend := time.Now()

	// Read back content
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	contentStr := string(content)

	// Verify iteration header is present
	assert.Contains(t, contentStr, "## Iteration 3")

	// Verify timestamp is present and reasonable
	// Should contain a date in YYYY-MM-DD format
	year := time.Now().Format("2006")
	assert.Contains(t, contentStr, year)

	// Verify the learning content is present
	assert.Contains(t, contentStr, "- Pattern: Use table-driven tests in Go")
	assert.Contains(t, contentStr, "- Gotcha: Remember to handle nil maps")

	// Verify timestamp format (rough check)
	// Format should be: 2006-01-02 15:04:05
	lines := strings.Split(contentStr, "\n")
	var headerLine string
	for _, line := range lines {
		if strings.Contains(line, "## Iteration 3") {
			headerLine = line
			break
		}
	}
	require.NotEmpty(t, headerLine)

	// Extract timestamp from header (format: ## Iteration 3 (2026-01-30 14:30:00))
	assert.Contains(t, headerLine, "(")
	assert.Contains(t, headerLine, ")")
	assert.Contains(t, headerLine, ":")

	// Parse timestamp to verify it's in valid range
	startIdx := strings.Index(headerLine, "(") + 1
	endIdx := strings.Index(headerLine, ")")
	timestampStr := headerLine[startIdx:endIdx]

	// Parse in local timezone to match how it was written
	parsedTime, err := time.ParseInLocation("2006-01-02 15:04:05", timestampStr, time.Local)
	require.NoError(t, err)

	// Timestamp should be between before and after append (within a few seconds tolerance)
	assert.True(t, !parsedTime.Before(beforeAppend.Add(-2*time.Second)), "timestamp should not be before beforeAppend")
	assert.True(t, !parsedTime.After(afterAppend.Add(2*time.Second)), "timestamp should not be after afterAppend")
}

func TestAppendLearnings_EmptyContentDoesNotAppend(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "learnings.md")

	// Initialize file
	err := InitLearnings(filePath)
	require.NoError(t, err)

	// Get initial content
	initialContent, err := os.ReadFile(filePath)
	require.NoError(t, err)

	// Append empty content
	err = AppendLearnings(filePath, 1, "")
	require.NoError(t, err)

	// Verify content unchanged
	currentContent, err := os.ReadFile(filePath)
	require.NoError(t, err)

	assert.Equal(t, string(initialContent), string(currentContent))
	assert.NotContains(t, string(currentContent), "## Iteration 1")
}

func TestAppendLearnings_MultipleAppends(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "learnings.md")

	// Initialize file
	err := InitLearnings(filePath)
	require.NoError(t, err)

	// Append first learning
	err = AppendLearnings(filePath, 1, "- Pattern: First learning")
	require.NoError(t, err)

	// Append second learning
	err = AppendLearnings(filePath, 2, "- Pattern: Second learning")
	require.NoError(t, err)

	// Append third learning
	err = AppendLearnings(filePath, 5, "- Gotcha: Third learning")
	require.NoError(t, err)

	// Read final content
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	contentStr := string(content)

	// Verify all iterations are present
	assert.Contains(t, contentStr, "## Iteration 1")
	assert.Contains(t, contentStr, "## Iteration 2")
	assert.Contains(t, contentStr, "## Iteration 5")

	// Verify all learnings are present
	assert.Contains(t, contentStr, "- Pattern: First learning")
	assert.Contains(t, contentStr, "- Pattern: Second learning")
	assert.Contains(t, contentStr, "- Gotcha: Third learning")

	// Verify template header is still present
	assert.Contains(t, contentStr, "# Ralph Loop Learnings")
	assert.Contains(t, contentStr, "## Iteration Log")

	// Verify order (iteration 1 should come before iteration 2)
	idx1 := strings.Index(contentStr, "## Iteration 1")
	idx2 := strings.Index(contentStr, "## Iteration 2")
	idx5 := strings.Index(contentStr, "## Iteration 5")
	assert.True(t, idx1 < idx2)
	assert.True(t, idx2 < idx5)
}

func TestAppendLearnings_CreatesFileIfNotExists(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "learnings.md")

	// Append without initializing first
	err := AppendLearnings(filePath, 1, "- Pattern: First learning")
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(filePath)
	require.NoError(t, err)

	// Verify content
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "- Pattern: First learning")
	assert.Contains(t, string(content), "## Iteration 1")
}

func TestReadLearnings_ReadsFullContent(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "learnings.md")

	// Initialize and add some content
	err := InitLearnings(filePath)
	require.NoError(t, err)

	err = AppendLearnings(filePath, 1, "- Pattern: Test learning")
	require.NoError(t, err)

	// Read back
	content := ReadLearnings(filePath)

	assert.Contains(t, content, "# Ralph Loop Learnings")
	assert.Contains(t, content, "## Codebase Patterns")
	assert.Contains(t, content, "## Iteration Log")
	assert.Contains(t, content, "## Iteration 1")
	assert.Contains(t, content, "- Pattern: Test learning")
}

func TestReadLearnings_FileNotExists(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "nonexistent.md")

	content := ReadLearnings(filePath)

	assert.Equal(t, "", content)
}

func TestReadLearnings_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "empty.md")

	// Create empty file
	err := os.WriteFile(filePath, []byte(""), 0644)
	require.NoError(t, err)

	content := ReadLearnings(filePath)

	assert.Equal(t, "", content)
}

func TestReadLearnings_MultipleIterations(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "learnings.md")

	// Initialize file
	err := InitLearnings(filePath)
	require.NoError(t, err)

	// Add multiple iterations
	err = AppendLearnings(filePath, 1, "- Pattern: First")
	require.NoError(t, err)

	err = AppendLearnings(filePath, 2, "- Pattern: Second")
	require.NoError(t, err)

	err = AppendLearnings(filePath, 3, "- Gotcha: Third")
	require.NoError(t, err)

	// Read all content
	content := ReadLearnings(filePath)

	// Verify all iterations are in the content
	assert.Contains(t, content, "## Iteration 1")
	assert.Contains(t, content, "## Iteration 2")
	assert.Contains(t, content, "## Iteration 3")
	assert.Contains(t, content, "- Pattern: First")
	assert.Contains(t, content, "- Pattern: Second")
	assert.Contains(t, content, "- Gotcha: Third")
}

func TestAppendLearnings_WithMultilineContent(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "learnings.md")

	err := InitLearnings(filePath)
	require.NoError(t, err)

	multilineContent := `- Pattern: Use context for timeout control
  Always propagate context through function calls
  Use context.WithTimeout for operations with deadlines
- Gotcha: Defer in loops can cause memory issues
  Consider using a closure or refactoring the loop`

	err = AppendLearnings(filePath, 1, multilineContent)
	require.NoError(t, err)

	content := ReadLearnings(filePath)

	assert.Contains(t, content, "Always propagate context through function calls")
	assert.Contains(t, content, "Consider using a closure or refactoring the loop")
}

func TestAppendLearnings_FormattingPreservation(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "learnings.md")

	err := InitLearnings(filePath)
	require.NoError(t, err)

	// Content with specific formatting
	content := `- Pattern: Use these steps:
  1. Initialize state
  2. Validate input
  3. Execute operation
- Context: Project structure:
  - cmd/ for CLI entry points
  - internal/ for private packages`

	err = AppendLearnings(filePath, 1, content)
	require.NoError(t, err)

	result := ReadLearnings(filePath)

	// Verify formatting is preserved
	assert.Contains(t, result, "1. Initialize state")
	assert.Contains(t, result, "2. Validate input")
	assert.Contains(t, result, "- cmd/ for CLI entry points")
	assert.Contains(t, result, "- internal/ for private packages")
}

func TestAppendLearnings_FailsWithReadOnlyFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "readonly.md")

	// Create a file and make it read-only
	err := os.WriteFile(filePath, []byte("initial content"), 0444)
	require.NoError(t, err)

	// Try to append - should fail
	err = AppendLearnings(filePath, 1, "- Pattern: This should fail")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open learnings file")
}

func TestAppendLearnings_FailsWithDirectoryAsFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a directory with the name we want to use as a file
	dirPath := filepath.Join(tempDir, "learnings-dir")
	err := os.Mkdir(dirPath, 0755)
	require.NoError(t, err)

	// Try to append to the directory (OpenFile should fail)
	err = AppendLearnings(dirPath, 1, "- Pattern: This should fail")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open learnings file")
}

func TestAppendLearnings_WriteStringErrorPath(t *testing.T) {
	// This test attempts to trigger the WriteString error path at manager.go:63
	// This error is difficult to trigger portably because it requires:
	// 1. Successfully opening a file for appending, but
	// 2. Failing to write to it after opening
	//
	// On Linux, /dev/full provides this behavior (opens successfully, writes fail)
	// On macOS/BSD, /dev/full doesn't exist
	//
	// Try to use /dev/full if available (Linux systems)
	devFull := "/dev/full"
	if _, err := os.Stat(devFull); os.IsNotExist(err) {
		// On macOS, we'll try /dev/zero which is read-only
		// Actually, /dev/zero is readable, not writable
		// There's no portable way to test this without /dev/full
		t.Skip("Skipping test: /dev/full not available (Linux-only test)")
		return
	}

	// /dev/full is a special file that:
	// - Opens successfully with O_WRONLY
	// - Returns ENOSPC (no space left) on write operations
	// This triggers the WriteString error path
	err := AppendLearnings(devFull, 1, "- Pattern: This should fail with ENOSPC")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to append learnings")
}



func TestAppendLearnings_WriteStringError_RLIMIT(t *testing.T) {
	// Trigger WriteString error by setting RLIMIT_FSIZE to 0.
	// This causes any write to a new file to fail with EFBIG ("file too large").
	// Works on macOS and Linux without needing /dev/full.
	//
	// SIGXFSZ must be ignored to prevent process termination.

	// Ignore SIGXFSZ to prevent default signal handler from terminating process
	signal.Ignore(syscall.SIGXFSZ)
	defer signal.Reset(syscall.SIGXFSZ)

	// Save current RLIMIT_FSIZE
	var origRlim syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &origRlim)
	require.NoError(t, err, "should be able to get RLIMIT_FSIZE")

	// Set RLIMIT_FSIZE to 0 bytes
	newRlim := syscall.Rlimit{Cur: 0, Max: origRlim.Max}
	err = syscall.Setrlimit(syscall.RLIMIT_FSIZE, &newRlim)
	require.NoError(t, err, "should be able to set RLIMIT_FSIZE to 0")

	// Restore original limit when done
	defer func() {
		_ = syscall.Setrlimit(syscall.RLIMIT_FSIZE, &origRlim)
	}()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "learnings.md")

	// OpenFile with O_CREATE|O_APPEND|O_WRONLY should succeed (creating empty file),
	// but WriteString should fail with EFBIG
	err = AppendLearnings(filePath, 1, "- Pattern: This write should fail with EFBIG")
	assert.Error(t, err, "WriteString should fail when RLIMIT_FSIZE is 0")
	assert.Contains(t, err.Error(), "failed to append learnings", "should wrap as learnings error")
}

func TestReadLearnings_WithPermissionError(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "nopermission.md")

	// Create a file with no read permissions
	err := os.WriteFile(filePath, []byte("secret content"), 0200)
	require.NoError(t, err)

	// Reading should return empty string (silent failure)
	content := ReadLearnings(filePath)
	assert.Equal(t, "", content)
}
