package tasks

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestDiscoverTasksFile_ExplicitFlag_PermissionDenied(t *testing.T) {
	tmp := realPath(t, t.TempDir())
	tasksPath := filepath.Join(tmp, "no-read.md")
	writeFile(t, tasksPath, "- [ ] task one\n")

	// Make file unreadable
	require.NoError(t, os.Chmod(tasksPath, 0000))
	t.Cleanup(func() {
		// Restore permissions for cleanup
		_ = os.Chmod(tasksPath, 0644)
	})

	// DiscoverTasksFile checks file existence with os.Stat, which succeeds
	// even on unreadable files. The error occurs when trying to read the file.
	// Since DiscoverTasksFile only calls os.Stat, this should succeed.
	got, err := DiscoverTasksFile(tasksPath)
	require.NoError(t, err)
	assert.Equal(t, tasksPath, got)
}

func TestDiscoverTasksFile_ExplicitFlag_Directory(t *testing.T) {
	tmp := realPath(t, t.TempDir())
	dirPath := filepath.Join(tmp, "tasks-dir")
	require.NoError(t, os.Mkdir(dirPath, 0755))

	// On Unix systems, os.Stat succeeds on directories, so DiscoverTasksFile
	// will return the path. The error will occur when trying to open it for reading.
	// This tests that DiscoverTasksFile accepts any path that exists.
	got, err := DiscoverTasksFile(dirPath)
	require.NoError(t, err)
	assert.Equal(t, dirPath, got)
}

// TestDiscoverTasksFile_AbsError_ExplicitFlag exercises the filepath.Abs
// error path (line 39) when a relative flag path is provided but the current
// working directory is unreadable. On macOS/Linux, removing read+execute
// permissions from the CWD causes os.Getwd (and therefore filepath.Abs) to
// return an error.
func TestDiscoverTasksFile_AbsError_ExplicitFlag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission removal trick does not work on Windows")
	}

	tmpDir := realPath(t, t.TempDir())
	orig, err := os.Getwd()
	require.NoError(t, err)

	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		// Restore permissions before restoring CWD
		_ = os.Chmod(tmpDir, 0755)
		_ = os.Chdir(orig)
	})

	// Remove read+execute from the CWD so os.Getwd fails
	require.NoError(t, os.Chmod(tmpDir, 0000))

	_, err = DiscoverTasksFile("relative-flag.md")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolving tasks file path")
}

// TestDiscoverTasksFile_AbsError_WellKnownAndSubdirPaths exercises the
// filepath.Abs error continue branches in the well-known paths loop (line 53)
// and the subdirectory glob loop (lines 67, 70). When the CWD is unreadable,
// every filepath.Abs call fails and the function falls through to the
// "no tasks file found" error.
func TestDiscoverTasksFile_AbsError_WellKnownAndSubdirPaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission removal trick does not work on Windows")
	}

	tmpDir := realPath(t, t.TempDir())
	orig, err := os.Getwd()
	require.NoError(t, err)

	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		_ = os.Chmod(tmpDir, 0755)
		_ = os.Chdir(orig)
	})

	// Remove read+execute from the CWD so os.Getwd fails
	require.NoError(t, os.Chmod(tmpDir, 0000))

	// With empty flag, the function will iterate well-known paths and subdir
	// roots, hitting filepath.Abs errors each time (continue), then return
	// the final "no tasks file found" error.
	_, err = DiscoverTasksFile("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no tasks file found")
}

// TestDiscoverTasksFile_GlobError exercises the filepath.Glob error continue
// branch (line 71). When the CWD path contains a '[' character, the
// constructed glob pattern becomes syntactically invalid, causing
// filepath.Glob to return filepath.ErrBadPattern.
func TestDiscoverTasksFile_GlobError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory name with '[' may not be supported on Windows")
	}

	tmpDir := realPath(t, t.TempDir())
	// Create a directory whose name contains '[', which makes any glob
	// pattern rooted in this directory syntactically invalid.
	badDir := filepath.Join(tmpDir, "bad[dir")
	require.NoError(t, os.MkdirAll(badDir, 0755))

	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(badDir))
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})

	// No well-known files exist, so the function will reach the subdirectory
	// glob phase. filepath.Abs succeeds but filepath.Glob fails because the
	// absolute path contains '[', which is an unclosed bracket in glob syntax.
	_, err = DiscoverTasksFile("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no tasks file found")
}
