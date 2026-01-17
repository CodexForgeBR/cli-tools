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
IMPL_MODEL="opus"
VAL_MODEL="opus"
TASKS_FILE=""
VERBOSE=""
STATE_DIR=".ralph-loop"
SCRIPT_START_TIME=""
ITERATION_START_TIME=""

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
    save_state "INTERRUPTED"
    echo -e "${GREEN}State saved to ${STATE_DIR}/${NC}"
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
  --implementation-model M   Model for implementation (default: opus)
  --validation-model M       Model for validation (default: sonnet)
  --tasks-file PATH          Path to tasks.md (auto-detects if not specified)
  -h, --help                 Show this help message

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
            --implementation-model)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --implementation-model"
                    exit 1
                fi
                IMPL_MODEL=$2
                shift 2
                ;;
            --validation-model)
                if [[ -z "$2" ]]; then
                    log_error "Missing value for --validation-model"
                    exit 1
                fi
                VAL_MODEL=$2
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

# Initialize state directory
init_state_dir() {
    mkdir -p "$STATE_DIR"
    echo "{\"started_at\": \"$(date -Iseconds)\", \"iteration\": 0, \"status\": \"running\"}" > "$STATE_DIR/current-state.json"
    log_info "State directory initialized: $STATE_DIR"
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

# Save current state
save_state() {
    local status=$1
    local iteration=${2:-0}
    local verdict=${3:-""}

    cat > "$STATE_DIR/current-state.json" << EOF
{
    "started_at": "$(date -Iseconds)",
    "last_updated": "$(date -Iseconds)",
    "iteration": $iteration,
    "status": "$status",
    "verdict": "$verdict",
    "tasks_file": "$TASKS_FILE",
    "implementation_model": "$IMPL_MODEL",
    "validation_model": "$VAL_MODEL"
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

    if [[ $iteration -gt 1 ]]; then
        claude_args+=(--continue)
    fi

    # Prompt goes as positional argument at the end
    claude_args+=("$prompt")

    log_info "Running claude..." >&2

    # Run claude with tee to show output AND save to file
    set +e  # Temporarily disable exit on error
    claude "${claude_args[@]}" 2>&1 | tee "$output_file" >&2
    local claude_exit=$?
    set -e  # Re-enable exit on error

    if [[ $claude_exit -eq 0 ]]; then
        log_success "Implementation phase completed" >&2
    else
        log_error "Implementation phase failed with exit code $claude_exit" >&2
        log_warn "Check if claude CLI is working: claude --print 'hello'" >&2
    fi

    save_iteration_state "$iteration" "implementation" "$output_file"
    log_summary "Iteration $iteration: Implementation phase completed"

    # Only this goes to stdout - the file path
    echo "$output_file"
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

    # Run claude with tee to show output AND save to file (output to stderr for display)
    set +e  # Temporarily disable exit on error
    claude "${claude_args[@]}" 2>&1 | tee "$output_file" >&2
    local claude_exit=$?
    set -e  # Re-enable exit on error

    if [[ $claude_exit -eq 0 ]]; then
        log_success "Validation phase completed" >&2
    else
        log_error "Validation phase failed with exit code $claude_exit" >&2
        log_warn "Check if claude CLI is working: claude --print 'hello'" >&2
    fi

    save_iteration_state "$iteration" "validation" "$output_file"
    log_summary "Iteration $iteration: Validation phase completed"

    echo "$output_file"
}

# Main loop
main() {
    parse_args "$@"

    echo -e "${CYAN}"
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║                     RALPH LOOP                                ║"
    echo "║         Dual-Model Validation for Spec-Driven Dev             ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"

    # Find tasks.md
    TASKS_FILE=$(find_tasks_file) || exit 1
    log_info "Tasks file: $TASKS_FILE"

    # Count initial tasks
    local initial_unchecked
    local initial_checked
    initial_unchecked=$(count_unchecked_tasks "$TASKS_FILE")
    initial_checked=$(count_checked_tasks "$TASKS_FILE")

    log_info "Initial state: $initial_checked checked, $initial_unchecked unchecked"

    if [[ "$initial_unchecked" -eq 0 ]]; then
        log_success "All tasks already completed!"
        exit 0
    fi

    # Initialize state
    init_state_dir
    log_summary "Started Ralph Loop with $initial_unchecked unchecked tasks"
    log_summary "Implementation model: $IMPL_MODEL, Validation model: $VAL_MODEL"

    log_info "Max iterations: $MAX_ITERATIONS"
    log_info "Implementation model: $IMPL_MODEL"
    log_info "Validation model: $VAL_MODEL"

    SCRIPT_START_TIME=$(get_timestamp)

    local iteration=0
    local feedback=""
    local last_unchecked=$initial_unchecked

    while [[ $iteration -lt $MAX_ITERATIONS ]]; do
        iteration=$((iteration + 1))
        ITERATION_START_TIME=$(get_timestamp)

        echo -e "\n${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${YELLOW}                    ITERATION $iteration / $MAX_ITERATIONS${NC}"
        echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}\n"

        save_state "running" "$iteration"

        # Run implementation
        local impl_output_file
        impl_output_file=$(run_implementation "$iteration" "$feedback")

        # Run validation
        local val_output_file
        val_output_file=$(run_validation "$iteration" "$impl_output_file")

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
                fi
                ;;

            NEEDS_MORE_WORK)
                feedback=$(parse_feedback "$val_json")
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
                log_info "Escalation reason: $feedback"
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
                ;;
        esac

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
