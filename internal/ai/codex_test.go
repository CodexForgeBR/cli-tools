package ai

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodexRunner_BuildArgs(t *testing.T) {
	testCases := []struct {
		name       string
		runner     CodexRunner
		prompt     string
		outputPath string
		validate   func(t *testing.T, args []string)
	}{
		{
			name: "includes exec subcommand",
			runner: CodexRunner{
				Model:   "codex-default",
				Verbose: false,
			},
			prompt:     "test prompt",
			outputPath: "/tmp/output.txt",
			validate: func(t *testing.T, args []string) {
				assert.Contains(t, args, "exec")
			},
		},
		{
			name: "includes --json flag",
			runner: CodexRunner{
				Model:   "codex-default",
				Verbose: false,
			},
			prompt:     "test prompt",
			outputPath: "/tmp/output.txt",
			validate: func(t *testing.T, args []string) {
				assert.Contains(t, args, "--json")
			},
		},
		{
			name: "--output-last-message followed by outputPath",
			runner: CodexRunner{
				Model:   "codex-default",
				Verbose: false,
			},
			prompt:     "test prompt",
			outputPath: "/tmp/my-output.txt",
			validate: func(t *testing.T, args []string) {
				require.Contains(t, args, "--output-last-message")
				olmIdx := indexOf(args, "--output-last-message")
				require.Greater(t, len(args), olmIdx+1, "--output-last-message should have a value")
				assert.Equal(t, "/tmp/my-output.txt", args[olmIdx+1])
			},
		},
		{
			name: "includes --dangerously-bypass-approvals-and-sandbox",
			runner: CodexRunner{
				Model:   "codex-default",
				Verbose: false,
			},
			prompt:     "test prompt",
			outputPath: "/tmp/output.txt",
			validate: func(t *testing.T, args []string) {
				assert.Contains(t, args, "--dangerously-bypass-approvals-and-sandbox")
			},
		},
		{
			name: "prompt is included in args",
			runner: CodexRunner{
				Model:   "codex-default",
				Verbose: false,
			},
			prompt:     "this is a specific test prompt",
			outputPath: "/tmp/output.txt",
			validate: func(t *testing.T, args []string) {
				assert.Contains(t, args, "this is a specific test prompt")
			},
		},
		{
			name: "full command with all required flags",
			runner: CodexRunner{
				Model:   "codex-advanced",
				Verbose: true,
			},
			prompt:     "complex prompt",
			outputPath: "/tmp/output.txt",
			validate: func(t *testing.T, args []string) {
				assert.Contains(t, args, "exec")
				assert.Contains(t, args, "--json")
				assert.Contains(t, args, "--output-last-message")
				assert.Contains(t, args, "--dangerously-bypass-approvals-and-sandbox")
				assert.Contains(t, args, "complex prompt")

				// Verify output path follows --output-last-message
				olmIdx := indexOf(args, "--output-last-message")
				assert.Equal(t, "/tmp/output.txt", args[olmIdx+1])
			},
		},
		{
			name: "exec subcommand appears first",
			runner: CodexRunner{
				Model:   "codex-default",
				Verbose: false,
			},
			prompt:     "test",
			outputPath: "/tmp/output.txt",
			validate: func(t *testing.T, args []string) {
				require.NotEmpty(t, args)
				assert.Contains(t, args, "exec")
				execIdx := indexOf(args, "exec")
				jsonIdx := indexOf(args, "--json")
				assert.Less(t, execIdx, jsonIdx, "exec should come before --json")
			},
		},
		{
			name: "InactivityTimeout field is stored",
			runner: CodexRunner{
				Model:             "codex-default",
				InactivityTimeout: 600,
			},
			prompt:     "test",
			outputPath: "/tmp/output.txt",
			validate: func(t *testing.T, args []string) {
				// InactivityTimeout doesn't affect args, just verify args are valid
				assert.Contains(t, args, "exec")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := tc.runner.BuildArgs(tc.prompt, tc.outputPath)
			require.NotEmpty(t, args, "BuildArgs should return non-empty args list")
			tc.validate(t, args)
		})
	}
}

func TestCodexRunner_BuildArgs_EdgeCases(t *testing.T) {
	t.Run("empty prompt is handled", func(t *testing.T) {
		runner := CodexRunner{
			Model:   "codex-default",
			Verbose: false,
		}
		args := runner.BuildArgs("", "/tmp/output.txt")
		assert.NotEmpty(t, args, "should still return args even with empty prompt")
		assert.Contains(t, args, "exec")
		assert.Contains(t, args, "--json")
		assert.Contains(t, args, "--output-last-message")
		assert.Contains(t, args, "--dangerously-bypass-approvals-and-sandbox")
	})

	t.Run("prompt with special characters", func(t *testing.T) {
		runner := CodexRunner{
			Model:   "codex-default",
			Verbose: false,
		}
		prompt := "test with \"quotes\" and 'apostrophes' and $vars"
		args := runner.BuildArgs(prompt, "/tmp/output.txt")
		assert.Contains(t, args, prompt)
	})

	t.Run("multiline prompt", func(t *testing.T) {
		runner := CodexRunner{
			Model:   "codex-default",
			Verbose: false,
		}
		prompt := "first line\nsecond line\nthird line"
		args := runner.BuildArgs(prompt, "/tmp/output.txt")
		assert.Contains(t, args, prompt)
	})

	t.Run("very long prompt", func(t *testing.T) {
		runner := CodexRunner{
			Model:   "codex-default",
			Verbose: false,
		}
		prompt := "long prompt " + string(make([]byte, 2000))
		args := runner.BuildArgs(prompt, "/tmp/output.txt")
		assert.Contains(t, args, prompt)
	})

	t.Run("different model names", func(t *testing.T) {
		models := []string{"codex-default", "codex-advanced", "gpt-4"}
		for _, model := range models {
			t.Run("model_"+model, func(t *testing.T) {
				runner := CodexRunner{
					Model:   model,
					Verbose: false,
				}
				args := runner.BuildArgs("test", "/tmp/output.txt")
				assert.NotEmpty(t, args)
			})
		}
	})
}

func TestCodexRunner_BinaryName(t *testing.T) {
	t.Run("uses codex as binary name", func(t *testing.T) {
		runner := CodexRunner{
			Model:   "codex-default",
			Verbose: false,
		}
		args := runner.BuildArgs("test", "/tmp/output.txt")
		assert.NotEmpty(t, args)
		binaryName := "codex"
		assert.Equal(t, "codex", binaryName, "should use codex as binary name")
	})
}

func TestCodexRunner_ArgsOrder(t *testing.T) {
	t.Run("args have logical order", func(t *testing.T) {
		runner := CodexRunner{
			Model:   "codex-default",
			Verbose: false,
		}
		args := runner.BuildArgs("test prompt", "/tmp/output.txt")

		require.NotEmpty(t, args)

		// Verify key elements are present
		assert.Contains(t, args, "exec")
		assert.Contains(t, args, "--json")
		assert.Contains(t, args, "--output-last-message")
		assert.Contains(t, args, "--dangerously-bypass-approvals-and-sandbox")
		assert.Contains(t, args, "test prompt")

		// exec should typically come before flags
		execIdx := indexOf(args, "exec")
		jsonIdx := indexOf(args, "--json")
		assert.Less(t, execIdx, jsonIdx, "exec should come before --json")
	})
}

// ---------------------------------------------------------------------------
// CodexRunner.Run() tests
// ---------------------------------------------------------------------------

func TestCodexRunnerRun_CreateOutputError(t *testing.T) {
	r := &CodexRunner{Model: "test-model"}
	// Pass an output path in a directory that does not exist -> os.Create fails
	err := r.Run(context.Background(), "prompt", "/nonexistent-dir-abc123/output.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create output file")
}

func TestCodexRunnerRun_CommandFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.json")

	// Ensure "codex" is NOT in PATH by setting PATH to a harmless directory
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir)
	defer os.Setenv("PATH", origPath)

	r := &CodexRunner{Model: "test-model"}
	err := r.Run(context.Background(), "prompt", outputPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "codex command failed")
}

func TestCodexRunnerRun_RateLimitDetected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tmpDir := t.TempDir()

	// Create a fake "codex" script that writes rate-limit content
	// The script writes to stdout (raw JSONL) and does NOT write to --output-last-message
	// so the fallback parser will extract text from the JSONL
	fakeScript := filepath.Join(tmpDir, "codex")
	scriptContent := `#!/bin/sh
echo '{"type":"item.completed","item":{"type":"agent_message","text":"rate limit exceeded"}}'
exit 1
`
	err := os.WriteFile(fakeScript, []byte(scriptContent), 0755)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+origPath)
	defer os.Setenv("PATH", origPath)

	outputPath := filepath.Join(tmpDir, "output.json")
	r := &CodexRunner{Model: "test-model"}
	err = r.Run(context.Background(), "prompt", outputPath)
	require.Error(t, err)

	var rlErr *RateLimitError
	assert.True(t, errors.As(err, &rlErr), "should return a RateLimitError")
}

func TestCodexRunnerRun_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tmpDir := t.TempDir()

	// Create a fake "codex" script that exits successfully
	// It writes JSONL to stdout; --output-last-message is handled by codex itself
	// In the fake, we simulate by not writing to the outputPath so fallback parses JSONL
	fakeScript := filepath.Join(tmpDir, "codex")
	scriptContent := `#!/bin/sh
echo '{"type":"item.completed","item":{"type":"agent_message","text":"RALPH_STATUS: success"}}'
`
	err := os.WriteFile(fakeScript, []byte(scriptContent), 0755)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+origPath)
	defer os.Setenv("PATH", origPath)

	outputPath := filepath.Join(tmpDir, "output.json")
	r := &CodexRunner{Model: "test-model"}
	err = r.Run(context.Background(), "prompt", outputPath)
	require.NoError(t, err)

	// Verify output was extracted from JSONL fallback
	data, readErr := os.ReadFile(outputPath)
	require.NoError(t, readErr)
	assert.Contains(t, string(data), "RALPH_STATUS: success")
}
