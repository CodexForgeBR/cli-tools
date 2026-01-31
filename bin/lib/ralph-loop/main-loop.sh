#!/usr/bin/env bash
#
# main-loop.sh - Decomposed main() function for ralph-loop.sh
#
# This file contains the main orchestrator and sub-functions that implement
# the core loop logic, previously in a monolithic main() function.
#
# Sub-functions:
# - main_init() - Load configs, parse args, apply defaults, set up models
# - main_handle_commands() - Handle --status, --clean, --cancel (early exits)
# - main_display_banner() - Show startup banner with config summary
# - main_find_tasks() - Find and validate tasks.md
# - main_handle_resume() - Detect interrupted session, load state, show resume info
# - main_validate_setup() - Validate model/AI combinations, tasks hash
# - main_fetch_github_issue() - Fetch issue body if --github-issue provided
# - main_tasks_validation() - Run tasks-vs-plan validation (iteration 1 only)
# - main_handle_schedule() - Wait for --start-at time
# - main_iteration_loop() - The while loop driving impl + validation iterations
# - main_run_post_validation_chain() - Cross-validation -> final plan validation -> success/reject
# - main_handle_verdict() - Case statement on verdict (COMPLETE/NEEDS_MORE_WORK/etc.)
# - main_exit_success() - Success banner, notification, cleanup, exit 0
# - cleanup() - Trap handler for SIGINT/SIGTERM
# - main() - Orchestrator calling sub-functions in sequence

# Error handling is set in the main entry point script

# ============================================================================
# Cleanup Handler
# ============================================================================

cleanup() {
    local signal="${1:-EXIT}"
    log_warn "Caught signal $signal - cleaning up..."
    
    # Save interrupted state if we're mid-iteration
    if [[ -n "${CURRENT_ITERATION:-}" && "${CURRENT_ITERATION}" -gt 0 ]]; then
        log_info "Saving interrupted state at iteration $CURRENT_ITERATION, phase: ${CURRENT_PHASE:-unknown}"
        save_state "INTERRUPTED" "$CURRENT_ITERATION" "INTERRUPTED"
    fi
    
    log_info "Session interrupted - use --resume to continue"
    exit 130  # Standard exit code for SIGINT
}

# ============================================================================
# Main Sub-Functions
# ============================================================================

# ----------------------------------------------------------------------------
# main_init() - Load configs, parse args, apply defaults, set up models
# ----------------------------------------------------------------------------
main_init() {
    log_debug "Initializing ralph-loop..."
    
    # Load config files before parsing args (precedence: CLI > project > global > defaults)
    load_config "$HOME/.config/ralph-loop/config"  # Global config
    load_config ".ralph-loop/config"               # Project config

    parse_args "$@"

    # Apply config values where CLI flags were not provided
    apply_config

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
    
    log_debug "Initialization complete"
}

# ----------------------------------------------------------------------------
# main_handle_commands() - Handle --status, --clean, --cancel (early exits)
# ----------------------------------------------------------------------------
main_handle_commands() {
    log_debug "Checking for command flags..."
    
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
    
    log_debug "No command flags detected, continuing to main loop"
}

# ----------------------------------------------------------------------------
# main_display_banner() - Show startup banner with config summary
# ----------------------------------------------------------------------------
main_display_banner() {
    log_debug "Displaying banner..."
    
    echo -e "${CYAN}"
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║                     RALPH LOOP                                ║"
    echo "║         Dual-Model Validation for Spec-Driven Dev             ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# ----------------------------------------------------------------------------
# main_find_tasks() - Find and validate tasks.md
# ----------------------------------------------------------------------------
main_find_tasks() {
    log_debug "Finding tasks file..."
    
    # Find tasks.md
    TASKS_FILE=$(find_tasks_file) || exit 1
    log_info "Tasks file: $TASKS_FILE"
}

# ----------------------------------------------------------------------------
# main_handle_resume() - Detect interrupted session, load state, show resume info
# ----------------------------------------------------------------------------
main_handle_resume() {
    
    log_debug "Checking for resume state..."
    
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
    
    log_debug "Resume handling complete (resuming=${resuming})"
}

# ----------------------------------------------------------------------------
# main_validate_setup() - Validate model/AI combinations, tasks hash
# ----------------------------------------------------------------------------
main_validate_setup() {

    
    log_debug "Validating setup..."
    
    validate_models_for_ai

    # Parse schedule time if provided (for new sessions only)
    if [[ -n "$SCHEDULE_INPUT" && $resuming -eq 0 ]]; then
        log_info "Parsing scheduled start time: $SCHEDULE_INPUT"
        parse_schedule_time "$SCHEDULE_INPUT"

        # Validate not in the past (for full datetime)
        local now_epoch
        now_epoch=$(date +%s)
        if [[ $SCHEDULE_TARGET_EPOCH -le $now_epoch ]]; then
            log_warn "Scheduled time is in the past, proceeding immediately"
            SCHEDULE_TARGET_EPOCH=""
            SCHEDULE_TARGET_HUMAN=""
        else
            log_info "Implementation will start at: $SCHEDULE_TARGET_HUMAN"
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
        if [[ $resuming -eq 1 && ("$CURRENT_PHASE" == "implementation" || "$CURRENT_PHASE" == "validation" || "$CURRENT_PHASE" == "cross_validation" || "$CURRENT_PHASE" == "final_plan_validation") ]]; then
            log_warn "All tasks appear checked, but session was interrupted during $CURRENT_PHASE phase"
            log_info "Continuing to let validator verify the work..."
        else
            log_success "All tasks already completed!"
            exit 0
        fi
    fi

    log_info "Max iterations: $MAX_ITERATIONS"
    log_info "AI CLI: $AI_CLI"
    log_info "Implementation model: $IMPL_MODEL"
    log_info "Validation model: $VAL_MODEL"
    
    log_debug "Setup validation complete"
}

# ----------------------------------------------------------------------------
# main_fetch_github_issue() - Fetch issue body if --github-issue provided
# ----------------------------------------------------------------------------
main_fetch_github_issue() {

    
    log_debug "Checking for GitHub issue to fetch..."
    
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
    
    log_debug "GitHub issue handling complete"
}

# ----------------------------------------------------------------------------
# main_tasks_validation() - Run tasks-vs-plan validation (iteration 1 only)
# ----------------------------------------------------------------------------
main_tasks_validation() {

    
    log_debug "Checking if tasks validation should run..."
    
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

        # Ensure state directory exists before saving state
        mkdir -p "$STATE_DIR"
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

            # Send tasks invalid notification
            send_notification "tasks_invalid" "Tasks don't properly implement the plan (template violations detected)" $EXIT_TASKS_INVALID

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

        # Programmatic enforcement: override VALID verdict if contradictions, missing requirements, or scope narrowing exist
        if [[ "$tasks_verdict" == "VALID" ]]; then
            local contradictions_count missing_req_count scope_narrowing_count
            contradictions_count=$(echo "$tasks_val_json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    tasks_val = data.get('RALPH_TASKS_VALIDATION', {})
    print(tasks_val.get('analysis', {}).get('contradictions_found', 0))
except:
    print(0)
" 2>/dev/null || echo "0")

            missing_req_count=$(echo "$tasks_val_json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    tasks_val = data.get('RALPH_TASKS_VALIDATION', {})
    print(tasks_val.get('analysis', {}).get('missing_requirements', 0))
except:
    print(0)
" 2>/dev/null || echo "0")

            scope_narrowing_count=$(echo "$tasks_val_json" | python3 -c "
import sys
import json

try:
    data = json.load(sys.stdin)
    tasks_val = data.get('RALPH_TASKS_VALIDATION', {})
    print(tasks_val.get('analysis', {}).get('scope_narrowing_detected', 0))
except:
    print(0)
" 2>/dev/null || echo "0")

            if [[ "$contradictions_count" -gt 0 ]] || [[ "$missing_req_count" -gt 0 ]] || [[ "$scope_narrowing_count" -gt 0 ]]; then
                log_warning "AI returned VALID despite finding contradictions ($contradictions_count), missing requirements ($missing_req_count), or scope narrowing ($scope_narrowing_count) - overriding to INVALID"
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

            # Send tasks invalid notification
            send_notification "tasks_invalid" "Tasks don't properly implement the original plan" $EXIT_TASKS_INVALID

            exit $EXIT_TASKS_INVALID
        fi

        log_success "Tasks validation VALID: tasks.md properly implements the original plan"
    fi
    
    log_debug "Tasks validation complete"
}

# ----------------------------------------------------------------------------
# main_handle_schedule() - Wait for --start-at time
# ----------------------------------------------------------------------------
main_handle_schedule() {

    log_debug "Checking for scheduled start time..."

    # Initialize state directory if not resuming (moved here to run AFTER tasks validation)
    if [[ $resuming -eq 0 ]]; then
        init_state_dir
        init_learnings_file

        local initial_unchecked
        initial_unchecked=$(count_unchecked_tasks "$TASKS_FILE")

        log_summary "Started Ralph Loop with $initial_unchecked unchecked tasks"
        log_summary "AI CLI: $AI_CLI"
        log_summary "Implementation model: $IMPL_MODEL, Validation model: $VAL_MODEL"

        SCRIPT_START_TIME=$(get_timestamp)
    else
        # Resuming - use existing state
        log_summary "Resumed Ralph Loop at iteration $iteration"

        # Initialize learnings file if needed (for resumed sessions)
        init_learnings_file

        # Convert started_at from ISO format to timestamp if needed
        if [[ "$SCRIPT_START_TIME" =~ ^[0-9]{4}- ]]; then
            SCRIPT_START_TIME=$(date -d "$SCRIPT_START_TIME" +%s 2>/dev/null || get_timestamp)
        fi
    fi

    # Handle scheduled start time
    if [[ -n "$SCHEDULE_TARGET_EPOCH" ]]; then
        # Check if resuming during waiting phase
        if [[ $resuming -eq 1 && "$CURRENT_PHASE" == "waiting_for_schedule" ]]; then
            local now_epoch
            now_epoch=$(date +%s)

            if [[ $SCHEDULE_TARGET_EPOCH -le $now_epoch ]]; then
                log_info "Scheduled time has passed during interruption, proceeding immediately"
            else
                log_info "Resuming wait for scheduled time: $SCHEDULE_TARGET_HUMAN"
                wait_until_scheduled_time "$SCHEDULE_TARGET_EPOCH"
            fi
        elif [[ $resuming -eq 0 ]]; then
            # Fresh start - wait for scheduled time
            wait_until_scheduled_time "$SCHEDULE_TARGET_EPOCH"
        fi
        # If resuming but not in waiting phase, skip wait (already past scheduled time)
    fi
    
    log_debug "Schedule handling complete"
}

# ----------------------------------------------------------------------------
# main_run_post_validation_chain() - Cross-validation -> final plan validation -> success/reject
# This eliminates duplication between the two code paths that run these validations
# ----------------------------------------------------------------------------
main_run_post_validation_chain() {
    local iteration=$1
    local val_output_file=$2
    local impl_output_file=$3
    
    log_debug "Running post-validation chain (cross-validation + final plan validation)..."
    
    # Check if cross-validation should run
    if [[ "$CROSS_VALIDATE" -eq 1 && "$CROSS_AI_AVAILABLE" -eq 1 ]]; then
        log_info "Running cross-validation with $CROSS_AI..."

        # Run cross-validation
        CURRENT_PHASE="cross_validation"
        save_state "running" "$iteration"

        local cross_val_file
        cross_val_file=$(run_cross_validation "$iteration" "$val_output_file" "$impl_output_file")

        # Parse cross-validation output
        local cross_val_json
        cross_val_json=$(extract_json_from_file "$cross_val_file" "RALPH_CROSS_VALIDATION") || true

        if [[ -z "$cross_val_json" ]]; then
            log_warn "Could not parse cross-validation JSON - treating as REJECTED"
            # Treat as REJECTED if we can't parse - cannot safely verify without structured output
            echo "REJECTED:Cross-validation by $CROSS_AI did not provide structured JSON output. This is required for independent verification."
            return 1
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

        if [[ "$cross_verdict" != "CONFIRMED" ]]; then
            # REJECTED - extract feedback
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

            echo "REJECTED:Cross-validation by $CROSS_AI found issues: $cross_feedback"
            return 1
        fi
        
        # Cross-validation CONFIRMED - continue to final plan validation
        log_success "Cross-validation CONFIRMED"
    elif [[ "$CROSS_VALIDATE" -eq 1 && "$CROSS_AI_AVAILABLE" -eq 0 ]]; then
        # Alternate AI not available, skip with warning (already logged at startup)
        log_warn "Skipping cross-validation ($CROSS_AI not installed)"
    fi

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
                # Extract feedback and return rejection
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

                echo "NOT_IMPLEMENTED:Final plan validation found missing requirements: $final_plan_feedback"
                return 1
            fi

            # CONFIRMED - fall through to success
            log_success "Final plan validation CONFIRMED - original plan fully implemented"
        else
            log_warn "Could not parse final plan validation JSON, assuming confirmed"
        fi
    fi

    # All validations passed
    echo "CONFIRMED"
    return 0
}

# ----------------------------------------------------------------------------
# main_exit_success() - Success banner, notification, cleanup, exit 0
# This eliminates duplication between the multiple success exit paths
# ----------------------------------------------------------------------------
main_exit_success() {
    local iteration=$1
    local skip_cross_validation=${2:-0}
    
    log_debug "Exiting with success..."
    
    local iter_elapsed=$(($(get_timestamp) - ITERATION_START_TIME))
    local total_elapsed=$(($(get_timestamp) - SCRIPT_START_TIME))

    if [[ $skip_cross_validation -eq 0 ]]; then
        log_success "Cross-validation CONFIRMED completion"
    else
        log_success "All tasks completed and verified!"
    fi
    
    CURRENT_PHASE="complete"
    save_state "COMPLETE" "$iteration" "COMPLETE"
    log_summary "SUCCESS: All tasks completed after $iteration iterations in $(format_duration $total_elapsed)"

    echo -e "\n${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                    RALPH LOOP COMPLETE                        ║${NC}"
    
    if [[ $skip_cross_validation -eq 0 ]]; then
        echo -e "${GREEN}║         All tasks verified and cross-validated!               ║${NC}"
    else
        echo -e "${GREEN}║              All tasks verified and complete!                 ║${NC}"
    fi
    
    echo -e "${GREEN}╠═══════════════════════════════════════════════════════════════╣${NC}"
    printf "${GREEN}║  Iterations: %-3d              Total time: %-18s║${NC}\n" "$iteration" "$(format_duration $total_elapsed)"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}\n"

    log_info "Cleaning up session directory..."
    rm -rf "$STATE_DIR"

    # Send success notification
    send_notification "completed" "All tasks completed in $iteration iterations ($(format_duration $total_elapsed))" $EXIT_SUCCESS

    exit $EXIT_SUCCESS
}

# ----------------------------------------------------------------------------
# main_iteration_loop() - The while loop driving impl + validation iterations
# ----------------------------------------------------------------------------
main_iteration_loop() {
    
    log_debug "Starting iteration loop..."
    
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

                    # Run post-validation chain (cross-validation + final plan validation)
                    local chain_result
                    set +e
                    chain_result=$(main_run_post_validation_chain "$iteration" "$val_output_file" "$impl_output_file")
                    local chain_exit=$?
                    set -e

                    if [[ $chain_exit -eq 0 && "$chain_result" == "CONFIRMED" ]]; then
                        # SUCCESS - all validations passed
                        main_exit_success "$iteration" 0
                    else
                        # REJECTED or NOT_IMPLEMENTED - extract feedback and continue
                        local verdict_type="${chain_result%%:*}"
                        local chain_feedback="${chain_result#*:}"
                        
                        feedback="$chain_feedback"
                        LAST_FEEDBACK="$feedback"
                        log_warn "$verdict_type - continuing to next iteration"
                        continue
                    fi
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

        # Parse validation output and handle verdict
        main_handle_verdict "$iteration" "$val_output_file" "$impl_output_file" feedback
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

    # Send max iterations notification
    send_notification "max_iterations" "Exhausted $MAX_ITERATIONS iterations with $final_unchecked tasks remaining ($(format_duration $total_elapsed))" $EXIT_MAX_ITERATIONS

    exit $EXIT_MAX_ITERATIONS
}

# ----------------------------------------------------------------------------
# main_handle_verdict() - Case statement on verdict (COMPLETE/NEEDS_MORE_WORK/etc.)
# ----------------------------------------------------------------------------
main_handle_verdict() {
    local iteration=$1
    local val_output_file=$2
    local impl_output_file=$3
    
    log_debug "Handling validation verdict..."
    
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
        return
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
                # Run post-validation chain (cross-validation + final plan validation)
                local chain_result
                set +e
                chain_result=$(main_run_post_validation_chain "$iteration" "$val_output_file" "$impl_output_file")
                local chain_exit=$?
                set -e

                if [[ $chain_exit -eq 0 && "$chain_result" == "CONFIRMED" ]]; then
                    # SUCCESS - all validations passed
                    local skip_cross=$([[ "$CROSS_VALIDATE" -eq 0 || "$CROSS_AI_AVAILABLE" -eq 0 ]] && echo 1 || echo 0)
                    main_exit_success "$iteration" $skip_cross
                else
                    # REJECTED or NOT_IMPLEMENTED - extract feedback and continue
                    local verdict_type="${chain_result%%:*}"
                    local chain_feedback="${chain_result#*:}"
                    
                    feedback="$chain_feedback"
                    LAST_FEEDBACK="$feedback"
                    log_warn "$verdict_type - continuing loop"
                    log_info "Feedback: $feedback"
                    # Continue loop (don't exit)
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

                # Send blocked notification
                send_notification "blocked" "All $blocked_count remaining tasks blocked - human intervention required ($(format_duration $total_elapsed))" $EXIT_BLOCKED

                exit $EXIT_BLOCKED
            fi
            ;;

        NEEDS_MORE_WORK)
            feedback=$(parse_feedback "$val_json")
            LAST_FEEDBACK="$feedback"  # Store for state saving
            log_info "Feedback: $feedback"
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

            # Send escalation notification
            send_notification "escalate" "Needs human escalation: $feedback ($(format_duration $total_elapsed))" $EXIT_ESCALATE

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

                # Send inadmissible notification
                send_notification "inadmissible" "Repeated inadmissible practices ($INADMISSIBLE_COUNT violations) - needs redesign ($(format_duration $total_elapsed))" $EXIT_INADMISSIBLE

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
            return
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

                # Send blocked notification
                send_notification "blocked" "All $blocked_count remaining tasks blocked - human intervention required ($(format_duration $total_elapsed))" $EXIT_BLOCKED

                exit $EXIT_BLOCKED
            fi
            ;;

        *)
            log_warn "Unknown verdict: $verdict, continuing"
            feedback="Validation returned unclear verdict ($verdict). Please continue with remaining tasks."
            LAST_FEEDBACK="$feedback"  # Store for state saving
            ;;
    esac

    # Display iteration elapsed time
    local iter_elapsed=$(($(get_timestamp) - ITERATION_START_TIME))
    local total_elapsed=$(($(get_timestamp) - SCRIPT_START_TIME))
    log_info "Iteration $iteration completed in $(format_duration $iter_elapsed) (total: $(format_duration $total_elapsed))"
}

# ============================================================================
# Main Orchestrator
# ============================================================================

main() {
    # Set up trap handlers
    trap 'cleanup SIGINT' SIGINT
    trap 'cleanup SIGTERM' SIGTERM

    # Declare shared variables (used across sub-functions)
    iteration=0
    feedback=""
    resuming=0

    # Phase 1: Initialize configuration and parse arguments
    main_init "$@"

    # Phase 2: Handle command flags (--status, --clean, --cancel)
    main_handle_commands

    # Phase 3: Display startup banner
    main_display_banner

    # Phase 4: Find and validate tasks.md
    main_find_tasks

    # Phase 5: Handle resume logic (load state if resuming)
    main_handle_resume

    # Phase 6: Validate setup (models, schedule, initial task count)
    main_validate_setup

    # Phase 7: Fetch GitHub issue if needed
    main_fetch_github_issue

    # Phase 8: Run tasks validation (if original plan provided)
    main_tasks_validation

    # Phase 9: Handle scheduled start time
    main_handle_schedule

    # Phase 10: Run the main iteration loop
    main_iteration_loop
}

# Only run main if this script is executed directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
