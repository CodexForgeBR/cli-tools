#!/bin/bash
# json-parsing.sh - JSON extraction and parsing for ralph-loop
# Part of Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

# PYTHON_DIR should be set by the main script
# Default fallback for standalone testing
: "${PYTHON_DIR:=$(dirname "${BASH_SOURCE[0]}")/../ralph-loop-python}"

extract_json_from_file() {
    local file_path=$1
    local json_type=$2  # RALPH_STATUS or RALPH_VALIDATION

    # Use standalone Python script for robust JSON extraction
    python3 "$PYTHON_DIR/json_extractor.py" "$file_path" "$json_type"
}

# Parse validation verdict
parse_verdict() {
    local json=$1
    echo "$json" | python3 "$PYTHON_DIR/json_field.py" verdict 2>/dev/null || echo "PARSE_ERROR"
}

# Parse validation feedback
parse_feedback() {
    local json=$1
    echo "$json" | python3 "$PYTHON_DIR/json_field.py" feedback 2>/dev/null || echo "Could not parse feedback"
}

# Parse remaining unchecked count from validation
parse_remaining() {
    local json=$1
    echo "$json" | python3 "$PYTHON_DIR/json_field.py" remaining 2>/dev/null || echo "-1"
}

# Parse confirmed blocked count from validation
parse_blocked_count() {
    local json=$1
    echo "$json" | python3 "$PYTHON_DIR/json_field.py" blocked_count 2>/dev/null || echo "0"
}

# Parse blocked tasks list from validation (returns formatted string)
parse_blocked_tasks() {
    local json=$1
    echo "$json" | python3 "$PYTHON_DIR/json_field.py" blocked_tasks 2>/dev/null || echo "Could not parse blocked tasks"
}
