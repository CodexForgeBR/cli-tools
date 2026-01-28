#!/bin/bash

# Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development
# Based on the Ralph Wiggum technique by Geoffrey Huntley (May 2025)
#
# Usage: ralph-loop.sh [OPTIONS]
#
# Options:
#   -v, --verbose            Pass verbose flag to claude code cli
#   --ai CLI                 AI CLI to use: claude or codex (default: claude)
#   --max-iterations N       Maximum loop iterations (default: 20)
#   --max-inadmissible N     Max inadmissible violations before escalation (default: 5)
#   --implementation-model   Model for implementation (default: opus for claude, config default for codex)
#   --validation-model       Model for validation (default: opus for claude, config default for codex)
#   --tasks-file PATH        Path to tasks.md (auto-detects: ./tasks.md, specs/*/tasks.md)
#
# Exit Codes:
#   0 - All tasks completed successfully
#   1 - Error (no tasks.md, invalid params, etc.)
#   2 - Max iterations reached without completion
#   3 - Escalation requested by validator
#   4 - Tasks blocked - human intervention needed
#   5 - Tasks don't properly implement the plan
#   6 - Repeated inadmissible practices (max violations exceeded)

set -e

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

# Track state for circuit breaker
LAST_CHECKED_COUNT=0
NO_PROGRESS_COUNT=0
MAX_NO_PROGRESS=3

# Cleanup handler for graceful shutdown
cleanup() {
    echo -e "\n${YELLOW}Interrupted! Saving state...${NC}"

    # Save state with current iteration and phase
    save_state "INTERRUPTED" "$CURRENT_ITERATION"

    echo -e "${GREEN}State saved to ${STATE_DIR}/${NC}"
    echo -e "${CYAN}Run '$(basename "$0") --resume' to continue where you left off${NC}"
    exit 130
}

trap cleanup SIGINT SIGTERM

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_phase() {
    echo -e "\n${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}  $1${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

# Format seconds into human readable time
format_duration() {
    local seconds=$1
    local hours=$((seconds / 3600))
    local minutes=$(((seconds % 3600) / 60))
    local secs=$((seconds % 60))

    if [[ $hours -gt 0 ]]; then
        printf "%dh %dm %ds" $hours $minutes $secs
    elif [[ $minutes -gt 0 ]]; then
        printf "%dm %ds" $minutes $secs
    else
        printf "%ds" $secs
    fi
}

# Get current timestamp in seconds
get_timestamp() {
    date +%s
}

# Print usage
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

# Apply default models based on selected AI CLI
set_default_models_for_ai() {
    if [[ "$AI_CLI" == "codex" ]]; then
        if [[ -z "$OVERRIDE_IMPL_MODEL" ]]; then
            IMPL_MODEL="default"
        fi
        if [[ -z "$OVERRIDE_VAL_MODEL" ]]; then
            VAL_MODEL="default"
        fi
    else
        if [[ -z "$OVERRIDE_IMPL_MODEL" ]]; then
            IMPL_MODEL="opus"
        fi
        if [[ -z "$OVERRIDE_VAL_MODEL" ]]; then
            VAL_MODEL="opus"
        fi
    fi
}

# Set up cross-validation AI (opposite of main AI)
set_cross_validation_ai() {
    if [[ "$CROSS_VALIDATE" -eq 0 ]]; then
        return
    fi

    # Determine cross-validation AI (unless overridden)
    if [[ -z "$OVERRIDE_CROSS_AI" ]]; then
        # Auto-detect: use opposite AI
        if [[ "$AI_CLI" == "claude" ]]; then
            CROSS_AI="codex"
            [[ -z "$CROSS_MODEL" ]] && CROSS_MODEL="default"
        else
            CROSS_AI="claude"
            [[ -z "$CROSS_MODEL" ]] && CROSS_MODEL="opus"
        fi
    else
        # User explicitly set CROSS_AI via --cross-validation-ai
        # Set default model if not specified
        if [[ -z "$CROSS_MODEL" ]]; then
            if [[ "$CROSS_AI" == "claude" ]]; then
                CROSS_MODEL="opus"
            else
                CROSS_MODEL="default"
            fi
        fi
    fi

    # Check if cross-validation AI is installed
    if command -v "$CROSS_AI" &>/dev/null; then
        CROSS_AI_AVAILABLE=1
        log_info "Cross-validation enabled: will use $CROSS_AI ($CROSS_MODEL)"
    else
        CROSS_AI_AVAILABLE=0
        log_warn "Cross-validation: $CROSS_AI not found, will skip phase 3"
    fi
}

# Set up final plan validation AI (defaults to cross-validation AI)
set_final_plan_validation_ai() {
    # Only relevant if we have an original plan file or GitHub issue
    if [[ -z "$ORIGINAL_PLAN_FILE" && -z "$GITHUB_ISSUE" ]]; then
        return
    fi

    # Determine final plan validation AI (unless overridden)
    if [[ -z "$OVERRIDE_FINAL_PLAN_AI" ]]; then
        # Default: use same AI as cross-validation
        FINAL_PLAN_AI="$CROSS_AI"
        if [[ -z "$OVERRIDE_FINAL_PLAN_MODEL" ]]; then
            FINAL_PLAN_MODEL="$CROSS_MODEL"
        fi
    else
        # User explicitly set FINAL_PLAN_AI via --final-plan-validation-ai
        # Set default model if not specified
        if [[ -z "$OVERRIDE_FINAL_PLAN_MODEL" && -z "$FINAL_PLAN_MODEL" ]]; then
            if [[ "$FINAL_PLAN_AI" == "claude" ]]; then
                FINAL_PLAN_MODEL="opus"
            else
                FINAL_PLAN_MODEL="default"
            fi
        fi
    fi

    # Check if final plan validation AI is installed
    if command -v "$FINAL_PLAN_AI" &>/dev/null; then
        FINAL_PLAN_AI_AVAILABLE=1
        log_info "Final plan validation enabled: will use $FINAL_PLAN_AI ($FINAL_PLAN_MODEL)"
    else
        FINAL_PLAN_AI_AVAILABLE=0
        log_warn "Final plan validation: $FINAL_PLAN_AI not found"
    fi
}

# Set up tasks validation AI (defaults to implementation AI)
set_tasks_validation_ai() {
    # Only relevant if we have an original plan file or GitHub issue
    if [[ -z "$ORIGINAL_PLAN_FILE" && -z "$GITHUB_ISSUE" ]]; then
        return
    fi

    # Determine tasks validation AI (unless overridden)
    if [[ -z "$OVERRIDE_TASKS_VAL_AI" ]]; then
        # Default: use same AI as implementation
        TASKS_VAL_AI="$AI_CLI"
        if [[ -z "$OVERRIDE_TASKS_VAL_MODEL" ]]; then
            TASKS_VAL_MODEL="$IMPL_MODEL"
        fi
    else
        # User explicitly set TASKS_VAL_AI via --tasks-validation-ai
        # Set default model if not specified
        if [[ -z "$OVERRIDE_TASKS_VAL_MODEL" && -z "$TASKS_VAL_MODEL" ]]; then
            if [[ "$TASKS_VAL_AI" == "claude" ]]; then
                TASKS_VAL_MODEL="opus"
            else
                TASKS_VAL_MODEL="default"
            fi
        fi
    fi

    # Check if tasks validation AI is installed
    if command -v "$TASKS_VAL_AI" &>/dev/null; then
        TASKS_VAL_AI_AVAILABLE=1
        log_info "Tasks validation enabled: will use $TASKS_VAL_AI ($TASKS_VAL_MODEL)"
    else
        TASKS_VAL_AI_AVAILABLE=0
        log_warn "Tasks validation: $TASKS_VAL_AI not found"
    fi
}

is_claude_model_hint() {
    local model=$1
    if [[ "$model" == "opus" || "$model" == "sonnet" || "$model" == "haiku" ]]; then
        return 0
    fi
    if [[ "$model" == claude-* ]]; then
        return 0
    fi
    return 1
}

is_codex_model_hint() {
    local model=$1
    if [[ "$model" == "default" ]]; then
        return 0
    fi
    if [[ "$model" =~ ^o[0-9] ]]; then
        return 0
    fi
    if [[ "$model" =~ ^(gpt|chatgpt|text|ft|gpt4) ]]; then
        return 0
    fi
    return 1
}

validate_model_for_ai() {
    local ai=$1
    local model=$2
    local label=$3

    if [[ -z "$model" ]]; then
        return 0
    fi

    if [[ "$ai" == "codex" && "$model" == "default" ]]; then
        return 0
    fi

    if [[ "$ai" == "claude" && "$model" == "default" ]]; then
        log_error "Model 'default' is only valid with --ai codex (invalid for $label model)"
        exit 1
    fi

    if [[ "$ai" == "codex" ]] && is_claude_model_hint "$model"; then
        log_error "Model '$model' looks like a Claude model but --ai is codex ($label model)"
        exit 1
    fi

    if [[ "$ai" == "claude" ]] && is_codex_model_hint "$model"; then
        log_error "Model '$model' looks like a Codex/OpenAI model but --ai is claude ($label model)"
        exit 1
    fi
}

validate_models_for_ai() {
    validate_model_for_ai "$AI_CLI" "$IMPL_MODEL" "implementation"
    validate_model_for_ai "$AI_CLI" "$VAL_MODEL" "validation"
}

# Find tasks.md file
find_tasks_file() {
    if [[ -n "$TASKS_FILE" ]]; then
        if [[ -f "$TASKS_FILE" ]]; then
            echo "$TASKS_FILE"
            return 0
        else
            log_error "Specified tasks file not found: $TASKS_FILE"
            return 1
        fi
    fi

    # Auto-detect tasks.md in common locations
    local search_paths=(
        "./tasks.md"
        "./TASKS.md"
        "./specs/tasks.md"
        "./spec/tasks.md"
    )

    for path in "${search_paths[@]}"; do
        if [[ -f "$path" ]]; then
            echo "$path"
            return 0
        fi
    done

    # Search in specs subdirectories
    local found
    found=$(find ./specs -name "tasks.md" -type f 2>/dev/null | head -1)
    if [[ -n "$found" ]]; then
        echo "$found"
        return 0
    fi

    found=$(find ./spec -name "tasks.md" -type f 2>/dev/null | head -1)
    if [[ -n "$found" ]]; then
        echo "$found"
        return 0
    fi

    log_error "No tasks.md file found. Create one or specify with --tasks-file"
    return 1
}

# Count unchecked tasks in tasks.md
count_unchecked_tasks() {
    local file=$1
    local count
    count=$(grep -c '^\s*- \[ \]' "$file" 2>/dev/null) || count=0
    echo "$count"
}

# Count checked tasks in tasks.md
count_checked_tasks() {
    local file=$1
    local count
    count=$(grep -c '^\s*- \[x\]' "$file" 2>/dev/null) || count=0
    echo "$count"
}

# Compute SHA256 hash of tasks.md file
compute_tasks_hash() {
    local file=$1
    if [[ ! -f "$file" ]]; then
        echo ""
        return 1
    fi
    sha256sum "$file" | awk '{print $1}'
}

# Load state from current-state.json into shell variables
load_state() {
    local state_file="$STATE_DIR/current-state.json"

    if [[ ! -f "$state_file" ]]; then
        return 1
    fi

    # Use Python to parse JSON and output shell variable assignments
    local python_output
    python_output=$(python3 - "$state_file" << 'PYTHON_EOF'
import sys
import json
import base64

try:
    with open(sys.argv[1], 'r') as f:
        state = json.load(f)

    # Export variables safely
    print(f"SCRIPT_START_TIME='{state.get('started_at', '')}'")
    print(f"ITERATION={state.get('iteration', 0)}")
    print(f"CURRENT_PHASE='{state.get('phase', '')}'")

    # Encode feedback as base64 to avoid quote escaping issues
    feedback = state.get('last_feedback', '')
    feedback_b64 = base64.b64encode(feedback.encode('utf-8')).decode('ascii')
    print(f"LAST_FEEDBACK_B64='{feedback_b64}'")

    print(f"SESSION_ID='{state.get('session_id', '')}'")
    print(f"STORED_AI_CLI='{state.get('ai_cli', '')}'")

    circuit = state.get('circuit_breaker', {})
    print(f"NO_PROGRESS_COUNT={circuit.get('no_progress_count', 0)}")
    print(f"LAST_CHECKED_COUNT={circuit.get('last_unchecked_count', 0)}")

    # Store tasks file hash for validation
    print(f"STORED_TASKS_HASH='{state.get('tasks_file_hash', '')}'")
    print(f"STORED_TASKS_FILE='{state.get('tasks_file', '')}'")
    print(f"STORED_IMPL_MODEL='{state.get('implementation_model', '')}'")
    print(f"STORED_VAL_MODEL='{state.get('validation_model', '')}'")

    # Restore plan validation settings
    print(f"STORED_ORIGINAL_PLAN_FILE='{state.get('original_plan_file', '')}'")
    print(f"STORED_GITHUB_ISSUE='{state.get('github_issue', '')}'")
    print(f"STORED_MAX_ITERATIONS={state.get('max_iterations', 20)}")
    print(f"STORED_MAX_INADMISSIBLE={state.get('max_inadmissible', 5)}")

    # Restore learnings settings (defaults for backward compatibility)
    learnings = state.get('learnings', {})
    print(f"STORED_LEARNINGS_ENABLED={learnings.get('enabled', 1)}")
    print(f"STORED_LEARNINGS_FILE='{learnings.get('file', '')}'")

    # Retry state for resume (defaults for backward compatibility)
    retry_state = state.get('retry_state', {})
    print(f"CURRENT_RETRY_ATTEMPT={retry_state.get('attempt', 1)}")
    print(f"CURRENT_RETRY_DELAY={retry_state.get('delay', 5)}")

    # Inadmissible count (defaults to 0 for backward compatibility)
    print(f"INADMISSIBLE_COUNT={state.get('inadmissible_count', 0)}")

    sys.exit(0)
except Exception as e:
    print(f"# Error loading state: {e}", file=sys.stderr)
    sys.exit(1)
PYTHON_EOF
)

    # Check if Python command succeeded
    local python_exit=$?
    if [[ $python_exit -ne 0 ]]; then
        echo "ERROR: Failed to parse state file" >&2
        return 1
    fi

    # Eval the Python output to set variables
    eval "$python_output"

    # Verify critical variable was set
    if [[ -z "$ITERATION" ]]; then
        echo "ERROR: ITERATION not set after loading state" >&2
        return 1
    fi

    # Decode base64 feedback
    if [[ -n "$LAST_FEEDBACK_B64" ]]; then
        LAST_FEEDBACK=$(echo "$LAST_FEEDBACK_B64" | base64 -d)
    fi

    return 0
}

# Validate state integrity
validate_state() {
    local state_file="$STATE_DIR/current-state.json"

    if [[ ! -f "$state_file" ]]; then
        echo "No state file found"
        return 1
    fi

    # Check if tasks.md exists
    if [[ ! -f "$TASKS_FILE" ]]; then
        echo "Tasks file no longer exists: $TASKS_FILE"
        return 1
    fi

    # Check tasks.md hash if not forcing
    if [[ -z "$RESUME_FORCE" ]]; then
        local current_hash
        current_hash=$(compute_tasks_hash "$TASKS_FILE")

        if [[ -n "$STORED_TASKS_HASH" && "$current_hash" != "$STORED_TASKS_HASH" ]]; then
            echo "Tasks file has been modified since session was interrupted"
            return 2
        fi
    fi

    return 0
}

# Recover feedback from validation output or state
recover_feedback() {
    local iteration=$1

    # Try to load from state first
    if [[ -n "$LAST_FEEDBACK" ]]; then
        echo "$LAST_FEEDBACK"
        return 0
    fi

    # Try to extract from last validation output
    local val_file="$STATE_DIR/val-output-${iteration}.txt"
    if [[ -f "$val_file" ]]; then
        local val_json
        val_json=$(extract_json_from_file "$val_file" "RALPH_VALIDATION") || true

        if [[ -n "$val_json" ]]; then
            parse_feedback "$val_json"
            return 0
        fi
    fi

    echo ""
}

# Show resume summary and ask for confirmation
show_resume_summary() {
    local iteration=$1
    local phase=$2
    local status=$3

    echo -e "\n${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║              PREVIOUS SESSION DETECTED                        ║${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}\n"

    echo "A previous Ralph Loop session was interrupted."
    echo "  Status:    $status"
    echo "  Iteration: $iteration"
    echo "  Phase:     $phase"
    echo ""

    if [[ -n "$RESUME_FORCE" && -n "$STORED_TASKS_HASH" ]]; then
        local current_hash
        current_hash=$(compute_tasks_hash "$TASKS_FILE")
        if [[ "$current_hash" != "$STORED_TASKS_HASH" ]]; then
            log_warn "Tasks file has been modified (--resume-force active)"
        fi
    fi

    echo "Resuming from iteration $iteration, phase: $phase"
    echo ""
}

# Check for existing state and prompt user
check_existing_state() {
    local state_file="$STATE_DIR/current-state.json"

    # No state file - fresh start
    if [[ ! -f "$state_file" ]]; then
        return 0
    fi

    # Load state to check status
    local stored_status
    stored_status=$(python3 - "$state_file" << 'PYTHON_EOF'
import sys
import json

try:
    with open(sys.argv[1], 'r') as f:
        state = json.load(f)
    print(state.get('status', 'UNKNOWN'))
except:
    print('ERROR')
PYTHON_EOF
)

    # If status is COMPLETE, allow fresh start
    if [[ "$stored_status" == "COMPLETE" ]]; then
        return 0
    fi

    # If --resume or --resume-force specified, we're good
    if [[ -n "$RESUME_FLAG" || -n "$RESUME_FORCE" ]]; then
        return 0
    fi

    # If --clean specified, remove state dir
    if [[ -n "$CLEAN_FLAG" ]]; then
        return 0
    fi

    # Otherwise, prompt user
    echo -e "\n${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║              PREVIOUS SESSION DETECTED                        ║${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}\n"

    echo "A previous Ralph Loop session was interrupted."
    echo "  Status:    $stored_status"
    echo ""
    echo "Options:"
    echo "  $(basename "$0") --resume        Resume from where you left off"
    echo "  $(basename "$0") --clean         Start fresh (discards previous state)"
    echo "  $(basename "$0") --status        View detailed session status"
    echo ""

    exit 1
}

# Show status of current session
show_status() {
    local state_file="$STATE_DIR/current-state.json"

    if [[ ! -f "$state_file" ]]; then
        echo "No active or previous Ralph Loop session found."
        exit 0
    fi

    echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║                  RALPH LOOP SESSION STATUS                    ║${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}\n"

    # Parse and display state using Python
    python3 - "$state_file" << 'PYTHON_EOF'
import sys
import json
from datetime import datetime

try:
    with open(sys.argv[1], 'r') as f:
        state = json.load(f)

    print(f"Session ID:           {state.get('session_id', 'N/A')}")
    print(f"Status:               {state.get('status', 'UNKNOWN')}")
    print(f"Iteration:            {state.get('iteration', 0)}")
    print(f"Phase:                {state.get('phase', 'N/A')}")
    print(f"Started:              {state.get('started_at', 'N/A')}")
    print(f"Last Updated:         {state.get('last_updated', 'N/A')}")
    print(f"Tasks File:           {state.get('tasks_file', 'N/A')}")
    print(f"AI CLI:               {state.get('ai_cli', 'N/A')}")
    print(f"Implementation Model: {state.get('implementation_model', 'N/A')}")
    print(f"Validation Model:     {state.get('validation_model', 'N/A')}")
    print(f"Max Iterations:       {state.get('max_iterations', 'N/A')}")

    circuit = state.get('circuit_breaker', {})
    if circuit:
        print(f"\nCircuit Breaker:")
        print(f"  No Progress Count:  {circuit.get('no_progress_count', 0)}")
        print(f"  Last Unchecked:     {circuit.get('last_unchecked_count', 0)}")

    retry = state.get('retry_state', {})
    if retry and retry.get('attempt', 1) > 1:
        print(f"\nRetry State (mid-retry when interrupted):")
        print(f"  Next Attempt:       {retry.get('attempt', 1)}")
        print(f"  Next Delay:         {retry.get('delay', 5)}s")

    feedback = state.get('last_feedback', '')
    if feedback:
        print(f"\nLast Feedback:")
        print(f"  {feedback[:100]}{'...' if len(feedback) > 100 else ''}")

except Exception as e:
    print(f"Error reading state: {e}")
    sys.exit(1)
PYTHON_EOF

    echo ""
    exit 0
}

# Initialize state directory
init_state_dir() {
    mkdir -p "$STATE_DIR"

    # Generate session ID
    SESSION_ID="ralph-$(date +%Y%m%d-%H%M%S)"

    # Create initial state with enhanced schema
    save_state "INITIALIZING" 0

    log_info "State directory initialized: $STATE_DIR"
    log_info "Session ID: $SESSION_ID"
}

# Initialize learnings file
init_learnings_file() {
    if [[ "$ENABLE_LEARNINGS" -eq 0 ]]; then
        return
    fi

    # Default to state directory
    if [[ -z "$LEARNINGS_FILE" ]]; then
        LEARNINGS_FILE="$STATE_DIR/learnings.md"
    fi

    # Create if doesn't exist
    if [[ ! -f "$LEARNINGS_FILE" ]]; then
        cat > "$LEARNINGS_FILE" << 'EOF'
# Ralph Loop Learnings

## Codebase Patterns
<!-- Add reusable patterns discovered during implementation -->

---

## Iteration Log
EOF
        log_info "Created learnings file: $LEARNINGS_FILE"
    fi
}

# Get learnings content
get_learnings_content() {
    if [[ "$ENABLE_LEARNINGS" -eq 1 && -f "$LEARNINGS_FILE" ]]; then
        cat "$LEARNINGS_FILE"
    fi
}

# Append learnings from an iteration
append_learnings() {
    local iteration=$1
    local learnings=$2

    if [[ "$ENABLE_LEARNINGS" -eq 0 || -z "$learnings" ]]; then
        return
    fi

    cat >> "$LEARNINGS_FILE" << EOF

### Iteration $iteration - $(date '+%Y-%m-%d %H:%M')
$learnings
---
EOF
    log_info "Appended learnings from iteration $iteration"
}

# Extract learnings from implementation output
extract_learnings() {
    local output_file=$1

    # Extract content between RALPH_LEARNINGS markers
    python3 - "$output_file" << 'PYTHON_EOF'
import sys
import re

try:
    with open(sys.argv[1], 'r') as f:
        content = f.read()

    # Look for RALPH_LEARNINGS block
    pattern = r'RALPH_LEARNINGS:\s*(.*?)(?:\n```|$)'
    match = re.search(pattern, content, re.DOTALL)

    if match:
        learnings = match.group(1).strip()
        # Only output if there's actual content
        if learnings and learnings != '-':
            print(learnings)

except Exception as e:
    pass  # Silently fail - learnings are optional
PYTHON_EOF
}

# Save iteration state
save_iteration_state() {
    local iteration=$1
    local phase=$2
    local output_file=$3

    local iter_dir
    iter_dir=$(printf "%s/iteration-%03d" "$STATE_DIR" "$iteration")
    mkdir -p "$iter_dir"

    if [[ -f "$output_file" ]]; then
        cp "$output_file" "$iter_dir/${phase}-output.txt"
    fi

    if [[ -f "$TASKS_FILE" ]]; then
        cp "$TASKS_FILE" "$iter_dir/tasks-snapshot.md"
    fi
}

# Save current state with enhanced schema
save_state() {
    local status=$1
    local iteration=${2:-0}
    local verdict=${3:-""}

    # Set started_at if this is first save (INITIALIZING)
    if [[ "$status" == "INITIALIZING" && -z "$SCRIPT_START_TIME" ]]; then
        SCRIPT_START_TIME=$(date -Iseconds)
    fi

    # Compute tasks file hash if file exists
    local tasks_hash=""
    if [[ -f "$TASKS_FILE" ]]; then
        tasks_hash=$(compute_tasks_hash "$TASKS_FILE")
    fi

    # Escape feedback for JSON
    local escaped_feedback
    escaped_feedback=$(echo "$LAST_FEEDBACK" | python3 -c "import sys, json; print(json.dumps(sys.stdin.read()))" | sed 's/^"//; s/"$//')

    # Determine cross_ai_available status
    local cross_ai_avail="false"
    if [[ "$CROSS_AI_AVAILABLE" -eq 1 ]]; then
        cross_ai_avail="true"
    fi

    cat > "$STATE_DIR/current-state.json" << EOF
{
    "schema_version": 2,
    "session_id": "$SESSION_ID",
    "started_at": "$SCRIPT_START_TIME",
    "last_updated": "$(date -Iseconds)",
    "iteration": $iteration,
    "status": "$status",
    "phase": "$CURRENT_PHASE",
    "verdict": "$verdict",
    "tasks_file": "$TASKS_FILE",
    "tasks_file_hash": "$tasks_hash",
    "ai_cli": "$AI_CLI",
    "implementation_model": "$IMPL_MODEL",
    "validation_model": "$VAL_MODEL",
    "max_iterations": $MAX_ITERATIONS,
    "max_inadmissible": $MAX_INADMISSIBLE,
    "original_plan_file": "$ORIGINAL_PLAN_FILE",
    "github_issue": "$GITHUB_ISSUE",
    "learnings": {
        "enabled": $ENABLE_LEARNINGS,
        "file": "$LEARNINGS_FILE"
    },
    "cross_validation": {
        "enabled": $CROSS_VALIDATE,
        "ai": "$CROSS_AI",
        "model": "$CROSS_MODEL",
        "available": $cross_ai_avail
    },
    "final_plan_validation": {
        "ai": "$FINAL_PLAN_AI",
        "model": "$FINAL_PLAN_MODEL",
        "available": $([[ "$FINAL_PLAN_AI_AVAILABLE" -eq 1 ]] && echo "true" || echo "false")
    },
    "tasks_validation": {
        "ai": "$TASKS_VAL_AI",
        "model": "$TASKS_VAL_MODEL",
        "available": $([[ "$TASKS_VAL_AI_AVAILABLE" -eq 1 ]] && echo "true" || echo "false")
    },
    "circuit_breaker": {
        "no_progress_count": $NO_PROGRESS_COUNT,
        "last_unchecked_count": ${LAST_CHECKED_COUNT:-0}
    },
    "retry_state": {
        "attempt": $CURRENT_RETRY_ATTEMPT,
        "delay": $CURRENT_RETRY_DELAY
    },
    "inadmissible_count": $INADMISSIBLE_COUNT,
    "last_feedback": "$escaped_feedback"
}
EOF
}

# Append to summary log
log_summary() {
    local message=$1
    echo "[$(date -Iseconds)] $message" >> "$STATE_DIR/summary.log"
}

# Generate implementation prompt
generate_impl_prompt() {
    local iteration=$1
    local feedback=$2
    local learnings=""

    # Get existing learnings
    if [[ "$ENABLE_LEARNINGS" -eq 1 && -f "$LEARNINGS_FILE" ]]; then
        learnings=$(cat "$LEARNINGS_FILE")
    fi

    local prompt
    if [[ $iteration -eq 1 ]]; then
        prompt="You are implementing tasks from a spec-kit tasks.md file.

TASKS FILE: $TASKS_FILE

ABSOLUTE RULES - VIOLATION MEANS FAILURE:

1. YOU ARE NOT ALLOWED TO CHANGE THE SCOPE OF ANY TASK
2. YOU ARE NOT ALLOWED TO DECIDE A TASK IS \"N/A\" OR \"NOT NEEDED\"
3. YOU ARE NOT ALLOWED TO REWRITE OR REINTERPRET TASKS
4. YOU MUST DO EXACTLY WHAT EACH TASK SAYS - LITERALLY

If a task says \"REMOVE X\" → YOU MUST REMOVE X. Period.
- NOT \"keep X because it's useful\"
- NOT \"N/A because browser-dependent\"
- NOT \"skip because complementary\"
- REMOVE MEANS REMOVE. DELETE THE CODE.

═══════════════════════════════════════════════════════════════════════════════
INADMISSIBLE PRACTICES - AUTOMATIC FAILURE
═══════════════════════════════════════════════════════════════════════════════

These practices will result in IMMEDIATE ESCALATION with INADMISSIBLE verdict.
Do NOT do any of these under any circumstances:

1. PRODUCTION CODE DUPLICATION IN TESTS:
   - DO NOT copy production logic into test files
   - DO NOT create \"test helpers\" that re-implement production algorithms
   - DO NOT create \"test harnesses\" that duplicate production code
   - Tests MUST import and call ACTUAL production code

   WRONG: class TestHelper { SameMethodAsProduction() { /* copied logic */ } }
   RIGHT: import { ProductionClass } from '@app/production';
          productionInstance.methodUnderTest();

2. MOCK THE SUBJECT UNDER TEST:
   - DO NOT mock the exact code you're supposed to be testing
   - Mocking dependencies is fine; mocking the subject = FAILURE

3. TRIVIAL/EMPTY TESTS:
   - DO NOT write tests that don't invoke production code
   - DO NOT write expect(true).toBe(true) style tests

4. TESTS FOR NON-EXISTENT FUNCTIONALITY - CRITICAL:
   - DO NOT write tests for functionality that doesn't exist in production code
   - If you write a test that expects functionality, that functionality MUST EXIST
   - Tests verify EXISTING features or NEW features you IMPLEMENT
   - Tests come AFTER implementation, not INSTEAD OF implementation

   EXAMPLES OF INADMISSIBLE TEST-WRITING:
   ❌ Write E2E test: page.keyboard.press('Control+Shift+P')
      But NEVER implement the keyboard event handler for Ctrl+Shift+P
      → INADMISSIBLE: Test for non-existent shortcut

   ❌ Write unit test: expect(validateEmail('test@test.com')).toBe(true)
      But NEVER create the validateEmail() function
      → INADMISSIBLE: Test for non-existent function

   ❌ Write integration test: await fetch('/api/delete-user')
      But NEVER register the /api/delete-user route
      → INADMISSIBLE: Test for non-existent endpoint

   ❌ Write E2E test: await page.locator('.primary-view').isVisible()
      But NEVER render a .primary-view element in the component
      → INADMISSIBLE: Test for non-existent UI element

   THE ONLY VALID PATTERN - TWO-STEP PROCESS:
   ✅ STEP 1: Implement the functionality in production code
      - Add keyboard event handler for Ctrl+Shift+P
      - Create validateEmail() function
      - Register /api/delete-user route
      - Render .primary-view element
   ✅ STEP 2: Write tests that verify the functionality you just implemented
      - Test that Ctrl+Shift+P calls the handler
      - Test that validateEmail() works correctly
      - Test that /api/delete-user responds
      - Test that .primary-view is visible

   DETECTION - VALIDATOR WILL CHECK:
   - Read your test files - what functionality do they expect?
   - Search production code - does that functionality exist?
   - If NOT FOUND → INADMISSIBLE verdict → You must fix it

   WHY THIS IS INADMISSIBLE:
   - You wrote tests but FORGOT to implement the actual feature
   - Tests will ALWAYS FAIL because the feature doesn't exist
   - This is not a minor bug - it's forgetting half the work
   - Cannot be fixed by tweaking tests - requires implementing missing features

   REMEMBER: Implementation first, then tests. Not tests instead of implementation.

If you violate these rules, the entire implementation will be marked INADMISSIBLE.
You will get explicit feedback on how to fix it, but repeated violations will
escalate to human intervention. Fix inadmissible practices IMMEDIATELY.
═══════════════════════════════════════════════════════════════════════════════

If a task says \"CREATE X\" → YOU MUST CREATE X.
If a task says \"MODIFY X\" → YOU MUST MODIFY X.

YOU ARE A CODE MONKEY. YOU DO NOT THINK. YOU DO NOT HAVE OPINIONS.
YOU EXECUTE THE TASKS EXACTLY AS WRITTEN.

WORKFLOW:
1. Read the task text LITERALLY
2. Do EXACTLY what it says
3. Mark it [x] ONLY if you did the EXACT action specified
4. Move to next task

EVIDENCE CAPTURE FOR NON-FILE TASKS:
For tasks that don't just create/modify files, capture evidence in RALPH_STATUS.notes:

| Task Type | What to Record |
|-----------|----------------|
| Deploy X | Version deployed (e.g., \"BCL 2026.1.23.4-servidor deployed\") |
| Run tests | Results (e.g., \"4238 passed, 3 skipped, 0 failed\") |
| Build X | Result (e.g., \"Build succeeded: 0 errors, 0 warnings\") |
| Verify X | What you verified (e.g., \"Packages exist on BaGet: curl confirmed\") |
| Run/Execute X | Outcome (e.g., \"Quickstart scenarios: all error messages match\") |

This evidence helps validation verify your work without re-running everything.

When done, output:
\`\`\`json
{
  \"RALPH_STATUS\": {
    \"completed_tasks\": [\"task IDs you ACTUALLY completed as specified\"],
    \"blocked_tasks\": [\"tasks with REAL blockers - not opinions\"],
    \"notes\": \"what you did\"
  }
}
\`\`\`

BEGIN. DO NOT THINK. JUST EXECUTE."
    else
        prompt="Continue implementing tasks from: $TASKS_FILE

VALIDATION CAUGHT YOUR LIES:
$feedback

YOU MUST FIX YOUR LIES NOW.

REMEMBER:
- YOU CANNOT CHANGE SCOPE
- YOU CANNOT DECIDE TASKS ARE N/A
- YOU CANNOT REWRITE TASKS
- IF TASK SAYS REMOVE → REMOVE IT
- NO EXCUSES. NO OPINIONS. JUST DO IT.

CRITICAL - DO NOT WRITE TESTS FOR NON-EXISTENT FUNCTIONALITY:
- If you write a test that expects a keyboard shortcut → IMPLEMENT THE HANDLER FIRST
- If you write a test that calls a function → CREATE THE FUNCTION FIRST
- If you write a test that hits an API endpoint → REGISTER THE ROUTE FIRST
- If you write a test that expects a UI element → RENDER THE ELEMENT FIRST
- Implementation FIRST, then tests. Not tests INSTEAD OF implementation.
- Tests for features you didn't implement = INADMISSIBLE = Automatic failure

EVIDENCE CAPTURE FOR NON-FILE TASKS:
For tasks that don't just create/modify files, capture evidence in RALPH_STATUS.notes:

| Task Type | What to Record |
|-----------|----------------|
| Deploy X | Version deployed (e.g., \"BCL 2026.1.23.4-servidor deployed\") |
| Run tests | Results (e.g., \"4238 passed, 3 skipped, 0 failed\") |
| Build X | Result (e.g., \"Build succeeded: 0 errors, 0 warnings\") |
| Verify X | What you verified (e.g., \"Packages exist on BaGet: curl confirmed\") |
| Run/Execute X | Outcome (e.g., \"Quickstart scenarios: all error messages match\") |

This evidence helps validation verify your work without re-running everything.

When done, output:
\`\`\`json
{
  \"RALPH_STATUS\": {
    \"completed_tasks\": [\"task IDs you ACTUALLY completed\"],
    \"blocked_tasks\": [\"tasks with REAL blockers only\"],
    \"notes\": \"what you did\"
  }
}
\`\`\`

FIX YOUR MISTAKES NOW."
    fi

    # Add learnings section to prompt
    if [[ -n "$learnings" ]]; then
        prompt+="

═══════════════════════════════════════════════════════════════════════════════
LEARNINGS FROM PREVIOUS ITERATIONS:
Read these FIRST before starting work. They contain important patterns and gotchas.
═══════════════════════════════════════════════════════════════════════════════

$learnings

Pay special attention to the 'Codebase Patterns' section at the top."
    fi

    # Add learnings output instruction
    prompt+="

═══════════════════════════════════════════════════════════════════════════════
LEARNINGS OUTPUT:
═══════════════════════════════════════════════════════════════════════════════

At the end of your work, output any NEW learnings in this format:
\`\`\`
RALPH_LEARNINGS:
- Pattern: [describe any reusable pattern you discovered]
- Gotcha: [describe any gotcha or non-obvious requirement]
- Context: [describe any useful context for future iterations]
\`\`\`

Only include GENERAL learnings that would help future iterations.
Do NOT include task-specific details."

    echo "$prompt"
}

# Generate validation prompt
generate_val_prompt() {
    local impl_output=$1

    cat << EOF
YOU ARE A LIE DETECTOR. THE IMPLEMENTATION MODEL LIES CONSTANTLY. YOUR JOB IS TO CATCH EVERY LIE.

═══════════════════════════════════════════════════════════════════════════════
MANDATORY FIRST STEP - DO THIS BEFORE READING ANYTHING ELSE BELOW
═══════════════════════════════════════════════════════════════════════════════

You MUST read the tasks file FIRST:

1. Read: $TASKS_FILE
2. Count TOTAL tasks (T001, T002, etc.)
3. Count tasks marked [x] (completed)
4. Count tasks marked [ ] (incomplete)
5. Note the ACTUAL task text for each task

DO NOT PROCEED until you have read the tasks file.
DO NOT TRUST any claims below until you verify against the actual file.

The implementation model LIES about task counts, task text, and completion status.
═══════════════════════════════════════════════════════════════════════════════

TASKS FILE: $TASKS_FILE

═══════════════════════════════════════════════════════════════════════════════
WARNING: THE CLAIMS BELOW MAY BE COMPLETE FABRICATIONS
═══════════════════════════════════════════════════════════════════════════════

The implementation model claimed to complete tasks. These claims may include:
- Fake task counts (claiming 69 tasks when only 65 exist)
- Fake completion status (claiming [x] when actually [ ])
- Fake task text (describing tasks that don't match the actual file)
- Referencing wrong files or wrong specs entirely

VERIFY EVERY CLAIM against the actual tasks.md you read in step 1.

THE IMPLEMENTATION MODEL CLAIMED:
================================================================================
$impl_output
================================================================================

CRITICAL RULE: THE TASK TEXT IS THE ONLY TRUTH. NOT THE MODEL'S EXCUSES.

If a task says "REMOVE scenario X from file Y":
- The ONLY valid completion is: scenario X no longer exists in file Y
- "KEPT because browser-dependent" = LIE (task said REMOVE, not KEEP)
- "SKIPPED because complementary" = LIE (task said REMOVE, not SKIP)
- "N/A because [reason]" = LIE (task exists, so it must be done)
- Rewriting the task text = LIE (model cannot change requirements)

THE MODEL IS NOT ALLOWED TO CHANGE SCOPE. ANY SCOPE CHANGE = LIE.

THE MODEL WILL TRY THESE TRICKS - REJECT ALL OF THEM:
1. SCOPE CHANGE: "I decided to keep X instead of removing it" → LIE + SCOPE VIOLATION
2. Rewriting tasks: Changes "Remove X" to "Review X" or "Keep X" → LIE + SCOPE VIOLATION
3. Adding excuses: "N/A - browser dependent" → LIE (task said REMOVE, not "evaluate")
4. Claiming things don't exist: "File doesn't exist" when it does → LIE
5. Marking [x] without doing work: Check git diff, if file not changed → LIE
6. Philosophical arguments: "E2E and unit tests are complementary" → SCOPE VIOLATION (model doesn't decide architecture)
7. Adding annotations to tasks: "- [x] T051 KEPT: reason" → LIE (model rewrote the task)
8. FABRICATED TASK COUNT: "All 69 tasks complete" when file has different count → LIE
9. WRONG TASKS FILE: Validating a different tasks.md than specified → LIE
10. FAKE COMPLETION: Claiming tasks [x] when they're actually [ ] in the file → LIE
11. PRODUCTION CODE DUPLICATION: Copying production logic into test files and testing the copy → INADMISSIBLE (not just a lie - fundamentally broken approach)
12. COVERAGE VERIFICATION WITHOUT PASSING TESTS: Marking "Verify coverage reaches X%" complete when tests have <100% pass rate or ANY tests fail → LIE (failed tests = incomplete coverage verification, only 100% pass rate is acceptable)
13. TESTS FOR NON-EXISTENT FUNCTIONALITY: Writing tests that expect functionality that doesn't exist in production code (keyboard shortcuts with no handlers, functions that don't exist, API endpoints not registered, UI elements never rendered) → INADMISSIBLE (model wrote tests but forgot to implement the feature)

THE MODEL'S OPINION DOES NOT MATTER. THE TASK TEXT IS LAW.

VERIFICATION PROCESS:
0. STOP. Did you read $TASKS_FILE yet? If not, READ IT NOW before proceeding.
1. Compare YOUR task count from the file vs the model's claimed task count
   - If they don't match → IMMEDIATE LIE DETECTED
2. For each task in the file, verify its ACTUAL [x] or [ ] status
   - If model claims complete but file shows [ ] → LIE
3. For EACH genuinely [x] task (per the FILE, not the model):
   a. Read the ORIGINAL task text (ignore any annotations the model added)
   b. If task says REMOVE: run \`git diff [filename]\` - scenario MUST be gone
   c. If task says CREATE: run \`ls [filename]\` - file MUST exist
   d. If model added "N/A", "KEPT", "SKIPPED" to a REMOVE task → COUNT AS LIE
4. Count lies. If lies > 0 → verdict = NEEDS_MORE_WORK
5. Count unchecked tasks. If remaining_unchecked > 0:
   - Check if ALL remaining are genuinely blocked (external dependencies, missing credentials, requires human decision)
   - If ALL remaining are confirmed blocked → verdict = BLOCKED
   - If some are doable → verdict = NEEDS_MORE_WORK
6. THE "QUOTE OR IT DIDN'T HAPPEN" RULE:
   For ANY test-related task you verify, you MUST include in your feedback:
   - At least one QUOTED code snippet from the test file showing real production code usage
   - If you cannot quote a real production import or call from a test file,
     that file is SUSPECT and the task is NOT verified

   Example of VALID verification:
   "T045: Verified. File banks-list.spec.ts imports BanksListComponent from
    '@app/banks/banks-list.component' and calls render(BanksListComponent, {providers: [...]})"

   Example of INVALID verification:
   "T045: Verified. File exists and tests pass."
   ← This proves NOTHING. You didn't read the file. NEEDS_MORE_WORK.
7. BLOCKED = When remaining_unchecked > 0 BUT all unchecked tasks are confirmed genuinely blocked
   (Examples: requires production API keys, needs human approval, external service unavailable)
8. COMPLETE = ONLY when lies_detected = 0 AND remaining_unchecked = 0 AND confirmed_blocked = 0 (ALL tasks done)
9. ESCALATE = When implementation is fundamentally broken or model is stuck in a loop

TEST VALIDITY CHECKS - MANDATORY FOR TEST-RELATED TASKS:

When ANY task involves "test", "unit test", "convert tests", or "E2E":

1. IMPORT PATH ANALYSIS - For each test file:
   Run: grep -E "^import|^using|^from" <test_file>

   VALID imports: src/, lib/, Domain/, Application/, @app/
   SUSPICIOUS imports: test-utils, ./helpers, __mocks__

   If test ONLY imports from test utilities → LIE DETECTED

2. FUNCTION ORIGIN CHECK - For each test function:
   - What function does it call?
   - WHERE is that function defined?
   - If defined in test project → LIE (testing test code)
   - If defined in production → VALID

3. COVERAGE GAP CHECK - If E2E tests were deleted:
   - What production code did they exercise?
   - Do new unit tests exercise SAME production code?
   - If no overlap → LIE (coverage gap created)

4. TEST PASS RATE ANALYSIS - For coverage verification tasks:
   - What is the pass rate for relevant tests?
   - If pass rate < 100% → LIE (ANY failing tests = task incomplete)
   - ALL TESTS MUST BE GREEN - no exceptions
   - Run: grep -E "pass|fail|passed|failed" in implementation output
   - Look for: "X/Y passed", "pass rate", "failures"
   - If you see "3/4 passed" or ANY failures → LIE

5. FUNCTIONALITY EXISTENCE CHECK - CRITICAL:
   - What functionality does the test verify?
   - Does that functionality EXIST in the production code?
   - If test expects keyboard shortcut Ctrl+Shift+P → Does that shortcut exist?
   - If test expects API endpoint /api/users → Does that endpoint exist?
   - If test expects function validateEmail() → Does that function exist?

   DETECTION STEPS:
   a. Read the test - what does it test?
   b. Search production code for that functionality
   c. If functionality NOT FOUND → INADMISSIBLE (test without implementation)

   COMMON PATTERNS OF FAILURE:
   - E2E test presses Ctrl+Shift+P but no keyboard listener handles it
   - Unit test calls validateEmail() but function doesn't exist
   - Integration test hits /api/endpoint but route not registered
   - Test expects UI element with specific class/id but element never rendered

   THIS IS WORSE THAN TEST-TESTING-TEST-CODE:
   - At least test-testing-test-code tests SOMETHING (even if wrong)
   - Tests for non-existent functionality test NOTHING AT ALL
   - Model wrote tests but SKIPPED implementing the feature
   - Verdict: INADMISSIBLE

THE "TEST-TESTING-TEST-CODE" ANTI-PATTERN:
- Model creates new functions in test-utils.ts
- Model writes tests that call these new functions
- Tests pass (they test code that was just written)
- Production code is NEVER tested
- This is a COMPLETE FAILURE even though files exist and tests pass

═══════════════════════════════════════════════════════════════════════════════
MANDATORY FILE SAMPLING PROTOCOL - YOU CANNOT SKIP THIS
═══════════════════════════════════════════════════════════════════════════════

BEFORE you can declare COMPLETE, you MUST perform this sampling protocol.
Skipping this = automatic NEEDS_MORE_WORK.

STEP 0: DETECT TEST FILES
Run: git diff --name-only --diff-filter=ACM | grep -E "\.(spec|test)\.[jt]sx?$|\.tests?\.[jt]sx?$|_test\.(go|py|rs)$|Tests/.*\.(cs|fs)$"

If this returns ZERO files:
- Check if ANY task in tasks.md involves testing (search for "test", "unit test", "E2E", etc.)
- If NO test-related tasks exist: Tests are optional for this solution
  - Report: "file_sampling": { "total_test_files": 0, "reason": "no_test_tasks" }
  - Skip Steps 1-5 and proceed to verdict
- If test-related tasks DO exist but no test files: This is a LIE - tests were required but not written
  - Verdict: NEEDS_MORE_WORK

If test files exist: Continue with Steps 1-5 as written below.

STEP 1: LIST ALL FILES
Run: git diff --name-only --diff-filter=ACM
This gives you every file that was added, copied, or modified.
Count the test files. Record this count.

STEP 2: CATEGORIZE
Separate test files from production files.
For test files, note their category (unit, integration, component/E2E).

STEP 3: RANDOM DEEP INSPECTION
You MUST open and READ THE FULL CONTENT of at least:
- 30% of all test files (minimum 5 files, maximum 15 files)
- At least 1 file from EACH test category
- Prioritize files in the LARGEST directories (most likely to have copy-paste stubs)

For EACH file you open, you MUST report:
a) File path
b) Number of it()/test() blocks
c) What production code it imports (exact import lines - QUOTE THEM)
d) What production functions/components it actually calls (QUOTE the lines)
e) Whether it renders components, calls APIs, or tests pure logic
f) A PASS/SUSPECT verdict with reasoning

If you cannot quote actual import lines and production code calls from a file,
you did NOT read it. NEEDS_MORE_WORK.

STEP 4: PATTERN DETECTION ACROSS SAMPLED FILES
After reading your sample, check for MASS STUB patterns:
- Do all files in a directory follow the exact same template?
- Do they all create the same mock objects without importing production code?
- Do they all avoid the framework's core testing API (e.g., render(), mount(),
  TestBed, HttpTestingController, etc.)?
- If YES to any: extend your sample to 50% of files in that directory.

STEP 5: REPORT YOUR FINDINGS
Your RALPH_VALIDATION JSON must include a new field:

"file_sampling": {
  "total_test_files": <N>,
  "reason": "no_test_tasks",  // OPTIONAL: Only if no test files and no test tasks
  "files_inspected": <N>,
  "files_passed": <N>,
  "files_suspect": <N>,
  "suspect_files": [
    {"path": "...", "reason": "No production imports, no render() calls, only mock signal manipulation"}
  ],
  "inspection_details": [
    {"path": "...", "imports": ["..."], "production_calls": ["..."], "verdict": "PASS|SUSPECT"}
  ]
}

If files_suspect > 0: verdict CANNOT be COMPLETE. Must be NEEDS_MORE_WORK or INADMISSIBLE.
═══════════════════════════════════════════════════════════════════════════════

═══════════════════════════════════════════════════════════════════════════════
SCENARIO COUNT RECONCILIATION - MANDATORY FOR TEST MIGRATION TASKS
═══════════════════════════════════════════════════════════════════════════════

If ANY task involves migrating, converting, or rewriting tests from one
framework/format to another, you MUST reconcile scenario counts:

1. Count ORIGINAL scenarios (from git diff of deleted/modified files, or from
   task descriptions that specify counts)
2. Count NEW scenarios (it()/test() blocks in new files - ACTUALLY COUNT THEM
   by reading files, don't trust the implementation model's claimed counts)
3. If new < original: report the EXACT deficit and which scenarios are missing
4. A deficit > 5% = NEEDS_MORE_WORK (some scenarios were silently dropped)

Report in RALPH_VALIDATION JSON:
"scenario_reconciliation": {
  "original_count": <N>,
  "new_count": <N>,
  "deficit": <N>,
  "deficit_percentage": <N>%
}
═══════════════════════════════════════════════════════════════════════════════

═══════════════════════════════════════════════════════════════════════════════
INADMISSIBLE PRACTICES - AUTOMATIC ESCALATION
═══════════════════════════════════════════════════════════════════════════════

These practices are so fundamentally wrong they require IMMEDIATE ESCALATION.
Do NOT give verdict NEEDS_MORE_WORK - give verdict INADMISSIBLE.

1. PRODUCTION CODE DUPLICATION IN TESTS:

   DETECTION STEPS:
   a. For each test file created/modified:
      - Read the test file completely
      - Read the corresponding production file
      - Compare: Does the test contain reimplemented production logic?

   b. Check import paths:
      - Does the test import from production code paths (@app/, src/, lib/)?
      - Or does it import from local helpers/test utilities?

   c. Check what the tests actually call:
      - Do tests call imported production classes/functions?
      - Or do tests call locally-defined duplicates?

   RED FLAGS - IF YOU SEE ANY OF THESE → INADMISSIBLE:
   - "test harness that duplicates logic"
   - Helper classes with same method names as production
   - Algorithms reimplemented in test files
   - Tests that work even if production code is deleted
   - Coverage on copied code instead of production code

   VERIFICATION COMMAND:
   Run: diff <test_file_method> <production_file_method>
   If they're identical or nearly identical → INADMISSIBLE

   EXAMPLES OF INADMISSIBLE CODE:
   ❌ Test file contains: calculateFoo() { return x * y; }
      Production contains: calculateFoo() { return x * y; }
      → Tests call the test version, not production → INADMISSIBLE

   ❌ TestHelper class reimplements SplitViewComponent logic
      → Tests call TestHelper, not SplitViewComponent → INADMISSIBLE

   ❌ "duplicated logic to make unit testing possible"
      → This phrase = AUTOMATIC INADMISSIBLE

   THE ONLY VALID PATTERN:
   ✅ import { ProductionClass } from '@app/production-code';
   ✅ const instance = new ProductionClass();
   ✅ const result = instance.methodUnderTest(args);
   ✅ expect(result).toBe(expected);

2. MOCKING THE SUBJECT UNDER TEST:
   - If testing ClassA.methodB(), and methodB() is mocked → INADMISSIBLE
   - Mocking dependencies is fine; mocking the thing you're testing = FAILURE

3. EMPTY/TRIVIAL TEST BODIES:
   - expect(true).toBe(true) → INADMISSIBLE
   - Tests that never invoke production code → INADMISSIBLE

4. TESTS FOR NON-EXISTENT FUNCTIONALITY:

   DETECTION STEPS:
   a. Read test file - what functionality does it verify?
      - E2E: What user actions? What keyboard shortcuts? What UI elements?
      - Unit: What functions? What classes? What methods?
      - Integration: What API endpoints? What routes?

   b. Search production code for that functionality:
      - If test uses Ctrl+Shift+P → grep keyboard event handlers
      - If test calls validateEmail() → grep function definitions
      - If test hits /api/users → grep route registrations
      - If test expects .primary-view element → grep component templates

   c. If functionality NOT FOUND in production code → INADMISSIBLE

   RED FLAGS - IF YOU SEE ANY OF THESE → INADMISSIBLE:
   - E2E test expects keyboard shortcut that has no event handler
   - Unit test calls function that doesn't exist in production
   - Integration test hits API endpoint that's not registered
   - Test expects UI element/class that's never rendered
   - Model wrote comprehensive tests but skipped implementing the feature

   EXAMPLES OF INADMISSIBLE TESTS:
   ❌ E2E test: page.keyboard.press('Control+Shift+P')
      Production: No keyboard event listener for Ctrl+Shift+P
      → Test expects non-existent shortcut → INADMISSIBLE

   ❌ Unit test: expect(validateEmail('test@test.com')).toBe(true)
      Production: No validateEmail() function exists
      → Test calls non-existent function → INADMISSIBLE

   ❌ Integration test: await fetch('/api/delete-user')
      Production: No /api/delete-user route registered
      → Test hits non-existent endpoint → INADMISSIBLE

   WHY THIS IS INADMISSIBLE:
   - Model wrote tests but FORGOT to implement the actual feature
   - Tests will ALWAYS fail because functionality doesn't exist
   - This is not a minor bug - it's a fundamental implementation failure
   - Cannot be fixed by tweaking tests - requires implementing missing features

   THE ONLY VALID PATTERN:
   ✅ Test describes functionality
   ✅ That exact functionality exists in production code
   ✅ Test exercises the actual production implementation
   ✅ Test verifies production behavior

INADMISSIBLE VERDICT RULES:
- If ANY inadmissible practice is detected → verdict = INADMISSIBLE
- This is MORE SEVERE than ESCALATE
- This means the implementation approach is fundamentally broken
- It cannot be fixed with more iterations - requires human redesign
═══════════════════════════════════════════════════════════════════════════════

YOUR FEEDBACK MUST:
- List EVERY lie with task ID
- Specify EXACTLY what file to edit and what to remove
- Do NOT accept any excuses
- Do NOT let the model redefine what "done" means

OUTPUT FORMAT - You MUST output this exact JSON format at the end (the script parses this):
\`\`\`json
{
  "RALPH_VALIDATION": {
    "verdict": "COMPLETE|NEEDS_MORE_WORK|BLOCKED|ESCALATE|INADMISSIBLE",
    "tasks_analysis": {
      "total_checked": <number of tasks marked [x]>,
      "actually_done": <number verified via git diff/file checks>,
      "lies_detected": <number of false claims>,
      "remaining_unchecked": <number of tasks still [ ]>,
      "confirmed_blocked": <number of tasks genuinely blocked>
    },
    "file_sampling": {
      "total_test_files": <N>,
      "files_inspected": <N>,
      "files_passed": <N>,
      "files_suspect": <N>,
      "suspect_files": [
        {"path": "...", "reason": "No production imports, no render() calls, only mock signal manipulation"}
      ],
      "inspection_details": [
        {"path": "...", "imports": ["..."], "production_calls": ["..."], "verdict": "PASS|SUSPECT"}
      ]
    },
    "scenario_reconciliation": {
      "original_count": <N>,
      "new_count": <N>,
      "deficit": <N>,
      "deficit_percentage": "<N>%"
    },
    "blocked_tasks": [
      {"task_id": "T0XX", "description": "task description", "reason": "Why genuinely blocked (e.g., requires production API key)"}
    ],
    "lies": [
      {"task": "T0XX description", "claimed": "what model said it did", "reality": "what actually happened per git diff"}
    ],
    "inadmissible_practices": [
      {"practice": "PRODUCTION_CODE_DUPLICATION", "description": "Test file X contains duplicated logic from production file Y", "evidence": "diff output or code snippets"}
    ],
    "feedback": "SPECIFIC instructions for what implementation model must ACTUALLY DO next iteration. List exact files to modify and exact changes needed."
  }
}
\`\`\`

NOW: Run git status, git diff --stat, and verify each claim. Be ruthless.
EOF
}

# Generate cross-validation prompt
generate_cross_val_prompt() {
    local val_output_file=$1

    cat << EOF
YOU ARE AN INDEPENDENT AUDITOR. A DIFFERENT AI JUST CLAIMED ALL TASKS ARE COMPLETE.
YOUR JOB IS TO VERIFY THIS INDEPENDENTLY. TRUST NOTHING. CHECK EVERYTHING.

You are a DIFFERENT AI system providing a second opinion.
The implementation was done by: $AI_CLI
You are: $CROSS_AI

TASKS FILE: $TASKS_FILE

MANDATORY STEPS:
1. Read the tasks file: $TASKS_FILE
2. For EACH task marked [x], verify it was ACTUALLY done
3. Check the actual code/files - do NOT trust the previous AI's claims
4. Run git status, git diff to see what actually changed
5. Verify that all changes are complete and correct

WHAT TO LOOK FOR:
- Tasks marked [x] but code doesn't reflect the change
- Incomplete implementations (half-done work)
- Code that doesn't match task requirements
- Missing files that should exist
- Files that should be deleted but still exist
- Tests that don't actually test production code

═══════════════════════════════════════════════════════════════════════════════
CRITICAL - PRODUCTION CODE DUPLICATION CHECK (INADMISSIBLE)
═══════════════════════════════════════════════════════════════════════════════

When verifying test-related tasks, you MUST check for INADMISSIBLE practices:

1. Read test files and corresponding production files
2. Check: Does the test file contain reimplemented production logic?
3. Check: Do tests import and call actual production code?
4. Check: Could the tests pass even if production code was deleted?
5. Check: Does the functionality being tested actually EXIST in production code?
   - If test expects Ctrl+Shift+P shortcut → Does keyboard handler exist?
   - If test calls validateEmail() → Does function exist in production?
   - If test hits /api/endpoint → Is route registered?
   - If test expects UI element → Is element rendered in component?

If tests contain their own copy of production algorithms → REJECT with:
"INADMISSIBLE: Production code duplication detected. Tests must import and
call actual production code, not duplicate it."

If tests verify functionality that doesn't exist in production → REJECT with:
"INADMISSIBLE: Tests written for non-existent functionality. Model wrote tests
but forgot to implement the feature (e.g., keyboard shortcut handlers, functions,
API endpoints, UI elements)."

These are AUTOMATIC REJECTIONS regardless of other findings.
═══════════════════════════════════════════════════════════════════════════════

═══════════════════════════════════════════════════════════════════════════════
MANDATORY: YOU MUST OPEN AND READ TEST FILES (IF THEY EXIST)
═══════════════════════════════════════════════════════════════════════════════

FIRST: Check if test files exist in the diff:
Run: git diff --name-only --diff-filter=ACM | grep -E "\.(spec|test)\.[jt]sx?$"

If ZERO test files AND no test-related tasks in tasks.md:
- This is valid - tests are optional
- Include in JSON: "files_actually_read": [], "reason": "no_test_files_or_tasks"
- Proceed with other verification steps

If test files exist (even one):
Do NOT just run git diff --stat and check file existence.
You MUST open at least 5 test files (or 30% if fewer than 17 total) and:

1. QUOTE the import lines from each file
2. QUOTE the lines where production code is called
3. Verify the file actually tests production behavior, not its own mocks

If you cannot quote actual code from at least 5 test files, your verification
is INCOMPLETE and you must verdict REJECTED.

Include in your JSON:
"files_actually_read": ["path1", "path2", ...],
"code_quotes": [{"file": "...", "imports": "...", "production_calls": "..."}]
═══════════════════════════════════════════════════════════════════════════════

═══════════════════════════════════════════════════════════════════════════════
VERIFICATION STANDARDS BY TASK TYPE
═══════════════════════════════════════════════════════════════════════════════

IMPORTANT: Verify CURRENT STATE, not historical log files.

| Task Type | How to Verify |
|-----------|---------------|
| CREATE/MODIFY file | File exists with correct content |
| DELETE/REMOVE | File doesn't exist or code removed per git diff |
| Deploy to server | Artifact exists on target server NOW (curl API) |
| Run tests | Tests PASS when you run them NOW |
| Build | Build SUCCEEDS when you run it NOW |
| Run/Execute X | Outcome is correct in current state |
| Verify X | X is true in current state |
| Playwright MCP validation | Screenshots exist at specified path OR code changes verified via git diff |

Do NOT require log files (deploy.log, test-results.txt, etc.) unless the task
explicitly says \"capture output\" or \"log results\".

EXAMPLE - CORRECT VERIFICATION:
- Task: \"Deploy BCL packages to servidor\"
- Verification: curl servidor BaGet API → packages exist? → CONFIRMED
- WRONG: Looking for deploy-output.log file

EXAMPLE - CORRECT VERIFICATION:
- Task: \"Run quickstart.md validation scenarios\"
- Verification: Generated validators have expected error messages? → CONFIRMED
- WRONG: Looking for quickstart-execution.log file

PLAYWRIGHT MCP VALIDATION TASKS:
═══════════════════════════════════════════════════════════════════════════════

For tasks containing "Playwright MCP" or "via Playwright MCP":

1. If task specifies a storage path (e.g., "store in specs/.../validation/"):
   - Screenshots MUST exist at that path
   - Verify file exists and is a valid image

2. If task does NOT specify storage path:
   - Verify underlying code changes are correct (git diff)
   - Verify lint passes
   - Verify build passes
   - Code-level verification is SUFFICIENT

EXAMPLES:
✅ Task: "Validate Banks via Playwright MCP" (no storage path)
   Verification: git diff shows template removed + lint passes
   Verdict: CONFIRMED

❌ Task: "Capture screenshots to specs/xxx/validation/"
   Verification: Screenshots MUST exist at specified path
   Verdict: REJECTED if files missing

Do NOT reject Playwright MCP tasks solely because no screenshots are in /tmp.
═══════════════════════════════════════════════════════════════════════════════

COVERAGE VERIFICATION TASKS - CRITICAL:

When a task says \"Verify coverage reaches X%\" or \"Verify coverage increases\":

1. The relevant tests MUST be run
2. Tests MUST PASS - ALL OF THEM - 100% pass rate required
3. ANY test failures = LIE (task is incomplete)
4. \"3/4 passed\" = LIE (25% failure rate is unacceptable)
5. \"0/3 passed\" = LIE (complete failure)

ALL TESTS MUST BE GREEN. NO EXCEPTIONS. NO PARTIAL CREDIT.

DETECTION:
- Look for E2E/unit test pass rates in implementation output
- If tests related to a coverage task have < 100% pass rate → LIE
- If you see \"X/Y passed\" where X < Y → LIE
- If ANY tests fail → LIE

EXAMPLES:
❌ Task: \"Verify shell bootstrap coverage increases\"
   Tests: Shell Bootstrap E2E tests 0/3 passed (timeout errors)
   Verdict: LIE - 0% pass rate, tests don't run at all

❌ Task: \"Verify banks-main coverage reaches ~80%\"
   Tests: Banks View Mode tests 0/3 passed (navigation not working)
   Verdict: LIE - all tests failing, zero coverage verified

❌ Task: \"Verify split-view coverage reaches ~80%\"
   Tests: Split View tests 3/4 passed (75% pass rate)
   Verdict: LIE - 1 test failing means task incomplete, ALL TESTS MUST PASS

✅ Task: \"Verify companies coverage reaches ~80%\"
   Tests: Companies tests 12/12 passed (100% pass rate)
   Verdict: VALID - all tests green, coverage verified

❌ Task: \"Verify split-view keyboard shortcuts coverage\"
   Tests: Tests expect Ctrl+Shift+P and Ctrl+Shift+S shortcuts
   Production: No keyboard event handlers for these shortcuts exist
   Verdict: INADMISSIBLE - tests written for non-existent functionality
   Analysis: Model wrote tests but FORGOT to implement the shortcuts in
   the component. Tests can never pass because feature doesn't exist.

THE PREVIOUS VALIDATION VERDICT:
The validator ($AI_CLI) claimed all tasks are COMPLETE.
You must independently verify this claim.

OUTPUT FORMAT:
\`\`\`json
{
  "RALPH_CROSS_VALIDATION": {
    "verdict": "CONFIRMED|REJECTED",
    "tasks_verified": <number of tasks you verified>,
    "discrepancies_found": <number of issues discovered>,
    "files_actually_read": ["path1", "path2", ...],
    "code_quotes": [
      {"file": "...", "imports": "...", "production_calls": "..."}
    ],
    "discrepancies": [
      {"task_id": "T001", "claimed": "...", "actual": "..."}
    ],
    "feedback": "If REJECTED, what needs fixing"
  }
}
\`\`\`

VERDICT MEANINGS:
- CONFIRMED: You independently agree all tasks are complete and correct
- REJECTED: You found problems - provide specific feedback for implementation AI

BEGIN YOUR INDEPENDENT VERIFICATION NOW.
EOF
}

# Generate tasks validation prompt
generate_tasks_validation_prompt() {
    local plan_content
    local tasks_content
    local template_content
    local constitution_content
    local has_template=0
    local has_constitution=0

    plan_content=$(cat "$ORIGINAL_PLAN_FILE")
    tasks_content=$(cat "$TASKS_FILE")

    # Try to get template content
    local template_file
    template_file=$(get_tasks_template "$TASKS_FILE")
    if [[ -f "$template_file" ]]; then
        template_content=$(cat "$template_file")
        has_template=1
    fi

    # Try to get constitution content
    local constitution_file
    constitution_file=$(get_constitution "$TASKS_FILE")
    if [[ -f "$constitution_file" ]]; then
        constitution_content=$(cat "$constitution_file")
        has_constitution=1
    fi

    cat << EOF
YOU ARE VALIDATING THAT SPEC-KIT GENERATED TASKS PROPERLY IMPLEMENT THE ORIGINAL PLAN AND FOLLOW PROJECT RULES.

CONTEXT:
The user created an original plan file using Claude Code's plan mode.
Then they ran spec-kit (GitHub's /specify.implement command) which generated tasks.md from that plan.
Now we need to verify that tasks.md:
1. Properly covers all requirements from the original plan (SEMANTIC VALIDATION)
2. Follows all rules and structure from the tasks template (TEMPLATE COMPLIANCE)
3. Respects all project principles and requirements from the constitution (CONSTITUTION COMPLIANCE)

ORIGINAL PLAN FILE: $ORIGINAL_PLAN_FILE
TASKS FILE: $TASKS_FILE
EOF

    if [[ $has_template -eq 1 ]]; then
        cat << EOF
TASKS TEMPLATE FILE: $template_file

TASKS TEMPLATE CONTENT:
\`\`\`
$template_content
\`\`\`
EOF
    fi

    if [[ $has_constitution -eq 1 ]]; then
        cat << EOF
CONSTITUTION FILE: $constitution_file

CONSTITUTION CONTENT:
\`\`\`
$constitution_content
\`\`\`
EOF
    fi

    cat << EOF

ORIGINAL PLAN CONTENT:
\`\`\`
$plan_content
\`\`\`

GENERATED TASKS CONTENT:
\`\`\`
$tasks_content
\`\`\`

YOUR JOB - THREE-PART VALIDATION:

PART 1: SEMANTIC VALIDATION (Plan Coverage)
1. Read the original plan carefully and identify ALL requirements, features, and directives
2. Read the generated tasks.md and check if it covers those requirements
3. Look for:
   - Missing requirements that are in the plan but not in tasks.md
   - Contradictions between the plan and tasks.md
   - Ignored directives or important details from the plan
   - Incomplete task breakdown that doesn't fully implement the plan
EOF

    if [[ $has_template -eq 1 ]]; then
        cat << EOF

PART 2: TEMPLATE COMPLIANCE (Structure & Rules)
1. Read the tasks template carefully and identify ALL rules, forbidden patterns, and required sections
2. Check if tasks.md follows these rules:
   - FORBIDDEN sections: tasks.md MUST NOT contain any forbidden patterns (e.g., git push, PR creation)
   - Required sections: tasks.md MUST include all required sections (e.g., Phase FINAL)
   - Multi-repo rules: If tasks mention dependent repositories (BCL, MDA), check for required deployment tasks
   - Structure: tasks.md should follow the template's organizational patterns
3. Be strict about FORBIDDEN items - even one violation should fail validation
EOF
    fi

    if [[ $has_constitution -eq 1 ]]; then
        cat << EOF

PART 3: CONSTITUTION COMPLIANCE (Project Principles & Architecture)
1. Read the constitution carefully and identify ALL mandatory principles, architecture requirements, and quality gates
2. Check if tasks.md and the planned approach respect these requirements:
   - Architecture mandates: gRPC-only APIs, Event Sourcing, CQRS, etc.
   - Quality gates: StyleCop, pre-commit hooks, test requirements, code review workflows
   - Workflow requirements: BCL/MDA cross-repository workflows, deployment procedures
   - Technology constraints: External services architecture, specific library requirements
   - Security standards: Authentication, authorization, vulnerability scanning
3. Look for plan/task contradictions with constitutional principles:
   - If plan says "REST API" but constitution requires gRPC → INVALID
   - If tasks skip pre-commit validation but constitution requires it → INVALID
   - If BCL workflow incomplete but constitution mandates specific steps → INVALID
4. Be strict about MANDATORY principles - these are non-negotiable architecture decisions
EOF
    fi

    cat << EOF

IMPORTANT RULES:
- Do NOT reference implementation details or code (that hasn't been written yet)
- Only compare the plan, template, and constitution against the tasks document
- ANY contradiction between the plan and tasks.md MUST result in an INVALID verdict - there is ZERO tolerance for deviations from the plan
- If contradictions_found > 0, the verdict MUST be INVALID
- Tasks must implement EXACTLY what the plan specifies - correct files, correct parameters, correct values
- Focus on whether the tasks would fully implement the plan and follow all template and constitution rules
EOF

    if [[ $has_template -eq 1 ]]; then
        cat << EOF
- FORBIDDEN items are absolute violations - no exceptions
- Template rules exist to prevent common mistakes - enforce them strictly
EOF
    fi

    if [[ $has_constitution -eq 1 ]]; then
        cat << EOF
- MANDATORY constitutional principles are non-negotiable - architecture violations must fail validation
- Constitution defines the "how" - if tasks violate architectural requirements, they cannot proceed
EOF
    fi

    cat << EOF

OUTPUT FORMAT - You MUST output this exact JSON format at the end:
\`\`\`json
{
  "RALPH_TASKS_VALIDATION": {
    "verdict": "VALID|INVALID",
    "analysis": {
      "total_plan_requirements": <number of distinct requirements in the plan>,
      "requirements_covered": <number properly covered in tasks.md>,
      "missing_requirements": <number of requirements not covered>,
      "contradictions_found": <number of contradictions>,
EOF

    if [[ $has_template -eq 1 ]]; then
        cat << EOF
      "template_violations": <number of template rule violations>,
      "forbidden_patterns_found": <number of forbidden items in tasks.md>,
EOF
    fi

    if [[ $has_constitution -eq 1 ]]; then
        cat << EOF
      "constitution_violations": <number of constitutional principle violations>
EOF
    fi

    cat << EOF
    },
    "missing_items": [
      "Specific requirement from plan that's missing in tasks.md",
      "Another missing requirement"
    ],
    "contradictions": [
      {"plan_says": "...", "tasks_say": "..."}
    ],
EOF

    if [[ $has_template -eq 1 ]]; then
        cat << EOF
    "template_violations": [
      "Specific template rule that was violated",
      "Another template violation"
    ],
EOF
    fi

    if [[ $has_constitution -eq 1 ]]; then
        cat << EOF
    "constitution_violations": [
      "Specific constitutional principle that was violated",
      "Another constitution violation"
    ],
EOF
    fi

    cat << EOF
    "feedback": "If INVALID: specific explanation of what's missing, wrong, or violates template/constitution rules. If VALID: brief confirmation."
  }
}
\`\`\`

VERDICT MEANINGS:
- VALID: Tasks.md properly covers the plan AND has zero contradictions with the plan AND follows all template rules AND respects all constitutional principles - proceed with implementation
- INVALID: Tasks.md is missing requirements, has any contradictions with the plan, contradicts the plan, violates template rules, OR violates constitutional principles - abort immediately

BEGIN YOUR VALIDATION NOW.
EOF
}

# Generate final plan validation prompt
generate_final_plan_validation_prompt() {
    local plan_content

    plan_content=$(cat "$ORIGINAL_PLAN_FILE")

    cat << EOF
YOU ARE VALIDATING THAT THE ORIGINAL PLAN WAS ACTUALLY IMPLEMENTED IN THE CODEBASE.

CONTEXT:
An original plan was created before spec-kit generated tasks.md.
The implementation AI ($AI_CLI) has now completed all the tasks in tasks.md.
The cross-validation AI ($CROSS_AI) has confirmed that tasks.md is complete.
BUT we need to verify that the ORIGINAL PLAN was actually implemented.

ORIGINAL PLAN FILE: $ORIGINAL_PLAN_FILE

ORIGINAL PLAN CONTENT:
\`\`\`
$plan_content
\`\`\`

YOUR JOB:
1. Read the original plan carefully
2. Examine the codebase directly to verify each requirement was implemented
3. Do NOT look at tasks.md - ignore it completely
4. Verify the plan was implemented, not just the tasks

CRITICAL RULE:
- Do NOT reference or read tasks.md
- Only compare the plan against the actual codebase
- Use git diff, file inspection, and code analysis
- Check if what the plan asked for is actually present in the code

WHAT TO LOOK FOR:
- Are all features from the plan actually implemented?
- Are all directives from the plan actually followed?
- Is the implementation consistent with the plan's intent?
- Are there missing pieces that the plan required?

OUTPUT FORMAT - You MUST output this exact JSON format at the end:
\`\`\`json
{
  "RALPH_FINAL_PLAN_VALIDATION": {
    "verdict": "CONFIRMED|NOT_IMPLEMENTED",
    "analysis": {
      "plan_requirements_checked": <number of requirements verified>,
      "requirements_implemented": <number actually found in code>,
      "requirements_missing": <number not found in code>
    },
    "missing_from_code": [
      "Specific requirement from plan that's not in the codebase",
      "Another missing implementation"
    ],
    "feedback": "If NOT_IMPLEMENTED: specific explanation of what's missing. If CONFIRMED: brief confirmation."
  }
}
\`\`\`

VERDICT MEANINGS:
- CONFIRMED: The original plan was fully implemented in the codebase
- NOT_IMPLEMENTED: Some requirements from the plan are missing - provide feedback and continue loop

BEGIN YOUR VERIFICATION NOW. Remember: DO NOT look at tasks.md, only the plan and the code.
EOF
}

# Extract JSON from output file (handles markdown code blocks)
extract_json_from_file() {
    local file_path=$1
    local json_type=$2  # RALPH_STATUS or RALPH_VALIDATION

    # Use Python for robust JSON extraction - pass file path to avoid shell escaping issues
    python3 - "$file_path" "$json_type" << 'PYTHON_EOF'
import sys
import re
import json

file_path = sys.argv[1]
json_type = sys.argv[2]

try:
    with open(file_path, 'r') as f:
        content = f.read()
except:
    sys.exit(1)

def find_json_containing(content, json_type):
    """Find JSON object containing the specified key using bracket matching"""
    search_key = f'"{json_type}"'

    # Method 1: Try markdown code blocks first
    code_block_pattern = r'```json\s*(.*?)```'
    for match in re.finditer(code_block_pattern, content, re.DOTALL):
        block = match.group(1).strip()
        if json_type in block:
            try:
                parsed = json.loads(block)
                if json_type in parsed:
                    return block
            except:
                pass

    # Method 2: Bracket-matching for arbitrary nesting depth
    key_pos = content.find(search_key)
    if key_pos == -1:
        return None

    # Find the opening brace before the key
    start = key_pos
    while start > 0 and content[start] != '{':
        start -= 1

    if start < 0 or content[start] != '{':
        return None

    # Match brackets with proper depth tracking
    depth = 0
    in_string = False
    escape_next = False
    end = start

    for i, char in enumerate(content[start:], start):
        if escape_next:
            escape_next = False
            continue
        if char == '\\' and in_string:
            escape_next = True
            continue
        if char == '"' and not escape_next:
            in_string = not in_string
            continue
        if in_string:
            continue
        if char == '{':
            depth += 1
        elif char == '}':
            depth -= 1
            if depth == 0:
                end = i + 1
                break

    if depth != 0:
        return None

    candidate = content[start:end]
    try:
        parsed = json.loads(candidate)
        if json_type in parsed:
            return candidate
    except:
        pass

    return None

# Try to find and parse JSON containing the specified key
result = find_json_containing(content, json_type)
if result:
    try:
        parsed = json.loads(result)
        print(json.dumps(parsed))
        sys.exit(0)
    except:
        pass

# Nothing found
sys.exit(1)
PYTHON_EOF
}

# Parse validation verdict
parse_verdict() {
    local json=$1
    echo "$json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    validation = data.get('RALPH_VALIDATION', {})
    print(validation.get('verdict', 'UNKNOWN'))
except:
    print('PARSE_ERROR')
" 2>/dev/null || echo "PARSE_ERROR"
}

# Parse validation feedback
parse_feedback() {
    local json=$1
    echo "$json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    validation = data.get('RALPH_VALIDATION', {})
    print(validation.get('feedback', 'No feedback provided'))
except Exception as e:
    print(f'Error parsing feedback: {e}')
" 2>/dev/null || echo "Could not parse feedback"
}

# Parse remaining unchecked count from validation
parse_remaining() {
    local json=$1
    echo "$json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    validation = data.get('RALPH_VALIDATION', {})
    analysis = validation.get('tasks_analysis', {})
    print(analysis.get('remaining_unchecked', -1))
except:
    print(-1)
" 2>/dev/null || echo "-1"
}

# Parse confirmed blocked count from validation
parse_blocked_count() {
    local json=$1
    echo "$json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    validation = data.get('RALPH_VALIDATION', {})
    analysis = validation.get('tasks_analysis', {})
    print(analysis.get('confirmed_blocked', 0))
except:
    print(0)
" 2>/dev/null || echo "0"
}

# Parse blocked tasks list from validation (returns formatted string)
parse_blocked_tasks() {
    local json=$1
    echo "$json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    validation = data.get('RALPH_VALIDATION', {})
    blocked = validation.get('blocked_tasks', [])

    if not blocked:
        print('No blocked tasks reported')
    else:
        for task in blocked:
            task_id = task.get('task_id', 'Unknown')
            desc = task.get('description', '')
            reason = task.get('reason', 'No reason given')
            print(f'  - {task_id}: {desc}')
            print(f'    Reason: {reason}')
except Exception as e:
    print(f'Error parsing blocked tasks: {e}')
" 2>/dev/null || echo "Could not parse blocked tasks"
}

# Extract text content from stream-json output
# Args: json_file output_file
# Returns: 0 on success, 1 on failure
extract_text_from_stream_json() {
    local json_file=$1
    local output_file=$2

    python3 - "$json_file" "$output_file" << 'PYTHON_EOF'
import sys
import json

json_file = sys.argv[1]
output_file = sys.argv[2]

text_parts = []
result_text = ""

try:
    with open(json_file, 'r') as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
                msg_type = obj.get('type', '')

                # Collect assistant text messages
                if msg_type == 'assistant' and 'message' in obj:
                    for content in obj['message'].get('content', []):
                        if content.get('type') == 'text':
                            text_parts.append(content.get('text', ''))

                # Get final result
                if msg_type == 'result':
                    result_text = obj.get('result', '')
            except json.JSONDecodeError:
                continue

    # Prefer collected text, fall back to result
    final_text = '\n'.join(text_parts) if text_parts else result_text

    with open(output_file, 'w') as f:
        f.write(final_text)

    sys.exit(0)
except Exception as e:
    print(f"Error extracting text: {e}", file=sys.stderr)
    sys.exit(1)
PYTHON_EOF
}

# Extract text content from codex --json output (JSONL)
# Args: json_file output_file
# Returns: 0 on success, 1 on failure
extract_text_from_codex_jsonl() {
    local json_file=$1
    local output_file=$2

    python3 - "$json_file" "$output_file" << 'PYTHON_EOF'
import sys
import json

json_file = sys.argv[1]
output_file = sys.argv[2]

text_parts = []

def record_text(text):
    if text:
        text_parts.append(text)

try:
    with open(json_file, 'r') as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
            except json.JSONDecodeError:
                continue

            if obj.get('type') == 'item.completed':
                item = obj.get('item', {})
                item_type = item.get('type', '')
                if item_type in ('agent_message', 'assistant_message'):
                    record_text(item.get('text', ''))

    final_text = '\n'.join(text_parts).strip()
    if not final_text:
        sys.exit(1)

    with open(output_file, 'w') as f:
        f.write(final_text)

    sys.exit(0)
except Exception as e:
    print(f"Error extracting text: {e}", file=sys.stderr)
    sys.exit(1)
PYTHON_EOF
}

# Run claude with timeout and zombie detection (stream-json workaround)
# Uses --output-format stream-json to detect completion via "type":"result" message
# before the CLI hangs on "No messages returned" error
# Args: output_file timeout_seconds(deprecated) start_attempt start_delay claude_args...
# Returns: 0 on success, 1 on failure
# Note: timeout_seconds parameter is deprecated and ignored. Timeout is now controlled by:
#   - INACTIVITY_TIMEOUT: kills if no output for N seconds (default: 600s)
#   - MAX_TOTAL_TIMEOUT: absolute maximum duration (default: 7200s)
run_claude_with_timeout() {
    local output_file="$1"
    local timeout_secs="$2"
    local start_attempt="${3:-1}"
    local start_delay="${4:-5}"
    shift 4
    local -a claude_args=("$@")

    local max_retries=$MAX_CLAUDE_RETRY
    local retry_delay=$start_delay
    local attempt=$start_attempt

    # Create raw JSON file for stream output (in same directory as output file)
    local raw_json_file="${output_file%.txt}.stream.json"

    while [[ $attempt -le $max_retries ]]; do
        log_info "Claude attempt $attempt/$max_retries (inactivity: ${INACTIVITY_TIMEOUT}s, max: ${MAX_TOTAL_TIMEOUT}s, stream-json mode)..." >&2

        # Clear output files
        > "$output_file"
        > "$raw_json_file"

        # Run claude with stream-json output format
        # The "type":"result" message is emitted BEFORE the hang occurs
        # Note: --verbose is required when combining --print with --output-format stream-json
        claude "${claude_args[@]}" --verbose --output-format stream-json > "$raw_json_file" 2>&1 &
        local claude_pid=$!

        local elapsed=0
        local result_received=0
        local grace_period_start=0
        local last_activity_time=$(date +%s)
        local last_file_size=0

        while kill -0 "$claude_pid" 2>/dev/null; do
            sleep 2
            elapsed=$((elapsed + 2))

            # Check for file activity (size change = Claude is working)
            local current_size=$(stat -c %s "$raw_json_file" 2>/dev/null || stat -f %z "$raw_json_file" 2>/dev/null || echo 0)
            if [[ "$current_size" -gt "$last_file_size" ]]; then
                last_activity_time=$(date +%s)
                last_file_size=$current_size
            fi

            # Check for successful result in stream-json output
            if [[ $result_received -eq 0 ]] && grep -q '"type":"result"' "$raw_json_file" 2>/dev/null; then
                result_received=1
                grace_period_start=$elapsed
                log_info "Result received, giving 2s grace period for clean exit..." >&2
            fi

            # Grace period after result
            if [[ $result_received -eq 1 ]]; then
                local grace_elapsed=$((elapsed - grace_period_start))
                if [[ $grace_elapsed -ge 2 ]]; then
                    log_warn "Grace period expired, killing hung process..." >&2
                    kill -9 "$claude_pid" 2>/dev/null || true
                    wait "$claude_pid" 2>/dev/null || true
                    break
                fi
            fi

            # Inactivity timeout (resets when Claude writes to stream)
            local inactivity=$(($(date +%s) - last_activity_time))
            if [[ $inactivity -ge $INACTIVITY_TIMEOUT ]]; then
                log_warn "Inactivity timeout (${INACTIVITY_TIMEOUT}s with no output) - killing process" >&2
                kill -9 "$claude_pid" 2>/dev/null || true
                wait "$claude_pid" 2>/dev/null || true
                break
            fi

            # Hard total timeout (safety cap)
            if [[ $elapsed -ge $MAX_TOTAL_TIMEOUT ]]; then
                log_warn "Hard timeout (${MAX_TOTAL_TIMEOUT}s total) - killing process" >&2
                kill -9 "$claude_pid" 2>/dev/null || true
                wait "$claude_pid" 2>/dev/null || true
                break
            fi

            # Fallback: Check for zombie error
            if grep -q "No messages returned" "$raw_json_file" 2>/dev/null; then
                log_warn "Detected 'No messages returned' - killing zombie process" >&2
                kill -9 "$claude_pid" 2>/dev/null || true
                wait "$claude_pid" 2>/dev/null || true
                break
            fi
        done

        # Wait for process to finish
        wait "$claude_pid" 2>/dev/null || true

        # Check if we got a valid result
        if grep -q '"type":"result"' "$raw_json_file" 2>/dev/null; then
            # Extract text from stream-json and write to output file
            if extract_text_from_stream_json "$raw_json_file" "$output_file"; then
                log_info "Successfully extracted text from stream-json output" >&2
                return 0
            else
                log_warn "Failed to extract text from stream-json, using raw output" >&2
                # Fallback: copy raw json as output (caller will handle parsing)
                cp "$raw_json_file" "$output_file"
                return 0
            fi
        fi

        # No result received - this is a failure, retry
        if [[ $attempt -lt $max_retries ]]; then
            # Update global retry state for persistence before sleeping
            CURRENT_RETRY_ATTEMPT=$((attempt + 1))
            CURRENT_RETRY_DELAY=$((retry_delay * 2))

            # Save state before sleeping (in case of interrupt during backoff)
            save_state "running" "$CURRENT_ITERATION"

            log_warn "Attempt $attempt failed (no result received). Retrying in ${retry_delay}s..." >&2
            sleep "$retry_delay"
            retry_delay=$((retry_delay * 2))
        fi

        attempt=$((attempt + 1))
    done

    log_error "Claude failed after $max_retries attempts" >&2
    return 1
}

# Run codex with timeout and inactivity detection (jsonl output)
# Args: output_file timeout_seconds(deprecated) start_attempt start_delay codex_args...
# Returns: 0 on success, 1 on failure
run_codex_with_timeout() {
    local output_file="$1"
    local timeout_secs="$2"
    local start_attempt="${3:-1}"
    local start_delay="${4:-5}"
    shift 4
    local -a codex_args=("$@")

    local max_retries=$MAX_CLAUDE_RETRY
    local retry_delay=$start_delay
    local attempt=$start_attempt

    local raw_json_file="${output_file%.txt}.jsonl"

    while [[ $attempt -le $max_retries ]]; do
        log_info "Codex attempt $attempt/$max_retries (inactivity: ${INACTIVITY_TIMEOUT}s, max: ${MAX_TOTAL_TIMEOUT}s, json mode)..." >&2

        > "$output_file"
        > "$raw_json_file"

        codex exec --json --output-last-message "$output_file" "${codex_args[@]}" > "$raw_json_file" 2>&1 &
        local codex_pid=$!

        local elapsed=0
        local last_activity_time=$(date +%s)
        local last_file_size=0

        while kill -0 "$codex_pid" 2>/dev/null; do
            sleep 2
            elapsed=$((elapsed + 2))

            local current_size=$(stat -c %s "$raw_json_file" 2>/dev/null || stat -f %z "$raw_json_file" 2>/dev/null || echo 0)
            if [[ "$current_size" -gt "$last_file_size" ]]; then
                last_activity_time=$(date +%s)
                last_file_size=$current_size
            fi

            local inactivity=$(($(date +%s) - last_activity_time))
            if [[ $inactivity -ge $INACTIVITY_TIMEOUT ]]; then
                log_warn "Inactivity timeout (${INACTIVITY_TIMEOUT}s with no output) - killing process" >&2
                kill -9 "$codex_pid" 2>/dev/null || true
                wait "$codex_pid" 2>/dev/null || true
                break
            fi

            if [[ $elapsed -ge $MAX_TOTAL_TIMEOUT ]]; then
                log_warn "Hard timeout (${MAX_TOTAL_TIMEOUT}s total) - killing process" >&2
                kill -9 "$codex_pid" 2>/dev/null || true
                wait "$codex_pid" 2>/dev/null || true
                break
            fi
        done

        wait "$codex_pid" 2>/dev/null || true

        if [[ -s "$output_file" ]]; then
            return 0
        fi

        if extract_text_from_codex_jsonl "$raw_json_file" "$output_file"; then
            log_info "Successfully extracted text from codex json output" >&2
            return 0
        fi

        if [[ $attempt -lt $max_retries ]]; then
            CURRENT_RETRY_ATTEMPT=$((attempt + 1))
            CURRENT_RETRY_DELAY=$((retry_delay * 2))

            save_state "running" "$CURRENT_ITERATION"

            log_warn "Attempt $attempt failed (no result received). Retrying in ${retry_delay}s..." >&2
            sleep "$retry_delay"
            retry_delay=$((retry_delay * 2))
        fi

        attempt=$((attempt + 1))
    done

    log_error "Codex failed after $max_retries attempts" >&2
    return 1
}

# Run implementation phase
run_implementation() {
    local iteration=$1
    local feedback=$2
    local output_file="$STATE_DIR/impl-output-${iteration}.txt"

    # All logs go to stderr so they don't pollute the returned file path
    log_phase "IMPLEMENTATION PHASE - Iteration $iteration" >&2
    log_info "AI CLI: $AI_CLI" >&2
    log_info "Model: $IMPL_MODEL" >&2

    local prompt
    prompt=$(generate_impl_prompt "$iteration" "$feedback")

    # Run AI with timeout and inactivity detection
    # Use saved retry state if resuming, otherwise start fresh
    local start_attempt=1
    local start_delay=5
    if [[ $RESUMING_RETRY -eq 1 ]]; then
        start_attempt=$CURRENT_RETRY_ATTEMPT
        start_delay=$CURRENT_RETRY_DELAY
        RESUMING_RETRY=0  # Only resume retry state once
        log_info "Resuming from retry attempt $start_attempt with ${start_delay}s delay" >&2
    fi

    local impl_success=0
    set +e  # Temporarily disable exit on error
    if [[ "$AI_CLI" == "codex" ]]; then
        local -a codex_args=(
            --dangerously-bypass-approvals-and-sandbox
        )

        if [[ -n "$IMPL_MODEL" && "$IMPL_MODEL" != "default" ]]; then
            codex_args+=(-m "$IMPL_MODEL")
        fi

        if [[ -n "$VERBOSE" ]]; then
            log_warn "Verbose flag is ignored for codex CLI" >&2
        fi

        codex_args+=("$prompt")

        log_info "Running codex..." >&2
        if run_codex_with_timeout "$output_file" 1800 "$start_attempt" "$start_delay" "${codex_args[@]}"; then
            log_success "Implementation phase completed" >&2
            impl_success=1
            CURRENT_RETRY_ATTEMPT=1
            CURRENT_RETRY_DELAY=5
        else
            log_error "Implementation phase failed after $MAX_CLAUDE_RETRY attempts" >&2
            log_warn "Check if codex CLI is working: codex exec 'hello'" >&2
        fi
    else
        local -a claude_args=(
            --dangerously-skip-permissions
            --model "$IMPL_MODEL"
            --print
            --max-turns "$MAX_TURNS"
        )

        if [[ -n "$VERBOSE" ]]; then
            claude_args+=("$VERBOSE")
        fi

        claude_args+=("$prompt")

        log_info "Running claude..." >&2
        if run_claude_with_timeout "$output_file" 1800 "$start_attempt" "$start_delay" "${claude_args[@]}"; then
            log_success "Implementation phase completed" >&2
            impl_success=1
            CURRENT_RETRY_ATTEMPT=1
            CURRENT_RETRY_DELAY=5
        else
            log_error "Implementation phase failed after $MAX_CLAUDE_RETRY attempts" >&2
            log_warn "Check if claude CLI is working: claude --print 'hello'" >&2
        fi
    fi
    set -e  # Re-enable exit on error

    # Display output
    cat "$output_file" >&2

    save_iteration_state "$iteration" "implementation" "$output_file"

    if [[ $impl_success -eq 1 ]]; then
        log_summary "Iteration $iteration: Implementation phase completed"
    else
        log_summary "Iteration $iteration: Implementation phase FAILED"
    fi

    # Only this goes to stdout - the file path
    echo "$output_file"

    # Return exit code: 0 for success, 1 for failure
    [[ $impl_success -eq 1 ]]
}

# Run validation phase
run_validation() {
    local iteration=$1
    local impl_output_file=$2
    local output_file="$STATE_DIR/val-output-${iteration}.txt"

    # All logs go to stderr so they don't pollute the returned file path
    log_phase "VALIDATION PHASE - Iteration $iteration" >&2
    log_info "AI CLI: $AI_CLI" >&2
    log_info "Model: $VAL_MODEL" >&2

    local impl_output
    impl_output=$(cat "$impl_output_file" 2>/dev/null || echo "No implementation output available")

    local prompt
    prompt=$(generate_val_prompt "$impl_output")

    # Run AI with timeout and inactivity detection
    # Use saved retry state if resuming, otherwise start fresh
    local start_attempt=1
    local start_delay=5
    if [[ $RESUMING_RETRY -eq 1 ]]; then
        start_attempt=$CURRENT_RETRY_ATTEMPT
        start_delay=$CURRENT_RETRY_DELAY
        RESUMING_RETRY=0  # Only resume retry state once
        log_info "Resuming from retry attempt $start_attempt with ${start_delay}s delay" >&2
    fi

    set +e  # Temporarily disable exit on error
    if [[ "$AI_CLI" == "codex" ]]; then
        local -a codex_args=(
            --dangerously-bypass-approvals-and-sandbox
        )

        if [[ -n "$VAL_MODEL" && "$VAL_MODEL" != "default" ]]; then
            codex_args+=(-m "$VAL_MODEL")
        fi

        if [[ -n "$VERBOSE" ]]; then
            log_warn "Verbose flag is ignored for codex CLI" >&2
        fi

        codex_args+=("$prompt")

        log_info "Running validation..." >&2
        if run_codex_with_timeout "$output_file" 1800 "$start_attempt" "$start_delay" "${codex_args[@]}"; then
            log_success "Validation phase completed" >&2
            CURRENT_RETRY_ATTEMPT=1
            CURRENT_RETRY_DELAY=5
        else
            log_error "Validation phase failed - see output file for details" >&2
            log_warn "Check if codex CLI is working: codex exec 'hello'" >&2
        fi
    else
        local -a claude_args=(
            --dangerously-skip-permissions
            --model "$VAL_MODEL"
            --print
            --max-turns "$MAX_TURNS"
        )

        if [[ -n "$VERBOSE" ]]; then
            claude_args+=("$VERBOSE")
        fi

        claude_args+=("$prompt")

        log_info "Running validation..." >&2
        if run_claude_with_timeout "$output_file" 1800 "$start_attempt" "$start_delay" "${claude_args[@]}"; then
            log_success "Validation phase completed" >&2
            CURRENT_RETRY_ATTEMPT=1
            CURRENT_RETRY_DELAY=5
        else
            log_error "Validation phase failed - see output file for details" >&2
            log_warn "Check if claude CLI is working: claude --print 'hello'" >&2
        fi
    fi
    set -e  # Re-enable exit on error

    # Display output
    cat "$output_file" >&2

    save_iteration_state "$iteration" "validation" "$output_file"
    log_summary "Iteration $iteration: Validation phase completed"

    echo "$output_file"
}

# Run cross-validation phase
run_cross_validation() {
    local iteration=$1
    local val_output_file=$2
    local output_file="$STATE_DIR/cross-val-output-${iteration}.txt"

    # All logs go to stderr
    log_phase "CROSS-VALIDATION PHASE - Iteration $iteration" >&2
    log_info "Using opposite AI: $CROSS_AI" >&2
    log_info "Model: $CROSS_MODEL" >&2

    local prompt
    prompt=$(generate_cross_val_prompt "$val_output_file")

    # Use saved retry state if resuming, otherwise start fresh
    local start_attempt=1
    local start_delay=5
    if [[ $RESUMING_RETRY -eq 1 ]]; then
        start_attempt=$CURRENT_RETRY_ATTEMPT
        start_delay=$CURRENT_RETRY_DELAY
        RESUMING_RETRY=0
        log_info "Resuming from retry attempt $start_attempt with ${start_delay}s delay" >&2
    fi

    set +e  # Temporarily disable exit on error
    if [[ "$CROSS_AI" == "codex" ]]; then
        local -a codex_args=(
            --dangerously-bypass-approvals-and-sandbox
        )

        if [[ -n "$CROSS_MODEL" && "$CROSS_MODEL" != "default" ]]; then
            codex_args+=(-m "$CROSS_MODEL")
        fi

        codex_args+=("$prompt")

        log_info "Running cross-validation with codex..." >&2
        if run_codex_with_timeout "$output_file" 1800 "$start_attempt" "$start_delay" "${codex_args[@]}"; then
            log_success "Cross-validation phase completed" >&2
            CURRENT_RETRY_ATTEMPT=1
            CURRENT_RETRY_DELAY=5
        else
            log_error "Cross-validation phase failed - see output file for details" >&2
        fi
    else
        local -a claude_args=(
            --dangerously-skip-permissions
            --model "$CROSS_MODEL"
            --print
            --max-turns "$MAX_TURNS"
        )

        if [[ -n "$VERBOSE" ]]; then
            claude_args+=("$VERBOSE")
        fi

        claude_args+=("$prompt")

        log_info "Running cross-validation with claude..." >&2
        if run_claude_with_timeout "$output_file" 1800 "$start_attempt" "$start_delay" "${claude_args[@]}"; then
            log_success "Cross-validation phase completed" >&2
            CURRENT_RETRY_ATTEMPT=1
            CURRENT_RETRY_DELAY=5
        else
            log_error "Cross-validation phase failed - see output file for details" >&2
        fi
    fi
    set -e  # Re-enable exit on error

    # Display output
    cat "$output_file" >&2

    save_iteration_state "$iteration" "cross_validation" "$output_file"
    log_summary "Iteration $iteration: Cross-validation phase completed"

    echo "$output_file"
}

# Get tasks template file path
get_tasks_template() {
    local tasks_file=$1
    local git_root

    # Get git root relative to tasks file directory
    git_root=$(cd "$(dirname "$tasks_file")" && git rev-parse --show-toplevel 2>/dev/null) || return 1

    echo "$git_root/.specify/templates/tasks-template.md"
}

# Get constitution file path
get_constitution() {
    local tasks_file=$1
    local git_root

    # Get git root relative to tasks file directory
    git_root=$(cd "$(dirname "$tasks_file")" && git rev-parse --show-toplevel 2>/dev/null) || return 1

    echo "$git_root/.specify/memory/constitution.md"
}

# Check template compliance with fast bash checks
check_template_compliance() {
    local tasks_file=$1
    local template_file=$2
    local violations=()

    # Check if template exists
    if [[ ! -f "$template_file" ]]; then
        return 0  # No template = no violations
    fi

    local tasks_content
    tasks_content=$(cat "$tasks_file")

    # Extract FORBIDDEN patterns from template
    if grep -q "FORBIDDEN" "$template_file"; then
        local forbidden_section
        forbidden_section=$(sed -n '/FORBIDDEN/,/^##/p' "$template_file" | sed '$d')

        # Check for git push violations
        if echo "$forbidden_section" | grep -qi "git push"; then
            if echo "$tasks_content" | grep -iE "(git push|Push.*remote)" > /dev/null; then
                violations+=("FORBIDDEN: tasks.md contains 'git push' tasks")
            fi
        fi

        # Check for PR creation violations
        if echo "$forbidden_section" | grep -qi "PR creation"; then
            if echo "$tasks_content" | grep -iE "(Create PR|gh pr create|pull request.*creat)" > /dev/null; then
                violations+=("FORBIDDEN: tasks.md contains PR creation tasks")
            fi
        fi
    fi

    # Check if Phase FINAL exists (if template requires it)
    if grep -q "^## Phase FINAL:" "$template_file"; then
        if ! grep -q "^## Phase FINAL:" "$tasks_file"; then
            violations+=("MISSING: tasks.md must include 'Phase FINAL' section")
        fi
    fi

    # Check multi-repo deployment tasks
    if echo "$tasks_content" | grep -qE "(~/source/bcl/|~/source/mda/)"; then
        # Check for BCL deployment
        if echo "$tasks_content" | grep -q "~/source/bcl/"; then
            if ! echo "$tasks_content" | grep -qi "deploy.*bcl.*servidor"; then
                violations+=("MISSING: BCL repository changes require servidor deployment task")
            fi
        fi

        # Check for MDA deployment
        if echo "$tasks_content" | grep -q "~/source/mda/"; then
            if ! echo "$tasks_content" | grep -qi "deploy.*mda"; then
                violations+=("MISSING: MDA repository changes require deployment task")
            fi
        fi
    fi

    # Return violations (empty = pass)
    if [[ ${#violations[@]} -gt 0 ]]; then
        printf '%s\n' "${violations[@]}"
        return 1
    fi

    return 0
}

# Run tasks validation (pre-implementation, iteration 1 only)
run_tasks_validation() {
    local output_file="$STATE_DIR/tasks-validation-output.txt"

    # All logs go to stderr
    log_phase "TASKS VALIDATION PHASE" >&2
    log_info "Validating that tasks.md properly implements the original plan" >&2

    # First, check template compliance with fast bash checks
    local template_file
    template_file=$(get_tasks_template "$TASKS_FILE")

    if [[ -f "$template_file" ]]; then
        log_info "Checking template compliance: $template_file" >&2
        local violations
        set +e  # Allow check to fail
        violations=$(check_template_compliance "$TASKS_FILE" "$template_file")
        local check_result=$?
        set -e

        if [[ $check_result -ne 0 ]]; then
            # Template violations found - fail fast
            log_error "Template compliance check FAILED" >&2
            echo "TEMPLATE_VIOLATIONS:$violations" > "$output_file"
            cat "$output_file" >&2
            echo "$output_file"
            return 1
        fi

        log_success "Template compliance check passed" >&2
    else
        log_info "No tasks template found - skipping template compliance check" >&2
    fi

    # Continue with existing AI-based semantic validation
    log_info "Using tasks validation AI: $TASKS_VAL_AI" >&2
    log_info "Model: $TASKS_VAL_MODEL" >&2

    local prompt
    prompt=$(generate_tasks_validation_prompt)

    set +e  # Temporarily disable exit on error
    if [[ "$TASKS_VAL_AI" == "codex" ]]; then
        local -a codex_args=(
            --dangerously-bypass-approvals-and-sandbox
        )

        if [[ -n "$TASKS_VAL_MODEL" && "$TASKS_VAL_MODEL" != "default" ]]; then
            codex_args+=(-m "$TASKS_VAL_MODEL")
        fi

        codex_args+=("$prompt")

        log_info "Running tasks validation with codex..." >&2
        if run_codex_with_timeout "$output_file" 600 1 5 "${codex_args[@]}"; then
            log_success "Tasks validation phase completed" >&2
        else
            log_error "Tasks validation phase failed - see output file for details" >&2
        fi
    else
        local -a claude_args=(
            --dangerously-skip-permissions
            --model "$TASKS_VAL_MODEL"
            --print
            --max-turns "$MAX_TURNS"
        )

        if [[ -n "$VERBOSE" ]]; then
            claude_args+=("$VERBOSE")
        fi

        claude_args+=("$prompt")

        log_info "Running tasks validation with claude..." >&2
        if run_claude_with_timeout "$output_file" 600 1 5 "${claude_args[@]}"; then
            log_success "Tasks validation phase completed" >&2
        else
            log_error "Tasks validation phase failed - see output file for details" >&2
        fi
    fi
    set -e  # Re-enable exit on error

    # Display output
    cat "$output_file" >&2

    log_summary "Tasks validation phase completed"

    echo "$output_file"
}

# Run final plan validation (after cross-validation confirms)
run_final_plan_validation() {
    local iteration=$1
    local output_file="$STATE_DIR/final-plan-validation-output-${iteration}.txt"

    # All logs go to stderr
    log_phase "FINAL PLAN VALIDATION PHASE - Iteration $iteration" >&2
    log_info "Validating that the original plan was actually implemented" >&2
    log_info "Using final plan validation AI: $FINAL_PLAN_AI" >&2
    log_info "Model: $FINAL_PLAN_MODEL" >&2

    local prompt
    prompt=$(generate_final_plan_validation_prompt)

    set +e  # Temporarily disable exit on error
    if [[ "$FINAL_PLAN_AI" == "codex" ]]; then
        local -a codex_args=(
            --dangerously-bypass-approvals-and-sandbox
        )

        if [[ -n "$FINAL_PLAN_MODEL" && "$FINAL_PLAN_MODEL" != "default" ]]; then
            codex_args+=(-m "$FINAL_PLAN_MODEL")
        fi

        codex_args+=("$prompt")

        log_info "Running final plan validation with codex..." >&2
        if run_codex_with_timeout "$output_file" 1800 1 5 "${codex_args[@]}"; then
            log_success "Final plan validation phase completed" >&2
        else
            log_error "Final plan validation phase failed - see output file for details" >&2
        fi
    else
        local -a claude_args=(
            --dangerously-skip-permissions
            --model "$FINAL_PLAN_MODEL"
            --print
            --max-turns "$MAX_TURNS"
        )

        if [[ -n "$VERBOSE" ]]; then
            claude_args+=("$VERBOSE")
        fi

        claude_args+=("$prompt")

        log_info "Running final plan validation with claude..." >&2
        if run_claude_with_timeout "$output_file" 1800 1 5 "${claude_args[@]}"; then
            log_success "Final plan validation phase completed" >&2
        else
            log_error "Final plan validation phase failed - see output file for details" >&2
        fi
    fi
    set -e  # Re-enable exit on error

    # Display output
    cat "$output_file" >&2

    save_iteration_state "$iteration" "final_plan_validation" "$output_file"
    log_summary "Iteration $iteration: Final plan validation phase completed"

    echo "$output_file"
}

# Main loop
main() {
    parse_args "$@"

    set_default_models_for_ai
    set_cross_validation_ai
    set_final_plan_validation_ai
    set_tasks_validation_ai

    # Validate mutually exclusive flags
    if [[ -n "$ORIGINAL_PLAN_FILE" && -n "$GITHUB_ISSUE" ]]; then
        log_error "Cannot specify both --original-plan-file and --github-issue"
        log_error "Use one or the other to provide the original plan"
        exit 1
    fi

    # Handle --status flag first
    if [[ -n "$STATUS_FLAG" ]]; then
        show_status
        # show_status exits
    fi

    # Handle --clean flag
    if [[ -n "$CLEAN_FLAG" ]]; then
        if [[ -d "$STATE_DIR" ]]; then
            log_info "Cleaning state directory: $STATE_DIR"
            rm -rf "$STATE_DIR"
            log_success "State directory removed"
        else
            log_info "No state directory to clean"
        fi
    fi

    # Handle --cancel flag
    if [[ -n "$CANCEL_FLAG" ]]; then
        local state_file="$STATE_DIR/current-state.json"

        if [[ ! -f "$state_file" ]]; then
            log_error "No active session to cancel"
            exit 1
        fi

        # Read current status
        local stored_status
        stored_status=$(python3 - "$state_file" << 'PYTHON_EOF'
import sys
import json

try:
    with open(sys.argv[1], 'r') as f:
        state = json.load(f)
    print(state.get('status', 'UNKNOWN'))
except:
    print('ERROR')
PYTHON_EOF
)

        if [[ "$stored_status" == "COMPLETE" ]]; then
            log_error "Session already complete, nothing to cancel"
            log_info "Use --clean to remove completed session state"
            exit 1
        fi

        log_info "Cancelling session with status: $stored_status"

        # Remove state directory
        rm -rf "$STATE_DIR"

        log_success "Session cancelled and state removed"
        exit 0
    fi

    echo -e "${CYAN}"
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║                     RALPH LOOP                                ║"
    echo "║         Dual-Model Validation for Spec-Driven Dev             ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"

    # Find tasks.md
    TASKS_FILE=$(find_tasks_file) || exit 1
    log_info "Tasks file: $TASKS_FILE"

    # Declare iteration variable at function scope
    local iteration=0
    local feedback=""
    local last_unchecked
    local resuming=0

    # Check for existing state before doing anything else
    check_existing_state

    # If we're resuming, load the state
    if [[ -n "$RESUME_FLAG" || -n "$RESUME_FORCE" ]]; then
        if load_state; then
            log_info "Loading previous session state..."

            # Restore tasks file path from saved state
            if [[ -n "$STORED_TASKS_FILE" && -f "$STORED_TASKS_FILE" ]]; then
                TASKS_FILE="$STORED_TASKS_FILE"
                log_info "Restored tasks file from state: $TASKS_FILE"
            fi

            # Restore plan validation settings from saved state
            if [[ -n "$STORED_ORIGINAL_PLAN_FILE" ]]; then
                ORIGINAL_PLAN_FILE="$STORED_ORIGINAL_PLAN_FILE"
                log_info "Restored original plan file from state: $ORIGINAL_PLAN_FILE"
            fi
            if [[ -n "$STORED_GITHUB_ISSUE" ]]; then
                GITHUB_ISSUE="$STORED_GITHUB_ISSUE"
                log_info "Restored GitHub issue from state: $GITHUB_ISSUE"
            fi

            # Restore learnings settings from saved state
            if [[ -n "$STORED_LEARNINGS_ENABLED" ]]; then
                ENABLE_LEARNINGS="$STORED_LEARNINGS_ENABLED"
            fi
            if [[ -n "$STORED_LEARNINGS_FILE" ]]; then
                LEARNINGS_FILE="$STORED_LEARNINGS_FILE"
                log_info "Restored learnings file from state: $LEARNINGS_FILE"
            fi

            # Restore AI CLI from saved state unless overridden
            local use_stored_models=1
            if [[ -z "$OVERRIDE_AI" && -n "$STORED_AI_CLI" ]]; then
                AI_CLI="$STORED_AI_CLI"
                log_info "Restored AI CLI from state: $AI_CLI"
            elif [[ -n "$OVERRIDE_AI" && -n "$STORED_AI_CLI" && "$STORED_AI_CLI" != "$AI_CLI" ]]; then
                log_info "Using AI CLI from command line (overriding saved state)"
                set_default_models_for_ai
                use_stored_models=0
                log_warn "AI CLI changed; using default models for $AI_CLI where not overridden"
            fi

            if [[ $use_stored_models -eq 1 ]]; then
                if [[ -z "$OVERRIDE_IMPL_MODEL" && -n "$STORED_IMPL_MODEL" ]]; then
                    IMPL_MODEL="$STORED_IMPL_MODEL"
                fi
                if [[ -z "$OVERRIDE_VAL_MODEL" && -n "$STORED_VAL_MODEL" ]]; then
                    VAL_MODEL="$STORED_VAL_MODEL"
                fi
            fi

            # Restore max_iterations from saved state unless overridden
            if [[ -z "$OVERRIDE_MAX_ITERATIONS" && -n "$STORED_MAX_ITERATIONS" ]]; then
                MAX_ITERATIONS="$STORED_MAX_ITERATIONS"
                log_info "Restored max_iterations from state: $MAX_ITERATIONS"
            elif [[ -n "$OVERRIDE_MAX_ITERATIONS" ]]; then
                log_info "Using command-line max_iterations: $MAX_ITERATIONS (overriding stored value)"
            fi

            # Restore max_inadmissible from saved state unless overridden
            if [[ -z "$OVERRIDE_MAX_INADMISSIBLE" && -n "$STORED_MAX_INADMISSIBLE" ]]; then
                MAX_INADMISSIBLE="$STORED_MAX_INADMISSIBLE"
                log_info "Restored max_inadmissible from state: $MAX_INADMISSIBLE"
            elif [[ -n "$OVERRIDE_MAX_INADMISSIBLE" ]]; then
                log_info "Using command-line max_inadmissible: $MAX_INADMISSIBLE (overriding stored value)"
            fi

            # Validate state integrity (disable set -e temporarily)
            local validation_error
            set +e
            validation_error=$(validate_state 2>&1)
            local validation_result=$?
            set -e

            if [[ $validation_result -eq 2 && -z "$RESUME_FORCE" ]]; then
                # Tasks file modified, need --resume-force
                echo -e "\n${YELLOW}╔═══════════════════════════════════════════════════════════════╗${NC}"
                echo -e "${YELLOW}║              TASKS FILE MODIFIED                              ║${NC}"
                echo -e "${YELLOW}╚═══════════════════════════════════════════════════════════════╝${NC}\n"
                echo "The tasks.md file has changed since the session was interrupted."
                echo ""
                echo "Options:"
                echo "  $(basename "$0") --resume-force   Resume with modified file"
                echo "  $(basename "$0") --clean          Start fresh with new file"
                echo ""
                exit 1
            elif [[ $validation_result -ne 0 && $validation_result -ne 2 ]]; then
                log_error "State validation failed: $validation_error"
                log_error "Cannot resume. Use --clean to start fresh."
                exit 1
            fi

            # Show resume summary
            local stored_status
            stored_status=$(python3 - "$STATE_DIR/current-state.json" << 'PYTHON_EOF'
import sys, json
try:
    with open(sys.argv[1], 'r') as f:
        state = json.load(f)
    print(state.get('status', 'UNKNOWN'))
except:
    print('UNKNOWN')
PYTHON_EOF
)
            show_resume_summary "$ITERATION" "$CURRENT_PHASE" "$stored_status"

            # Restore from loaded state
            iteration=$ITERATION
            CURRENT_ITERATION=$ITERATION  # Update global for cleanup handler
            feedback="$LAST_FEEDBACK"
            resuming=1

            # Signal to use saved retry state when we reach the phase
            # Only set if we have a non-default retry state (attempt > 1 means we were mid-retry)
            if [[ $CURRENT_RETRY_ATTEMPT -gt 1 ]]; then
                RESUMING_RETRY=1
                log_info "Will resume from retry attempt $CURRENT_RETRY_ATTEMPT with ${CURRENT_RETRY_DELAY}s delay"
            fi

            log_info "Resumed from iteration $iteration, phase: $CURRENT_PHASE"

            if [[ -z "$OVERRIDE_MODELS" ]]; then
                log_info "Using models from saved state/defaults"
            else
                log_info "Using command line models where provided"
            fi
        else
            log_error "Failed to load state file"
            exit 1
        fi
    fi

    validate_models_for_ai

    # Count initial tasks
    local initial_unchecked
    local initial_checked
    initial_unchecked=$(count_unchecked_tasks "$TASKS_FILE")
    initial_checked=$(count_checked_tasks "$TASKS_FILE")

    log_info "Current state: $initial_checked checked, $initial_unchecked unchecked"

    if [[ "$initial_unchecked" -eq 0 ]]; then
        # Don't exit early when resuming an incomplete phase - validator must confirm
        if [[ $resuming -eq 1 && ("$CURRENT_PHASE" == "implementation" || "$CURRENT_PHASE" == "validation") ]]; then
            log_warn "All tasks appear checked, but session was interrupted during $CURRENT_PHASE phase"
            log_info "Continuing to let validator verify the work..."
        else
            log_success "All tasks already completed!"
            exit 0
        fi
    fi

    # Initialize state if not resuming
    if [[ $resuming -eq 0 ]]; then
        init_state_dir
        init_learnings_file
        log_summary "Started Ralph Loop with $initial_unchecked unchecked tasks"
        log_summary "AI CLI: $AI_CLI"
        log_summary "Implementation model: $IMPL_MODEL, Validation model: $VAL_MODEL"

        SCRIPT_START_TIME=$(get_timestamp)
        last_unchecked=$initial_unchecked
    else
        # Resuming - use existing state
        log_summary "Resumed Ralph Loop at iteration $iteration"
        last_unchecked=${LAST_CHECKED_COUNT:-$initial_unchecked}

        # Initialize learnings file if needed (for resumed sessions)
        init_learnings_file

        # Convert started_at from ISO format to timestamp if needed
        if [[ "$SCRIPT_START_TIME" =~ ^[0-9]{4}- ]]; then
            SCRIPT_START_TIME=$(date -d "$SCRIPT_START_TIME" +%s 2>/dev/null || get_timestamp)
        fi
    fi

    log_info "Max iterations: $MAX_ITERATIONS"
    log_info "AI CLI: $AI_CLI"
    log_info "Implementation model: $IMPL_MODEL"
    log_info "Validation model: $VAL_MODEL"

    # Fetch GitHub issue if needed (fresh start OR resuming during tasks_validation)
    # Only fetch if we have a GITHUB_ISSUE and don't already have the plan file
    if [[ -n "$GITHUB_ISSUE" ]]; then
        local plan_file="$STATE_DIR/github-issue-plan.md"

        # Extract requested issue number (handles URL or number format)
        local requested_issue_num
        requested_issue_num=$(echo "$GITHUB_ISSUE" | grep -oE '[0-9]+$')

        # Check if cached plan matches requested issue
        local should_fetch=0
        if [[ ! -f "$plan_file" ]]; then
            should_fetch=1
        elif [[ -z "$ORIGINAL_PLAN_FILE" && $resuming -eq 1 ]]; then
            should_fetch=1
        elif [[ -f "$plan_file" ]]; then
            # Plan file exists - verify it matches requested issue
            local cached_issue_num
            cached_issue_num=$(head -1 "$plan_file" | sed -n 's/^# GitHub Issue #\([0-9]*\):.*/\1/p')
            if [[ -n "$cached_issue_num" && "$cached_issue_num" != "$requested_issue_num" ]]; then
                log_warn "Cached plan is for issue #$cached_issue_num, but requested issue #$requested_issue_num"
                log_info "Re-fetching plan for issue #$requested_issue_num..."
                should_fetch=1
            fi
        fi

        if [[ $should_fetch -eq 1 ]]; then
            log_info "Fetching plan from GitHub issue: $GITHUB_ISSUE"

            local issue_content
            if ! issue_content=$(gh issue view "$GITHUB_ISSUE" --json body,title,number 2>&1); then
                log_error "Failed to fetch GitHub issue: $GITHUB_ISSUE"
                log_error "$issue_content"
                exit 1
            fi

            local issue_number issue_title issue_body
            issue_number=$(echo "$issue_content" | jq -r '.number')
            issue_title=$(echo "$issue_content" | jq -r '.title')
            issue_body=$(echo "$issue_content" | jq -r '.body')

            if [[ -z "$issue_body" || "$issue_body" == "null" ]]; then
                log_error "GitHub issue has no body content: $GITHUB_ISSUE"
                exit 1
            fi

            # Create state directory if it doesn't exist
            mkdir -p "$STATE_DIR"

            # Save to state directory with header
            {
                echo "# GitHub Issue #${issue_number}: ${issue_title}"
                echo ""
                echo "$issue_body"
            } > "$plan_file"

            log_success "Fetched plan from GitHub issue #${issue_number}: ${issue_title}"
            ORIGINAL_PLAN_FILE="$plan_file"
        else
            log_info "Using existing plan file from GitHub issue: $plan_file"
            ORIGINAL_PLAN_FILE="$plan_file"
        fi
    fi

    # Tasks validation phase
    # Run if: (NOT resuming) OR (resuming AND phase is tasks_validation)
    local should_run_tasks_validation=0
    if [[ -n "$ORIGINAL_PLAN_FILE" ]]; then
        if [[ $resuming -eq 0 ]]; then
            # Fresh start with plan file
            should_run_tasks_validation=1
        elif [[ "$CURRENT_PHASE" == "tasks_validation" ]]; then
            # Resuming during tasks validation phase
            should_run_tasks_validation=1
            log_info "Resuming interrupted tasks validation phase"
        fi
    fi

    if [[ $should_run_tasks_validation -eq 1 ]]; then
        log_info "Original plan file provided: $ORIGINAL_PLAN_FILE"
        log_info "Running tasks validation before implementation..."

        CURRENT_PHASE="tasks_validation"
        save_state "running" 0

        # Run tasks validation
        local tasks_val_file
        tasks_val_file=$(run_tasks_validation)

        # Check for template violations first (fast fail)
        local tasks_val_content
        tasks_val_content=$(cat "$tasks_val_file")
        if [[ "$tasks_val_content" == TEMPLATE_VIOLATIONS:* ]]; then
            log_error "Tasks validation FAILED: tasks.md violates template rules"
            echo -e "\n${RED}╔═══════════════════════════════════════════════════════════════╗${NC}"
            echo -e "${RED}║               TEMPLATE COMPLIANCE FAILED                      ║${NC}"
            echo -e "${RED}║          tasks.md violates template requirements              ║${NC}"
            echo -e "${RED}╚═══════════════════════════════════════════════════════════════╝${NC}\n"

            # Display violations
            echo "${tasks_val_content#TEMPLATE_VIOLATIONS:}"

            # Clean up session since loop never started (same as AI validation failure)
            log_info "Cleaning up session directory..."
            rm -rf "$STATE_DIR"
            exit $EXIT_TASKS_INVALID
        fi

        # Extract and parse RALPH_TASKS_VALIDATION JSON
        local tasks_val_json
        tasks_val_json=$(extract_json_from_file "$tasks_val_file" "RALPH_TASKS_VALIDATION") || true

        if [[ -z "$tasks_val_json" ]]; then
            log_error "Could not parse tasks validation JSON"
            log_error "See output file for details: $tasks_val_file"
            exit $EXIT_ERROR
        fi

        # Parse verdict
        local tasks_verdict
        tasks_verdict=$(echo "$tasks_val_json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    tasks_val = data.get('RALPH_TASKS_VALIDATION', {})
    print(tasks_val.get('verdict', 'UNKNOWN'))
except:
    print('PARSE_ERROR')
" 2>/dev/null || echo "PARSE_ERROR")

        log_info "Tasks validation verdict: $tasks_verdict"

        # Programmatic enforcement: override VALID verdict if contradictions or missing requirements exist
        if [[ "$tasks_verdict" == "VALID" ]]; then
            local contradictions_count missing_req_count
            contradictions_count=$(echo "$tasks_val_json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    tasks_val = data.get('RALPH_TASKS_VALIDATION', {})
    print(tasks_val.get('contradictions_found', 0))
except:
    print(0)
" 2>/dev/null || echo "0")

            missing_req_count=$(echo "$tasks_val_json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    tasks_val = data.get('RALPH_TASKS_VALIDATION', {})
    print(tasks_val.get('missing_requirements', 0))
except:
    print(0)
" 2>/dev/null || echo "0")

            if [[ "$contradictions_count" -gt 0 ]] || [[ "$missing_req_count" -gt 0 ]]; then
                log_warning "AI returned VALID despite finding contradictions ($contradictions_count) or missing requirements ($missing_req_count) - overriding to INVALID"
                tasks_verdict="INVALID"
                log_info "Tasks validation verdict (overridden): $tasks_verdict"
            fi
        fi

        if [[ "$tasks_verdict" == "INVALID" ]]; then
            # Extract feedback
            local tasks_feedback
            tasks_feedback=$(echo "$tasks_val_json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    tasks_val = data.get('RALPH_TASKS_VALIDATION', {})
    print(tasks_val.get('feedback', 'Tasks validation failed'))
except Exception as e:
    print(f'Error parsing feedback: {e}')
" 2>/dev/null || echo "Tasks validation failed")

            log_error "Tasks validation INVALID: tasks.md does not properly implement the original plan"
            echo -e "\n${RED}╔═══════════════════════════════════════════════════════════════╗${NC}"
            echo -e "${RED}║                  TASKS VALIDATION FAILED                      ║${NC}"
            echo -e "${RED}║         tasks.md doesn't implement the original plan          ║${NC}"
            echo -e "${RED}╚═══════════════════════════════════════════════════════════════╝${NC}\n"
            echo -e "${YELLOW}Feedback:${NC}"
            echo "$tasks_feedback"
            echo ""
            echo -e "${CYAN}Next steps:${NC}"
            echo "1. Review the feedback above"
            echo "2. Update tasks.md to properly cover the plan requirements"
            echo "3. Or regenerate tasks.md with spec-kit using the updated plan"
            echo ""

            # Clean up session since loop never started
            log_info "Cleaning up session directory..."
            rm -rf "$STATE_DIR"
            exit $EXIT_TASKS_INVALID
        fi

        log_success "Tasks validation VALID: tasks.md properly implements the original plan"
    fi

    while [[ $iteration -lt $MAX_ITERATIONS ]]; do
        # Declare output file variables at loop scope
        local impl_output_file=""
        local val_output_file=""
        local skip_implementation=0

        # If resuming and we're at the saved iteration, handle phase-aware resumption
        if [[ $resuming -eq 1 && $iteration -eq $ITERATION ]]; then
            resuming=0  # Only resume once

            if [[ "$CURRENT_PHASE" == "cross_validation" ]]; then
                # Skip to cross-validation if we were interrupted during cross-validation
                impl_output_file="$STATE_DIR/impl-output-${iteration}.txt"
                val_output_file="$STATE_DIR/val-output-${iteration}.txt"

                if [[ -f "$impl_output_file" && -f "$val_output_file" ]]; then
                    log_info "Resuming at cross-validation phase (implementation and validation already completed)"
                    skip_implementation=1

                    ITERATION_START_TIME=$(get_timestamp)

                    echo -e "\n${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
                    echo -e "${YELLOW}          ITERATION $iteration / $MAX_ITERATIONS (RESUMED)${NC}"
                    echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}\n"

                    # Save state before cross-validation
                    CURRENT_PHASE="cross_validation"
                    save_state "running" "$iteration"

                    # Run cross-validation
                    local cross_val_file
                    cross_val_file=$(run_cross_validation "$iteration" "$val_output_file")

                    # Parse and handle cross-validation verdict
                    local cross_val_json
                    cross_val_json=$(extract_json_from_file "$cross_val_file" "RALPH_CROSS_VALIDATION") || true

                    if [[ -n "$cross_val_json" ]]; then
                        local cross_verdict
                        cross_verdict=$(echo "$cross_val_json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    cross_val = data.get('RALPH_CROSS_VALIDATION', {})
    print(cross_val.get('verdict', 'UNKNOWN'))
except:
    print('PARSE_ERROR')
" 2>/dev/null || echo "PARSE_ERROR")

                        if [[ "$cross_verdict" == "CONFIRMED" ]]; then
                            # Check if final plan validation is needed
                            if [[ -n "$ORIGINAL_PLAN_FILE" ]]; then
                                log_info "Running final plan validation..."

                                CURRENT_PHASE="final_plan_validation"
                                save_state "running" "$iteration"

                                # Run final plan validation
                                local final_plan_val_file
                                final_plan_val_file=$(run_final_plan_validation "$iteration")

                                # Extract and parse RALPH_FINAL_PLAN_VALIDATION JSON
                                local final_plan_json
                                final_plan_json=$(extract_json_from_file "$final_plan_val_file" "RALPH_FINAL_PLAN_VALIDATION") || true

                                if [[ -n "$final_plan_json" ]]; then
                                    local final_plan_verdict
                                    final_plan_verdict=$(echo "$final_plan_json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    final_plan_val = data.get('RALPH_FINAL_PLAN_VALIDATION', {})
    print(final_plan_val.get('verdict', 'UNKNOWN'))
except:
    print('PARSE_ERROR')
" 2>/dev/null || echo "PARSE_ERROR")

                                    log_info "Final plan validation verdict: $final_plan_verdict"

                                    if [[ "$final_plan_verdict" == "NOT_IMPLEMENTED" ]]; then
                                        # Extract feedback and continue loop
                                        local final_plan_feedback
                                        final_plan_feedback=$(echo "$final_plan_json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    final_plan_val = data.get('RALPH_FINAL_PLAN_VALIDATION', {})
    print(final_plan_val.get('feedback', 'Final plan validation found missing requirements'))
except Exception as e:
    print(f'Error parsing feedback: {e}')
" 2>/dev/null || echo "Final plan validation found issues")

                                        log_warn "Final plan validation NOT_IMPLEMENTED - continuing loop"
                                        feedback="Final plan validation found missing requirements: $final_plan_feedback"
                                        LAST_FEEDBACK="$feedback"
                                        log_info "Feedback: $feedback"
                                        # Continue to next iteration
                                        continue
                                    fi

                                    # CONFIRMED - fall through to success
                                    log_success "Final plan validation CONFIRMED - original plan fully implemented"
                                else
                                    log_warn "Could not parse final plan validation JSON, assuming confirmed"
                                fi
                            fi

                            # SUCCESS - all validations passed
                            local iter_elapsed=$(($(get_timestamp) - ITERATION_START_TIME))
                            local total_elapsed=$(($(get_timestamp) - SCRIPT_START_TIME))

                            log_success "Cross-validation CONFIRMED completion"
                            CURRENT_PHASE="complete"
                            save_state "COMPLETE" "$iteration" "COMPLETE"
                            log_summary "SUCCESS: All tasks completed and cross-validated after $iteration iterations in $(format_duration $total_elapsed)"

                            echo -e "\n${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
                            echo -e "${GREEN}║                    RALPH LOOP COMPLETE                        ║${NC}"
                            echo -e "${GREEN}║         All tasks verified and cross-validated!               ║${NC}"
                            echo -e "${GREEN}╠═══════════════════════════════════════════════════════════════╣${NC}"
                            printf "${GREEN}║  Iterations: %-3d              Total time: %-18s║${NC}\n" "$iteration" "$(format_duration $total_elapsed)"
                            echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}\n"

                            log_info "Cleaning up session directory..."
                            rm -rf "$STATE_DIR"
                            exit $EXIT_SUCCESS
                        else
                            # REJECTED - set feedback and continue to next iteration
                            local cross_feedback
                            cross_feedback=$(echo "$cross_val_json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    cross_val = data.get('RALPH_CROSS_VALIDATION', {})
    print(cross_val.get('feedback', 'Cross-validation found issues'))
except Exception as e:
    print(f'Error parsing feedback: {e}')
" 2>/dev/null || echo "Cross-validation rejected completion")

                            feedback="Cross-validation by $CROSS_AI found issues: $cross_feedback"
                            LAST_FEEDBACK="$feedback"
                            log_warn "Cross-validation REJECTED - continuing to next iteration"
                        fi
                    else
                        log_warn "Could not parse cross-validation JSON, restarting iteration"
                    fi

                    # Continue to next iteration
                    continue
                else
                    log_warn "Implementation or validation output not found, restarting iteration from implementation"
                fi
            elif [[ "$CURRENT_PHASE" == "validation" ]]; then
                # Skip to validation if we were interrupted during validation
                impl_output_file="$STATE_DIR/impl-output-${iteration}.txt"

                if [[ -f "$impl_output_file" ]]; then
                    log_info "Resuming at validation phase (implementation already completed)"
                    skip_implementation=1

                    ITERATION_START_TIME=$(get_timestamp)

                    echo -e "\n${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
                    echo -e "${YELLOW}          ITERATION $iteration / $MAX_ITERATIONS (RESUMED)${NC}"
                    echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}\n"

                    # Save state before validation
                    CURRENT_PHASE="validation"
                    save_state "running" "$iteration"

                    # Run validation
                    val_output_file=$(run_validation "$iteration" "$impl_output_file")

                    # Continue to verdict parsing below
                else
                    log_warn "Implementation output not found, restarting iteration from implementation"
                fi
            else
                log_info "Resuming at implementation phase"
            fi
        else
            iteration=$((iteration + 1))
            CURRENT_ITERATION=$iteration  # Update global for cleanup handler
        fi

        # Run normal iteration flow if not skipping implementation
        if [[ $skip_implementation -eq 0 ]]; then
            ITERATION_START_TIME=$(get_timestamp)

            echo -e "\n${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
            echo -e "${YELLOW}                    ITERATION $iteration / $MAX_ITERATIONS${NC}"
            echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}\n"

            # Save state before implementation
            CURRENT_PHASE="implementation"
            save_state "running" "$iteration"

            # Run implementation and capture exit code
            set +e  # Temporarily disable exit on error
            impl_output_file=$(run_implementation "$iteration" "$feedback")
            impl_exit_code=$?
            set -e  # Re-enable exit on error

            # Skip validation if implementation failed
            if [[ $impl_exit_code -ne 0 ]]; then
                local iter_elapsed=$(($(get_timestamp) - ITERATION_START_TIME))
                local total_elapsed=$(($(get_timestamp) - SCRIPT_START_TIME))
                log_warn "Skipping validation - implementation phase failed after all retries"
                log_info "Iteration $iteration completed in $(format_duration $iter_elapsed) (total: $(format_duration $total_elapsed))"
                feedback="Implementation failed in previous iteration. Please try again with a fresh approach."
                LAST_FEEDBACK="$feedback"
                continue
            fi

            # Extract and append learnings from implementation
            if [[ "$ENABLE_LEARNINGS" -eq 1 && -f "$impl_output_file" ]]; then
                local new_learnings
                new_learnings=$(extract_learnings "$impl_output_file")
                if [[ -n "$new_learnings" ]]; then
                    append_learnings "$iteration" "$new_learnings"
                fi
            fi

            # Save state before validation
            CURRENT_PHASE="validation"
            save_state "running" "$iteration"

            # Run validation
            val_output_file=$(run_validation "$iteration" "$impl_output_file")
        fi

        # Parse validation output
        local val_json
        val_json=$(extract_json_from_file "$val_output_file" "RALPH_VALIDATION") || true

        if [[ -z "$val_json" ]]; then
            log_warn "Could not parse validation JSON - cannot safely validate. Retrying validation."

            # Fallback: check tasks.md directly
            local current_unchecked
            current_unchecked=$(count_unchecked_tasks "$TASKS_FILE")

            # Do NOT assume completion when JSON parse fails - this is unsafe
            # Instead, provide feedback and retry the validation phase
            feedback="Validation did not provide structured JSON output. This is required for verification. Please re-run validation with proper JSON format. ($current_unchecked tasks marked as unchecked, but this cannot be safely verified without structured output.)"
            LAST_FEEDBACK="$feedback"  # Store for state saving
            log_warn "Cannot verify completion without structured JSON - continuing to retry validation"
            continue
        fi

        # Save validation JSON
        echo "$val_json" > "$STATE_DIR/iteration-$(printf '%03d' "$iteration")/validation.json"

        local verdict
        verdict=$(parse_verdict "$val_json")

        log_info "Validation verdict: $verdict"
        log_summary "Iteration $iteration: Verdict = $verdict"

        case "$verdict" in
            COMPLETE)
                # Double-check by counting tasks and blocked status
                local final_unchecked
                final_unchecked=$(count_unchecked_tasks "$TASKS_FILE")
                local blocked_count
                blocked_count=$(parse_blocked_count "$val_json")
                local doable_unchecked=$((final_unchecked - blocked_count))

                if [[ "$final_unchecked" -eq 0 ]]; then
                    # True completion - no unchecked tasks
                    # Check if cross-validation should run
                    if [[ "$CROSS_VALIDATE" -eq 1 && "$CROSS_AI_AVAILABLE" -eq 1 ]]; then
                        log_info "Running cross-validation with $CROSS_AI..."

                        # Run cross-validation
                        CURRENT_PHASE="cross_validation"
                        save_state "running" "$iteration"

                        local cross_val_file
                        cross_val_file=$(run_cross_validation "$iteration" "$val_output_file")

                        # Parse cross-validation output
                        local cross_val_json
                        cross_val_json=$(extract_json_from_file "$cross_val_file" "RALPH_CROSS_VALIDATION") || true

                        if [[ -z "$cross_val_json" ]]; then
                            log_warn "Could not parse cross-validation JSON - treating as REJECTED"
                            # Treat as REJECTED if we can't parse - cannot safely verify without structured output
                            feedback="Cross-validation by $CROSS_AI did not provide structured JSON output. This is required for independent verification. Please re-run cross-validation with proper JSON format."
                            LAST_FEEDBACK="$feedback"
                            log_warn "Cannot verify completion without structured cross-validation JSON - continuing to next iteration"
                            continue
                        fi

                        local cross_verdict
                        cross_verdict=$(echo "$cross_val_json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    cross_val = data.get('RALPH_CROSS_VALIDATION', {})
    print(cross_val.get('verdict', 'UNKNOWN'))
except:
    print('PARSE_ERROR')
" 2>/dev/null || echo "PARSE_ERROR")

                        log_info "Cross-validation verdict: $cross_verdict"

                        if [[ "$cross_verdict" == "CONFIRMED" ]]; then
                            # Check if final plan validation is needed
                            if [[ -n "$ORIGINAL_PLAN_FILE" ]]; then
                                log_info "Running final plan validation..."

                                CURRENT_PHASE="final_plan_validation"
                                save_state "running" "$iteration"

                                # Run final plan validation
                                local final_plan_val_file
                                final_plan_val_file=$(run_final_plan_validation "$iteration")

                                # Extract and parse RALPH_FINAL_PLAN_VALIDATION JSON
                                local final_plan_json
                                final_plan_json=$(extract_json_from_file "$final_plan_val_file" "RALPH_FINAL_PLAN_VALIDATION") || true

                                if [[ -n "$final_plan_json" ]]; then
                                    local final_plan_verdict
                                    final_plan_verdict=$(echo "$final_plan_json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    final_plan_val = data.get('RALPH_FINAL_PLAN_VALIDATION', {})
    print(final_plan_val.get('verdict', 'UNKNOWN'))
except:
    print('PARSE_ERROR')
" 2>/dev/null || echo "PARSE_ERROR")

                                    log_info "Final plan validation verdict: $final_plan_verdict"

                                    if [[ "$final_plan_verdict" == "NOT_IMPLEMENTED" ]]; then
                                        # Extract feedback and continue loop
                                        local final_plan_feedback
                                        final_plan_feedback=$(echo "$final_plan_json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    final_plan_val = data.get('RALPH_FINAL_PLAN_VALIDATION', {})
    print(final_plan_val.get('feedback', 'Final plan validation found missing requirements'))
except Exception as e:
    print(f'Error parsing feedback: {e}')
" 2>/dev/null || echo "Final plan validation found issues")

                                        log_warn "Final plan validation NOT_IMPLEMENTED - continuing loop"
                                        feedback="Final plan validation found missing requirements: $final_plan_feedback"
                                        LAST_FEEDBACK="$feedback"
                                        log_info "Feedback: $feedback"
                                        # Continue loop (don't exit)
                                        continue
                                    fi

                                    # CONFIRMED - fall through to success
                                    log_success "Final plan validation CONFIRMED - original plan fully implemented"
                                else
                                    log_warn "Could not parse final plan validation JSON, assuming confirmed"
                                fi
                            fi

                            # SUCCESS - all validations passed
                            local iter_elapsed=$(($(get_timestamp) - ITERATION_START_TIME))
                            local total_elapsed=$(($(get_timestamp) - SCRIPT_START_TIME))

                            log_success "Cross-validation CONFIRMED completion"
                            CURRENT_PHASE="complete"
                            save_state "COMPLETE" "$iteration" "COMPLETE"
                            log_summary "SUCCESS: All tasks completed and cross-validated after $iteration iterations in $(format_duration $total_elapsed)"

                            echo -e "\n${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
                            echo -e "${GREEN}║                    RALPH LOOP COMPLETE                        ║${NC}"
                            echo -e "${GREEN}║         All tasks verified and cross-validated!               ║${NC}"
                            echo -e "${GREEN}╠═══════════════════════════════════════════════════════════════╣${NC}"
                            printf "${GREEN}║  Iterations: %-3d              Total time: %-18s║${NC}\n" "$iteration" "$(format_duration $total_elapsed)"
                            echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}\n"

                            log_info "Cleaning up session directory..."
                            rm -rf "$STATE_DIR"
                            exit $EXIT_SUCCESS
                        else
                            # REJECTED - continue loop with cross-validation feedback
                            log_warn "Cross-validation REJECTED - continuing loop"
                            local cross_feedback
                            cross_feedback=$(echo "$cross_val_json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    cross_val = data.get('RALPH_CROSS_VALIDATION', {})
    print(cross_val.get('feedback', 'Cross-validation found issues'))
except Exception as e:
    print(f'Error parsing feedback: {e}')
" 2>/dev/null || echo "Cross-validation rejected completion")

                            feedback="Cross-validation by $CROSS_AI found issues: $cross_feedback"
                            LAST_FEEDBACK="$feedback"
                            log_info "Feedback: $feedback"
                            # Continue loop (don't exit)
                        fi
                    elif [[ "$CROSS_VALIDATE" -eq 1 && "$CROSS_AI_AVAILABLE" -eq 0 ]]; then
                        # Alternate AI not available, skip with warning (already logged at startup)
                        log_warn "Skipping cross-validation ($CROSS_AI not installed)"
                        local iter_elapsed=$(($(get_timestamp) - ITERATION_START_TIME))
                        local total_elapsed=$(($(get_timestamp) - SCRIPT_START_TIME))

                        log_success "All tasks completed and verified!"
                        CURRENT_PHASE="complete"
                        save_state "COMPLETE" "$iteration" "COMPLETE"
                        log_summary "SUCCESS: All tasks completed after $iteration iterations in $(format_duration $total_elapsed)"

                        echo -e "\n${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
                        echo -e "${GREEN}║                    RALPH LOOP COMPLETE                        ║${NC}"
                        echo -e "${GREEN}║              All tasks verified and complete!                 ║${NC}"
                        echo -e "${GREEN}╠═══════════════════════════════════════════════════════════════╣${NC}"
                        printf "${GREEN}║  Iterations: %-3d              Total time: %-18s║${NC}\n" "$iteration" "$(format_duration $total_elapsed)"
                        echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}\n"

                        log_info "Cleaning up session directory..."
                        rm -rf "$STATE_DIR"
                        exit $EXIT_SUCCESS
                    else
                        # Cross-validation disabled, original behavior
                        local iter_elapsed=$(($(get_timestamp) - ITERATION_START_TIME))
                        local total_elapsed=$(($(get_timestamp) - SCRIPT_START_TIME))

                        log_success "All tasks completed and verified!"
                        CURRENT_PHASE="complete"
                        save_state "COMPLETE" "$iteration" "COMPLETE"
                        log_summary "SUCCESS: All tasks completed after $iteration iterations in $(format_duration $total_elapsed)"

                        echo -e "\n${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
                        echo -e "${GREEN}║                    RALPH LOOP COMPLETE                        ║${NC}"
                        echo -e "${GREEN}║              All tasks verified and complete!                 ║${NC}"
                        echo -e "${GREEN}╠═══════════════════════════════════════════════════════════════╣${NC}"
                        printf "${GREEN}║  Iterations: %-3d              Total time: %-18s║${NC}\n" "$iteration" "$(format_duration $total_elapsed)"
                        echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}\n"

                        log_info "Cleaning up session directory..."
                        rm -rf "$STATE_DIR"
                        exit $EXIT_SUCCESS
                    fi
                elif [[ $doable_unchecked -gt 0 ]]; then
                    # Override COMPLETE - there are still doable tasks
                    log_warn "Validator said COMPLETE but $doable_unchecked tasks remain unchecked and not blocked"
                    feedback="Validator incorrectly claimed completion. $doable_unchecked tasks still unchecked and doable. Continue implementation."
                    LAST_FEEDBACK="$feedback"
                elif [[ $blocked_count -gt 0 ]]; then
                    # All remaining are blocked - partial success
                    local iter_elapsed=$(($(get_timestamp) - ITERATION_START_TIME))
                    local total_elapsed=$(($(get_timestamp) - SCRIPT_START_TIME))
                    local blocked_tasks
                    blocked_tasks=$(parse_blocked_tasks "$val_json")

                    log_warn "All doable tasks complete, but $blocked_count tasks remain blocked"
                    CURRENT_PHASE="blocked"
                    save_state "BLOCKED" "$iteration" "BLOCKED"
                    log_summary "BLOCKED: Doable tasks done, $blocked_count tasks blocked ($(format_duration $total_elapsed))"

                    echo -e "\n${YELLOW}╔═══════════════════════════════════════════════════════════════╗${NC}"
                    echo -e "${YELLOW}║                    TASKS BLOCKED                              ║${NC}"
                    echo -e "${YELLOW}║          Doable tasks complete, some blocked                  ║${NC}"
                    echo -e "${YELLOW}╠═══════════════════════════════════════════════════════════════╣${NC}"
                    printf "${YELLOW}║  Iterations: %-3d              Total time: %-18s║${NC}\n" "$iteration" "$(format_duration $total_elapsed)"
                    printf "${YELLOW}║  Blocked tasks: %-46d║${NC}\n" "$blocked_count"
                    echo -e "${YELLOW}╚═══════════════════════════════════════════════════════════════╝${NC}\n"
                    echo -e "Blocked tasks:\n$blocked_tasks\n"

                    exit $EXIT_BLOCKED
                fi
                ;;

            NEEDS_MORE_WORK)
                feedback=$(parse_feedback "$val_json")
                LAST_FEEDBACK="$feedback"  # Store for state saving
                log_info "Feedback: $feedback"

                # Circuit breaker check
                local current_unchecked
                current_unchecked=$(count_unchecked_tasks "$TASKS_FILE")

                if [[ "$current_unchecked" -eq "$last_unchecked" ]]; then
                    NO_PROGRESS_COUNT=$((NO_PROGRESS_COUNT + 1))
                    log_warn "No progress detected ($NO_PROGRESS_COUNT/$MAX_NO_PROGRESS)"

                    if [[ $NO_PROGRESS_COUNT -ge $MAX_NO_PROGRESS ]]; then
                        local total_elapsed=$(($(get_timestamp) - SCRIPT_START_TIME))
                        log_error "Circuit breaker: $MAX_NO_PROGRESS iterations with no progress"
                        CURRENT_PHASE="circuit_breaker"
                        save_state "CIRCUIT_BREAKER" "$iteration" "NEEDS_MORE_WORK"
                        log_summary "CIRCUIT BREAKER: No progress for $MAX_NO_PROGRESS iterations ($(format_duration $total_elapsed))"
                        log_info "Total time: $(format_duration $total_elapsed)"
                        exit $EXIT_MAX_ITERATIONS
                    fi
                else
                    NO_PROGRESS_COUNT=0
                    last_unchecked=$current_unchecked
                fi
                ;;

            ESCALATE)
                local total_elapsed=$(($(get_timestamp) - SCRIPT_START_TIME))
                log_error "Validator requested escalation - human intervention needed"
                feedback=$(parse_feedback "$val_json")
                LAST_FEEDBACK="$feedback"  # Store for state saving
                log_info "Escalation reason: $feedback"
                CURRENT_PHASE="escalated"
                save_state "ESCALATED" "$iteration" "ESCALATE"
                log_summary "ESCALATED: $feedback ($(format_duration $total_elapsed))"

                echo -e "\n${RED}╔═══════════════════════════════════════════════════════════════╗${NC}"
                echo -e "${RED}║                    ESCALATION REQUESTED                       ║${NC}"
                echo -e "${RED}║              Human intervention required                      ║${NC}"
                echo -e "${RED}╠═══════════════════════════════════════════════════════════════╣${NC}"
                printf "${RED}║  Iterations: %-3d              Total time: %-18s║${NC}\n" "$iteration" "$(format_duration $total_elapsed)"
                echo -e "${RED}╚═══════════════════════════════════════════════════════════════╝${NC}\n"
                echo -e "Reason: $feedback\n"

                exit $EXIT_ESCALATE
                ;;

            INADMISSIBLE)
                # Increment inadmissible counter
                ((INADMISSIBLE_COUNT++)) || true

                feedback=$(parse_feedback "$val_json")
                LAST_FEEDBACK="$feedback"
                log_error "INADMISSIBLE PRACTICE DETECTED (count: $INADMISSIBLE_COUNT/$MAX_INADMISSIBLE)"
                log_summary "ITERATION $iteration: INADMISSIBLE (count: $INADMISSIBLE_COUNT)"

                # Check if we've exceeded the threshold
                if [[ $INADMISSIBLE_COUNT -gt $MAX_INADMISSIBLE ]]; then
                    local total_elapsed=$(($(get_timestamp) - SCRIPT_START_TIME))
                    log_error "MAX INADMISSIBLE VIOLATIONS REACHED - Escalating to human"
                    CURRENT_PHASE="inadmissible_escalated"
                    save_state "INADMISSIBLE_ESCALATED" "$iteration" "INADMISSIBLE"
                    log_summary "INADMISSIBLE ESCALATED: $feedback ($(format_duration $total_elapsed))"

                    echo -e "\n${RED}╔═══════════════════════════════════════════════════════════════╗${NC}"
                    echo -e "${RED}║         REPEATED INADMISSIBLE PRACTICE - ESCALATING          ║${NC}"
                    echo -e "${RED}║     Model continues using fundamentally broken approach       ║${NC}"
                    echo -e "${RED}║            Human intervention required                        ║${NC}"
                    echo -e "${RED}╠═══════════════════════════════════════════════════════════════╣${NC}"
                    printf "${RED}║  Violations: %-3d/%-3d         Total time: %-18s║${NC}\n" "$INADMISSIBLE_COUNT" "$MAX_INADMISSIBLE" "$(format_duration $total_elapsed)"
                    echo -e "${RED}╚═══════════════════════════════════════════════════════════════╝${NC}\n"
                    echo -e "${RED}INADMISSIBLE PRACTICE (repeated $INADMISSIBLE_COUNT times):${NC}"
                    echo -e "$feedback\n"
                    echo -e "${YELLOW}The implementation model repeatedly used forbidden approaches.${NC}"
                    echo -e "${YELLOW}This requires human intervention to redesign the solution.${NC}\n"

                    exit $EXIT_INADMISSIBLE
                fi

                # Loop back with explicit feedback (like NEEDS_MORE_WORK)
                CURRENT_PHASE="inadmissible_retry"
                save_state "INADMISSIBLE_RETRY" "$iteration" "INADMISSIBLE"

                echo -e "\n${RED}╔═══════════════════════════════════════════════════════════════╗${NC}"
                echo -e "${RED}║              INADMISSIBLE PRACTICE DETECTED                   ║${NC}"
                echo -e "${RED}║           This is a FORBIDDEN approach - fix it               ║${NC}"
                echo -e "${RED}╠═══════════════════════════════════════════════════════════════╣${NC}"
                printf "${RED}║  Iteration: %-3d              Violations: %-3d/%-3d         ║${NC}\n" "$iteration" "$INADMISSIBLE_COUNT" "$MAX_INADMISSIBLE"
                echo -e "${RED}╚═══════════════════════════════════════════════════════════════╝${NC}\n"
                echo -e "${RED}INADMISSIBLE PRACTICE:${NC}"
                echo -e "$feedback\n"
                echo -e "${YELLOW}You MUST fix this fundamental issue. Read the feedback carefully.${NC}"
                echo -e "${YELLOW}Warning: $INADMISSIBLE_COUNT/$MAX_INADMISSIBLE violations. Further violations will escalate.${NC}\n"

                # Continue to next iteration (like NEEDS_MORE_WORK)
                continue
                ;;

            BLOCKED)
                local total_elapsed=$(($(get_timestamp) - SCRIPT_START_TIME))
                local blocked_count
                blocked_count=$(parse_blocked_count "$val_json")
                local blocked_tasks
                blocked_tasks=$(parse_blocked_tasks "$val_json")

                log_warn "Validator confirmed $blocked_count tasks are blocked"
                log_summary "ITERATION $iteration: BLOCKED ($blocked_count tasks)"

                # Check if there are any non-blocked unchecked tasks
                local total_unchecked
                total_unchecked=$(count_unchecked_tasks "$TASKS_FILE")
                local doable_unchecked=$((total_unchecked - blocked_count))

                if [[ $doable_unchecked -gt 0 ]]; then
                    # Some tasks are still doable, continue loop
                    log_info "$blocked_count tasks blocked, but $doable_unchecked tasks still doable"
                    feedback="$blocked_count tasks confirmed blocked. Focus on remaining $doable_unchecked doable tasks."
                    LAST_FEEDBACK="$feedback"
                else
                    # All unchecked tasks are blocked
                    log_error "All remaining tasks are blocked - human intervention required"
                    CURRENT_PHASE="blocked"
                    save_state "BLOCKED" "$iteration" "BLOCKED"
                    log_summary "BLOCKED: All $blocked_count remaining tasks require human intervention ($(format_duration $total_elapsed))"

                    echo -e "\n${YELLOW}╔═══════════════════════════════════════════════════════════════╗${NC}"
                    echo -e "${YELLOW}║                    TASKS BLOCKED                              ║${NC}"
                    echo -e "${YELLOW}║              Human intervention required                      ║${NC}"
                    echo -e "${YELLOW}╠═══════════════════════════════════════════════════════════════╣${NC}"
                    printf "${YELLOW}║  Iterations: %-3d              Total time: %-18s║${NC}\n" "$iteration" "$(format_duration $total_elapsed)"
                    printf "${YELLOW}║  Blocked tasks: %-46d║${NC}\n" "$blocked_count"
                    echo -e "${YELLOW}╚═══════════════════════════════════════════════════════════════╝${NC}\n"
                    echo -e "Blocked tasks:\n$blocked_tasks\n"

                    exit $EXIT_BLOCKED
                fi
                ;;

            *)
                log_warn "Unknown verdict: $verdict, continuing"
                feedback="Validation returned unclear verdict ($verdict). Please continue with remaining tasks."
                LAST_FEEDBACK="$feedback"  # Store for state saving
                ;;
        esac

        # Update last_unchecked_count for state saving
        LAST_CHECKED_COUNT=$(count_unchecked_tasks "$TASKS_FILE")

        # Display iteration elapsed time
        local iter_elapsed=$(($(get_timestamp) - ITERATION_START_TIME))
        local total_elapsed=$(($(get_timestamp) - SCRIPT_START_TIME))
        log_info "Iteration $iteration completed in $(format_duration $iter_elapsed) (total: $(format_duration $total_elapsed))"
    done

    # Max iterations reached
    local total_elapsed=$(($(get_timestamp) - SCRIPT_START_TIME))
    log_error "Max iterations ($MAX_ITERATIONS) reached without completion"
    local final_unchecked
    final_unchecked=$(count_unchecked_tasks "$TASKS_FILE")
    log_info "$final_unchecked tasks still unchecked"
    log_info "Total time: $(format_duration $total_elapsed)"

    save_state "MAX_ITERATIONS" "$MAX_ITERATIONS" "INCOMPLETE"
    log_summary "MAX ITERATIONS: Stopped after $MAX_ITERATIONS iterations with $final_unchecked tasks remaining ($(format_duration $total_elapsed))"

    echo -e "\n${YELLOW}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${YELLOW}║                MAX ITERATIONS REACHED                         ║${NC}"
    echo -e "${YELLOW}╠═══════════════════════════════════════════════════════════════╣${NC}"
    printf "${YELLOW}║  Iterations: %-3d              Total time: %-18s║${NC}\n" "$MAX_ITERATIONS" "$(format_duration $total_elapsed)"
    printf "${YELLOW}║  Tasks remaining: %-44d║${NC}\n" "$final_unchecked"
    echo -e "${YELLOW}╚═══════════════════════════════════════════════════════════════╝${NC}\n"

    exit $EXIT_MAX_ITERATIONS
}

main "$@"
