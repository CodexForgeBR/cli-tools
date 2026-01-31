package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/CodexForgeBR/cli-tools/internal/ai"
	"github.com/CodexForgeBR/cli-tools/internal/cli"
	"github.com/CodexForgeBR/cli-tools/internal/config"
	"github.com/CodexForgeBR/cli-tools/internal/logging"
	"github.com/CodexForgeBR/cli-tools/internal/model"
	"github.com/CodexForgeBR/cli-tools/internal/phases"
	sighandler "github.com/CodexForgeBR/cli-tools/internal/signal"
)

// version vars injected via ldflags at build time
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cfg := config.NewDefaultConfig()

	rootCmd := &cobra.Command{
		Use:     "ralph-loop",
		Short:   "Dual-model AI implementation-validation loop orchestrator",
		Long:    "Ralph Loop orchestrates AI-powered implementation and validation cycles for spec-driven development.",
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate flags after parsing
			if err := cli.ValidateFlags(cmd, cfg); err != nil {
				return err
			}
			return runOrchestrator(cmd, cfg)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Bind all CLI flags to the config
	cli.BindFlags(rootCmd, cfg)

	// Set custom help template
	cli.SetCustomHelp(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// buildCLIOverrides creates a map of CLI flag overrides from the config.
// Uses cmd.Flags().Changed() to only include flags explicitly set by the user,
// ensuring config file values are not accidentally overridden by default values.
func buildCLIOverrides(cmd *cobra.Command, cfg *config.Config) map[string]string {
	overrides := make(map[string]string)

	// String flags: only include if explicitly set via CLI
	stringFlags := map[string]struct {
		key string
		val string
	}{
		"ai":                          {"AI_CLI", cfg.AIProvider},
		"implementation-model":        {"IMPL_MODEL", cfg.ImplModel},
		"validation-model":            {"VAL_MODEL", cfg.ValModel},
		"cross-validation-ai":         {"CROSS_AI", cfg.CrossAI},
		"cross-model":                 {"CROSS_MODEL", cfg.CrossModel},
		"final-plan-validation-ai":    {"FINAL_PLAN_AI", cfg.FinalPlanAI},
		"final-plan-validation-model": {"FINAL_PLAN_MODEL", cfg.FinalPlanModel},
		"tasks-validation-ai":         {"TASKS_VAL_AI", cfg.TasksValAI},
		"tasks-validation-model":      {"TASKS_VAL_MODEL", cfg.TasksValModel},
		"tasks-file":                  {"TASKS_FILE", cfg.TasksFile},
		"original-plan-file":          {"ORIGINAL_PLAN_FILE", cfg.OriginalPlanFile},
		"github-issue":                {"GITHUB_ISSUE", cfg.GithubIssue},
		"learnings-file":              {"LEARNINGS_FILE", cfg.LearningsFile},
		"notify-webhook":              {"NOTIFY_WEBHOOK", cfg.NotifyWebhook},
		"notify-channel":              {"NOTIFY_CHANNEL", cfg.NotifyChannel},
		"notify-chat-id":              {"NOTIFY_CHAT_ID", cfg.NotifyChatID},
	}
	for flag, mapping := range stringFlags {
		if cmd.Flags().Changed(flag) {
			overrides[mapping.key] = mapping.val
		}
	}

	// Int flags
	intFlags := map[string]struct {
		key string
		val int
	}{
		"max-iterations":     {"MAX_ITERATIONS", cfg.MaxIterations},
		"max-inadmissible":   {"MAX_INADMISSIBLE", cfg.MaxInadmissible},
		"max-claude-retry":   {"MAX_CLAUDE_RETRY", cfg.MaxClaudeRetry},
		"max-turns":          {"MAX_TURNS", cfg.MaxTurns},
		"inactivity-timeout": {"INACTIVITY_TIMEOUT", cfg.InactivityTimeout},
	}
	for flag, mapping := range intFlags {
		if cmd.Flags().Changed(flag) {
			overrides[mapping.key] = fmt.Sprintf("%d", mapping.val)
		}
	}

	// Bool flags
	boolFlags := map[string]struct {
		key string
		val bool
	}{
		"verbose": {"VERBOSE", cfg.Verbose},
	}
	for flag, mapping := range boolFlags {
		if cmd.Flags().Changed(flag) {
			if mapping.val {
				overrides[mapping.key] = "true"
			} else {
				overrides[mapping.key] = "false"
			}
		}
	}

	// Handle negation flags
	if cmd.Flags().Changed("no-learnings") {
		overrides["ENABLE_LEARNINGS"] = "false"
	}
	if cmd.Flags().Changed("no-cross-validate") {
		overrides["CROSS_VALIDATE"] = "false"
	}

	return overrides
}

func runOrchestrator(cmd *cobra.Command, cfg *config.Config) error {
	// Load config with full precedence chain
	// CLI flags are already bound to cfg, now load file-based configs
	globalConfigPath := ""
	projectConfigPath := ""
	explicitConfigPath := cfg.ConfigFile

	// Build CLI overrides map using Changed() for accurate detection
	cliOverrides := buildCLIOverrides(cmd, cfg)

	// Load config with precedence
	finalCfg, err := config.LoadWithPrecedence(globalConfigPath, projectConfigPath, explicitConfigPath, cliOverrides)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Merge CLI-only flags (not in config files)
	finalCfg.ConfigFile = cfg.ConfigFile
	finalCfg.Resume = cfg.Resume
	finalCfg.ResumeForce = cfg.ResumeForce
	finalCfg.Clean = cfg.Clean
	finalCfg.Status = cfg.Status
	finalCfg.Cancel = cfg.Cancel
	finalCfg.StartAt = cfg.StartAt

	// Replace cfg reference for subsequent use
	cfg = finalCfg

	// Set verbose mode
	logging.SetVerbose(cfg.Verbose)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Build AI runners based on config
	orch := phases.NewOrchestrator(cfg)

	retryCfg := ai.RetryConfig{
		MaxRetries: cfg.MaxClaudeRetry,
		BaseDelay:  5,
	}

	// Setup implementation and validation runners
	var rawImpl, rawVal ai.AIRunner
	if cfg.AIProvider == model.Claude {
		rawImpl = &ai.ClaudeRunner{
			Model:             cfg.ImplModel,
			MaxTurns:          cfg.MaxTurns,
			Verbose:           cfg.Verbose,
			InactivityTimeout: cfg.InactivityTimeout,
		}
		rawVal = &ai.ClaudeRunner{
			Model:             cfg.ValModel,
			MaxTurns:          cfg.MaxTurns,
			Verbose:           cfg.Verbose,
			InactivityTimeout: cfg.InactivityTimeout,
		}
	} else {
		rawImpl = &ai.CodexRunner{
			Model:             cfg.ImplModel,
			Verbose:           cfg.Verbose,
			InactivityTimeout: cfg.InactivityTimeout,
		}
		rawVal = &ai.CodexRunner{
			Model:             cfg.ValModel,
			Verbose:           cfg.Verbose,
			InactivityTimeout: cfg.InactivityTimeout,
		}
	}
	orch.ImplRunner = &ai.RetryRunner{Inner: rawImpl, RetryCfg: retryCfg}
	orch.ValRunner = &ai.RetryRunner{Inner: rawVal, RetryCfg: retryCfg}

	// Setup cross-validation runner
	if cfg.CrossValidate {
		crossAI, crossModel := model.SetupCrossValidation(cfg.AIProvider, cfg.CrossAI, cfg.CrossModel)
		cfg.CrossAI = crossAI
		cfg.CrossModel = crossModel

		avail := ai.CheckAvailability(crossAI)
		if avail[crossAI] {
			var rawCross ai.AIRunner
			if crossAI == model.Claude {
				rawCross = &ai.ClaudeRunner{Model: crossModel, MaxTurns: cfg.MaxTurns, Verbose: cfg.Verbose, InactivityTimeout: cfg.InactivityTimeout}
			} else {
				rawCross = &ai.CodexRunner{Model: crossModel, Verbose: cfg.Verbose, InactivityTimeout: cfg.InactivityTimeout}
			}
			orch.CrossRunner = &ai.RetryRunner{Inner: rawCross, RetryCfg: retryCfg}
		} else {
			cfg.CrossValidate = false
		}
	}

	// Setup final-plan validation runner
	if cfg.CrossValidate || cfg.FinalPlanAI != "" {
		fpAI, fpModel := model.SetupFinalPlanValidation(cfg.CrossAI, cfg.CrossModel, cfg.FinalPlanAI, cfg.FinalPlanModel)
		cfg.FinalPlanAI = fpAI
		cfg.FinalPlanModel = fpModel

		avail := ai.CheckAvailability(fpAI)
		if avail[fpAI] {
			var rawFP ai.AIRunner
			if fpAI == model.Claude {
				rawFP = &ai.ClaudeRunner{Model: fpModel, MaxTurns: cfg.MaxTurns, Verbose: cfg.Verbose, InactivityTimeout: cfg.InactivityTimeout}
			} else {
				rawFP = &ai.CodexRunner{Model: fpModel, Verbose: cfg.Verbose, InactivityTimeout: cfg.InactivityTimeout}
			}
			orch.FinalPlanRunner = &ai.RetryRunner{Inner: rawFP, RetryCfg: retryCfg}
		}
	}

	// Setup tasks validation runner
	tvAI, tvModel := model.SetupTasksValidation(cfg.AIProvider, cfg.ImplModel, cfg.TasksValAI, cfg.TasksValModel)
	cfg.TasksValAI = tvAI
	cfg.TasksValModel = tvModel
	if cfg.OriginalPlanFile != "" || cfg.GithubIssue != "" {
		var rawTV ai.AIRunner
		if tvAI == model.Claude {
			rawTV = &ai.ClaudeRunner{Model: tvModel, MaxTurns: cfg.MaxTurns, Verbose: cfg.Verbose, InactivityTimeout: cfg.InactivityTimeout}
		} else {
			rawTV = &ai.CodexRunner{Model: tvModel, Verbose: cfg.Verbose, InactivityTimeout: cfg.InactivityTimeout}
		}
		orch.TasksValRunner = &ai.RetryRunner{Inner: rawTV, RetryCfg: retryCfg}
	}

	// Setup signal handler to save state on interrupt
	sighandler.SetupSignalHandler(ctx, cancel, func() {
		logging.Warn("Interrupted — saving state...")
	})

	// Run orchestrator
	exitCode := orch.Run(ctx)
	os.Exit(exitCode)
	return nil // unreachable
}
