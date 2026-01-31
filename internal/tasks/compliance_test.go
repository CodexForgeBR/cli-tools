package tasks

import (
	"os"
	"path/filepath"
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

// writeComplianceTempFile creates a temp file for compliance tests.
func writeComplianceTempFile(t *testing.T, content string) string {
	t.Helper()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "tasks.md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}
