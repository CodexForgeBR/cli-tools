package config_test

import (
	"testing"

	"github.com/CodexForgeBR/cli-tools/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultConfigValues(t *testing.T) {
	cfg := config.NewDefaultConfig()
	require.NotNil(t, cfg)

	// AI provider and model selection.
	assert.Equal(t, "claude", cfg.AIProvider)
	assert.Equal(t, "opus", cfg.ImplModel)
	assert.Equal(t, "opus", cfg.ValModel)

	// Cross-validation.
	assert.True(t, cfg.CrossValidate)
	assert.Empty(t, cfg.CrossAI)
	assert.Empty(t, cfg.CrossModel)

	// Final plan validation.
	assert.Empty(t, cfg.FinalPlanAI)
	assert.Empty(t, cfg.FinalPlanModel)

	// Tasks validation.
	assert.Empty(t, cfg.TasksValAI)
	assert.Empty(t, cfg.TasksValModel)

	// Iteration limits.
	assert.Equal(t, 20, cfg.MaxIterations)
	assert.Equal(t, 5, cfg.MaxInadmissible)
	assert.Equal(t, 10, cfg.MaxClaudeRetry)
	assert.Equal(t, 100, cfg.MaxTurns)

	// Timeouts.
	assert.Equal(t, 1800, cfg.InactivityTimeout)

	// File paths.
	assert.Empty(t, cfg.TasksFile)
	assert.Empty(t, cfg.OriginalPlanFile)
	assert.Empty(t, cfg.GithubIssue)
	assert.Equal(t, ".ralph-loop/learnings.md", cfg.LearningsFile)
	assert.True(t, cfg.EnableLearnings)

	// Runtime flags.
	assert.False(t, cfg.Verbose)

	// Notification settings.
	assert.Equal(t, "http://127.0.0.1:18789/webhook", cfg.NotifyWebhook)
	assert.Equal(t, "telegram", cfg.NotifyChannel)
	assert.Empty(t, cfg.NotifyChatID)

	// CLI-only flags default to zero values.
	assert.Empty(t, cfg.ConfigFile)
	assert.False(t, cfg.Resume)
	assert.False(t, cfg.ResumeForce)
	assert.False(t, cfg.Clean)
	assert.False(t, cfg.Status)
	assert.False(t, cfg.Cancel)
	assert.Empty(t, cfg.StartAt)
}

func TestWhitelistedVarsContains24Entries(t *testing.T) {
	assert.Len(t, config.WhitelistedVars, 24)
}

func TestWhitelistedVarsContainsAllExpectedNames(t *testing.T) {
	expected := []string{
		"AI_CLI",
		"IMPL_MODEL",
		"VAL_MODEL",
		"CROSS_VALIDATE",
		"CROSS_AI",
		"CROSS_MODEL",
		"FINAL_PLAN_AI",
		"FINAL_PLAN_MODEL",
		"TASKS_VAL_AI",
		"TASKS_VAL_MODEL",
		"MAX_ITERATIONS",
		"MAX_INADMISSIBLE",
		"MAX_CLAUDE_RETRY",
		"MAX_TURNS",
		"INACTIVITY_TIMEOUT",
		"TASKS_FILE",
		"ORIGINAL_PLAN_FILE",
		"GITHUB_ISSUE",
		"LEARNINGS_FILE",
		"ENABLE_LEARNINGS",
		"VERBOSE",
		"NOTIFY_WEBHOOK",
		"NOTIFY_CHANNEL",
		"NOTIFY_CHAT_ID",
	}

	// Convert array to slice for comparison.
	vars := config.WhitelistedVars[:]
	assert.ElementsMatch(t, expected, vars)
}

func TestWhitelistedVarsHasNoDuplicates(t *testing.T) {
	seen := make(map[string]bool)
	for _, v := range config.WhitelistedVars {
		assert.False(t, seen[v], "duplicate whitelisted var: %s", v)
		seen[v] = true
	}
}
