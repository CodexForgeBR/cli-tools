package ai

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/CodexForgeBR/cli-tools/internal/parser"
	"github.com/CodexForgeBR/cli-tools/internal/ratelimit"
)

// ClaudeRunner implements AIRunner for Claude CLI.
type ClaudeRunner struct {
	Model             string
	MaxTurns          int
	Verbose           bool // Controls Go-level logging, not CLI flag
	InactivityTimeout int  // seconds before killing inactive process
}

// BuildArgs constructs the argument list for the claude CLI command.
// Always includes --verbose and --output-format stream-json (required for monitoring).
func (r *ClaudeRunner) BuildArgs(prompt string) []string {
	args := []string{
		"--print",
		"--verbose",
		"--output-format", "stream-json",
		"--dangerously-skip-permissions",
		"--model", r.Model,
		"--max-turns", fmt.Sprintf("%d", r.MaxTurns),
		"--", prompt,
	}
	return args
}

// Run executes the claude CLI with the given prompt and writes output to outputPath.
// Uses cmd.Start() + MonitorProcess + cmd.Wait() for process lifecycle management.
// Parses stream-json output to extract text content.
// Checks for rate limits after execution and returns a RateLimitError if detected.
func (r *ClaudeRunner) Run(ctx context.Context, prompt string, outputPath string) error {
	args := r.BuildArgs(prompt)

	// Create a cancellable context for the monitor to use
	monCtx, monCancel := context.WithCancel(ctx)
	defer monCancel()

	cmd := exec.CommandContext(monCtx, "claude", args...)

	// Raw stream-json output file
	rawPath := outputPath + ".stream.json"
	rawFile, err := os.Create(rawPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}

	// Merge stdout and stderr into the raw file
	cmd.Stdout = rawFile
	cmd.Stderr = rawFile

	// Start the process (non-blocking)
	if err := cmd.Start(); err != nil {
		rawFile.Close()
		return fmt.Errorf("claude command failed: %w", err)
	}

	// Start monitor in a goroutine
	go MonitorProcess(monCtx, monCancel, MonitorConfig{
		InactivityTimeout: r.InactivityTimeout,
		OutputPath:        rawPath,
	})

	// Wait for process to complete (or be killed by monitor)
	runErr := cmd.Wait()
	rawFile.Close()

	// Parse stream-json output to extract text
	rawData, readErr := os.ReadFile(rawPath)
	if readErr == nil {
		extracted := parser.ParseStreamJSON(string(rawData))
		if writeErr := os.WriteFile(outputPath, []byte(extracted), 0644); writeErr != nil {
			return fmt.Errorf("write parsed output: %w", writeErr)
		}
	} else {
		// If we can't read the raw file, create an empty output
		if writeErr := os.WriteFile(outputPath, []byte(""), 0644); writeErr != nil {
			return fmt.Errorf("write empty output: %w", writeErr)
		}
	}

	// Check for rate limit in extracted output regardless of command success
	rateLimitInfo, checkErr := ratelimit.CheckRateLimit(outputPath)
	if checkErr == nil && rateLimitInfo != nil && rateLimitInfo.Detected {
		return &RateLimitError{
			Info:          rateLimitInfo,
			UnderlyingErr: runErr,
		}
	}

	if runErr != nil {
		return fmt.Errorf("claude command failed: %w", runErr)
	}

	return nil
}
