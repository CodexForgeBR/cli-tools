#!/bin/bash
# models.sh - Model configuration and validation for ralph-loop  
# Part of Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

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
