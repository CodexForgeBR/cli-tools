package github

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseIssueRef_ValidReferences tests parsing valid GitHub issue references.
func TestParseIssueRef_ValidReferences(t *testing.T) {
	tests := []struct {
		name          string
		ref           string
		expectedOwner string
		expectedRepo  string
		expectedNum   int
	}{
		{
			name:          "standard reference",
			ref:           "CodexForgeBR/cli-tools#42",
			expectedOwner: "CodexForgeBR",
			expectedRepo:  "cli-tools",
			expectedNum:   42,
		},
		{
			name:          "single digit number",
			ref:           "owner/repo#1",
			expectedOwner: "owner",
			expectedRepo:  "repo",
			expectedNum:   1,
		},
		{
			name:          "large issue number",
			ref:           "user/project#9999",
			expectedOwner: "user",
			expectedRepo:  "project",
			expectedNum:   9999,
		},
		{
			name:          "owner with dash",
			ref:           "my-org/my-repo#123",
			expectedOwner: "my-org",
			expectedRepo:  "my-repo",
			expectedNum:   123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, number, err := ParseIssueRef(tt.ref)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedOwner, owner)
			assert.Equal(t, tt.expectedRepo, repo)
			assert.Equal(t, tt.expectedNum, number)
		})
	}
}

// TestParseIssueRef_InvalidReferences tests parsing invalid GitHub issue references.
func TestParseIssueRef_InvalidReferences(t *testing.T) {
	tests := []struct {
		name        string
		ref         string
		expectedErr string
	}{
		{
			name:        "empty reference",
			ref:         "",
			expectedErr: "empty issue reference",
		},
		{
			name:        "missing issue number",
			ref:         "owner/repo",
			expectedErr: "invalid issue reference format",
		},
		{
			name:        "missing repo path",
			ref:         "#123",
			expectedErr: "invalid repo path",
		},
		{
			name:        "only owner no repo",
			ref:         "owner#123",
			expectedErr: "invalid repo path",
		},
		{
			name:        "non-numeric issue number",
			ref:         "owner/repo#abc",
			expectedErr: "invalid issue number",
		},
		{
			name:        "zero issue number",
			ref:         "owner/repo#0",
			expectedErr: "issue number must be positive",
		},
		{
			name:        "negative issue number",
			ref:         "owner/repo#-5",
			expectedErr: "issue number must be positive",
		},
		{
			name:        "multiple hash symbols",
			ref:         "owner/repo#123#456",
			expectedErr: "invalid issue reference format",
		},
		{
			name:        "too many path segments",
			ref:         "org/owner/repo#123",
			expectedErr: "invalid repo path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, number, err := ParseIssueRef(tt.ref)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
			assert.Empty(t, owner)
			assert.Empty(t, repo)
			assert.Equal(t, 0, number)
		})
	}
}

// TestFetchIssue_InvalidInputs tests FetchIssue with invalid inputs.
func TestFetchIssue_InvalidInputs(t *testing.T) {
	tests := []struct {
		name        string
		owner       string
		repo        string
		number      int
		expectedErr string
	}{
		{
			name:        "empty owner",
			owner:       "",
			repo:        "repo",
			number:      1,
			expectedErr: "owner cannot be empty",
		},
		{
			name:        "empty repo",
			owner:       "owner",
			repo:        "",
			number:      1,
			expectedErr: "repo cannot be empty",
		},
		{
			name:        "zero issue number",
			owner:       "owner",
			repo:        "repo",
			number:      0,
			expectedErr: "issue number must be positive",
		},
		{
			name:        "negative issue number",
			owner:       "owner",
			repo:        "repo",
			number:      -1,
			expectedErr: "issue number must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := FetchIssue(tt.owner, tt.repo, tt.number)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
			assert.Empty(t, content)
		})
	}
}

// TestCacheIssue_ValidContent tests caching issue content to a directory.
func TestCacheIssue_ValidContent(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	content := `# Feature Request: Add dark mode

This is the issue body with some details about the feature request.

## Requirements

- Dark theme colors
- Toggle switch in settings
`

	err := CacheIssue(tmpDir, content)
	require.NoError(t, err)

	// Verify the file was created
	cachePath := filepath.Join(tmpDir, "github-issue.md")
	assert.FileExists(t, cachePath)

	// Verify the content matches
	savedContent, err := os.ReadFile(cachePath)
	require.NoError(t, err)
	assert.Equal(t, content, string(savedContent))
}

// TestCacheIssue_CreatesDirectory tests that CacheIssue creates the directory if it doesn't exist.
func TestCacheIssue_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested", "cache", "dir")

	content := "Issue content"

	err := CacheIssue(nestedDir, content)
	require.NoError(t, err)

	// Verify the nested directory was created
	assert.DirExists(t, nestedDir)

	// Verify the file was created
	cachePath := filepath.Join(nestedDir, "github-issue.md")
	assert.FileExists(t, cachePath)
}

// TestCacheIssue_OverwritesExistingFile tests that CacheIssue overwrites existing cached files.
func TestCacheIssue_OverwritesExistingFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Write initial content
	initialContent := "Initial issue content"
	err := CacheIssue(tmpDir, initialContent)
	require.NoError(t, err)

	// Verify initial content
	cachePath := filepath.Join(tmpDir, "github-issue.md")
	savedContent, err := os.ReadFile(cachePath)
	require.NoError(t, err)
	assert.Equal(t, initialContent, string(savedContent))

	// Overwrite with new content
	newContent := "Updated issue content"
	err = CacheIssue(tmpDir, newContent)
	require.NoError(t, err)

	// Verify new content
	savedContent, err = os.ReadFile(cachePath)
	require.NoError(t, err)
	assert.Equal(t, newContent, string(savedContent))
}

// TestCacheIssue_InvalidInputs tests CacheIssue with invalid inputs.
func TestCacheIssue_InvalidInputs(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		content     string
		expectedErr string
	}{
		{
			name:        "empty directory",
			dir:         "",
			content:     "content",
			expectedErr: "directory cannot be empty",
		},
		{
			name:        "empty content",
			dir:         "/tmp/cache",
			content:     "",
			expectedErr: "content cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CacheIssue(tt.dir, tt.content)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// TestCacheIssue_FilePermissions tests that cached files have correct permissions.
func TestCacheIssue_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	content := "Issue content"

	err := CacheIssue(tmpDir, content)
	require.NoError(t, err)

	cachePath := filepath.Join(tmpDir, "github-issue.md")
	fileInfo, err := os.Stat(cachePath)
	require.NoError(t, err)

	// Verify file is readable and writable by owner
	mode := fileInfo.Mode()
	assert.True(t, mode&0600 == 0600, "file should be readable and writable by owner")
}

// TestParseIssueRef_EdgeCases tests edge cases in issue reference parsing.
func TestParseIssueRef_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		ref         string
		shouldError bool
	}{
		{
			name:        "whitespace in reference",
			ref:         "owner/repo #123",
			shouldError: false, // This actually parses as owner="owner", repo="repo ", number=123
		},
		{
			name:        "trailing slash",
			ref:         "owner/repo/#123",
			shouldError: true,
		},
		{
			name:        "leading hash",
			ref:         "#owner/repo#123",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := ParseIssueRef(tt.ref)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCacheIssue_SpecialCharacters tests caching content with special characters.
func TestCacheIssue_SpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()

	content := `# Issue with "quotes" and special chars

This has:
- Newlines
- Unicode: 你好 世界
- Emojis: 🚀 ✅
- Backslashes: C:\path\to\file
`

	err := CacheIssue(tmpDir, content)
	require.NoError(t, err)

	// Verify content is preserved exactly
	cachePath := filepath.Join(tmpDir, "github-issue.md")
	savedContent, err := os.ReadFile(cachePath)
	require.NoError(t, err)
	assert.Equal(t, content, string(savedContent))
}
