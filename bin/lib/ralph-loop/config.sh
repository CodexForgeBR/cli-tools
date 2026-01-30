#!/bin/bash
# config.sh - Configuration loading and applying for ralph-loop
# Part of Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

# Load configuration from file
# Usage: load_config <config_file_path>
load_config() {
    local config_file="$1"

    [[ ! -f "$config_file" ]] && return 0

    log_debug "Loading config from: $config_file"

    # Whitelist of allowed config variables (security: prevent arbitrary code execution)
    local allowed_vars=(
        AI_CLI IMPL_MODEL VAL_MODEL
        CROSS_VALIDATE CROSS_AI CROSS_MODEL
        FINAL_PLAN_AI FINAL_PLAN_MODEL
        TASKS_VAL_AI TASKS_VAL_MODEL
        MAX_ITERATIONS MAX_INADMISSIBLE MAX_CLAUDE_RETRY MAX_TURNS
        INACTIVITY_TIMEOUT
        TASKS_FILE ORIGINAL_PLAN_FILE LEARNINGS_FILE
        ENABLE_LEARNINGS VERBOSE
        NOTIFY_WEBHOOK NOTIFY_CHANNEL NOTIFY_CHAT_ID
    )

    # Read config file line by line
    while IFS='=' read -r key value || [[ -n "$key" ]]; do
        # Skip comments and empty lines
        [[ "$key" =~ ^[[:space:]]*# ]] && continue
        [[ -z "$key" ]] && continue

        # Trim whitespace
        key=$(echo "$key" | xargs)
        value=$(echo "$value" | xargs)

        # Skip if key is empty after trimming
        [[ -z "$key" ]] && continue

        # Check if variable is whitelisted
        local allowed=0
        for allowed_var in "${allowed_vars[@]}"; do
            if [[ "$key" == "$allowed_var" ]]; then
                allowed=1
                break
            fi
        done

        if [[ $allowed -eq 0 ]]; then
            log_debug "Skipping unknown config variable: $key"
            continue
        fi

        # Store config value with CONFIG_ prefix (to not override CLI flags)
        eval "CONFIG_${key}=\"${value}\""
        log_debug "Loaded config: $key=$value"
    done < "$config_file"

    return 0
}

# Apply config values where CLI flags were NOT provided
# Called after parse_args, applies config values only if override flags are not set
apply_config() {
    # AI Configuration
    [[ -z "$OVERRIDE_AI" && -n "$CONFIG_AI_CLI" ]] && AI_CLI="$CONFIG_AI_CLI"
    [[ -z "$OVERRIDE_IMPL_MODEL" && -n "$CONFIG_IMPL_MODEL" ]] && IMPL_MODEL="$CONFIG_IMPL_MODEL"
    [[ -z "$OVERRIDE_VAL_MODEL" && -n "$CONFIG_VAL_MODEL" ]] && VAL_MODEL="$CONFIG_VAL_MODEL"

    # Cross-validation
    [[ -n "$CONFIG_CROSS_VALIDATE" ]] && CROSS_VALIDATE="$CONFIG_CROSS_VALIDATE"
    [[ -z "$OVERRIDE_CROSS_AI" && -n "$CONFIG_CROSS_AI" ]] && CROSS_AI="$CONFIG_CROSS_AI"
    [[ -n "$CONFIG_CROSS_MODEL" ]] && CROSS_MODEL="$CONFIG_CROSS_MODEL"

    # Final plan validation
    [[ -z "$OVERRIDE_FINAL_PLAN_AI" && -n "$CONFIG_FINAL_PLAN_AI" ]] && FINAL_PLAN_AI="$CONFIG_FINAL_PLAN_AI"
    [[ -z "$OVERRIDE_FINAL_PLAN_MODEL" && -n "$CONFIG_FINAL_PLAN_MODEL" ]] && FINAL_PLAN_MODEL="$CONFIG_FINAL_PLAN_MODEL"

    # Tasks validation
    [[ -z "$OVERRIDE_TASKS_VAL_AI" && -n "$CONFIG_TASKS_VAL_AI" ]] && TASKS_VAL_AI="$CONFIG_TASKS_VAL_AI"
    [[ -z "$OVERRIDE_TASKS_VAL_MODEL" && -n "$CONFIG_TASKS_VAL_MODEL" ]] && TASKS_VAL_MODEL="$CONFIG_TASKS_VAL_MODEL"

    # Iteration limits
    [[ -z "$OVERRIDE_MAX_ITERATIONS" && -n "$CONFIG_MAX_ITERATIONS" ]] && MAX_ITERATIONS="$CONFIG_MAX_ITERATIONS"
    [[ -z "$OVERRIDE_MAX_INADMISSIBLE" && -n "$CONFIG_MAX_INADMISSIBLE" ]] && MAX_INADMISSIBLE="$CONFIG_MAX_INADMISSIBLE"
    [[ -n "$CONFIG_MAX_CLAUDE_RETRY" ]] && MAX_CLAUDE_RETRY="$CONFIG_MAX_CLAUDE_RETRY"
    [[ -n "$CONFIG_MAX_TURNS" ]] && MAX_TURNS="$CONFIG_MAX_TURNS"
    [[ -n "$CONFIG_INACTIVITY_TIMEOUT" ]] && INACTIVITY_TIMEOUT="$CONFIG_INACTIVITY_TIMEOUT"

    # File paths
    [[ -n "$CONFIG_TASKS_FILE" ]] && TASKS_FILE="$CONFIG_TASKS_FILE"
    [[ -n "$CONFIG_ORIGINAL_PLAN_FILE" ]] && ORIGINAL_PLAN_FILE="$CONFIG_ORIGINAL_PLAN_FILE"
    [[ -n "$CONFIG_LEARNINGS_FILE" ]] && LEARNINGS_FILE="$CONFIG_LEARNINGS_FILE"

    # Features
    [[ -n "$CONFIG_ENABLE_LEARNINGS" ]] && ENABLE_LEARNINGS="$CONFIG_ENABLE_LEARNINGS"
    [[ -n "$CONFIG_VERBOSE" ]] && VERBOSE="$CONFIG_VERBOSE"

    # Notifications (apply defaults if not overridden)
    if [[ -z "$OVERRIDE_NOTIFY_WEBHOOK" ]]; then
        if [[ -n "$CONFIG_NOTIFY_WEBHOOK" ]]; then
            NOTIFY_WEBHOOK="$CONFIG_NOTIFY_WEBHOOK"
        else
            NOTIFY_WEBHOOK="http://127.0.0.1:18789/webhook"
        fi
    fi

    if [[ -z "$OVERRIDE_NOTIFY_CHANNEL" ]]; then
        if [[ -n "$CONFIG_NOTIFY_CHANNEL" ]]; then
            NOTIFY_CHANNEL="$CONFIG_NOTIFY_CHANNEL"
        else
            NOTIFY_CHANNEL="telegram"
        fi
    fi

    [[ -z "$OVERRIDE_NOTIFY_CHAT_ID" && -n "$CONFIG_NOTIFY_CHAT_ID" ]] && NOTIFY_CHAT_ID="$CONFIG_NOTIFY_CHAT_ID"
}
