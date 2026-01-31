#!/usr/bin/env bash
# Rate limit detection and sleep-until-reset functionality

# Python script location (LIB_DIR is /path/to/ralph-loop, so go up one level)
RATE_LIMIT_PARSER="${LIB_DIR}/../ralph-loop-python/rate_limit_parser.py"

# check_rate_limit(file)
#
# Checks if the given file contains rate limit messages and parses reset time.
# Sets global variables on success:
#   - RATE_LIMIT_RESET_EPOCH
#   - RATE_LIMIT_RESET_HUMAN
#   - RATE_LIMIT_RESET_TZ
#
# Returns:
#   0 - Rate limit detected and parsed
#   1 - No rate limit detected
#
# If rate limit is detected but time is unparseable, falls back to 15-minute wait.
check_rate_limit() {
    local file="$1"

    if [[ ! -f "$file" ]]; then
        log_error "check_rate_limit: File not found: $file" >&2
        return 1
    fi

    # Call Python parser
    local output
    local exit_code
    output=$(python3 "$RATE_LIMIT_PARSER" "$file" 2>&1)
    exit_code=$?

    case $exit_code in
        0)
            # Rate limit found and parsed successfully
            local epoch human tz
            epoch=$(echo "$output" | sed -n '1p')
            human=$(echo "$output" | sed -n '2p')
            tz=$(echo "$output" | sed -n '3p')

            if [[ -z "$epoch" || -z "$human" || -z "$tz" ]]; then
                log_error "check_rate_limit: Parser returned invalid output" >&2
                return 1
            fi

            # Set global variables
            RATE_LIMIT_RESET_EPOCH="$epoch"
            RATE_LIMIT_RESET_HUMAN="$human"
            RATE_LIMIT_RESET_TZ="$tz"

            log_info "Rate limit detected. Reset time: $human ($tz)" >&2
            return 0
            ;;

        1)
            # No rate limit detected
            return 1
            ;;

        2)
            # Rate limit detected but time is unparseable - fallback to 15-minute wait
            log_warn "Rate limit detected but reset time could not be parsed" >&2
            log_warn "Falling back to 15-minute wait" >&2

            local now fallback_epoch fallback_human
            now=$(date +%s)
            fallback_epoch=$((now + 900))  # 15 minutes from now
            fallback_human=$(date -r "$fallback_epoch" '+%Y-%m-%d %H:%M:%S %Z')

            RATE_LIMIT_RESET_EPOCH="$fallback_epoch"
            RATE_LIMIT_RESET_HUMAN="$fallback_human"
            RATE_LIMIT_RESET_TZ="local"

            return 0
            ;;

        *)
            # Parser error
            log_error "check_rate_limit: Parser failed with exit code $exit_code" >&2
            log_error "$output" >&2
            return 1
            ;;
    esac
}

# wait_for_rate_limit_reset(epoch, human, tz)
#
# Waits until the rate limit reset time with adaptive countdown.
# Does NOT change CURRENT_PHASE (keeps the parent phase for simplicity).
# Sends notification and saves state for Ctrl+C safety.
#
# Args:
#   epoch - Unix timestamp when rate limit resets
#   human - Human-readable reset time
#   tz    - Timezone string
wait_for_rate_limit_reset() {
    local reset_epoch="$1"
    local reset_human="$2"
    local reset_tz="$3"

    local now
    now=$(date +%s)

    local wait_seconds=$((reset_epoch - now))

    if [[ $wait_seconds -le 0 ]]; then
        log_info "Rate limit should have reset already, continuing immediately" >&2
        return 0
    fi

    # Send notification
    send_notification "rate_limited" "Rate limit hit" "Waiting until $reset_human ($reset_tz)"

    # Show banner
    log_info "" >&2
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" >&2
    log_info "  RATE LIMIT DETECTED" >&2
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" >&2
    log_info "" >&2
    log_info "  Reset time: $reset_human ($reset_tz)" >&2
    log_info "  Waiting: $(format_duration "$wait_seconds")" >&2
    log_info "" >&2
    log_info "  Press Ctrl+C to cancel" >&2
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" >&2
    log_info "" >&2

    # Adaptive countdown (same pattern as wait_until_scheduled_time)
    while [[ $wait_seconds -gt 0 ]]; do
        now=$(date +%s)
        wait_seconds=$((reset_epoch - now))

        if [[ $wait_seconds -le 0 ]]; then
            break
        fi

        # Adaptive sleep interval
        local sleep_interval=60  # Default: 1 minute
        if [[ $wait_seconds -lt 60 ]]; then
            sleep_interval=5  # Last minute: every 5 seconds
        elif [[ $wait_seconds -lt 300 ]]; then
            sleep_interval=30  # Last 5 minutes: every 30 seconds
        fi

        # Show countdown
        local formatted_time
        formatted_time=$(format_duration "$wait_seconds")
        echo -ne "\r  Time remaining: $formatted_time  " >&2

        # Sleep (capped to remaining time)
        if [[ $sleep_interval -gt $wait_seconds ]]; then
            sleep_interval=$wait_seconds
        fi
        sleep "$sleep_interval"

        # Save state periodically for Ctrl+C safety (every minute)
        if [[ $((wait_seconds % 60)) -eq 0 ]]; then
            save_state
        fi
    done

    echo "" >&2
    log_info "Rate limit reset time reached, resuming" >&2
    log_info "" >&2

    return 0
}

# format_duration(seconds)
#
# Formats seconds into human-readable duration.
# Examples: "2h 15m", "45m 30s", "30s"
format_duration() {
    local total_seconds="$1"
    local hours=$((total_seconds / 3600))
    local minutes=$(( (total_seconds % 3600) / 60 ))
    local seconds=$((total_seconds % 60))

    local parts=()
    if [[ $hours -gt 0 ]]; then
        parts+=("${hours}h")
    fi
    if [[ $minutes -gt 0 ]]; then
        parts+=("${minutes}m")
    fi
    if [[ $seconds -gt 0 || ${#parts[@]} -eq 0 ]]; then
        parts+=("${seconds}s")
    fi

    local result=""
    for part in "${parts[@]}"; do
        if [[ -n "$result" ]]; then
            result="$result "
        fi
        result="$result$part"
    done

    echo "$result"
}
