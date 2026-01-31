package tasks

import (
	"fmt"
	"os"
	"path/filepath"
)

// wellKnownPaths lists the fixed paths to check for a tasks file, in priority order.
var wellKnownPaths = []string{
	"tasks.md",
	"TASKS.md",
	"specs/tasks.md",
	"spec/tasks.md",
}

// subdirRoots lists directories whose immediate subdirectories are searched
// for a tasks.md file (e.g. specs/001-feature/tasks.md).
var subdirRoots = []string{
	"specs",
	"spec",
}

// DiscoverTasksFile locates a tasks file. If tasksFileFlag is provided it is
// used directly (must exist). Otherwise the function walks a deterministic set
// of well-known locations relative to the current working directory.
//
// Search order:
//  1. Explicit flag value (must exist)
//  2. ./tasks.md, ./TASKS.md, ./specs/tasks.md, ./spec/tasks.md
//  3. ./specs/*/tasks.md (first match, alphabetical)
//  4. ./spec/*/tasks.md  (first match, alphabetical)
func DiscoverTasksFile(tasksFileFlag string) (string, error) {
	// ---------------------------------------------------------------
	// 1. Explicit flag
	// ---------------------------------------------------------------
	if tasksFileFlag != "" {
		abs, err := filepath.Abs(tasksFileFlag)
		if err != nil {
			return "", fmt.Errorf("resolving tasks file path: %w", err)
		}
		if _, err := os.Stat(abs); err != nil {
			return "", fmt.Errorf("tasks file not found: %s", tasksFileFlag)
		}
		return abs, nil
	}

	// ---------------------------------------------------------------
	// 2. Well-known fixed paths
	// ---------------------------------------------------------------
	for _, rel := range wellKnownPaths {
		abs, err := filepath.Abs(rel)
		if err != nil {
			continue
		}
		if _, err := os.Stat(abs); err == nil {
			return abs, nil
		}
	}

	// ---------------------------------------------------------------
	// 3-4. Subdirectory glob patterns
	// ---------------------------------------------------------------
	for _, root := range subdirRoots {
		pattern := filepath.Join(root, "*", "tasks.md")
		absPattern, err := filepath.Abs(pattern)
		if err != nil {
			continue
		}
		matches, err := filepath.Glob(absPattern)
		if err != nil {
			continue
		}
		if len(matches) > 0 {
			return matches[0], nil
		}
	}

	return "", fmt.Errorf("no tasks file found (searched well-known paths and subdirectories)")
}
