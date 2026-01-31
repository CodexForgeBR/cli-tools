#!/bin/bash
# globals.sh - Global constants, defaults, and variable declarations for ralph-loop
# Part of Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

# Exit code constants
EXIT_SUCCESS=0
EXIT_ERROR=1
EXIT_MAX_ITERATIONS=2
EXIT_ESCALATE=3
EXIT_BLOCKED=4
EXIT_TASKS_INVALID=5            # Tasks don't properly implement the plan
EXIT_INADMISSIBLE=6              # Fundamentally broken - inadmissible practice detected

# Default configuration
MAX_ITERATIONS=20
MAX_CLAUDE_RETRY=10  # Default retries per AI call
IMPL_MODEL="opus"
VAL_MODEL="opus"
TASKS_FILE=""
VERBOSE=""
AI_CLI="claude"
OVERRIDE_AI=""
OVERRIDE_IMPL_MODEL=""
OVERRIDE_VAL_MODEL=""
OVERRIDE_MAX_ITERATIONS=""
OVERRIDE_MAX_INADMISSIBLE=""
STATE_DIR=".ralph-loop"
SCRIPT_START_TIME=""
ITERATION_START_TIME=""
SESSION_ID=""
CURRENT_ITERATION=0  # Global iteration counter for cleanup handler

# Timeout configuration
MAX_TURNS=100                # Default max turns per claude invocation
INACTIVITY_TIMEOUT=1800      # 30 min inactivity timeout (resets on activity)
MAX_TOTAL_TIMEOUT=7200       # 2 hour hard cap

# Resume-related flags
RESUME_FLAG=""
RESUME_FORCE=""
CLEAN_FLAG=""
STATUS_FLAG=""
CANCEL_FLAG=""
OVERRIDE_MODELS=""

# Original plan validation
ORIGINAL_PLAN_FILE=""           # Path to the original plan file (optional)
GITHUB_ISSUE=""                 # GitHub issue URL or number (optional)

# Learnings configuration
LEARNINGS_FILE=""           # Path to learnings file (default: .ralph-loop/learnings.md)
ENABLE_LEARNINGS=1          # ON by default

# Scheduling configuration
SCHEDULE_INPUT=""           # Raw user input for --start-at
SCHEDULE_TARGET_EPOCH=""    # Parsed target time in epoch seconds
SCHEDULE_TARGET_HUMAN=""    # Human-readable target time for display

# Rate limit tracking
RATE_LIMIT_RESET_EPOCH=""
RATE_LIMIT_RESET_HUMAN=""
RATE_LIMIT_RESET_TZ=""
RATE_LIMIT_BUFFER_SECONDS=60

# State tracking for resume
CURRENT_PHASE=""
LAST_FEEDBACK=""
STORED_AI_CLI=""
STORED_IMPL_MODEL=""
STORED_VAL_MODEL=""
INADMISSIBLE_COUNT=0         # Track inadmissible practice violations
MAX_INADMISSIBLE=5           # Escalate after this many inadmissible verdicts

# Cross-validation configuration
CROSS_VALIDATE=1              # ON by default
CROSS_MODEL=""                # Model for cross-validation AI
CROSS_AI=""                   # Auto-set: opposite of AI_CLI
CROSS_AI_AVAILABLE=0          # Whether alternate AI CLI is installed
OVERRIDE_CROSS_AI=""          # Override automatic opposite AI calculation

# Final plan validation configuration
FINAL_PLAN_AI=""              # AI for final plan validation (defaults to CROSS_AI)
FINAL_PLAN_MODEL=""           # Model for final plan validation (defaults to CROSS_MODEL)
FINAL_PLAN_AI_AVAILABLE=0    # Whether final plan AI is installed
OVERRIDE_FINAL_PLAN_AI=""    # Override final plan AI
OVERRIDE_FINAL_PLAN_MODEL="" # Override final plan model

# Tasks validation configuration (initial plan check)
TASKS_VAL_AI=""               # AI for tasks validation (defaults to AI_CLI)
TASKS_VAL_MODEL=""            # Model for tasks validation (defaults to IMPL_MODEL)
TASKS_VAL_AI_AVAILABLE=0     # Whether tasks validation AI is installed
OVERRIDE_TASKS_VAL_AI=""     # Override tasks validation AI
OVERRIDE_TASKS_VAL_MODEL=""  # Override tasks validation model

# Notification configuration
NOTIFY_WEBHOOK=""             # Webhook URL for notifications (defaults to http://127.0.0.1:18789/webhook)
NOTIFY_CHANNEL=""             # Channel name for routing (default: telegram)
NOTIFY_CHAT_ID=""             # Recipient chat ID
OVERRIDE_NOTIFY_WEBHOOK=""    # Override notification webhook
OVERRIDE_NOTIFY_CHANNEL=""    # Override notification channel
OVERRIDE_NOTIFY_CHAT_ID=""    # Override notification chat ID

# Retry state tracking for resume
CURRENT_RETRY_ATTEMPT=1
CURRENT_RETRY_DELAY=5
RESUMING_RETRY=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
