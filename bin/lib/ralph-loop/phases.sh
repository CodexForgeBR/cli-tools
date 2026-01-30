#!/bin/bash
# phases.sh - Phase execution and prompt generation for ralph-loop
# Part of Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

# This file contains the core phase execution functions and their prompt generators.
# Prompt generation is now split into separate files in the prompts/ directory.

# Get the directory where this script is located
PHASES_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source prompt generation modules
source "$PHASES_SCRIPT_DIR/prompts/impl-shared.sh"
source "$PHASES_SCRIPT_DIR/prompts/impl-first.prompt.sh"
source "$PHASES_SCRIPT_DIR/prompts/impl-continue.prompt.sh"
source "$PHASES_SCRIPT_DIR/prompts/validation.prompt.sh"
source "$PHASES_SCRIPT_DIR/prompts/cross-validation.prompt.sh"
source "$PHASES_SCRIPT_DIR/prompts/tasks-validation.prompt.sh"
source "$PHASES_SCRIPT_DIR/prompts/final-plan.prompt.sh"

# ============================================================================
# PROMPT GENERATION WRAPPER FUNCTIONS
# ============================================================================
# These functions wrap the prompt generators from the separate files
# and add learnings support

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
        prompt=$(_generate_impl_first_prompt "$TASKS_FILE")
    else
        prompt=$(_generate_impl_continue_prompt "$TASKS_FILE" "$feedback")
    fi

    # Add learnings section if available
    if [[ -n "$learnings" ]]; then
        prompt+="$(_get_learnings_section "$learnings")"
    fi

    # Add learnings output instruction
    prompt+="$(_get_learnings_output)"

    echo "$prompt"
}

# Generate validation prompt
generate_val_prompt() {
    local impl_output=$1
    _generate_validation_prompt "$TASKS_FILE" "$impl_output"
}

# Generate cross-validation prompt
generate_cross_val_prompt() {
    local impl_output=$1
    local val_output=$2
    _generate_cross_validation_prompt "$TASKS_FILE" "$impl_output" "$val_output"
}

# Generate tasks validation prompt
generate_tasks_validation_prompt() {
    local spec_file=$1
    local tasks_file=$2
    _generate_tasks_validation_prompt "$spec_file" "$tasks_file"
}

# Generate final plan validation prompt
generate_final_plan_validation_prompt() {
    local spec_file=$1
    local tasks_file=$2
    local plan_file=$3
    _generate_final_plan_validation_prompt "$spec_file" "$tasks_file" "$plan_file"
}

# ============================================================================
# HELPER FUNCTIONS FOR PHASE EXECUTION
# ============================================================================

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

# ============================================================================
# PHASE EXECUTION FUNCTIONS
# ============================================================================

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
    local impl_output_file=$3
    local output_file="$STATE_DIR/cross-val-output-${iteration}.txt"

    # All logs go to stderr
    log_phase "CROSS-VALIDATION PHASE - Iteration $iteration" >&2
    log_info "Using opposite AI: $CROSS_AI" >&2
    log_info "Model: $CROSS_MODEL" >&2

    local prompt
    prompt=$(generate_cross_val_prompt "$val_output_file" "$impl_output_file")

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

