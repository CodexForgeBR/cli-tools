// Package github provides utilities for interacting with GitHub issues.
package github

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ParseIssueRef parses a GitHub issue reference.
// Accepts plain numbers (e.g., "136") or "owner/repo#number" format.
// When a plain number is given, owner and repo are empty — the caller
// should pass the number directly to `gh issue view` which infers the
// repo from the current directory.
//
// Examples:
//   - "136" → ("", "", 136, nil)
//   - "CodexForgeBR/cli-tools#42" → ("CodexForgeBR", "cli-tools", 42, nil)
//   - "invalid" → ("", "", 0, error)
func ParseIssueRef(ref string) (owner, repo string, number int, err error) {
	if ref == "" {
		return "", "", 0, fmt.Errorf("empty issue reference")
	}

	// Try plain number first (e.g., "136")
	if n, parseErr := strconv.Atoi(ref); parseErr == nil {
		if n <= 0 {
			return "", "", 0, fmt.Errorf("issue number must be positive, got %d", n)
		}
		return "", "", n, nil
	}

	// Split by '#' to separate repo path from issue number
	parts := strings.Split(ref, "#")
	if len(parts) != 2 {
		return "", "", 0, fmt.Errorf("invalid issue reference format: expected number or 'owner/repo#number', got %q", ref)
	}

	// Parse the repo path (owner/repo)
	repoPath := parts[0]
	repoParts := strings.Split(repoPath, "/")
	if len(repoParts) != 2 {
		return "", "", 0, fmt.Errorf("invalid repo path: expected 'owner/repo', got %q", repoPath)
	}

	owner = repoParts[0]
	repo = repoParts[1]

	// Parse the issue number
	numberStr := parts[1]
	number, err = strconv.Atoi(numberStr)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid issue number %q: %w", numberStr, err)
	}

	if number <= 0 {
		return "", "", 0, fmt.Errorf("issue number must be positive, got %d", number)
	}

	return owner, repo, number, nil
}

// FetchIssue fetches a GitHub issue using the gh CLI tool.
// Returns the issue content (title and body) as a string.
// When owner and repo are empty, gh infers the repository from the
// current directory's git remote (matching the bash script behavior).
//
// Requires gh CLI to be installed and authenticated.
func FetchIssue(owner, repo string, number int) (string, error) {
	if number <= 0 {
		return "", fmt.Errorf("issue number must be positive, got %d", number)
	}

	// Build gh command args
	args := []string{"issue", "view", strconv.Itoa(number),
		"--json", "title,body",
		"--jq", `.title + "\n\n" + .body`}

	// Only add --repo if owner/repo were explicitly provided
	if owner != "" && repo != "" {
		args = append(args, "--repo", fmt.Sprintf("%s/%s", owner, repo))
	}

	cmd := exec.Command("gh", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		ref := fmt.Sprintf("#%d", number)
		if owner != "" {
			ref = fmt.Sprintf("%s/%s#%d", owner, repo, number)
		}
		return "", fmt.Errorf("failed to fetch issue %s: %w\nOutput: %s",
			ref, err, string(output))
	}

	content := strings.TrimSpace(string(output))
	if content == "" {
		return "", fmt.Errorf("issue #%d has no content", number)
	}

	return content, nil
}

// CacheIssue saves issue content to a cache directory.
// Creates a file named "github-issue-<number>.md" in the specified directory.
//
// If the directory does not exist, it will be created with 0755 permissions.
func CacheIssue(dir string, content string) error {
	if dir == "" {
		return fmt.Errorf("directory cannot be empty")
	}
	if content == "" {
		return fmt.Errorf("content cannot be empty")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory %q: %w", dir, err)
	}

	// Generate filename: github-issue.md
	filename := "github-issue.md"
	filepath := filepath.Join(dir, filename)

	// Write content to file
	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write issue cache to %q: %w", filepath, err)
	}

	return nil
}
