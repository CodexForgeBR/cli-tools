#!/bin/bash

# Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development
# Based on the Ralph Wiggum technique by Geoffrey Huntley (May 2025)
#
# Usage: ralph-loop.sh [OPTIONS]
#
# Options:
#   -v, --verbose            Pass verbose flag to claude code cli
#   --max-iterations N       Maximum loop iterations (default: 20)
#   --implementation-model   Model for implementation (default: opus)
#   --validation-model       Model for validation (default: sonnet)
#   --tasks-file PATH        Path to tasks.md (auto-detects: ./tasks.md, specs/*/tasks.md)
#
# Exit Codes:
#   0 - All tasks completed successfully
#   1 - Error (no tasks.md, invalid params, etc.)
#   2 - Max iterations reached without completion
#   3 - Escalation requested by validator

set -e

# Default configuration
MAX_ITERATIONS=20
MAX_CLAUDE_RETRY=10  # Default retries per claude call
IMPL_MODEL="opus"
VAL_MODEL="opus"
TASKS_FILE=""
VERBOSE=""
STATE_DIR=".ralph-loop"
SCRIPT_START_TIME=""
ITERATION_START_TIME=""
SESSION_ID=""
CURRENT_ITERATION=0  # Global iteration counter for cleanup handler

# Resume-related flags
RESUME_FLAG=""
RESUME_FORCE=""
CLEAN_FLAG=""
STATUS_FLAG=""
OVERRIDE_MODELS=""

# State tracking for resume
CURRENT_PHASE=""
LAST_FEEDBACK=""

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
  --max-iterations N         Maximum loop iterations (default: 20)
  --max-claude-retry N       Max retries per claude call (default: 10)
  --implementation-model M   Model for implementation (default: opus)
  --validation-model M       Model for validation (default: opus)
  --tasks-file PATH          Path to tasks.md (auto-detects if not specified)
  --resume                   Resume from last interrupted session
  --resume-force             Resume even if tasks.md has changed
  --clean                    Start fresh, delete existing .ralph-loop state
  --status                   Show current session status without running
  -h, --help                 Show this help message

Session Management:
  When a session is interrupted (Ctrl+C), state is automatically saved.
  Running the script again will detect the interrupted session and prompt you
  to either resume or start fresh.

Exit Codes:
  0 - All tasks completed successfully
  1 - Error (no tasks.md, invalid params, etc.)
  2 - Max iterations reached without completion
  3 - Escalation requested by validator

Examples:
  $(basename "$0")
  $(basename "$0") --max-iterations 10
  $(basename "$0") --implementation-model sonnet --validation-model haiku
  $(basename "$0") --tasks-file specs/feature/tasks.md -v
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
            --max-iterations)
                if [[ -z "$2" || ! "$2" =~ ^[0-9]+$ ]]; then
                    log_error "Invalid value for --max-iterations: $2"
                    exit 1
                fi
                MAX_ITERATIONS=$2
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
            --implementation-model)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --implementation-model"
                    exit 1
                fi
                IMPL_MODEL=$2
                OVERRIDE_MODELS="1"
                shift 2
                ;;
            --validation-model)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --validation-model"
                    exit 1
                fi
                VAL_MODEL=$2
                OVERRIDE_MODELS="1"
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

    circuit = state.get('circuit_breaker', {})
    print(f"NO_PROGRESS_COUNT={circuit.get('no_progress_count', 0)}")
    print(f"LAST_CHECKED_COUNT={circuit.get('last_unchecked_count', 0)}")

    # Store tasks file hash for validation
    print(f"STORED_TASKS_HASH='{state.get('tasks_file_hash', '')}'")

    # Retry state for resume (defaults for backward compatibility)
    retry_state = state.get('retry_state', {})
    print(f"CURRENT_RETRY_ATTEMPT={retry_state.get('attempt', 1)}")
    print(f"CURRENT_RETRY_DELAY={retry_state.get('delay', 5)}")

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
    "implementation_model": "$IMPL_MODEL",
    "validation_model": "$VAL_MODEL",
    "max_iterations": $MAX_ITERATIONS,
    "circuit_breaker": {
        "no_progress_count": $NO_PROGRESS_COUNT,
        "last_unchecked_count": ${LAST_CHECKED_COUNT:-0}
    },
    "retry_state": {
        "attempt": $CURRENT_RETRY_ATTEMPT,
        "delay": $CURRENT_RETRY_DELAY
    },
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

If a task says \"CREATE X\" → YOU MUST CREATE X.
If a task says \"MODIFY X\" → YOU MUST MODIFY X.

YOU ARE A CODE MONKEY. YOU DO NOT THINK. YOU DO NOT HAVE OPINIONS.
YOU EXECUTE THE TASKS EXACTLY AS WRITTEN.

WORKFLOW:
1. Read the task text LITERALLY
2. Do EXACTLY what it says
3. Mark it [x] ONLY if you did the EXACT action specified
4. Move to next task

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

    echo "$prompt"
}

# Generate validation prompt
generate_val_prompt() {
    local impl_output=$1

    cat << EOF
YOU ARE A LIE DETECTOR. THE IMPLEMENTATION MODEL LIES CONSTANTLY. YOUR JOB IS TO CATCH EVERY LIE.

TASKS FILE: $TASKS_FILE

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

THE MODEL'S OPINION DOES NOT MATTER. THE TASK TEXT IS LAW.

VERIFICATION PROCESS:
1. Read tasks.md - find ALL tasks marked [x]
2. For EACH [x] task:
   a. Read the ORIGINAL task text (ignore any annotations the model added)
   b. If task says REMOVE: run \`git diff [filename]\` - scenario MUST be gone
   c. If task says CREATE: run \`ls [filename]\` - file MUST exist
   d. If model added "N/A", "KEPT", "SKIPPED" to a REMOVE task → COUNT AS LIE
3. Count lies. If lies > 0 → verdict = NEEDS_MORE_WORK
4. Count unchecked tasks. If remaining_unchecked > 0 → verdict = NEEDS_MORE_WORK
5. COMPLETE = ONLY when lies_detected = 0 AND remaining_unchecked = 0 (ALL tasks done and verified)
6. ESCALATE = When implementation is fundamentally broken or model is stuck in a loop

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

THE "TEST-TESTING-TEST-CODE" ANTI-PATTERN:
- Model creates new functions in test-utils.ts
- Model writes tests that call these new functions
- Tests pass (they test code that was just written)
- Production code is NEVER tested
- This is a COMPLETE FAILURE even though files exist and tests pass

YOUR FEEDBACK MUST:
- List EVERY lie with task ID
- Specify EXACTLY what file to edit and what to remove
- Do NOT accept any excuses
- Do NOT let the model redefine what "done" means

OUTPUT FORMAT - You MUST output this exact JSON format at the end (the script parses this):
\`\`\`json
{
  "RALPH_VALIDATION": {
    "verdict": "COMPLETE|NEEDS_MORE_WORK|ESCALATE",
    "tasks_analysis": {
      "total_checked": <number of tasks marked [x]>,
      "actually_done": <number verified via git diff/file checks>,
      "lies_detected": <number of false claims>,
      "remaining_unchecked": <number of tasks still [ ]>
    },
    "lies": [
      {"task": "T0XX description", "claimed": "what model said it did", "reality": "what actually happened per git diff"}
    ],
    "feedback": "SPECIFIC instructions for what implementation model must ACTUALLY DO next iteration. List exact files to modify and exact changes needed."
  }
}
\`\`\`

NOW: Run git status, git diff --stat, and verify each claim. Be ruthless.
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

# Run claude with timeout and zombie detection (stream-json workaround)
# Uses --output-format stream-json to detect completion via "type":"result" message
# before the CLI hangs on "No messages returned" error
# Args: output_file timeout_seconds start_attempt start_delay claude_args...
# Returns: 0 on success, 1 on failure
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
        log_info "Claude attempt $attempt/$max_retries (timeout: ${timeout_secs}s, stream-json mode)..." >&2

        # Clear output files
        > "$output_file"
        > "$raw_json_file"

        # Run claude with stream-json output format
        # The "type":"result" message is emitted BEFORE the hang occurs
        claude "${claude_args[@]}" --output-format stream-json > "$raw_json_file" 2>&1 &
        local claude_pid=$!

        local elapsed=0
        local result_received=0
        local grace_period_start=0

        while kill -0 "$claude_pid" 2>/dev/null; do
            sleep 1
            elapsed=$((elapsed + 1))

            # Check for successful result in stream-json output
            if [[ $result_received -eq 0 ]] && grep -q '"type":"result"' "$raw_json_file" 2>/dev/null; then
                result_received=1
                grace_period_start=$elapsed
                log_info "Result received, giving 2s grace period for clean exit..." >&2
            fi

            # If result received, give 2 seconds grace period then force kill
            if [[ $result_received -eq 1 ]]; then
                local grace_elapsed=$((elapsed - grace_period_start))
                if [[ $grace_elapsed -ge 2 ]]; then
                    log_warn "Grace period expired, killing hung process..." >&2
                    kill -9 "$claude_pid" 2>/dev/null || true
                    wait "$claude_pid" 2>/dev/null || true
                    break
                fi
            fi

            # Fallback: Check for zombie error in output (keep as backup)
            if grep -q "No messages returned" "$raw_json_file" 2>/dev/null; then
                log_warn "Detected 'No messages returned' - killing zombie process" >&2
                kill -9 "$claude_pid" 2>/dev/null || true
                wait "$claude_pid" 2>/dev/null || true
                break
            fi

            # Hard timeout
            if [[ $elapsed -ge $timeout_secs ]]; then
                log_warn "Hard timeout (${timeout_secs}s) - killing process" >&2
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

# Run implementation phase
run_implementation() {
    local iteration=$1
    local feedback=$2
    local output_file="$STATE_DIR/impl-output-${iteration}.txt"

    # All logs go to stderr so they don't pollute the returned file path
    log_phase "IMPLEMENTATION PHASE - Iteration $iteration" >&2
    log_info "Model: $IMPL_MODEL" >&2

    local prompt
    prompt=$(generate_impl_prompt "$iteration" "$feedback")

    local claude_args=(
        --dangerously-skip-permissions
        --model "$IMPL_MODEL"
        --print
    )

    if [[ -n "$VERBOSE" ]]; then
        claude_args+=("$VERBOSE")
    fi

    # Prompt goes as positional argument at the end
    claude_args+=("$prompt")

    log_info "Running claude..." >&2

    # Run claude with timeout and zombie detection
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
    if run_claude_with_timeout "$output_file" 1800 "$start_attempt" "$start_delay" "${claude_args[@]}"; then
        log_success "Implementation phase completed" >&2
        impl_success=1
        # Reset retry state after successful phase completion
        CURRENT_RETRY_ATTEMPT=1
        CURRENT_RETRY_DELAY=5
    else
        log_error "Implementation phase failed after $MAX_CLAUDE_RETRY attempts" >&2
        log_warn "Check if claude CLI is working: claude --print 'hello'" >&2
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
    log_info "Model: $VAL_MODEL" >&2

    local impl_output
    impl_output=$(cat "$impl_output_file" 2>/dev/null || echo "No implementation output available")

    local prompt
    prompt=$(generate_val_prompt "$impl_output")

    local claude_args=(
        --dangerously-skip-permissions
        --model "$VAL_MODEL"
        --print
    )

    if [[ -n "$VERBOSE" ]]; then
        claude_args+=("$VERBOSE")
    fi

    # Prompt goes as positional argument at the end
    claude_args+=("$prompt")

    log_info "Running validation..." >&2

    # Run claude with timeout and zombie detection
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
    if run_claude_with_timeout "$output_file" 1800 "$start_attempt" "$start_delay" "${claude_args[@]}"; then
        log_success "Validation phase completed" >&2
        # Reset retry state after successful phase completion
        CURRENT_RETRY_ATTEMPT=1
        CURRENT_RETRY_DELAY=5
    else
        log_error "Validation phase failed - see output file for details" >&2
        log_warn "Check if claude CLI is working: claude --print 'hello'" >&2
    fi
    set -e  # Re-enable exit on error

    # Display output
    cat "$output_file" >&2

    save_iteration_state "$iteration" "validation" "$output_file"
    log_summary "Iteration $iteration: Validation phase completed"

    echo "$output_file"
}

# Main loop
main() {
    parse_args "$@"

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

            # If models were overridden via command line, use them
            if [[ -z "$OVERRIDE_MODELS" ]]; then
                log_info "Using models from saved state"
            else
                log_info "Using models from command line (overriding saved state)"
            fi
        else
            log_error "Failed to load state file"
            exit 1
        fi
    fi

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
        log_summary "Started Ralph Loop with $initial_unchecked unchecked tasks"
        log_summary "Implementation model: $IMPL_MODEL, Validation model: $VAL_MODEL"

        SCRIPT_START_TIME=$(get_timestamp)
        last_unchecked=$initial_unchecked
    else
        # Resuming - use existing state
        log_summary "Resumed Ralph Loop at iteration $iteration"
        last_unchecked=${LAST_CHECKED_COUNT:-$initial_unchecked}

        # Convert started_at from ISO format to timestamp if needed
        if [[ "$SCRIPT_START_TIME" =~ ^[0-9]{4}- ]]; then
            SCRIPT_START_TIME=$(date -d "$SCRIPT_START_TIME" +%s 2>/dev/null || get_timestamp)
        fi
    fi

    log_info "Max iterations: $MAX_ITERATIONS"
    log_info "Implementation model: $IMPL_MODEL"
    log_info "Validation model: $VAL_MODEL"

    while [[ $iteration -lt $MAX_ITERATIONS ]]; do
        # Declare output file variables at loop scope
        local impl_output_file=""
        local val_output_file=""
        local skip_implementation=0

        # If resuming and we're at the saved iteration, handle phase-aware resumption
        if [[ $resuming -eq 1 && $iteration -eq $ITERATION ]]; then
            resuming=0  # Only resume once

            if [[ "$CURRENT_PHASE" == "validation" ]]; then
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
            log_warn "Could not parse validation JSON, checking task counts directly"

            # Fallback: check tasks.md directly
            local current_unchecked
            current_unchecked=$(count_unchecked_tasks "$TASKS_FILE")

            if [[ "$current_unchecked" -eq 0 ]]; then
                local iter_elapsed=$(($(get_timestamp) - ITERATION_START_TIME))
                local total_elapsed=$(($(get_timestamp) - SCRIPT_START_TIME))

                log_success "All tasks appear to be checked off"
                CURRENT_PHASE="complete"
                save_state "COMPLETE" "$iteration" "COMPLETE"
                log_summary "Completed after $iteration iterations in $(format_duration $total_elapsed) (fallback check)"

                echo -e "\n${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
                echo -e "${GREEN}║                    RALPH LOOP COMPLETE                        ║${NC}"
                echo -e "${GREEN}║              All tasks checked off (fallback)                 ║${NC}"
                echo -e "${GREEN}╠═══════════════════════════════════════════════════════════════╣${NC}"
                printf "${GREEN}║  Iterations: %-3d              Total time: %-18s║${NC}\n" "$iteration" "$(format_duration $total_elapsed)"
                echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}\n"

                exit 0
            fi

            feedback="Validation did not provide structured output. $current_unchecked tasks remain unchecked. Please continue implementation."
            LAST_FEEDBACK="$feedback"  # Store for state saving
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
                # Double-check by counting tasks
                local final_unchecked
                final_unchecked=$(count_unchecked_tasks "$TASKS_FILE")

                if [[ "$final_unchecked" -eq 0 ]]; then
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

                    exit 0
                else
                    log_warn "Validator said COMPLETE but $final_unchecked tasks still unchecked"
                    feedback="Validator incorrectly claimed completion. $final_unchecked tasks still unchecked. Continue implementation."
                    LAST_FEEDBACK="$feedback"  # Store for state saving
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
                        exit 2
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

                exit 3
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

    exit 2
}

main "$@"
