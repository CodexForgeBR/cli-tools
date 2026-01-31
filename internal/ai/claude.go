package ai

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/CodexForgeBR/cli-tools/internal/ratelimit"
)

// ClaudeRunner implements AIRunner for Claude CLI.
type ClaudeRunner struct {
	Model    string
	MaxTurns int
	Verbose  bool
}

// BuildArgs constructs the argument list for the claude CLI command.
func (r *ClaudeRunner) BuildArgs(prompt string) []string {
	args := []string{
		"--print",
		"--dangerously-skip-permissions",
		"--output-format", "stream-json",
		"--model", r.Model,
		"--max-turns", fmt.Sprintf("%d", r.MaxTurns),
	}
	if r.Verbose {
		args = append(args, "--verbose")
	}
	args = append(args, "--prompt", prompt)
	return args
}

// Run executes the claude CLI with the given prompt and writes output to outputPath.
// Checks for rate limits after execution and returns a RateLimitError if detected.
func (r *ClaudeRunner) Run(ctx context.Context, prompt string, outputPath string) error {
	args := r.BuildArgs(prompt)
	cmd := exec.CommandContext(ctx, "claude", args...)

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer outFile.Close()

	cmd.Stdout = outFile
	cmd.Stderr = os.Stderr

	runErr := cmd.Run()

	// Check for rate limit in output regardless of command success
	rateLimitInfo, checkErr := ratelimit.CheckRateLimit(outputPath)
	if checkErr == nil && rateLimitInfo != nil && rateLimitInfo.Detected {
		// Rate limit detected - return special error type
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
