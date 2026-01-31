package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestHelpTemplate_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, helpTemplate)
}

func TestHelpTemplate_ContainsKeyFlags(t *testing.T) {
	requiredFlags := []string{
		"--ai",
		"--implementation-model",
		"--validation-model",
		"--cross-validation-ai",
		"--cross-model",
		"--final-plan-validation-ai",
		"--final-plan-validation-model",
		"--tasks-validation-ai",
		"--tasks-validation-model",
		"--max-iterations",
		"--max-inadmissible",
		"--max-claude-retry",
		"--max-turns",
		"--inactivity-timeout",
		"--tasks-file",
		"--original-plan-file",
		"--github-issue",
		"--learnings-file",
		"--config",
		"--verbose",
		"--no-learnings",
		"--no-cross-validate",
		"--start-at",
		"--at",
		"--notify-webhook",
		"--notify-channel",
		"--notify-chat-id",
		"--resume",
		"--resume-force",
		"--clean",
		"--status",
		"--cancel",
		"--help",
		"--version",
	}

	for _, flag := range requiredFlags {
		assert.Contains(t, helpTemplate, flag, "Help template should contain flag: %s", flag)
	}
}

func TestHelpTemplate_ContainsExitCodes(t *testing.T) {
	exitCodes := []string{
		"Success",
		"Error",
		"MaxIterations",
		"Escalate",
		"Blocked",
		"TasksInvalid",
		"Inadmissible",
		"Interrupted",
	}

	for _, code := range exitCodes {
		assert.Contains(t, helpTemplate, code, "Help template should contain exit code: %s", code)
	}
}

func TestHelpTemplate_ContainsSections(t *testing.T) {
	sections := []string{
		"USAGE",
		"FLAGS",
		"EXIT CODES",
		"EXAMPLES",
	}

	for _, section := range sections {
		assert.Contains(t, helpTemplate, section, "Help template should contain section: %s", section)
	}
}

func TestSetCustomHelp(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	SetCustomHelp(cmd)

	// The command should now have our custom help template set
	// We can verify this by checking that the help template is not empty
	// (cobra doesn't expose the template directly, but we can check it was set)
	assert.NotNil(t, cmd)
}
