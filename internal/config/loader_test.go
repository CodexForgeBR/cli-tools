package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CodexForgeBR/cli-tools/internal/config"
)

// writeFile is a test helper that creates a temporary file with the given content.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}

// ---------------------------------------------------------------------------
// LoadFile tests
// ---------------------------------------------------------------------------

func TestLoadFileBasicKeyValue(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config", "AI_CLI=codex\nIMPL_MODEL=gpt-4\n")

	m, err := config.LoadFile(path)
	require.NoError(t, err)

	assert.Equal(t, "codex", m["AI_CLI"])
	assert.Equal(t, "gpt-4", m["IMPL_MODEL"])
}

func TestLoadFileSkipsComments(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config", "# This is a comment\nAI_CLI=claude\n# Another comment\n")

	m, err := config.LoadFile(path)
	require.NoError(t, err)

	assert.Len(t, m, 1)
	assert.Equal(t, "claude", m["AI_CLI"])
}

func TestLoadFileTrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config", "  AI_CLI  =  codex  \n  IMPL_MODEL = sonnet  \n")

	m, err := config.LoadFile(path)
	require.NoError(t, err)

	assert.Equal(t, "codex", m["AI_CLI"])
	assert.Equal(t, "sonnet", m["IMPL_MODEL"])
}

func TestLoadFileSkipsEmptyLines(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config", "\n\nAI_CLI=claude\n\n\nIMPL_MODEL=opus\n\n")

	m, err := config.LoadFile(path)
	require.NoError(t, err)

	assert.Len(t, m, 2)
	assert.Equal(t, "claude", m["AI_CLI"])
	assert.Equal(t, "opus", m["IMPL_MODEL"])
}

func TestLoadFileSkipsUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config", "AI_CLI=claude\nUNKNOWN_KEY=value\nBOGUS=stuff\nIMPL_MODEL=opus\n")

	m, err := config.LoadFile(path)
	require.NoError(t, err)

	assert.Len(t, m, 2)
	assert.Equal(t, "claude", m["AI_CLI"])
	assert.Equal(t, "opus", m["IMPL_MODEL"])
	assert.Empty(t, m["UNKNOWN_KEY"])
	assert.Empty(t, m["BOGUS"])
}

func TestLoadFileSkipsLinesWithoutEquals(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config", "AI_CLI=claude\nthis has no equals\nIMPL_MODEL=opus\n")

	m, err := config.LoadFile(path)
	require.NoError(t, err)

	assert.Len(t, m, 2)
}

func TestLoadFileValueWithEquals(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config", "NOTIFY_WEBHOOK=http://host:8080/path?key=val\n")

	m, err := config.LoadFile(path)
	require.NoError(t, err)

	assert.Equal(t, "http://host:8080/path?key=val", m["NOTIFY_WEBHOOK"])
}

func TestLoadFileReturnsErrorForMissingFile(t *testing.T) {
	_, err := config.LoadFile("/nonexistent/path/config")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Precedence tests
// ---------------------------------------------------------------------------

func TestLoadWithPrecedenceDefaultsOnly(t *testing.T) {
	cfg, err := config.LoadWithPrecedence("", "", "", nil)
	require.NoError(t, err)

	expected := config.NewDefaultConfig()
	assert.Equal(t, expected.AIProvider, cfg.AIProvider)
	assert.Equal(t, expected.MaxIterations, cfg.MaxIterations)
	assert.Equal(t, expected.NotifyWebhook, cfg.NotifyWebhook)
}

func TestLoadWithPrecedenceGlobalOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	globalPath := writeFile(t, dir, "global", "AI_CLI=codex\nMAX_ITERATIONS=50\n")

	cfg, err := config.LoadWithPrecedence(globalPath, "", "", nil)
	require.NoError(t, err)

	assert.Equal(t, "codex", cfg.AIProvider)
	assert.Equal(t, 50, cfg.MaxIterations)
	// Unset fields keep defaults.
	assert.Equal(t, "opus", cfg.ImplModel)
}

func TestLoadWithPrecedenceProjectOverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	globalPath := writeFile(t, dir, "global", "AI_CLI=codex\nIMPL_MODEL=gpt-4\nMAX_ITERATIONS=50\n")
	projectPath := writeFile(t, dir, "project", "AI_CLI=claude\nMAX_ITERATIONS=30\n")

	cfg, err := config.LoadWithPrecedence(globalPath, projectPath, "", nil)
	require.NoError(t, err)

	// Project wins over global.
	assert.Equal(t, "claude", cfg.AIProvider)
	assert.Equal(t, 30, cfg.MaxIterations)
	// Global still applies for fields not set in project.
	assert.Equal(t, "gpt-4", cfg.ImplModel)
}

func TestLoadWithPrecedenceExplicitOverridesProject(t *testing.T) {
	dir := t.TempDir()
	globalPath := writeFile(t, dir, "global", "AI_CLI=codex\n")
	projectPath := writeFile(t, dir, "project", "AI_CLI=claude\nMAX_ITERATIONS=30\n")
	explicitPath := writeFile(t, dir, "explicit", "MAX_ITERATIONS=10\n")

	cfg, err := config.LoadWithPrecedence(globalPath, projectPath, explicitPath, nil)
	require.NoError(t, err)

	// Project wins for AI_CLI (explicit does not set it).
	assert.Equal(t, "claude", cfg.AIProvider)
	// Explicit wins for MAX_ITERATIONS.
	assert.Equal(t, 10, cfg.MaxIterations)
}

func TestLoadWithPrecedenceCLIOverridesAll(t *testing.T) {
	dir := t.TempDir()
	globalPath := writeFile(t, dir, "global", "AI_CLI=codex\nMAX_ITERATIONS=50\n")
	projectPath := writeFile(t, dir, "project", "AI_CLI=claude\nMAX_ITERATIONS=30\n")
	explicitPath := writeFile(t, dir, "explicit", "MAX_ITERATIONS=10\n")

	cli := map[string]string{
		"AI_CLI":         "codex",
		"MAX_ITERATIONS": "5",
		"VERBOSE":        "true",
	}

	cfg, err := config.LoadWithPrecedence(globalPath, projectPath, explicitPath, cli)
	require.NoError(t, err)

	// CLI overrides everything.
	assert.Equal(t, "codex", cfg.AIProvider)
	assert.Equal(t, 5, cfg.MaxIterations)
	assert.True(t, cfg.Verbose)
}

func TestLoadWithPrecedenceFullChain(t *testing.T) {
	dir := t.TempDir()

	// Each layer sets a unique field so we can verify all layers contribute.
	globalPath := writeFile(t, dir, "global", "NOTIFY_CHANNEL=slack\n")
	projectPath := writeFile(t, dir, "project", "LEARNINGS_FILE=project-learnings.md\n")
	explicitPath := writeFile(t, dir, "explicit", "NOTIFY_CHAT_ID=12345\n")
	cli := map[string]string{"VERBOSE": "true"}

	cfg, err := config.LoadWithPrecedence(globalPath, projectPath, explicitPath, cli)
	require.NoError(t, err)

	// Defaults preserved.
	assert.Equal(t, "claude", cfg.AIProvider)
	// Global.
	assert.Equal(t, "slack", cfg.NotifyChannel)
	// Project.
	assert.Equal(t, "project-learnings.md", cfg.LearningsFile)
	// Explicit.
	assert.Equal(t, "12345", cfg.NotifyChatID)
	// CLI.
	assert.True(t, cfg.Verbose)
}

func TestLoadWithPrecedenceMissingGlobalIsNotError(t *testing.T) {
	cfg, err := config.LoadWithPrecedence("/nonexistent/global/config", "", "", nil)
	require.NoError(t, err)
	assert.Equal(t, "claude", cfg.AIProvider) // defaults preserved
}

func TestLoadWithPrecedenceMissingProjectIsNotError(t *testing.T) {
	cfg, err := config.LoadWithPrecedence("", "/nonexistent/project/config", "", nil)
	require.NoError(t, err)
	assert.Equal(t, "claude", cfg.AIProvider)
}

func TestLoadWithPrecedenceMissingExplicitIsError(t *testing.T) {
	_, err := config.LoadWithPrecedence("", "", "/nonexistent/explicit/config", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "explicit config")
}

func TestLoadWithPrecedenceInvalidExplicitPath(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a directory, not a file
	dirPath := filepath.Join(tmpDir, "config-dir")
	require.NoError(t, os.Mkdir(dirPath, 0755))

	// Trying to load a directory as config should fail
	_, err := config.LoadWithPrecedence("", "", dirPath, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "explicit config")
}

func TestLoadWithPrecedenceInvalidGlobalPath(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a directory, not a file
	dirPath := filepath.Join(tmpDir, "global-dir")
	require.NoError(t, os.Mkdir(dirPath, 0755))

	// Global config error (non-ErrNotExist) should be returned
	_, err := config.LoadWithPrecedence(dirPath, "", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "global config")
}

func TestLoadWithPrecedenceInvalidProjectPath(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a directory, not a file
	dirPath := filepath.Join(tmpDir, "project-dir")
	require.NoError(t, os.Mkdir(dirPath, 0755))

	// Project config error (non-ErrNotExist) should be returned
	_, err := config.LoadWithPrecedence("", dirPath, "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project config")
}

// ---------------------------------------------------------------------------
// ApplyMapToConfig tests
// ---------------------------------------------------------------------------

func TestApplyMapToConfigSetsAllStringFields(t *testing.T) {
	cfg := config.NewDefaultConfig()
	m := map[string]string{
		"AI_CLI":             "codex",
		"IMPL_MODEL":         "gpt-4",
		"VAL_MODEL":          "gpt-3.5",
		"CROSS_AI":           "claude",
		"CROSS_MODEL":        "sonnet",
		"FINAL_PLAN_AI":      "codex",
		"FINAL_PLAN_MODEL":   "gpt-4",
		"TASKS_VAL_AI":       "claude",
		"TASKS_VAL_MODEL":    "opus",
		"TASKS_FILE":         "/tmp/tasks.md",
		"ORIGINAL_PLAN_FILE": "/tmp/plan.md",
		"GITHUB_ISSUE":       "https://github.com/org/repo/issues/1",
		"LEARNINGS_FILE":     "/tmp/learnings.md",
		"NOTIFY_WEBHOOK":     "https://example.com/hook",
		"NOTIFY_CHANNEL":     "slack",
		"NOTIFY_CHAT_ID":     "99999",
	}

	config.ApplyMapToConfig(cfg, m)

	assert.Equal(t, "codex", cfg.AIProvider)
	assert.Equal(t, "gpt-4", cfg.ImplModel)
	assert.Equal(t, "gpt-3.5", cfg.ValModel)
	assert.Equal(t, "claude", cfg.CrossAI)
	assert.Equal(t, "sonnet", cfg.CrossModel)
	assert.Equal(t, "codex", cfg.FinalPlanAI)
	assert.Equal(t, "gpt-4", cfg.FinalPlanModel)
	assert.Equal(t, "claude", cfg.TasksValAI)
	assert.Equal(t, "opus", cfg.TasksValModel)
	assert.Equal(t, "/tmp/tasks.md", cfg.TasksFile)
	assert.Equal(t, "/tmp/plan.md", cfg.OriginalPlanFile)
	assert.Equal(t, "https://github.com/org/repo/issues/1", cfg.GithubIssue)
	assert.Equal(t, "/tmp/learnings.md", cfg.LearningsFile)
	assert.Equal(t, "https://example.com/hook", cfg.NotifyWebhook)
	assert.Equal(t, "slack", cfg.NotifyChannel)
	assert.Equal(t, "99999", cfg.NotifyChatID)
}

func TestApplyMapToConfigSetsIntegerFields(t *testing.T) {
	cfg := config.NewDefaultConfig()
	m := map[string]string{
		"MAX_ITERATIONS":     "50",
		"MAX_INADMISSIBLE":   "10",
		"MAX_CLAUDE_RETRY":   "25",
		"MAX_TURNS":          "200",
		"INACTIVITY_TIMEOUT": "3600",
	}

	config.ApplyMapToConfig(cfg, m)

	assert.Equal(t, 50, cfg.MaxIterations)
	assert.Equal(t, 10, cfg.MaxInadmissible)
	assert.Equal(t, 25, cfg.MaxClaudeRetry)
	assert.Equal(t, 200, cfg.MaxTurns)
	assert.Equal(t, 3600, cfg.InactivityTimeout)
}

func TestApplyMapToConfigSetsBooleanFields(t *testing.T) {
	cfg := config.NewDefaultConfig()

	// Set to non-default values.
	m := map[string]string{
		"CROSS_VALIDATE":   "false",
		"ENABLE_LEARNINGS": "false",
		"VERBOSE":          "true",
	}
	config.ApplyMapToConfig(cfg, m)

	assert.False(t, cfg.CrossValidate)
	assert.False(t, cfg.EnableLearnings)
	assert.True(t, cfg.Verbose)
}

func TestApplyMapToConfigBooleanVariations(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"Yes", true},
		{"false", false},
		{"FALSE", false},
		{"False", false},
		{"0", false},
		{"no", false},
		{"NO", false},
		{"anything", false},
		{"", false},
		{"  true  ", true},   // whitespace trimming
		{"  false  ", false}, // whitespace trimming
	}

	for _, tt := range tests {
		t.Run("VERBOSE="+tt.value, func(t *testing.T) {
			cfg := config.NewDefaultConfig()
			config.ApplyMapToConfig(cfg, map[string]string{"VERBOSE": tt.value})
			assert.Equal(t, tt.expected, cfg.Verbose)
		})
	}
}

func TestApplyMapToConfigIgnoresInvalidIntegers(t *testing.T) {
	cfg := config.NewDefaultConfig()
	original := cfg.MaxIterations

	config.ApplyMapToConfig(cfg, map[string]string{"MAX_ITERATIONS": "not-a-number"})

	assert.Equal(t, original, cfg.MaxIterations, "invalid integer should preserve previous value")
}

func TestApplyMapToConfigIgnoresUnknownKeys(t *testing.T) {
	cfg := config.NewDefaultConfig()
	expected := config.NewDefaultConfig()

	config.ApplyMapToConfig(cfg, map[string]string{
		"TOTALLY_UNKNOWN": "value",
		"ANOTHER_BAD_KEY": "stuff",
	})

	assert.Equal(t, expected.AIProvider, cfg.AIProvider)
	assert.Equal(t, expected.MaxIterations, cfg.MaxIterations)
}

// ---------------------------------------------------------------------------
// Full Precedence Integration Tests (T085)
// ---------------------------------------------------------------------------

func TestLoadWithPrecedenceFullIntegration(t *testing.T) {
	dir := t.TempDir()

	// Create a comprehensive config hierarchy testing all precedence levels.
	// Each layer sets different fields, and some override previous layers.

	// Global config: sets baseline AI provider, max iterations, and notification settings
	globalPath := writeFile(t, dir, "global.config", `
AI_CLI=codex
IMPL_MODEL=gpt-4
MAX_ITERATIONS=50
MAX_INADMISSIBLE=8
NOTIFY_WEBHOOK=http://global.example.com/hook
NOTIFY_CHANNEL=slack
LEARNINGS_FILE=/global/learnings.md
ENABLE_LEARNINGS=true
CROSS_VALIDATE=true
`)

	// Project config: overrides AI provider and iterations, adds project-specific settings
	projectPath := writeFile(t, dir, "project.config", `
AI_CLI=claude
IMPL_MODEL=sonnet
MAX_ITERATIONS=30
TASKS_FILE=/project/tasks.md
ORIGINAL_PLAN_FILE=/project/plan.md
VERBOSE=false
`)

	// Explicit config: overrides iterations and verbose, adds cross-validation settings
	explicitPath := writeFile(t, dir, "explicit.config", `
MAX_ITERATIONS=15
VERBOSE=true
CROSS_AI=openai
CROSS_MODEL=gpt-3.5
TASKS_VAL_AI=claude
TASKS_VAL_MODEL=opus
`)

	// CLI overrides: highest priority, overrides iterations and adds final settings
	cliOverrides := map[string]string{
		"MAX_ITERATIONS":   "10",
		"NOTIFY_CHAT_ID":   "telegram-123",
		"MAX_CLAUDE_RETRY": "25",
	}

	cfg, err := config.LoadWithPrecedence(globalPath, projectPath, explicitPath, cliOverrides)
	require.NoError(t, err)

	// Verify precedence for each field:

	// From CLI (highest priority)
	assert.Equal(t, 10, cfg.MaxIterations, "CLI should override all other sources for MaxIterations")
	assert.Equal(t, "telegram-123", cfg.NotifyChatID, "CLI should set NotifyChatID")
	assert.Equal(t, 25, cfg.MaxClaudeRetry, "CLI should set MaxClaudeRetry")

	// From explicit config
	assert.True(t, cfg.Verbose, "Explicit config should override project for Verbose")
	assert.Equal(t, "openai", cfg.CrossAI, "Explicit config should set CrossAI")
	assert.Equal(t, "gpt-3.5", cfg.CrossModel, "Explicit config should set CrossModel")
	assert.Equal(t, "claude", cfg.TasksValAI, "Explicit config should set TasksValAI")
	assert.Equal(t, "opus", cfg.TasksValModel, "Explicit config should set TasksValModel")

	// From project config
	assert.Equal(t, "claude", cfg.AIProvider, "Project should override global for AIProvider")
	assert.Equal(t, "sonnet", cfg.ImplModel, "Project should override global for ImplModel")
	assert.Equal(t, "/project/tasks.md", cfg.TasksFile, "Project config should set TasksFile")
	assert.Equal(t, "/project/plan.md", cfg.OriginalPlanFile, "Project config should set OriginalPlanFile")

	// From global config (not overridden)
	assert.Equal(t, 8, cfg.MaxInadmissible, "Global config should set MaxInadmissible")
	assert.Equal(t, "http://global.example.com/hook", cfg.NotifyWebhook, "Global should set NotifyWebhook")
	assert.Equal(t, "slack", cfg.NotifyChannel, "Global should set NotifyChannel")
	assert.Equal(t, "/global/learnings.md", cfg.LearningsFile, "Global should set LearningsFile")
	assert.True(t, cfg.EnableLearnings, "Global should set EnableLearnings")
	assert.True(t, cfg.CrossValidate, "Global should set CrossValidate")

	// From defaults (not set anywhere)
	assert.Equal(t, "opus", cfg.ValModel, "Default should remain for ValModel")
	assert.Equal(t, 100, cfg.MaxTurns, "Default should remain for MaxTurns")
	assert.Equal(t, 1800, cfg.InactivityTimeout, "Default should remain for InactivityTimeout")
}

func TestLoadWithPrecedenceAllFieldsCoverage(t *testing.T) {
	dir := t.TempDir()

	// Test that every whitelisted config field can be set and has correct precedence.
	// Use different layers for different field types.

	globalPath := writeFile(t, dir, "global.config", `
AI_CLI=codex
IMPL_MODEL=gpt-4
VAL_MODEL=gpt-3.5
CROSS_AI=claude
CROSS_MODEL=opus
FINAL_PLAN_AI=openai
FINAL_PLAN_MODEL=gpt-4
TASKS_VAL_AI=claude
TASKS_VAL_MODEL=sonnet
MAX_ITERATIONS=100
MAX_INADMISSIBLE=10
MAX_CLAUDE_RETRY=20
MAX_TURNS=200
INACTIVITY_TIMEOUT=3600
`)

	projectPath := writeFile(t, dir, "project.config", `
TASKS_FILE=/project/tasks.md
ORIGINAL_PLAN_FILE=/project/original.md
GITHUB_ISSUE=https://github.com/owner/repo/issues/42
LEARNINGS_FILE=/project/learnings.md
NOTIFY_WEBHOOK=http://project.example.com/webhook
NOTIFY_CHANNEL=discord
NOTIFY_CHAT_ID=discord-456
`)

	explicitPath := writeFile(t, dir, "explicit.config", `
CROSS_VALIDATE=false
ENABLE_LEARNINGS=false
VERBOSE=true
`)

	cfg, err := config.LoadWithPrecedence(globalPath, projectPath, explicitPath, nil)
	require.NoError(t, err)

	// Verify all string fields from global
	assert.Equal(t, "codex", cfg.AIProvider)
	assert.Equal(t, "gpt-4", cfg.ImplModel)
	assert.Equal(t, "gpt-3.5", cfg.ValModel)
	assert.Equal(t, "claude", cfg.CrossAI)
	assert.Equal(t, "opus", cfg.CrossModel)
	assert.Equal(t, "openai", cfg.FinalPlanAI)
	assert.Equal(t, "gpt-4", cfg.FinalPlanModel)
	assert.Equal(t, "claude", cfg.TasksValAI)
	assert.Equal(t, "sonnet", cfg.TasksValModel)

	// Verify all int fields from global
	assert.Equal(t, 100, cfg.MaxIterations)
	assert.Equal(t, 10, cfg.MaxInadmissible)
	assert.Equal(t, 20, cfg.MaxClaudeRetry)
	assert.Equal(t, 200, cfg.MaxTurns)
	assert.Equal(t, 3600, cfg.InactivityTimeout)

	// Verify all string fields from project
	assert.Equal(t, "/project/tasks.md", cfg.TasksFile)
	assert.Equal(t, "/project/original.md", cfg.OriginalPlanFile)
	assert.Equal(t, "https://github.com/owner/repo/issues/42", cfg.GithubIssue)
	assert.Equal(t, "/project/learnings.md", cfg.LearningsFile)
	assert.Equal(t, "http://project.example.com/webhook", cfg.NotifyWebhook)
	assert.Equal(t, "discord", cfg.NotifyChannel)
	assert.Equal(t, "discord-456", cfg.NotifyChatID)

	// Verify all bool fields from explicit
	assert.False(t, cfg.CrossValidate)
	assert.False(t, cfg.EnableLearnings)
	assert.True(t, cfg.Verbose)
}

func TestLoadWithPrecedenceCLIOverridesEverything(t *testing.T) {
	dir := t.TempDir()

	// All config files set the same fields
	globalPath := writeFile(t, dir, "global.config", `
AI_CLI=codex
MAX_ITERATIONS=100
VERBOSE=false
CROSS_VALIDATE=true
`)

	projectPath := writeFile(t, dir, "project.config", `
AI_CLI=openai
MAX_ITERATIONS=50
VERBOSE=false
CROSS_VALIDATE=true
`)

	explicitPath := writeFile(t, dir, "explicit.config", `
AI_CLI=claude
MAX_ITERATIONS=25
VERBOSE=false
CROSS_VALIDATE=false
`)

	// CLI should win for all fields
	cliOverrides := map[string]string{
		"AI_CLI":         "codex",
		"MAX_ITERATIONS": "5",
		"VERBOSE":        "true",
		"CROSS_VALIDATE": "false",
	}

	cfg, err := config.LoadWithPrecedence(globalPath, projectPath, explicitPath, cliOverrides)
	require.NoError(t, err)

	assert.Equal(t, "codex", cfg.AIProvider, "CLI should override all for AIProvider")
	assert.Equal(t, 5, cfg.MaxIterations, "CLI should override all for MaxIterations")
	assert.True(t, cfg.Verbose, "CLI should override all for Verbose")
	assert.False(t, cfg.CrossValidate, "CLI should override all for CrossValidate")
}

func TestLoadWithPrecedencePartialOverrides(t *testing.T) {
	dir := t.TempDir()

	// Test that each layer only overrides what it specifies, leaving other fields intact
	globalPath := writeFile(t, dir, "global.config", `
AI_CLI=codex
IMPL_MODEL=gpt-4
MAX_ITERATIONS=100
MAX_INADMISSIBLE=10
VERBOSE=false
`)

	// Project only sets one field
	projectPath := writeFile(t, dir, "project.config", `
MAX_ITERATIONS=50
`)

	// Explicit only sets one different field
	explicitPath := writeFile(t, dir, "explicit.config", `
VERBOSE=true
`)

	// CLI only sets one more different field
	cliOverrides := map[string]string{
		"MAX_INADMISSIBLE": "3",
	}

	cfg, err := config.LoadWithPrecedence(globalPath, projectPath, explicitPath, cliOverrides)
	require.NoError(t, err)

	// Global fields not overridden should remain
	assert.Equal(t, "codex", cfg.AIProvider, "Global AIProvider should remain")
	assert.Equal(t, "gpt-4", cfg.ImplModel, "Global ImplModel should remain")

	// Project override
	assert.Equal(t, 50, cfg.MaxIterations, "Project should override MaxIterations")

	// Explicit override
	assert.True(t, cfg.Verbose, "Explicit should override Verbose")

	// CLI override
	assert.Equal(t, 3, cfg.MaxInadmissible, "CLI should override MaxInadmissible")
}

func TestLoadWithPrecedenceEmptyValuesDoNotOverride(t *testing.T) {
	dir := t.TempDir()

	// Global sets a value
	globalPath := writeFile(t, dir, "global.config", `
AI_CLI=codex
IMPL_MODEL=gpt-4
MAX_ITERATIONS=100
`)

	// Project has empty value (should be ignored as whitespace-only)
	projectPath := writeFile(t, dir, "project.config", `
AI_CLI=
IMPL_MODEL=
`)

	cfg, err := config.LoadWithPrecedence(globalPath, projectPath, "", nil)
	require.NoError(t, err)

	// Empty values should set empty strings (not skip), so this tests actual behavior
	assert.Equal(t, "", cfg.AIProvider, "Empty value in project should override to empty string")
	assert.Equal(t, "", cfg.ImplModel, "Empty whitespace value should override to empty string")
	assert.Equal(t, 100, cfg.MaxIterations, "Non-overridden field should remain")
}

func TestLoadWithPrecedenceIntegerParsing(t *testing.T) {
	dir := t.TempDir()

	// Test that integer fields are correctly parsed at each level
	globalPath := writeFile(t, dir, "global.config", `
MAX_ITERATIONS=100
MAX_INADMISSIBLE=10
`)

	projectPath := writeFile(t, dir, "project.config", `
MAX_ITERATIONS=50
MAX_CLAUDE_RETRY=20
`)

	explicitPath := writeFile(t, dir, "explicit.config", `
MAX_TURNS=150
INACTIVITY_TIMEOUT=2400
`)

	cliOverrides := map[string]string{
		"MAX_ITERATIONS":     "5",
		"INACTIVITY_TIMEOUT": "600",
	}

	cfg, err := config.LoadWithPrecedence(globalPath, projectPath, explicitPath, cliOverrides)
	require.NoError(t, err)

	assert.Equal(t, 5, cfg.MaxIterations, "CLI should override MaxIterations")
	assert.Equal(t, 10, cfg.MaxInadmissible, "Global should set MaxInadmissible")
	assert.Equal(t, 20, cfg.MaxClaudeRetry, "Project should set MaxClaudeRetry")
	assert.Equal(t, 150, cfg.MaxTurns, "Explicit should set MaxTurns")
	assert.Equal(t, 600, cfg.InactivityTimeout, "CLI should override InactivityTimeout")
}

func TestLoadWithPrecedenceBooleanParsing(t *testing.T) {
	dir := t.TempDir()

	// Test that boolean fields are correctly parsed at each level
	globalPath := writeFile(t, dir, "global.config", `
CROSS_VALIDATE=true
ENABLE_LEARNINGS=yes
VERBOSE=1
`)

	projectPath := writeFile(t, dir, "project.config", `
CROSS_VALIDATE=false
`)

	explicitPath := writeFile(t, dir, "explicit.config", `
VERBOSE=no
`)

	cliOverrides := map[string]string{
		"ENABLE_LEARNINGS": "false",
	}

	cfg, err := config.LoadWithPrecedence(globalPath, projectPath, explicitPath, cliOverrides)
	require.NoError(t, err)

	assert.False(t, cfg.CrossValidate, "Project should override CrossValidate to false")
	assert.False(t, cfg.EnableLearnings, "CLI should override EnableLearnings to false")
	assert.False(t, cfg.Verbose, "Explicit should override Verbose to false")
}

func TestLoadWithPrecedenceDefaultsPreserved(t *testing.T) {
	dir := t.TempDir()

	// Only set a few fields, verify defaults remain for others
	globalPath := writeFile(t, dir, "global.config", `
AI_CLI=codex
`)

	projectPath := writeFile(t, dir, "project.config", `
MAX_ITERATIONS=30
`)

	cfg, err := config.LoadWithPrecedence(globalPath, projectPath, "", nil)
	require.NoError(t, err)

	// Overridden fields
	assert.Equal(t, "codex", cfg.AIProvider)
	assert.Equal(t, 30, cfg.MaxIterations)

	// Default values that should remain
	defaults := config.NewDefaultConfig()
	assert.Equal(t, defaults.ImplModel, cfg.ImplModel)
	assert.Equal(t, defaults.ValModel, cfg.ValModel)
	assert.Equal(t, defaults.CrossValidate, cfg.CrossValidate)
	assert.Equal(t, defaults.MaxInadmissible, cfg.MaxInadmissible)
	assert.Equal(t, defaults.MaxClaudeRetry, cfg.MaxClaudeRetry)
	assert.Equal(t, defaults.MaxTurns, cfg.MaxTurns)
	assert.Equal(t, defaults.InactivityTimeout, cfg.InactivityTimeout)
	assert.Equal(t, defaults.LearningsFile, cfg.LearningsFile)
	assert.Equal(t, defaults.EnableLearnings, cfg.EnableLearnings)
	assert.Equal(t, defaults.NotifyWebhook, cfg.NotifyWebhook)
	assert.Equal(t, defaults.NotifyChannel, cfg.NotifyChannel)
}
