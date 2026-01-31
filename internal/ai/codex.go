package ai

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/CodexForgeBR/cli-tools/internal/ratelimit"
)

// CodexRunner implements AIRunner for Codex CLI.
type CodexRunner struct {
	Model   string
	Verbose bool
}

// BuildArgs constructs the argument list for the codex CLI command.
func (r *CodexRunner) BuildArgs(prompt string) []string {
	args := []string{
		"exec",
		"--json",
		"--output-last-message",
		"--dangerously-bypass-approvals-and-sandbox",
	}
	if r.Model != "" {
		args = append(args, "--model", r.Model)
	}
	args = append(args, prompt)
	return args
}

// Run executes the codex CLI with the given prompt and writes output to outputPath.
// Checks for rate limits after execution and returns a RateLimitError if detected.
func (r *CodexRunner) Run(ctx context.Context, prompt string, outputPath string) error {
	args := r.BuildArgs(prompt)
	cmd := exec.CommandContext(ctx, "codex", args...)

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
		return fmt.Errorf("codex command failed: %w", runErr)
	}

	return nil
}
