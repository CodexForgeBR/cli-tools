package tasks

import (
	"bufio"
	"os"
	"regexp"
)

// uncheckedRE matches a Markdown checkbox that has NOT been ticked.
// Allows leading whitespace: "  - [ ] some task"
var uncheckedRE = regexp.MustCompile(`^\s*- \[ \]`)

// checkedRE matches a Markdown checkbox that HAS been ticked (x or X).
// Allows leading whitespace: "  - [x] some task" or "  - [X] some task"
var checkedRE = regexp.MustCompile(`^\s*- \[[xX]\]`)

// CountUnchecked returns the number of unchecked task lines in filePath.
// A line is considered an unchecked task if it matches the pattern: ^\s*- \[ \]
func CountUnchecked(filePath string) (int, error) {
	return countMatches(filePath, uncheckedRE)
}

// CountChecked returns the number of checked task lines in filePath.
// A line is considered a checked task if it matches: ^\s*- \[[xX]\]
func CountChecked(filePath string) (int, error) {
	return countMatches(filePath, checkedRE)
}

// countMatches counts lines in filePath that match the given regexp.
func countMatches(filePath string, re *regexp.Regexp) (int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if re.MatchString(scanner.Text()) {
			count++
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return count, nil
}
