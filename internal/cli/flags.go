// Package cli provides flag binding and validation for the ralph-loop CLI.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/CodexForgeBR/cli-tools/internal/config"
)

// BindFlags registers all 32 CLI flags on the given cobra command.
// The flags directly modify fields in the provided config pointer.
// Call ValidateFlags after parsing to check flag combinations.
func BindFlags(cmd *cobra.Command, cfg *config.Config) {
	flags := cmd.Flags()

	// AI Provider & Models
	flags.StringVar(&cfg.AIProvider, "ai", "claude", "AI CLI to use: claude or codex")
	flags.StringVar(&cfg.ImplModel, "implementation-model", "", "Model for implementation phase")
	flags.StringVar(&cfg.ValModel, "validation-model", "", "Model for validation phase")
	flags.StringVar(&cfg.CrossModel, "cross-model", "", "Model for cross-validation")
	flags.StringVar(&cfg.CrossAI, "cross-validation-ai", "", "AI CLI for cross-validation")
	flags.StringVar(&cfg.FinalPlanAI, "final-plan-validation-ai", "", "AI CLI for final plan validation")
	flags.StringVar(&cfg.FinalPlanModel, "final-plan-validation-model", "", "Model for final plan validation")
	flags.StringVar(&cfg.TasksValAI, "tasks-validation-ai", "", "AI CLI for tasks validation")
	flags.StringVar(&cfg.TasksValModel, "tasks-validation-model", "", "Model for tasks validation")

	// Iteration Limits
	flags.IntVar(&cfg.MaxIterations, "max-iterations", 20, "Maximum loop iterations")
	flags.IntVar(&cfg.MaxInadmissible, "max-inadmissible", 5, "Max inadmissible verdicts before exit 6")
	flags.IntVar(&cfg.MaxClaudeRetry, "max-claude-retry", 10, "Max retries per AI invocation")
	flags.IntVar(&cfg.MaxTurns, "max-turns", 100, "Max agent turns per AI invocation")
	flags.IntVar(&cfg.InactivityTimeout, "inactivity-timeout", 1800, "Seconds of inactivity before kill")

	// Input Files
	flags.StringVar(&cfg.TasksFile, "tasks-file", "", "Path to tasks.md")
	flags.StringVar(&cfg.OriginalPlanFile, "original-plan-file", "", "Path to original plan (mutually exclusive with --github-issue)")
	flags.StringVar(&cfg.GithubIssue, "github-issue", "", "GitHub issue URL or number")
	flags.StringVar(&cfg.LearningsFile, "learnings-file", ".ralph-loop/learnings.md", "Path to learnings file")
	flags.StringVar(&cfg.ConfigFile, "config", "", "Path to additional config file")

	// Feature Toggles
	flags.BoolVarP(&cfg.Verbose, "verbose", "v", false, "Pass verbose flag to AI CLI")

	// Negation flags need special handling via Changed detection
	var noLearnings, noCrossValidate bool
	flags.BoolVar(&noLearnings, "no-learnings", false, "Disable learnings persistence")
	flags.BoolVar(&noCrossValidate, "no-cross-validate", false, "Disable cross-validation phase")

	// Scheduling
	flags.StringVar(&cfg.StartAt, "start-at", "", "Schedule start time (ISO 8601, HH:MM, YYYY-MM-DD HH:MM)")
	// Alias --at for --start-at
	flags.StringVar(&cfg.StartAt, "at", "", "Alias for --start-at")

	// Notifications
	flags.StringVar(&cfg.NotifyWebhook, "notify-webhook", "http://127.0.0.1:18789/webhook", "OpenClaw webhook URL")
	flags.StringVar(&cfg.NotifyChannel, "notify-channel", "telegram", "Notification channel")
	flags.StringVar(&cfg.NotifyChatID, "notify-chat-id", "", "Recipient chat ID")

	// Session Management
	flags.BoolVar(&cfg.Resume, "resume", false, "Resume from last interrupted session")
	flags.BoolVar(&cfg.ResumeForce, "resume-force", false, "Resume even if tasks.md changed (implies --resume)")
	flags.BoolVar(&cfg.Clean, "clean", false, "Delete state directory and start fresh")
	flags.BoolVar(&cfg.Status, "status", false, "Show session status and exit")
	flags.BoolVar(&cfg.Cancel, "cancel", false, "Cancel active session and exit")
}

// ValidateFlags checks for invalid flag combinations after parsing.
// Must be called after cmd.Execute() or cmd.ParseFlags().
func ValidateFlags(cmd *cobra.Command, cfg *config.Config) error {
	// Mutual exclusion: --original-plan-file and --github-issue
	if cfg.OriginalPlanFile != "" && cfg.GithubIssue != "" {
		return fmt.Errorf("--original-plan-file and --github-issue are mutually exclusive")
	}

	// --original-plan-file must exist if provided
	if cfg.OriginalPlanFile != "" {
		if _, err := os.Stat(cfg.OriginalPlanFile); err != nil {
			return fmt.Errorf("--original-plan-file: %w", err)
		}
	}

	// --config must exist if provided
	if cfg.ConfigFile != "" {
		if _, err := os.Stat(cfg.ConfigFile); err != nil {
			return fmt.Errorf("--config: %w", err)
		}
	}

	// --resume-force implies --resume
	if cfg.ResumeForce {
		cfg.Resume = true
	}

	// Handle negation flags via Changed detection
	if cmd.Flags().Changed("no-learnings") {
		cfg.EnableLearnings = false
	}
	if cmd.Flags().Changed("no-cross-validate") {
		cfg.CrossValidate = false
	}

	// Validate AI provider value
	if cfg.AIProvider != "claude" && cfg.AIProvider != "codex" {
		return fmt.Errorf("--ai must be 'claude' or 'codex', got: %s", cfg.AIProvider)
	}

	return nil
}
