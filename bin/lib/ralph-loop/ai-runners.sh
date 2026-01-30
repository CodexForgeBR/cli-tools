#!/bin/bash
# ai-runners.sh - AI execution and output parsing for ralph-loop
# Part of Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

# PYTHON_DIR should be set by the main script
# Default fallback for standalone testing
: "${PYTHON_DIR:=$(dirname "${BASH_SOURCE[0]}")/../ralph-loop-python}"

# Extract text content from claude --output-format stream-json output
# Args: json_file output_file
# Returns: 0 on success, 1 on failure
extract_text_from_stream_json() {
    local json_file=$1
    local output_file=$2

    # Use standalone Python script
    python3 "$PYTHON_DIR/stream_parser.py" stream "$json_file" "$output_file"
}

# Extract text content from codex JSONL output
# Args: json_file output_file
# Returns: 0 on success, 1 on failure
extract_text_from_codex_jsonl() {
    local json_file=$1
    local output_file=$2

    # Use standalone Python script
    python3 "$PYTHON_DIR/stream_parser.py" codex "$json_file" "$output_file"
}

# Run claude with timeout and zombie detection (stream-json workaround)
# Uses --output-format stream-json to detect completion via "type":"result" message
# before the CLI hangs on "No messages returned" error
# Args: output_file timeout_seconds(deprecated) start_attempt start_delay claude_args...
# Returns: 0 on success, 1 on failure
# Note: timeout_seconds parameter is deprecated and ignored. Timeout is now controlled by:
#   - INACTIVITY_TIMEOUT: kills if no output for N seconds (default: 600s)
#   - MAX_TOTAL_TIMEOUT: absolute maximum duration (default: 7200s)
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
        log_info "Claude attempt $attempt/$max_retries (inactivity: ${INACTIVITY_TIMEOUT}s, max: ${MAX_TOTAL_TIMEOUT}s, stream-json mode)..." >&2

        # Clear output files
        > "$output_file"
        > "$raw_json_file"

        # Run claude with stream-json output format
        # The "type":"result" message is emitted BEFORE the hang occurs
        # Note: --verbose is required when combining --print with --output-format stream-json
        claude "${claude_args[@]}" --verbose --output-format stream-json > "$raw_json_file" 2>&1 &
        local claude_pid=$!

        local elapsed=0
        local result_received=0
        local grace_period_start=0
        local last_activity_time=$(date +%s)
        local last_file_size=0

        while kill -0 "$claude_pid" 2>/dev/null; do
            sleep 2
            elapsed=$((elapsed + 2))

            # Check for file activity (size change = Claude is working)
            local current_size=$(stat -c %s "$raw_json_file" 2>/dev/null || stat -f %z "$raw_json_file" 2>/dev/null || echo 0)
            if [[ "$current_size" -gt "$last_file_size" ]]; then
                last_activity_time=$(date +%s)
                last_file_size=$current_size
            fi

            # Check for successful result in stream-json output
            if [[ $result_received -eq 0 ]] && grep -q '"type":"result"' "$raw_json_file" 2>/dev/null; then
                result_received=1
                grace_period_start=$elapsed
                log_info "Result received, giving 2s grace period for clean exit..." >&2
            fi

            # Grace period after result
            if [[ $result_received -eq 1 ]]; then
                local grace_elapsed=$((elapsed - grace_period_start))
                if [[ $grace_elapsed -ge 2 ]]; then
                    log_warn "Grace period expired, killing hung process..." >&2
                    kill -9 "$claude_pid" 2>/dev/null || true
                    wait "$claude_pid" 2>/dev/null || true
                    break
                fi
            fi

            # Inactivity timeout (resets when Claude writes to stream)
            local inactivity=$(($(date +%s) - last_activity_time))
            if [[ $inactivity -ge $INACTIVITY_TIMEOUT ]]; then
                log_warn "Inactivity timeout (${INACTIVITY_TIMEOUT}s with no output) - killing process" >&2
                kill -9 "$claude_pid" 2>/dev/null || true
                wait "$claude_pid" 2>/dev/null || true
                break
            fi

            # Hard total timeout (safety cap)
            if [[ $elapsed -ge $MAX_TOTAL_TIMEOUT ]]; then
                log_warn "Hard timeout (${MAX_TOTAL_TIMEOUT}s total) - killing process" >&2
                kill -9 "$claude_pid" 2>/dev/null || true
                wait "$claude_pid" 2>/dev/null || true
                break
            fi

            # Fallback: Check for zombie error
            if grep -q "No messages returned" "$raw_json_file" 2>/dev/null; then
                log_warn "Detected 'No messages returned' - killing zombie process" >&2
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

# Run codex with timeout and inactivity detection (jsonl output)
# Args: output_file timeout_seconds(deprecated) start_attempt start_delay codex_args...
# Returns: 0 on success, 1 on failure
run_codex_with_timeout() {
    local output_file="$1"
    local timeout_secs="$2"
    local start_attempt="${3:-1}"
    local start_delay="${4:-5}"
    shift 4
    local -a codex_args=("$@")

    local max_retries=$MAX_CLAUDE_RETRY
    local retry_delay=$start_delay
    local attempt=$start_attempt

    local raw_json_file="${output_file%.txt}.jsonl"

    while [[ $attempt -le $max_retries ]]; do
        log_info "Codex attempt $attempt/$max_retries (inactivity: ${INACTIVITY_TIMEOUT}s, max: ${MAX_TOTAL_TIMEOUT}s, json mode)..." >&2

        > "$output_file"
        > "$raw_json_file"

        codex exec --json --output-last-message "$output_file" "${codex_args[@]}" > "$raw_json_file" 2>&1 &
        local codex_pid=$!

        local elapsed=0
        local last_activity_time=$(date +%s)
        local last_file_size=0

        while kill -0 "$codex_pid" 2>/dev/null; do
            sleep 2
            elapsed=$((elapsed + 2))

            local current_size=$(stat -c %s "$raw_json_file" 2>/dev/null || stat -f %z "$raw_json_file" 2>/dev/null || echo 0)
            if [[ "$current_size" -gt "$last_file_size" ]]; then
                last_activity_time=$(date +%s)
                last_file_size=$current_size
            fi

            local inactivity=$(($(date +%s) - last_activity_time))
            if [[ $inactivity -ge $INACTIVITY_TIMEOUT ]]; then
                log_warn "Inactivity timeout (${INACTIVITY_TIMEOUT}s with no output) - killing process" >&2
                kill -9 "$codex_pid" 2>/dev/null || true
                wait "$codex_pid" 2>/dev/null || true
                break
            fi

            if [[ $elapsed -ge $MAX_TOTAL_TIMEOUT ]]; then
                log_warn "Hard timeout (${MAX_TOTAL_TIMEOUT}s total) - killing process" >&2
                kill -9 "$codex_pid" 2>/dev/null || true
                wait "$codex_pid" 2>/dev/null || true
                break
            fi
        done

        wait "$codex_pid" 2>/dev/null || true

        if [[ -s "$output_file" ]]; then
            return 0
        fi

        if extract_text_from_codex_jsonl "$raw_json_file" "$output_file"; then
            log_info "Successfully extracted text from codex json output" >&2
            return 0
        fi

        if [[ $attempt -lt $max_retries ]]; then
            CURRENT_RETRY_ATTEMPT=$((attempt + 1))
            CURRENT_RETRY_DELAY=$((retry_delay * 2))

            save_state "running" "$CURRENT_ITERATION"

            log_warn "Attempt $attempt failed (no result received). Retrying in ${retry_delay}s..." >&2
            sleep "$retry_delay"
            retry_delay=$((retry_delay * 2))
        fi

        attempt=$((attempt + 1))
    done

    log_error "Codex failed after $max_retries attempts" >&2
    return 1
}
