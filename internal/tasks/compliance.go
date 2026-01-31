package tasks

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// forbiddenPatterns maps each forbidden substring to a human-readable
// description used in violation messages.
var forbiddenPatterns = []struct {
	pattern     string
	description string
}{
	{"git push", "contains 'git push' command"},
	{"gh pr create", "contains 'gh pr create' command"},
}

// CheckCompliance scans the file at filePath for forbidden patterns and
// returns a slice of violation descriptions. An empty slice means the file
// is compliant.
func CheckCompliance(filePath string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var violations []string
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		for _, fp := range forbiddenPatterns {
			if strings.Contains(line, fp.pattern) {
				violations = append(violations,
					fmt.Sprintf("line %d: %s", lineNum, fp.description))
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return violations, nil
}
