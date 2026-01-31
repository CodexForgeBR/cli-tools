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

func TestClaudeRunner_BuildArgs(t *testing.T) {
	testCases := []struct {
		name     string
		runner   ClaudeRunner
		prompt   string
		validate func(t *testing.T, args []string)
	}{
		{
			name: "includes --print flag",
			runner: ClaudeRunner{
				Model:    "claude-sonnet-4-5",
				MaxTurns: 10,
				Verbose:  false,
			},
			prompt: "test prompt",
			validate: func(t *testing.T, args []string) {
				assert.Contains(t, args, "--print")
			},
		},
		{
			name: "includes --model flag with correct model",
			runner: ClaudeRunner{
				Model:    "claude-sonnet-4-5",
				MaxTurns: 10,
				Verbose:  false,
			},
			prompt: "test prompt",
			validate: func(t *testing.T, args []string) {
				require.Contains(t, args, "--model")
				modelIdx := indexOf(args, "--model")
				require.Greater(t, len(args), modelIdx+1, "--model should have a value")
				assert.Equal(t, "claude-sonnet-4-5", args[modelIdx+1])
			},
		},
		{
			name: "includes --max-turns flag",
			runner: ClaudeRunner{
				Model:    "claude-sonnet-4-5",
				MaxTurns: 15,
				Verbose:  false,
			},
			prompt: "test prompt",
			validate: func(t *testing.T, args []string) {
				require.Contains(t, args, "--max-turns")
				maxTurnsIdx := indexOf(args, "--max-turns")
				require.Greater(t, len(args), maxTurnsIdx+1, "--max-turns should have a value")
				assert.Equal(t, "15", args[maxTurnsIdx+1])
			},
		},
		{
			name: "--verbose is always present regardless of Verbose field",
			runner: ClaudeRunner{
				Model:    "claude-sonnet-4-5",
				MaxTurns: 10,
				Verbose:  false,
			},
			prompt: "test prompt",
			validate: func(t *testing.T, args []string) {
				assert.Contains(t, args, "--verbose", "--verbose should always be present for stream-json")
			},
		},
		{
			name: "--verbose is present when Verbose=true",
			runner: ClaudeRunner{
				Model:    "claude-sonnet-4-5",
				MaxTurns: 10,
				Verbose:  true,
			},
			prompt: "test prompt",
			validate: func(t *testing.T, args []string) {
				assert.Contains(t, args, "--verbose")
			},
		},
		{
			name: "--output-format stream-json always present",
			runner: ClaudeRunner{
				Model:    "claude-sonnet-4-5",
				MaxTurns: 10,
				Verbose:  false,
			},
			prompt: "test prompt",
			validate: func(t *testing.T, args []string) {
				require.Contains(t, args, "--output-format")
				formatIdx := indexOf(args, "--output-format")
				require.Greater(t, len(args), formatIdx+1, "--output-format should have a value")
				assert.Equal(t, "stream-json", args[formatIdx+1])
			},
		},
		{
			name: "includes --dangerously-skip-permissions",
			runner: ClaudeRunner{
				Model:    "claude-sonnet-4-5",
				MaxTurns: 10,
				Verbose:  false,
			},
			prompt: "test prompt",
			validate: func(t *testing.T, args []string) {
				assert.Contains(t, args, "--dangerously-skip-permissions")
			},
		},
		{
			name: "full command with all flags",
			runner: ClaudeRunner{
				Model:    "claude-opus-4-5",
				MaxTurns: 20,
				Verbose:  true,
			},
			prompt: "complex test prompt",
			validate: func(t *testing.T, args []string) {
				assert.Contains(t, args, "--print")
				assert.Contains(t, args, "--model")
				assert.Contains(t, args, "--max-turns")
				assert.Contains(t, args, "--verbose")
				assert.Contains(t, args, "--output-format")
				assert.Contains(t, args, "--dangerously-skip-permissions")
				assert.Contains(t, args, "complex test prompt")

				// Verify model value
				modelIdx := indexOf(args, "--model")
				assert.Equal(t, "claude-opus-4-5", args[modelIdx+1])

				// Verify max-turns value
				maxTurnsIdx := indexOf(args, "--max-turns")
				assert.Equal(t, "20", args[maxTurnsIdx+1])

				// Verify output-format value
				formatIdx := indexOf(args, "--output-format")
				assert.Equal(t, "stream-json", args[formatIdx+1])
			},
		},
		{
			name: "prompt is included in args",
			runner: ClaudeRunner{
				Model:    "claude-sonnet-4-5",
				MaxTurns: 10,
				Verbose:  false,
			},
			prompt: "this is my specific prompt",
			validate: func(t *testing.T, args []string) {
				assert.Contains(t, args, "this is my specific prompt")
			},
		},
		{
			name: "different max turns values",
			runner: ClaudeRunner{
				Model:    "claude-sonnet-4-5",
				MaxTurns: 5,
				Verbose:  false,
			},
			prompt: "test",
			validate: func(t *testing.T, args []string) {
				maxTurnsIdx := indexOf(args, "--max-turns")
				assert.Equal(t, "5", args[maxTurnsIdx+1])
			},
		},
		{
			name: "InactivityTimeout field is stored",
			runner: ClaudeRunner{
				Model:             "claude-sonnet-4-5",
				MaxTurns:          10,
				InactivityTimeout: 300,
			},
			prompt: "test",
			validate: func(t *testing.T, args []string) {
				// InactivityTimeout doesn't affect args, just verify args are valid
				assert.Contains(t, args, "--print")
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

func TestClaudeRunner_BuildArgs_EdgeCases(t *testing.T) {
	t.Run("empty prompt is handled", func(t *testing.T) {
		runner := ClaudeRunner{
			Model:    "claude-sonnet-4-5",
			MaxTurns: 10,
			Verbose:  false,
		}
		args := runner.BuildArgs("")
		assert.NotEmpty(t, args, "should still return args even with empty prompt")
	})

	t.Run("zero max turns", func(t *testing.T) {
		runner := ClaudeRunner{
			Model:    "claude-sonnet-4-5",
			MaxTurns: 0,
			Verbose:  false,
		}
		args := runner.BuildArgs("test")
		maxTurnsIdx := indexOf(args, "--max-turns")
		if maxTurnsIdx != -1 {
			assert.Equal(t, "0", args[maxTurnsIdx+1])
		}
	})

	t.Run("prompt with special characters", func(t *testing.T) {
		runner := ClaudeRunner{
			Model:    "claude-sonnet-4-5",
			MaxTurns: 10,
			Verbose:  false,
		}
		prompt := "test with \"quotes\" and 'apostrophes' and $special chars"
		args := runner.BuildArgs(prompt)
		assert.Contains(t, args, prompt)
	})

	t.Run("very long prompt", func(t *testing.T) {
		runner := ClaudeRunner{
			Model:    "claude-sonnet-4-5",
			MaxTurns: 10,
			Verbose:  false,
		}
		prompt := "very long prompt " + string(make([]byte, 1000))
		args := runner.BuildArgs(prompt)
		assert.Contains(t, args, prompt)
	})
}

// indexOf returns the index of the first occurrence of str in slice, or -1 if not found
func indexOf(slice []string, str string) int {
	for i, s := range slice {
		if s == str {
			return i
		}
	}
	return -1
}

// ---------------------------------------------------------------------------
// ClaudeRunner.Run() tests
// ---------------------------------------------------------------------------

func TestClaudeRunnerRun_CreateOutputError(t *testing.T) {
	r := &ClaudeRunner{Model: "test-model", MaxTurns: 1}
	// Pass an output path in a directory that does not exist -> os.Create fails
	err := r.Run(context.Background(), "prompt", "/nonexistent-dir-abc123/output.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create output file")
}

func TestClaudeRunnerRun_CommandFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.json")

	// Ensure "claude" is NOT in PATH by using a PATH with only a harmless directory
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir)
	defer os.Setenv("PATH", origPath)

	r := &ClaudeRunner{Model: "test-model", MaxTurns: 1}
	err := r.Run(context.Background(), "prompt", outputPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "claude command failed")
}

func TestClaudeRunnerRun_RateLimitDetected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tmpDir := t.TempDir()

	// Create a fake "claude" script that writes stream-json with rate-limit content
	fakeScript := filepath.Join(tmpDir, "claude")
	// Write a result event containing the rate limit text (small enough for bare pattern)
	scriptContent := `#!/bin/sh
echo '{"type":"result","result":"rate limit exceeded"}'
exit 1
`
	err := os.WriteFile(fakeScript, []byte(scriptContent), 0755)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+origPath)
	defer os.Setenv("PATH", origPath)

	outputPath := filepath.Join(tmpDir, "output.json")
	r := &ClaudeRunner{Model: "test-model", MaxTurns: 1}
	err = r.Run(context.Background(), "prompt", outputPath)
	require.Error(t, err)

	var rlErr *RateLimitError
	assert.True(t, errors.As(err, &rlErr), "should return a RateLimitError")
}

func TestClaudeRunnerRun_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tmpDir := t.TempDir()

	// Create a fake "claude" script that writes stream-json output
	fakeScript := filepath.Join(tmpDir, "claude")
	scriptContent := `#!/bin/sh
echo '{"type":"result","result":"RALPH_STATUS: success"}'
`
	err := os.WriteFile(fakeScript, []byte(scriptContent), 0755)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+origPath)
	defer os.Setenv("PATH", origPath)

	outputPath := filepath.Join(tmpDir, "output.json")
	r := &ClaudeRunner{Model: "test-model", MaxTurns: 1}
	err = r.Run(context.Background(), "prompt", outputPath)
	require.NoError(t, err)

	// Verify that the output was parsed from stream-json
	data, readErr := os.ReadFile(outputPath)
	require.NoError(t, readErr)
	assert.Contains(t, string(data), "RALPH_STATUS: success")
}
