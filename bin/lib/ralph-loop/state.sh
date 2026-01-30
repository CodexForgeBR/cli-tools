#!/bin/bash
# state.sh - State management and persistence for ralph-loop
# Part of Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

# PYTHON_DIR should be set by the main script
# Default fallback for standalone testing
: "${PYTHON_DIR:=$(dirname "${BASH_SOURCE[0]}")/../ralph-loop-python}"

load_state() {
    local state_file="$STATE_DIR/current-state.json"

    if [[ ! -f "$state_file" ]]; then
        return 1
    fi

    # Use standalone Python script to parse JSON and output shell variable assignments
    local python_output
    python_output=$(python3 "$PYTHON_DIR/state_parser.py" load "$state_file")

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

    # Restore schedule data if enabled
    if [[ "$STORED_SCHEDULE_ENABLED" -eq 1 ]]; then
        SCHEDULE_TARGET_EPOCH="$STORED_SCHEDULE_TARGET_EPOCH"
        SCHEDULE_TARGET_HUMAN="$STORED_SCHEDULE_TARGET_HUMAN"
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

    # Load state to check status using standalone Python script
    local stored_status
    stored_status=$(python3 "$PYTHON_DIR/state_parser.py" check "$state_file")

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

    # Parse and display state using standalone Python script
    python3 "$PYTHON_DIR/state_parser.py" status "$state_file"

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
        cat > "$LEARNINGS_FILE" << 'LEARNINGS_INNER_EOF'
# Ralph Loop Learnings

## Codebase Patterns
<!-- Add reusable patterns discovered during implementation -->

---

## Iteration Log
LEARNINGS_INNER_EOF
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

    cat >> "$LEARNINGS_FILE" << APPEND_LEARNINGS_EOF

### Iteration $iteration - $(date '+%Y-%m-%d %H:%M')
$learnings
---
APPEND_LEARNINGS_EOF
    log_info "Appended learnings from iteration $iteration"
}

# Extract learnings from implementation output
extract_learnings() {
    local output_file=$1

    # Use standalone Python script to extract learnings
    python3 "$PYTHON_DIR/learnings_extractor.py" "$output_file"
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

    # Escape feedback for JSON using standalone Python script
    local escaped_feedback
    escaped_feedback=$(echo "$LAST_FEEDBACK" | python3 "$PYTHON_DIR/json_field.py" escape | sed 's/^"//; s/"$//')

    # Determine cross_ai_available status
    local cross_ai_avail="false"
    if [[ "$CROSS_AI_AVAILABLE" -eq 1 ]]; then
        cross_ai_avail="true"
    fi

    cat > "$STATE_DIR/current-state.json" << SAVE_STATE_INNER_EOF
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
    "schedule": {
        "enabled": $([[ -n "$SCHEDULE_TARGET_EPOCH" ]] && echo "true" || echo "false"),
        "target_epoch": ${SCHEDULE_TARGET_EPOCH:-0},
        "target_human": "$SCHEDULE_TARGET_HUMAN"
    },
    "retry_state": {
        "attempt": $CURRENT_RETRY_ATTEMPT,
        "delay": $CURRENT_RETRY_DELAY
    },
    "inadmissible_count": $INADMISSIBLE_COUNT,
    "last_feedback": "$escaped_feedback"
}
SAVE_STATE_INNER_EOF
}

# Append to summary log
log_summary() {
    local message=$1
    echo "[$(date -Iseconds)] $message" >> "$STATE_DIR/summary.log"
}
