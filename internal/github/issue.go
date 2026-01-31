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

// ParseIssueRef parses a GitHub issue reference in format "owner/repo#number".
// Returns the owner, repo name, issue number, and any parsing error.
//
// Examples:
//   - "CodexForgeBR/cli-tools#42" → ("CodexForgeBR", "cli-tools", 42, nil)
//   - "owner/repo#123" → ("owner", "repo", 123, nil)
//   - "invalid" → ("", "", 0, error)
func ParseIssueRef(ref string) (owner, repo string, number int, err error) {
	if ref == "" {
		return "", "", 0, fmt.Errorf("empty issue reference")
	}

	// Split by '#' to separate repo path from issue number
	parts := strings.Split(ref, "#")
	if len(parts) != 2 {
		return "", "", 0, fmt.Errorf("invalid issue reference format: expected 'owner/repo#number', got %q", ref)
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
//
// Requires gh CLI to be installed and authenticated.
func FetchIssue(owner, repo string, number int) (string, error) {
	if owner == "" {
		return "", fmt.Errorf("owner cannot be empty")
	}
	if repo == "" {
		return "", fmt.Errorf("repo cannot be empty")
	}
	if number <= 0 {
		return "", fmt.Errorf("issue number must be positive, got %d", number)
	}

	// Use gh CLI to fetch the issue
	// Format: gh issue view NUMBER --repo OWNER/REPO --json title,body --jq '.title + "\n\n" + .body'
	repoPath := fmt.Sprintf("%s/%s", owner, repo)
	cmd := exec.Command("gh", "issue", "view", strconv.Itoa(number),
		"--repo", repoPath,
		"--json", "title,body",
		"--jq", `.title + "\n\n" + .body`)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to fetch issue %s/%s#%d: %w\nOutput: %s",
			owner, repo, number, err, string(output))
	}

	content := strings.TrimSpace(string(output))
	if content == "" {
		return "", fmt.Errorf("issue %s/%s#%d has no content", owner, repo, number)
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
