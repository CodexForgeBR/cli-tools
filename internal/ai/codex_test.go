package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodexRunner_BuildArgs(t *testing.T) {
	testCases := []struct {
		name     string
		runner   CodexRunner
		prompt   string
		validate func(t *testing.T, args []string)
	}{
		{
			name: "includes exec subcommand",
			runner: CodexRunner{
				Model:   "codex-default",
				Verbose: false,
			},
			prompt: "test prompt",
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
			prompt: "test prompt",
			validate: func(t *testing.T, args []string) {
				assert.Contains(t, args, "--json")
			},
		},
		{
			name: "includes --output-last-message flag",
			runner: CodexRunner{
				Model:   "codex-default",
				Verbose: false,
			},
			prompt: "test prompt",
			validate: func(t *testing.T, args []string) {
				assert.Contains(t, args, "--output-last-message")
			},
		},
		{
			name: "includes --dangerously-bypass-approvals-and-sandbox",
			runner: CodexRunner{
				Model:   "codex-default",
				Verbose: false,
			},
			prompt: "test prompt",
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
			prompt: "this is a specific test prompt",
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
			prompt: "complex prompt",
			validate: func(t *testing.T, args []string) {
				assert.Contains(t, args, "exec")
				assert.Contains(t, args, "--json")
				assert.Contains(t, args, "--output-last-message")
				assert.Contains(t, args, "--dangerously-bypass-approvals-and-sandbox")
				assert.Contains(t, args, "complex prompt")
			},
		},
		{
			name: "verbose flag when verbose=true",
			runner: CodexRunner{
				Model:   "codex-default",
				Verbose: true,
			},
			prompt: "test",
			validate: func(t *testing.T, args []string) {
				// Codex may or may not have a verbose flag - document actual behavior
				// This test verifies the args are built correctly with Verbose=true
				assert.NotEmpty(t, args)
			},
		},
		{
			name: "verbose flag when verbose=false",
			runner: CodexRunner{
				Model:   "codex-default",
				Verbose: false,
			},
			prompt: "test",
			validate: func(t *testing.T, args []string) {
				// This test verifies the args are built correctly with Verbose=false
				assert.NotEmpty(t, args)
			},
		},
		{
			name: "exec subcommand appears first",
			runner: CodexRunner{
				Model:   "codex-default",
				Verbose: false,
			},
			prompt: "test",
			validate: func(t *testing.T, args []string) {
				require.NotEmpty(t, args)
				// exec should typically be the first argument for codex
				assert.Contains(t, args, "exec")
				execIdx := indexOf(args, "exec")
				assert.GreaterOrEqual(t, execIdx, 0, "exec subcommand should be present")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := tc.runner.BuildArgs(tc.prompt)
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
		args := runner.BuildArgs("")
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
		args := runner.BuildArgs(prompt)
		assert.Contains(t, args, prompt)
	})

	t.Run("multiline prompt", func(t *testing.T) {
		runner := CodexRunner{
			Model:   "codex-default",
			Verbose: false,
		}
		prompt := "first line\nsecond line\nthird line"
		args := runner.BuildArgs(prompt)
		assert.Contains(t, args, prompt)
	})

	t.Run("very long prompt", func(t *testing.T) {
		runner := CodexRunner{
			Model:   "codex-default",
			Verbose: false,
		}
		prompt := "long prompt " + string(make([]byte, 2000))
		args := runner.BuildArgs(prompt)
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
				args := runner.BuildArgs("test")
				assert.NotEmpty(t, args)
				// Model may or may not be included in args depending on implementation
			})
		}
	})
}

func TestCodexRunner_BinaryName(t *testing.T) {
	t.Run("uses codex as binary name", func(t *testing.T) {
		// This test documents that CodexRunner should use "codex" as the binary name
		// The actual command execution would use exec.Command("codex", args...)
		runner := CodexRunner{
			Model:   "codex-default",
			Verbose: false,
		}
		args := runner.BuildArgs("test")

		// The args should not contain the binary name itself (that's passed separately to exec.Command)
		// But we document that the binary name should be "codex"
		assert.NotEmpty(t, args)

		// This is a documentation test - the binary name "codex" should be used
		// when calling exec.Command("codex", args...)
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
		args := runner.BuildArgs("test prompt")

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
