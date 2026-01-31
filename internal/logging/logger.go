// Package logging provides colored, leveled log output for the ralph-loop CLI.
//
// All output functions write a prefixed, color-coded line. Debug output is
// suppressed unless verbose mode is enabled via SetVerbose(true).
package logging

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

// verbose controls whether Debug() produces output.
var verbose bool

// Color printers for each log level.
var (
	infoPrefix    = color.New(color.FgBlue).SprintFunc()
	successPrefix = color.New(color.FgGreen).SprintFunc()
	warnPrefix    = color.New(color.FgYellow).SprintFunc()
	errorPrefix   = color.New(color.FgRed).SprintFunc()
	phasePrefix   = color.New(color.FgCyan).SprintFunc()
	debugPrefix   = color.New(color.FgBlue).SprintFunc()
)

// SetVerbose enables or disables Debug output.
func SetVerbose(v bool) {
	verbose = v
}

// Info prints an informational message to stderr in blue.
func Info(msg string) {
	fmt.Fprintln(os.Stderr, infoPrefix("[INFO]")+" "+msg)
}

// Success prints a success message to stderr in green.
func Success(msg string) {
	fmt.Fprintln(os.Stderr, successPrefix("[SUCCESS]")+" "+msg)
}

// Warn prints a warning message to stderr in yellow.
func Warn(msg string) {
	fmt.Fprintln(os.Stderr, warnPrefix("[WARN]")+" "+msg)
}

// Error prints an error message to stderr in red.
func Error(msg string) {
	fmt.Fprintln(os.Stderr, errorPrefix("[ERROR]")+" "+msg)
}

// Phase prints a phase header to stderr in cyan, surrounded by separator lines.
func Phase(msg string) {
	sep := phasePrefix("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(os.Stderr, sep)
	fmt.Fprintln(os.Stderr, phasePrefix("[PHASE]")+" "+msg)
	fmt.Fprintln(os.Stderr, sep)
}

// Debug prints a debug message to stderr in blue, only when verbose mode is enabled.
func Debug(msg string) {
	if !verbose {
		return
	}
	fmt.Fprintln(os.Stderr, debugPrefix("[DEBUG]")+" "+msg)
}

// FormatDuration converts a duration in seconds to a human-readable string.
//
// Examples:
//
//	FormatDuration(0)    => "0s"
//	FormatDuration(45)   => "45s"
//	FormatDuration(90)   => "1m 30s"
//	FormatDuration(3661) => "1h 1m 1s"
//	FormatDuration(7200) => "2h 0m 0s"
func FormatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		m := seconds / 60
		s := seconds % 60
		return fmt.Sprintf("%dm %ds", m, s)
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	return fmt.Sprintf("%dh %dm %ds", h, m, s)
}
