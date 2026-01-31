package learnings

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// Template for newly initialized learnings file
	learningsTemplate = `# Ralph Loop Learnings

## Codebase Patterns
<!-- Add reusable patterns discovered during implementation -->

---

## Iteration Log
`
)

// InitLearnings creates a new learnings markdown file with the standard template.
// Creates parent directories if needed. Returns error if file creation fails.
func InitLearnings(filePath string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Write template to file
	if err := os.WriteFile(filePath, []byte(learningsTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write learnings file: %w", err)
	}

	return nil
}

// AppendLearnings appends a new learning entry to the learnings file.
// Each entry includes the iteration number and timestamp in local timezone.
// Does nothing if content is empty.
// Returns error if file operations fail.
func AppendLearnings(filePath string, iteration int, content string) error {
	// Skip if content is empty
	if content == "" {
		return nil
	}

	// Format the entry with iteration number and local timestamp
	// Use Local timezone explicitly for consistent test behavior
	timestamp := time.Now().Local().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("\n## Iteration %d (%s)\n\n%s\n", iteration, timestamp, content)

	// Open file in append mode
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open learnings file: %w", err)
	}
	defer f.Close()

	// Write the entry
	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("failed to append learnings: %w", err)
	}

	return nil
}

// ReadLearnings reads the entire learnings file content.
// Returns empty string if file doesn't exist (not an error).
// Returns error only for actual I/O failures.
func ReadLearnings(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		// File not existing is not an error - return empty string
		if os.IsNotExist(err) {
			return ""
		}
		// For other errors, return empty string silently
		// (learnings are optional and shouldn't break workflow)
		return ""
	}

	return string(content)
}
