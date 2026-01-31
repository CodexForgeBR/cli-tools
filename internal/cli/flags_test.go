package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CodexForgeBR/cli-tools/internal/config"
)

func TestBindFlags_DefaultValues(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cmd := &cobra.Command{Use: "test"}
	BindFlags(cmd, cfg)

	err := cmd.ParseFlags([]string{})
	require.NoError(t, err)

	assert.Equal(t, "claude", cfg.AIProvider)
	assert.Equal(t, 20, cfg.MaxIterations)
	assert.Equal(t, 5, cfg.MaxInadmissible)
	assert.Equal(t, 10, cfg.MaxClaudeRetry)
	assert.Equal(t, 100, cfg.MaxTurns)
	assert.Equal(t, 1800, cfg.InactivityTimeout)
	assert.Equal(t, ".ralph-loop/learnings.md", cfg.LearningsFile)
	assert.Equal(t, "http://127.0.0.1:18789/webhook", cfg.NotifyWebhook)
	assert.Equal(t, "telegram", cfg.NotifyChannel)
	assert.False(t, cfg.Verbose)
	assert.True(t, cfg.EnableLearnings)
	assert.True(t, cfg.CrossValidate)
}

func TestBindFlags_AIProvider(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"default", []string{}, "claude"},
		{"claude", []string{"--ai", "claude"}, "claude"},
		{"codex", []string{"--ai", "codex"}, "codex"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.NewDefaultConfig()
			cmd := &cobra.Command{Use: "test"}
			BindFlags(cmd, cfg)

			err := cmd.ParseFlags(tt.args)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, cfg.AIProvider)
		})
	}
}

func TestValidateFlags_InvalidAI(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cmd := &cobra.Command{Use: "test"}
	BindFlags(cmd, cfg)

	// Parse flags with an invalid AI provider
	err := cmd.ParseFlags([]string{"--ai", "invalid"})
	require.NoError(t, err)

	// Validation should fail
	err = ValidateFlags(cmd, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be 'claude' or 'codex'")
}

func TestBindFlags_VerboseFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{"not set", []string{}, false},
		{"long form", []string{"--verbose"}, true},
		{"short form", []string{"-v"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.NewDefaultConfig()
			cmd := &cobra.Command{Use: "test"}
			BindFlags(cmd, cfg)

			err := cmd.ParseFlags(tt.args)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, cfg.Verbose)
		})
	}
}

func TestBindFlags_IntFlags(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		value    string
		check    func(*config.Config) int
		expected int
	}{
		{"max-iterations", "--max-iterations", "30", func(c *config.Config) int { return c.MaxIterations }, 30},
		{"max-inadmissible", "--max-inadmissible", "10", func(c *config.Config) int { return c.MaxInadmissible }, 10},
		{"max-claude-retry", "--max-claude-retry", "15", func(c *config.Config) int { return c.MaxClaudeRetry }, 15},
		{"max-turns", "--max-turns", "200", func(c *config.Config) int { return c.MaxTurns }, 200},
		{"inactivity-timeout", "--inactivity-timeout", "3600", func(c *config.Config) int { return c.InactivityTimeout }, 3600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.NewDefaultConfig()
			cmd := &cobra.Command{Use: "test"}
			BindFlags(cmd, cfg)

			err := cmd.ParseFlags([]string{tt.flag, tt.value})
			require.NoError(t, err)

			assert.Equal(t, tt.expected, tt.check(cfg))
		})
	}
}

func TestBindFlags_StringFlags(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		value    string
		check    func(*config.Config) string
		expected string
	}{
		{"implementation-model", "--implementation-model", "sonnet", func(c *config.Config) string { return c.ImplModel }, "sonnet"},
		{"validation-model", "--validation-model", "haiku", func(c *config.Config) string { return c.ValModel }, "haiku"},
		{"cross-model", "--cross-model", "default", func(c *config.Config) string { return c.CrossModel }, "default"},
		{"cross-validation-ai", "--cross-validation-ai", "codex", func(c *config.Config) string { return c.CrossAI }, "codex"},
		{"final-plan-validation-ai", "--final-plan-validation-ai", "claude", func(c *config.Config) string { return c.FinalPlanAI }, "claude"},
		{"final-plan-validation-model", "--final-plan-validation-model", "opus", func(c *config.Config) string { return c.FinalPlanModel }, "opus"},
		{"tasks-validation-ai", "--tasks-validation-ai", "codex", func(c *config.Config) string { return c.TasksValAI }, "codex"},
		{"tasks-validation-model", "--tasks-validation-model", "default", func(c *config.Config) string { return c.TasksValModel }, "default"},
		{"tasks-file", "--tasks-file", "custom-tasks.md", func(c *config.Config) string { return c.TasksFile }, "custom-tasks.md"},
		{"learnings-file", "--learnings-file", "custom-learnings.md", func(c *config.Config) string { return c.LearningsFile }, "custom-learnings.md"},
		{"notify-webhook", "--notify-webhook", "http://example.com", func(c *config.Config) string { return c.NotifyWebhook }, "http://example.com"},
		{"notify-channel", "--notify-channel", "slack", func(c *config.Config) string { return c.NotifyChannel }, "slack"},
		{"notify-chat-id", "--notify-chat-id", "12345", func(c *config.Config) string { return c.NotifyChatID }, "12345"},
		{"start-at", "--start-at", "14:30", func(c *config.Config) string { return c.StartAt }, "14:30"},
		{"at alias", "--at", "15:00", func(c *config.Config) string { return c.StartAt }, "15:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.NewDefaultConfig()
			cmd := &cobra.Command{Use: "test"}
			BindFlags(cmd, cfg)

			err := cmd.ParseFlags([]string{tt.flag, tt.value})
			require.NoError(t, err)

			assert.Equal(t, tt.expected, tt.check(cfg))
		})
	}
}

func TestBindFlags_BoolFlags(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		check    func(*config.Config) bool
		expected bool
	}{
		{"resume", "--resume", func(c *config.Config) bool { return c.Resume }, true},
		{"resume-force", "--resume-force", func(c *config.Config) bool { return c.ResumeForce }, true},
		{"clean", "--clean", func(c *config.Config) bool { return c.Clean }, true},
		{"status", "--status", func(c *config.Config) bool { return c.Status }, true},
		{"cancel", "--cancel", func(c *config.Config) bool { return c.Cancel }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.NewDefaultConfig()
			cmd := &cobra.Command{Use: "test"}
			BindFlags(cmd, cfg)

			err := cmd.ParseFlags([]string{tt.flag})
			require.NoError(t, err)

			assert.Equal(t, tt.expected, tt.check(cfg))
		})
	}
}

func TestValidateFlags_MutualExclusion(t *testing.T) {
	// Create temporary files for testing
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "plan.md")
	err := os.WriteFile(planFile, []byte("test plan"), 0644)
	require.NoError(t, err)

	cfg := config.NewDefaultConfig()
	cmd := &cobra.Command{Use: "test"}
	BindFlags(cmd, cfg)

	// Parse flags that set both mutually exclusive options
	err = cmd.ParseFlags([]string{"--original-plan-file", planFile, "--github-issue", "123"})
	require.NoError(t, err)

	// Validation should fail
	err = ValidateFlags(cmd, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestValidateFlags_OriginalPlanFileMustExist(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cmd := &cobra.Command{Use: "test"}
	BindFlags(cmd, cfg)

	// Parse flags with a nonexistent file
	err := cmd.ParseFlags([]string{"--original-plan-file", "/nonexistent/plan.md"})
	require.NoError(t, err)

	// Validation should fail
	err = ValidateFlags(cmd, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--original-plan-file")
}

func TestValidateFlags_ConfigFileMustExist(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cmd := &cobra.Command{Use: "test"}
	BindFlags(cmd, cfg)

	// Parse flags with a nonexistent config file
	err := cmd.ParseFlags([]string{"--config", "/nonexistent/config"})
	require.NoError(t, err)

	// Validation should fail
	err = ValidateFlags(cmd, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--config")
}

func TestValidateFlags_ResumeForceImpliesResume(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cmd := &cobra.Command{Use: "test"}
	BindFlags(cmd, cfg)

	// Parse flags with resume-force
	err := cmd.ParseFlags([]string{"--resume-force"})
	require.NoError(t, err)

	assert.False(t, cfg.Resume, "Resume should be false before validation")

	// Validation should set Resume to true
	err = ValidateFlags(cmd, cfg)
	require.NoError(t, err)
	assert.True(t, cfg.Resume, "--resume-force should imply --resume")
}

func TestValidateFlags_NoLearnings(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cmd := &cobra.Command{Use: "test"}
	BindFlags(cmd, cfg)

	err := cmd.ParseFlags([]string{"--no-learnings"})
	require.NoError(t, err)

	assert.True(t, cfg.EnableLearnings, "EnableLearnings should still be true before validation")

	err = ValidateFlags(cmd, cfg)
	require.NoError(t, err)
	assert.False(t, cfg.EnableLearnings, "--no-learnings should disable learnings")
}

func TestValidateFlags_NoCrossValidate(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cmd := &cobra.Command{Use: "test"}
	BindFlags(cmd, cfg)

	err := cmd.ParseFlags([]string{"--no-cross-validate"})
	require.NoError(t, err)

	assert.True(t, cfg.CrossValidate, "CrossValidate should still be true before validation")

	err = ValidateFlags(cmd, cfg)
	require.NoError(t, err)
	assert.False(t, cfg.CrossValidate, "--no-cross-validate should disable cross-validation")
}
