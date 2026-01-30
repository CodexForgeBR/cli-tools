#!/bin/bash
# scheduling.sh - Scheduling functions for ralph-loop
# Part of Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

# Dependencies: Requires log_info, log_error, and save_state functions from other modules
# Variables used: CURRENT_PHASE, SCHEDULE_TARGET_HUMAN, SCHEDULE_TARGET_EPOCH

# ============================================================================
# Time Formatting Helpers
# ============================================================================

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

detect_date_flavor() {
    if date --version >/dev/null 2>&1; then
        echo "gnu"
    else
        echo "bsd"
    fi
}

# ============================================================================
# Schedule Parsing
# ============================================================================

# Parse schedule time into epoch timestamp
# Arguments: $1 - date/time string in format: YYYY-MM-DD, HH:MM, "YYYY-MM-DD HH:MM", or YYYY-MM-DDTHH:MM
# Returns: epoch timestamp
# Sets globals: SCHEDULE_TARGET_EPOCH, SCHEDULE_TARGET_HUMAN
parse_schedule_time() {
    local input="$1"
    local flavor
    flavor=$(detect_date_flavor)
    local epoch=""

    # Current time for reference
    local now_epoch
    now_epoch=$(date +%s)

    # Try to parse different formats
    if [[ "$input" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
        # Date only: YYYY-MM-DD (start at 00:00:00)
        if [[ "$flavor" == "gnu" ]]; then
            epoch=$(date -d "$input 00:00:00" +%s 2>/dev/null)
        else
            epoch=$(date -j -f "%Y-%m-%d %H:%M:%S" "$input 00:00:00" +%s 2>/dev/null)
        fi
        SCHEDULE_TARGET_HUMAN="$input at 00:00:00"

    elif [[ "$input" =~ ^[0-9]{1,2}:[0-9]{2}$ ]]; then
        # Time only: HH:MM (assume today, or tomorrow if time passed)
        local today
        today=$(date +%Y-%m-%d)

        if [[ "$flavor" == "gnu" ]]; then
            epoch=$(date -d "$today $input:00" +%s 2>/dev/null)
        else
            epoch=$(date -j -f "%Y-%m-%d %H:%M:%S" "$today $input:00" +%s 2>/dev/null)
        fi

        # If time already passed today, schedule for tomorrow
        if [[ -n "$epoch" && $epoch -le $now_epoch ]]; then
            local tomorrow
            if [[ "$flavor" == "gnu" ]]; then
                tomorrow=$(date -d "+1 day" +%Y-%m-%d)
                epoch=$(date -d "$tomorrow $input:00" +%s 2>/dev/null)
                SCHEDULE_TARGET_HUMAN="tomorrow at $input"
            else
                tomorrow=$(date -v+1d +%Y-%m-%d)
                epoch=$(date -j -f "%Y-%m-%d %H:%M:%S" "$tomorrow $input:00" +%s 2>/dev/null)
                SCHEDULE_TARGET_HUMAN="tomorrow at $input"
            fi
        else
            SCHEDULE_TARGET_HUMAN="today at $input"
        fi

    elif [[ "$input" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}[T\ ][0-9]{1,2}:[0-9]{2}(:[0-9]{2})?$ ]]; then
        # Date + time: YYYY-MM-DD HH:MM or YYYY-MM-DDTHH:MM
        local normalized
        normalized="${input/T/ }"  # Convert T to space

        # Add seconds if not present
        if [[ ! "$normalized" =~ :[0-9]{2}$ ]]; then
            normalized="$normalized:00"
        fi

        if [[ "$flavor" == "gnu" ]]; then
            epoch=$(date -d "$normalized" +%s 2>/dev/null)
        else
            epoch=$(date -j -f "%Y-%m-%d %H:%M:%S" "$normalized" +%s 2>/dev/null)
        fi
        SCHEDULE_TARGET_HUMAN="$normalized"

    else
        log_error "Invalid date/time format: $input"
        log_error "Supported formats: YYYY-MM-DD, HH:MM, 'YYYY-MM-DD HH:MM', YYYY-MM-DDTHH:MM"
        exit 1
    fi

    if [[ -z "$epoch" ]]; then
        log_error "Failed to parse date/time: $input"
        exit 1
    fi

    SCHEDULE_TARGET_EPOCH="$epoch"
}

# ============================================================================
# Schedule Waiting
# ============================================================================

# Wait until scheduled time
# Arguments: $1 - target epoch timestamp
wait_until_scheduled_time() {
    local target_epoch=$1
    local now_epoch
    now_epoch=$(date +%s)
    local remaining=$((target_epoch - now_epoch))

    if [[ $remaining -le 0 ]]; then
        log_info "Scheduled time has passed, proceeding immediately"
        return
    fi

    # Update phase to waiting
    CURRENT_PHASE="waiting_for_schedule"
    save_state

    # Show banner
    echo ""
    log_info "════════════════════════════════════════════════════════════════"
    log_info "⏰ WAITING FOR SCHEDULED TIME"
    log_info "════════════════════════════════════════════════════════════════"
    log_info "Target time: $SCHEDULE_TARGET_HUMAN"
    log_info "Current time: $(date '+%Y-%m-%d %H:%M:%S')"
    log_info ""
    log_info "Implementation will begin in: $(format_duration $remaining)"
    log_info ""
    log_info "Press Ctrl+C to pause. Resume later with --resume"
    log_info "════════════════════════════════════════════════════════════════"
    echo ""

    # Countdown loop with adaptive sleep intervals
    while [[ $remaining -gt 0 ]]; do
        # Adaptive sleep: longer intervals when far away, shorter when close
        local sleep_interval=1
        if [[ $remaining -gt 3600 ]]; then
            sleep_interval=60  # 1 minute intervals when >1 hour away
        elif [[ $remaining -gt 600 ]]; then
            sleep_interval=30  # 30 second intervals when >10 min away
        elif [[ $remaining -gt 60 ]]; then
            sleep_interval=10  # 10 second intervals when >1 min away
        fi

        # Print absolute target time (in-place update)
        local target_time
        local flavor
        flavor=$(detect_date_flavor)
        if [[ "$flavor" == "gnu" ]]; then
            target_time=$(date -d "@$target_epoch" '+%Y-%m-%d %H:%M:%S')
        else
            target_time=$(date -r "$target_epoch" '+%Y-%m-%d %H:%M:%S')
        fi
        printf "\r⏳ Waiting until: %s" "$target_time"

        sleep $sleep_interval

        now_epoch=$(date +%s)
        remaining=$((target_epoch - now_epoch))
    done

    # Clear countdown line and show start message
    printf "\r%-80s\r" ""  # Clear the line
    log_info "✅ Scheduled time reached! Starting implementation loop..."
    echo ""
}

# ============================================================================
# Git Change Detection
# ============================================================================

# Check if there are uncommitted changes (staged or unstaged)
has_uncommitted_changes() {
    [[ -n "$(git status --porcelain 2>/dev/null)" ]]
}
