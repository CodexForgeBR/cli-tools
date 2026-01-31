package tasks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCompliance_CleanFile(t *testing.T) {
	content := `# Tasks

- [ ] Implement feature A
- [ ] Write tests for feature B
- [x] Review documentation

Some notes about the project.
`
	path := writeComplianceTempFile(t, content)

	violations, err := CheckCompliance(path)
	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestCheckCompliance_GitPush(t *testing.T) {
	content := `# Deploy Steps

- [ ] Run git push to deploy
`
	path := writeComplianceTempFile(t, content)

	violations, err := CheckCompliance(path)
	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Contains(t, violations[0], "git push")
}

func TestCheckCompliance_GhPrCreate(t *testing.T) {
	content := `# Workflow

Run gh pr create --title "Feature" to open a PR.
`
	path := writeComplianceTempFile(t, content)

	violations, err := CheckCompliance(path)
	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Contains(t, violations[0], "gh pr create")
}

func TestCheckCompliance_MultipleViolations(t *testing.T) {
	content := `# CI Steps

1. Build the project
2. Run git push origin main
3. Then run gh pr create --title "Release"
4. Done
`
	path := writeComplianceTempFile(t, content)

	violations, err := CheckCompliance(path)
	require.NoError(t, err)
	assert.Len(t, violations, 2)
	assert.Contains(t, violations[0], "git push")
	assert.Contains(t, violations[1], "gh pr create")
}

func TestCheckCompliance_BothPatternsOnSameLine(t *testing.T) {
	content := `Run git push && gh pr create to finish.
`
	path := writeComplianceTempFile(t, content)

	violations, err := CheckCompliance(path)
	require.NoError(t, err)
	// Both patterns appear on the same line, so both should be reported.
	assert.Len(t, violations, 2)
}

func TestCheckCompliance_EmptyFile(t *testing.T) {
	path := writeComplianceTempFile(t, "")

	violations, err := CheckCompliance(path)
	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestCheckCompliance_NonExistentFile(t *testing.T) {
	_, err := CheckCompliance(filepath.Join(t.TempDir(), "missing.md"))
	require.Error(t, err)
}

func TestCheckCompliance_ViolationIncludesLineNumber(t *testing.T) {
	content := `line one
line two
git push origin main
line four
`
	path := writeComplianceTempFile(t, content)

	violations, err := CheckCompliance(path)
	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Contains(t, violations[0], "line 3")
}

func TestCheckCompliance_PatternsInComments(t *testing.T) {
	content := `# Tasks

- [ ] Review the PR
- [ ] Don't run git push manually
- [ ] Avoid using gh pr create in scripts

# Notes
<!-- Comment: git push should be avoided -->
<!-- Also: gh pr create is automated -->
`
	path := writeComplianceTempFile(t, content)

	violations, err := CheckCompliance(path)
	require.NoError(t, err)
	// Violations should be found in the task lines and HTML comments
	require.Len(t, violations, 4)
	assert.Contains(t, violations[0], "line 4")
	assert.Contains(t, violations[0], "git push")
	assert.Contains(t, violations[1], "line 5")
	assert.Contains(t, violations[1], "gh pr create")
	assert.Contains(t, violations[2], "line 8")
	assert.Contains(t, violations[2], "git push")
	assert.Contains(t, violations[3], "line 9")
	assert.Contains(t, violations[3], "gh pr create")
}

func TestCheckCompliance_PartialMatches(t *testing.T) {
	content := `# Deploy Notes

- [ ] Use "git push" command carefully
- [ ] The "gh pr create" tool is helpful
- [ ] Check git pusher logs
`
	path := writeComplianceTempFile(t, content)

	violations, err := CheckCompliance(path)
	require.NoError(t, err)
	// Lines 3, 4, and 5 contain the forbidden patterns as substrings
	// "git pusher" contains "git push", but "gh pr creation" doesn't match "gh pr create" exactly
	require.Len(t, violations, 3)
	assert.Contains(t, violations[0], "line 3")
	assert.Contains(t, violations[0], "git push")
	assert.Contains(t, violations[1], "line 4")
	assert.Contains(t, violations[1], "gh pr create")
	assert.Contains(t, violations[2], "line 5")
	assert.Contains(t, violations[2], "git push")
}

func TestCheckCompliance_CaseSensitivity(t *testing.T) {
	// The patterns are case-sensitive (lowercase in forbiddenPatterns)
	content := `# Test Case Sensitivity

- [ ] GIT PUSH should not match
- [ ] GH PR CREATE should not match
- [ ] git push should match
- [ ] gh pr create should match
`
	path := writeComplianceTempFile(t, content)

	violations, err := CheckCompliance(path)
	require.NoError(t, err)
	// Only lowercase versions should match
	require.Len(t, violations, 2)
	assert.Contains(t, violations[0], "line 5")
	assert.Contains(t, violations[1], "line 6")
}

// TestCheckCompliance_ScannerError triggers a scanner error by writing a
// single line longer than bufio.MaxScanTokenSize (64 KB). The default
// bufio.Scanner returns bufio.ErrTooLong in this case.
func TestCheckCompliance_ScannerError(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "huge-line.md")

	// A single line of 70 KB (no newline) exceeds the 64 KB default buffer.
	hugeLine := strings.Repeat("a", 70*1024)
	require.NoError(t, os.WriteFile(path, []byte(hugeLine), 0644))

	_, err := CheckCompliance(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too long")
}

// writeComplianceTempFile creates a temp file for compliance tests.
func writeComplianceTempFile(t *testing.T, content string) string {
	t.Helper()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "tasks.md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}
