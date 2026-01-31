#!/bin/bash
# notifications.sh - Notification handling for ralph-loop
# Part of Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

# Send notification via OpenClaw CLI
# Usage: send_notification <event> <message> <exit_code>
send_notification() {
    local event="$1"
    local message="$2"
    local exit_code="${3:-0}"

    # Silent no-op if chat ID not configured
    [[ -z "$NOTIFY_CHAT_ID" ]] && return 0

    # Check if openclaw is installed
    if ! command -v openclaw >/dev/null 2>&1; then
        log_debug "openclaw not installed, skipping notification"
        return 0
    fi

    # Calculate elapsed time
    local elapsed=0
    if [[ -n "$SCRIPT_START_TIME" ]]; then
        elapsed=$(($(get_timestamp) - SCRIPT_START_TIME))
    fi

    # Build notification message with emoji based on event
    local emoji="ℹ️"
    case "$event" in
        completed) emoji="✅" ;;
        max_iterations) emoji="⏱️" ;;
        escalate) emoji="🚨" ;;
        blocked) emoji="🚧" ;;
        tasks_invalid) emoji="❌" ;;
        inadmissible) emoji="⛔" ;;
        interrupted) emoji="⏸️" ;;
        rate_limited) emoji="⏳" ;;
    esac

    # Format the full notification message
    local full_message
    full_message=$(cat <<NOTIF_MSG
$emoji Ralph Loop: $event

$message

Project: $(basename "$(pwd)")
Session: ${SESSION_ID:-unknown}
Iteration: ${CURRENT_ITERATION:-0}/${MAX_ITERATIONS}
Exit Code: $exit_code
NOTIF_MSG
)

    # Fire-and-forget: notifications must never block the loop
    # Send via OpenClaw CLI in background with timeout fallback
    # Use gtimeout on macOS (via brew install coreutils), or regular timeout on Linux
    local timeout_cmd=""
    if command -v timeout >/dev/null 2>&1; then
        timeout_cmd="timeout 10s"
    elif command -v gtimeout >/dev/null 2>&1; then
        timeout_cmd="gtimeout 10s"
    fi

    # Send notification (with timeout if available)
    if [[ -n "$timeout_cmd" ]]; then
        $timeout_cmd openclaw message send \
            --channel "${NOTIFY_CHANNEL:-telegram}" \
            --target "$NOTIFY_CHAT_ID" \
            --message "$full_message" \
            >/dev/null 2>&1 || true
    else
        # No timeout available, run with background job and kill after 10s
        openclaw message send \
            --channel "${NOTIFY_CHANNEL:-telegram}" \
            --target "$NOTIFY_CHAT_ID" \
            --message "$full_message" \
            >/dev/null 2>&1 &
        local notify_pid=$!
        ( sleep 10 && kill -9 $notify_pid 2>/dev/null ) &
    fi

    log_debug "Notification sent via OpenClaw: event=$event, exit_code=$exit_code"
}
