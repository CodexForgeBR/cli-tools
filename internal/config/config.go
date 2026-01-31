// Package config defines the ralph-loop configuration model and default values.
//
// Configuration is assembled from multiple sources with a strict precedence
// chain: built-in defaults < global config file < project config file <
// explicit config file < CLI flag overrides.
package config

// WhitelistedVars lists every configuration variable name that may appear in
// config files. Variables not in this list are silently ignored during loading.
// The list contains exactly 24 entries matching the data model specification.
var WhitelistedVars = [24]string{
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

// Config holds every configuration field for the ralph-loop CLI.
// See the data model specification for field semantics.
type Config struct {
	// AI provider and model selection.
	AIProvider string
	ImplModel  string
	ValModel   string

	// Cross-validation settings.
	CrossValidate bool
	CrossAI       string
	CrossModel    string

	// Final plan validation settings.
	FinalPlanAI    string
	FinalPlanModel string

	// Tasks validation settings.
	TasksValAI    string
	TasksValModel string

	// Iteration limits.
	MaxIterations   int
	MaxInadmissible int
	MaxClaudeRetry  int
	MaxTurns        int

	// Timeouts.
	InactivityTimeout int

	// File paths.
	TasksFile        string
	OriginalPlanFile string
	GithubIssue      string
	LearningsFile    string
	EnableLearnings  bool

	// Runtime flags.
	Verbose bool

	// Notification settings.
	NotifyWebhook string
	NotifyChannel string
	NotifyChatID  string

	// CLI-only flags (not loaded from config files).
	ConfigFile  string
	Resume      bool
	ResumeForce bool
	Clean       bool
	Status      bool
	Cancel      bool
	StartAt     string
}

// NewDefaultConfig returns a Config populated with all built-in default values.
func NewDefaultConfig() *Config {
	return &Config{
		AIProvider:        "claude",
		ImplModel:         "opus",
		ValModel:          "opus",
		CrossValidate:     true,
		MaxIterations:     20,
		MaxInadmissible:   5,
		MaxClaudeRetry:    10,
		MaxTurns:          100,
		InactivityTimeout: 1800,
		LearningsFile:     ".ralph-loop/learnings.md",
		EnableLearnings:   true,
		NotifyWebhook:     "http://127.0.0.1:18789/webhook",
		NotifyChannel:     "telegram",
	}
}
