package ai

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/CodexForgeBR/cli-tools/internal/parser"
	"github.com/CodexForgeBR/cli-tools/internal/ratelimit"
)

// CodexRunner implements AIRunner for Codex CLI.
type CodexRunner struct {
	Model             string
	Verbose           bool
	InactivityTimeout int // seconds before killing inactive process
}

// BuildArgs constructs the argument list for the codex CLI command.
// outputPath is the file where codex writes the extracted last message via --output-last-message.
func (r *CodexRunner) BuildArgs(prompt string, outputPath string) []string {
	args := []string{
		"exec",
		"--json",
		"--output-last-message", outputPath,
		"--dangerously-bypass-approvals-and-sandbox",
	}
	if r.Model != "" {
		args = append(args, "--model", r.Model)
	}
	args = append(args, prompt)
	return args
}

// Run executes the codex CLI with the given prompt and writes output to outputPath.
// Uses cmd.Start() + MonitorProcess + cmd.Wait() for process lifecycle management.
// Codex writes extracted text to outputPath via --output-last-message; raw JSONL goes to a separate file.
// Falls back to parsing JSONL if --output-last-message produces empty output.
// Checks for rate limits after execution and returns a RateLimitError if detected.
func (r *CodexRunner) Run(ctx context.Context, prompt string, outputPath string) error {
	args := r.BuildArgs(prompt, outputPath)

	// Create a cancellable context for the monitor to use
	monCtx, monCancel := context.WithCancel(ctx)
	defer monCancel()

	cmd := exec.CommandContext(monCtx, "codex", args...)

	// Raw JSONL output file (separate from the extracted text output)
	rawPath := outputPath + ".jsonl"
	rawFile, err := os.Create(rawPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}

	// Merge stdout and stderr into the raw JSONL file
	cmd.Stdout = rawFile
	cmd.Stderr = rawFile

	// Start the process (non-blocking)
	if err := cmd.Start(); err != nil {
		rawFile.Close()
		return fmt.Errorf("codex command failed: %w", err)
	}

	// Start monitor in a goroutine
	go MonitorProcess(monCtx, monCancel, MonitorConfig{
		InactivityTimeout: r.InactivityTimeout,
		OutputPath:        rawPath,
	})

	// Wait for process to complete (or be killed by monitor)
	runErr := cmd.Wait()
	rawFile.Close()

	// Check if outputPath has content from --output-last-message
	// If empty or missing, fallback to parsing raw JSONL
	outputContent, readErr := os.ReadFile(outputPath)
	if readErr != nil || len(bytes.TrimSpace(outputContent)) == 0 {
		rawData, rawReadErr := os.ReadFile(rawPath)
		if rawReadErr == nil {
			extracted := parser.ParseCodexJSONL(string(rawData))
			if writeErr := os.WriteFile(outputPath, []byte(extracted), 0644); writeErr != nil {
				return fmt.Errorf("write parsed output: %w", writeErr)
			}
		} else {
			// If we can't read the raw file either, create an empty output
			if writeErr := os.WriteFile(outputPath, []byte(""), 0644); writeErr != nil {
				return fmt.Errorf("write empty output: %w", writeErr)
			}
		}
	}

	// Check for rate limit in output regardless of command success
	rateLimitInfo, checkErr := ratelimit.CheckRateLimit(outputPath)
	if checkErr == nil && rateLimitInfo != nil && rateLimitInfo.Detected {
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
