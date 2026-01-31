// Package cli provides help text and usage formatting for the ralph-loop CLI.
package cli

import (
	"github.com/spf13/cobra"
)

const helpTemplate = `ralph-loop - Dual-model AI implementation-validation loop orchestrator

USAGE
  ralph-loop [flags]

FLAGS
  AI Provider & Models:
    --ai <claude|codex>                    AI CLI to use (default: claude)
    --implementation-model <model>         Model for implementation phase (default: opus/default)
    --validation-model <model>             Model for validation phase (default: opus/default)
    --cross-validation-ai <claude|codex>   AI CLI for cross-validation (default: auto-opposite)
    --cross-model <model>                  Model for cross-validation (default: auto)
    --final-plan-validation-ai <ai>        AI CLI for final plan validation (default: same as cross-val)
    --final-plan-validation-model <model>  Model for final plan validation (default: same as cross-val)
    --tasks-validation-ai <ai>             AI CLI for tasks validation (default: same as --ai)
    --tasks-validation-model <model>       Model for tasks validation (default: same as impl)

  Iteration Limits:
    --max-iterations <int>                 Maximum loop iterations (default: 20)
    --max-inadmissible <int>               Max inadmissible verdicts before exit 6 (default: 5)
    --max-claude-retry <int>               Max retries per AI invocation (default: 10)
    --max-turns <int>                      Max agent turns per AI invocation (default: 100)
    --inactivity-timeout <int>             Seconds of inactivity before kill (default: 1800)

  Input Files:
    --tasks-file <path>                    Path to tasks.md (default: auto-detect)
    --original-plan-file <path>            Path to original plan (mutually exclusive with --github-issue)
    --github-issue <url|number>            GitHub issue URL or number (mutually exclusive with --original-plan-file)
    --learnings-file <path>                Path to learnings file (default: .ralph-loop/learnings.md)
    --config <path>                        Path to additional config file

  Feature Toggles:
    -v, --verbose                          Pass verbose flag to AI CLI
    --no-learnings                         Disable learnings persistence
    --no-cross-validate                    Disable cross-validation phase

  Scheduling:
    --start-at <time>                      Schedule start time (ISO 8601, HH:MM, YYYY-MM-DD HH:MM)
    --at <time>                            Alias for --start-at

  Notifications:
    --notify-webhook <url>                 OpenClaw webhook URL (default: http://127.0.0.1:18789/webhook)
    --notify-channel <channel>             Notification channel (default: telegram)
    --notify-chat-id <id>                  Recipient chat ID (required to enable notifications)

  Session Management:
    --resume                               Resume from last interrupted session
    --resume-force                         Resume even if tasks.md changed (implies --resume)
    --clean                                Delete state directory and start fresh
    --status                               Show session status and exit
    --cancel                               Cancel active session and exit

  Help & Version:
    -h, --help                             Show this help text
    --version                              Show version, commit, build date

EXIT CODES
  0   Success              All tasks complete and validated
  1   Error                Invalid arguments, file not found, misconfiguration
  2   MaxIterations        Iteration limit reached without completion
  3   Escalate             Validation requested human intervention
  4   Blocked              All tasks blocked on external dependencies
  5   TasksInvalid         Tasks don't properly implement original plan
  6   Inadmissible         Inadmissible violation threshold exceeded
  130 Interrupted          SIGINT or SIGTERM received

EXAMPLES
  # Start a new loop with default settings
  ralph-loop

  # Use codex for implementation, claude for validation
  ralph-loop --ai codex --cross-validation-ai claude

  # Resume interrupted session
  ralph-loop --resume

  # Start fresh after clearing state
  ralph-loop --clean

  # Check session status
  ralph-loop --status

For more information, see: https://github.com/CodexForgeBR/cli-tools
`

// SetCustomHelp configures the cobra command to use our custom help template.
func SetCustomHelp(cmd *cobra.Command) {
	cmd.SetHelpTemplate(helpTemplate)
}
