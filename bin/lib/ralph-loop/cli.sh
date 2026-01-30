#!/bin/bash
# cli.sh - CLI argument parsing for ralph-loop
# Part of Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

usage() {
    cat << EOF
Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

Usage: $(basename "$0") [OPTIONS]

Options:
  -v, --verbose              Pass verbose flag to claude code cli
  --ai CLI                   AI CLI to use: claude or codex (default: claude)
  --max-iterations N         Maximum loop iterations (default: 20)
  --max-claude-retry N       Max retries per AI call (default: 10)
  --max-turns N              Max agent turns per claude invocation (default: 100)
  --inactivity-timeout N     Inactivity timeout in seconds (default: 1800 = 30 min)
  --implementation-model M   Model for implementation (default: opus for claude, config default for codex)
  --validation-model M       Model for validation (default: opus for claude, config default for codex)
  --tasks-file PATH          Path to tasks.md (auto-detects if not specified)
  --original-plan-file PATH  Path to original plan file for plan validation
  --github-issue <URL|NUM>   GitHub issue URL or number to use as original plan
                             (mutually exclusive with --original-plan-file)
  --learnings-file PATH      Path to learnings file (default: .ralph-loop/learnings.md)
  --no-learnings             Disable learnings persistence (enabled by default)
  --no-cross-validate        Disable cross-validation phase (enabled by default)
  --cross-model M            Model for cross-validation AI (default: opposite AI's default)
  --cross-validation-ai AI   Override cross-validation AI (default: opposite of --ai)
  --final-plan-validation-ai AI      AI for final plan validation (default: same as cross-validation)
  --final-plan-validation-model M    Model for final plan validation (default: same as cross-validation)
  --tasks-validation-ai AI           AI for tasks validation (default: same as implementation)
  --tasks-validation-model M         Model for tasks validation (default: same as implementation)
  --start-at DATETIME        Schedule when implementation begins (validation runs immediately)
                             Formats: YYYY-MM-DD, HH:MM, "YYYY-MM-DD HH:MM", YYYY-MM-DDTHH:MM
  --notify-webhook URL       Webhook URL for notifications (default: http://127.0.0.1:18789/webhook)
  --notify-channel NAME      Channel name for routing (default: telegram)
  --notify-chat-id ID        Recipient chat ID (for Telegram, Discord, etc.)
  --config PATH              Additional config file to load (highest priority after CLI flags)
  --resume                   Resume from last interrupted session
  --resume-force             Resume even if tasks.md has changed
  --clean                    Start fresh, delete existing .ralph-loop state
  --status                   Show current session status without running
  --cancel                   Cancel an active/interrupted session and exit
  -h, --help                 Show this help message

Timeout Configuration:
  --max-turns limits tool calls per invocation to prevent unbounded work
  Inactivity timeout: 1800s default (kills process if no output for 30 minutes)
    - Configurable with --inactivity-timeout
    - Resets when AI produces output (activity-based)
  Hard cap timeout: 7200s (absolute 2-hour maximum per invocation)
  Both timeouts reset when Claude produces output (activity-based)

Cross-Validation:
  By default, when validation returns COMPLETE, a cross-validation phase runs
  using the OPPOSITE AI (claude → codex, or codex → claude) to independently
  verify completion. This provides an additional layer of verification.

  Cross-validation verdicts:
    CONFIRMED - Agrees with validation, truly complete
    REJECTED  - Disagrees, provides feedback and continues loop

  Disable with --no-cross-validate if only single-AI validation is desired.

Original Plan Validation:
  When --original-plan-file is provided, two additional validations run:

  1. Tasks Validation (iteration 1 only, before implementation):
     - Validates that tasks.md properly covers the original plan
     - Uses the SAME AI as implementation (--ai flag value)
     - Aborts immediately if tasks don't cover the plan (exit code 5)

  2. Final Plan Validation (after cross-validation confirms COMPLETE):
     - Validates the original plan was actually implemented
     - Uses a DIFFERENT AI than implementation (like cross-validation)
     - Does NOT reference tasks.md - only the plan and codebase
     - If NOT_IMPLEMENTED: continues loop with feedback
     - If CONFIRMED: marks session as truly complete

Configuration Files:
  Config files provide defaults for all flags using layered precedence:
    1. Script defaults (hardcoded)
    2. Global config: ~/.config/ralph-loop/config
    3. Project config: .ralph-loop/config
    4. CLI flags (highest priority, always win)

  Config file format (shell-sourceable KEY=VALUE):
    AI_CLI=claude
    IMPL_MODEL=opus
    MAX_ITERATIONS=30
    NOTIFY_WEBHOOK=http://127.0.0.1:18789/webhook
    NOTIFY_CHANNEL=telegram
    NOTIFY_CHAT_ID=123456789

  See ~/.config/ralph-loop/config for full example with all options.

Notifications:
  Ralph loop can send notifications to OpenClaw or any webhook endpoint at
  completion, failure, or escalation events. Notifications are fire-and-forget
  (never block the loop). Configure via config file or CLI flags.

  Notification events:
    - completed: All tasks finished successfully
    - max_iterations: Iteration limit reached
    - escalate: Human intervention requested
    - blocked: Tasks blocked
    - inadmissible: Repeated inadmissible practices
    - tasks_invalid: Tasks don't implement plan
    - interrupted: User interrupted (Ctrl+C)

  OpenClaw integration:
    1. Install: npm install -g openclaw@latest
    2. Onboard: openclaw onboard --install-daemon
    3. Pair Telegram: openclaw pairing approve telegram <code>
    4. Configure: Set NOTIFY_CHAT_ID in config file
    5. Run ralph-loop - notifications go to your Telegram

Session Management:
  When a session is interrupted (Ctrl+C), state is automatically saved.
  Running the script again will detect the interrupted session and prompt you
  to either resume or start fresh.

  Use --cancel to abort an interrupted session without resuming or starting fresh.

Exit Codes:
  0 - All tasks completed successfully
  1 - Error (no tasks.md, invalid params, etc.)
  2 - Max iterations reached without completion
  3 - Escalation requested by validator
  4 - Tasks blocked - human intervention needed
  5 - Tasks don't properly implement the original plan

Examples:
  $(basename "$0")
  $(basename "$0") --max-iterations 10
  $(basename "$0") --implementation-model sonnet --validation-model haiku
  $(basename "$0") --tasks-file specs/feature/tasks.md -v
  $(basename "$0") --ai codex
  $(basename "$0") --ai claude --cross-model o1
  $(basename "$0") --no-cross-validate
  $(basename "$0") --resume
  $(basename "$0") --status
  $(basename "$0") --clean
EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                VERBOSE="--verbose"
                shift
                ;;
            --ai)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --ai (expected: claude or codex)"
                    exit 1
                fi
                case "$2" in
                    claude|codex)
                        AI_CLI="$2"
                        OVERRIDE_AI="1"
                        ;;
                    *)
                        log_error "Invalid value for --ai: $2 (expected: claude or codex)"
                        exit 1
                        ;;
                esac
                shift 2
                ;;
            --max-iterations)
                if [[ -z "$2" || ! "$2" =~ ^[0-9]+$ ]]; then
                    log_error "Invalid value for --max-iterations: $2"
                    exit 1
                fi
                MAX_ITERATIONS=$2
                OVERRIDE_MAX_ITERATIONS="1"
                shift 2
                ;;
            --max-inadmissible)
                if [[ -z "$2" || ! "$2" =~ ^[0-9]+$ ]]; then
                    log_error "Invalid value for --max-inadmissible: $2"
                    exit 1
                fi
                MAX_INADMISSIBLE=$2
                OVERRIDE_MAX_INADMISSIBLE="1"
                shift 2
                ;;
            --max-claude-retry)
                if [[ -z "$2" || ! "$2" =~ ^[0-9]+$ ]]; then
                    log_error "Invalid value for --max-claude-retry: $2"
                    exit 1
                fi
                MAX_CLAUDE_RETRY=$2
                shift 2
                ;;
            --max-turns)
                if [[ -z "$2" || ! "$2" =~ ^[0-9]+$ ]]; then
                    log_error "Invalid value for --max-turns: $2"
                    exit 1
                fi
                MAX_TURNS=$2
                shift 2
                ;;
            --inactivity-timeout)
                if [[ -z "$2" || ! "$2" =~ ^[0-9]+$ ]]; then
                    log_error "Invalid value for --inactivity-timeout: $2"
                    exit 1
                fi
                INACTIVITY_TIMEOUT=$2
                shift 2
                ;;
            --implementation-model)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --implementation-model"
                    exit 1
                fi
                IMPL_MODEL=$2
                OVERRIDE_MODELS="1"
                OVERRIDE_IMPL_MODEL="1"
                shift 2
                ;;
            --validation-model)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --validation-model"
                    exit 1
                fi
                VAL_MODEL=$2
                OVERRIDE_MODELS="1"
                OVERRIDE_VAL_MODEL="1"
                shift 2
                ;;
            --tasks-file)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --tasks-file"
                    exit 1
                fi
                TASKS_FILE=$2
                shift 2
                ;;
            --no-cross-validate)
                CROSS_VALIDATE=0
                shift
                ;;
            --cross-model)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --cross-model"
                    exit 1
                fi
                CROSS_MODEL="$2"
                shift 2
                ;;
            --cross-validation-ai)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --cross-validation-ai"
                    exit 1
                fi
                case "$2" in
                    claude|codex)
                        CROSS_AI="$2"
                        OVERRIDE_CROSS_AI="1"
                        ;;
                    *)
                        log_error "Invalid value for --cross-validation-ai: $2 (expected: claude or codex)"
                        exit 1
                        ;;
                esac
                shift 2
                ;;
            --final-plan-validation-ai)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --final-plan-validation-ai"
                    exit 1
                fi
                case "$2" in
                    claude|codex)
                        FINAL_PLAN_AI="$2"
                        OVERRIDE_FINAL_PLAN_AI="1"
                        ;;
                    *)
                        log_error "Invalid value for --final-plan-validation-ai: $2 (expected: claude or codex)"
                        exit 1
                        ;;
                esac
                shift 2
                ;;
            --final-plan-validation-model)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --final-plan-validation-model"
                    exit 1
                fi
                FINAL_PLAN_MODEL="$2"
                OVERRIDE_FINAL_PLAN_MODEL="1"
                shift 2
                ;;
            --tasks-validation-ai)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --tasks-validation-ai"
                    exit 1
                fi
                case "$2" in
                    claude|codex)
                        TASKS_VAL_AI="$2"
                        OVERRIDE_TASKS_VAL_AI="1"
                        ;;
                    *)
                        log_error "Invalid value for --tasks-validation-ai: $2 (expected: claude or codex)"
                        exit 1
                        ;;
                esac
                shift 2
                ;;
            --tasks-validation-model)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --tasks-validation-model"
                    exit 1
                fi
                TASKS_VAL_MODEL="$2"
                OVERRIDE_TASKS_VAL_MODEL="1"
                shift 2
                ;;
            --resume)
                RESUME_FLAG="1"
                shift
                ;;
            --resume-force)
                RESUME_FLAG="1"
                RESUME_FORCE="1"
                shift
                ;;
            --clean)
                CLEAN_FLAG="1"
                shift
                ;;
            --status)
                STATUS_FLAG="1"
                shift
                ;;
            --cancel)
                CANCEL_FLAG="1"
                shift
                ;;
            --original-plan-file)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --original-plan-file"
                    exit 1
                fi
                if [[ ! -f "$2" ]]; then
                    log_error "Original plan file not found: $2"
                    exit 1
                fi
                ORIGINAL_PLAN_FILE="$2"
                shift 2
                ;;
            --github-issue)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --github-issue"
                    exit 1
                fi
                GITHUB_ISSUE="$2"
                shift 2
                ;;
            --learnings-file)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --learnings-file"
                    exit 1
                fi
                LEARNINGS_FILE="$2"
                shift 2
                ;;
            --no-learnings)
                ENABLE_LEARNINGS=0
                shift
                ;;
            --notify-webhook)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --notify-webhook"
                    exit 1
                fi
                NOTIFY_WEBHOOK="$2"
                OVERRIDE_NOTIFY_WEBHOOK="1"
                shift 2
                ;;
            --notify-channel)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --notify-channel"
                    exit 1
                fi
                NOTIFY_CHANNEL="$2"
                OVERRIDE_NOTIFY_CHANNEL="1"
                shift 2
                ;;
            --notify-chat-id)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --notify-chat-id"
                    exit 1
                fi
                NOTIFY_CHAT_ID="$2"
                OVERRIDE_NOTIFY_CHAT_ID="1"
                shift 2
                ;;
            --config)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --config"
                    exit 1
                fi
                if [[ ! -f "$2" ]]; then
                    log_error "Config file not found: $2"
                    exit 1
                fi
                # Load additional config file (highest priority after CLI flags)
                load_config "$2"
                shift 2
                ;;
            --start-at|--at)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --start-at"
                    exit 1
                fi
                SCHEDULE_INPUT="$2"
                shift 2
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                log_error "Unknown parameter: $1"
                usage
                exit 1
                ;;
        esac
    done
}
