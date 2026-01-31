package tasks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// realPath resolves symlinks so that tests work on macOS where /var is a
// symlink to /private/var.
func realPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	require.NoError(t, err)
	return resolved
}

// chdirTemp changes the working directory to dir for the duration of the test,
// then restores the original.
func chdirTemp(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	require.NoError(t, err)
	resolved := realPath(t, dir)
	require.NoError(t, os.Chdir(resolved))
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
}

// writeFile writes content at path, creating intermediate directories.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestDiscoverTasksFile_ExplicitFlag_Exists(t *testing.T) {
	tmp := realPath(t, t.TempDir())
	tasksPath := filepath.Join(tmp, "my-tasks.md")
	writeFile(t, tasksPath, "- [ ] task one\n")

	got, err := DiscoverTasksFile(tasksPath)
	require.NoError(t, err)
	assert.Equal(t, tasksPath, got)
}

func TestDiscoverTasksFile_ExplicitFlag_Missing(t *testing.T) {
	tmp := realPath(t, t.TempDir())
	missingPath := filepath.Join(tmp, "no-such-file.md")

	_, err := DiscoverTasksFile(missingPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tasks file not found")
}

func TestDiscoverTasksFile_AutoDetect_TasksMD(t *testing.T) {
	tmp := realPath(t, t.TempDir())
	chdirTemp(t, tmp)

	tasksPath := filepath.Join(tmp, "tasks.md")
	writeFile(t, tasksPath, "- [ ] auto-detected\n")

	got, err := DiscoverTasksFile("")
	require.NoError(t, err)
	assert.Equal(t, tasksPath, got)
}

func TestDiscoverTasksFile_AutoDetect_TASKSMD(t *testing.T) {
	// On case-insensitive filesystems (macOS HFS+/APFS), tasks.md and TASKS.md
	// are the same file, so the search will match via the "tasks.md" entry in
	// wellKnownPaths. We verify the file is found regardless of the exact name.
	tmp := realPath(t, t.TempDir())
	chdirTemp(t, tmp)

	tasksPath := filepath.Join(tmp, "TASKS.md")
	writeFile(t, tasksPath, "- [ ] uppercase\n")

	got, err := DiscoverTasksFile("")
	require.NoError(t, err)
	// On case-insensitive FS, the returned path may be tasks.md instead of
	// TASKS.md. We just verify the directory and base file match (ignoring case).
	assert.Equal(t, tmp, filepath.Dir(got))
	assert.Contains(t, []string{"tasks.md", "TASKS.md"}, filepath.Base(got))
}

func TestDiscoverTasksFile_AutoDetect_SpecsSubdir(t *testing.T) {
	tmp := realPath(t, t.TempDir())
	chdirTemp(t, tmp)

	subPath := filepath.Join(tmp, "specs", "001-feature", "tasks.md")
	writeFile(t, subPath, "- [ ] from subdirectory\n")

	got, err := DiscoverTasksFile("")
	require.NoError(t, err)
	assert.Equal(t, subPath, got)
}

func TestDiscoverTasksFile_AutoDetect_SpecSubdir(t *testing.T) {
	tmp := realPath(t, t.TempDir())
	chdirTemp(t, tmp)

	subPath := filepath.Join(tmp, "spec", "my-branch", "tasks.md")
	writeFile(t, subPath, "- [ ] from spec subdirectory\n")

	got, err := DiscoverTasksFile("")
	require.NoError(t, err)
	assert.Equal(t, subPath, got)
}

func TestDiscoverTasksFile_PriorityOrder(t *testing.T) {
	// When both ./tasks.md and ./specs/*/tasks.md exist, ./tasks.md wins.
	tmp := realPath(t, t.TempDir())
	chdirTemp(t, tmp)

	rootTasks := filepath.Join(tmp, "tasks.md")
	writeFile(t, rootTasks, "- [ ] root\n")

	subTasks := filepath.Join(tmp, "specs", "feat", "tasks.md")
	writeFile(t, subTasks, "- [ ] sub\n")

	got, err := DiscoverTasksFile("")
	require.NoError(t, err)
	assert.Equal(t, rootTasks, got)
}

func TestDiscoverTasksFile_NotFound(t *testing.T) {
	tmp := realPath(t, t.TempDir())
	chdirTemp(t, tmp)

	_, err := DiscoverTasksFile("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no tasks file found")
}
