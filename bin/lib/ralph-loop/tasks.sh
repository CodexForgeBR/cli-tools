#!/bin/bash
# tasks.sh - Tasks file discovery and validation for ralph-loop
# Part of Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

# Find tasks.md file in project
# Returns: Path to tasks.md file
# Exit code: 0 if found, 1 if not found
find_tasks_file() {
    if [[ -n "$TASKS_FILE" ]]; then
        if [[ -f "$TASKS_FILE" ]]; then
            echo "$TASKS_FILE"
            return 0
        else
            log_error "Specified tasks file not found: $TASKS_FILE"
            return 1
        fi
    fi

    # Auto-detect tasks.md in common locations
    local search_paths=(
        "./tasks.md"
        "./TASKS.md"
        "./specs/tasks.md"
        "./spec/tasks.md"
    )

    for path in "${search_paths[@]}"; do
        if [[ -f "$path" ]]; then
            echo "$path"
            return 0
        fi
    done

    # Search in specs subdirectories
    local found
    found=$(find ./specs -name "tasks.md" -type f 2>/dev/null | head -1)
    if [[ -n "$found" ]]; then
        echo "$found"
        return 0
    fi

    found=$(find ./spec -name "tasks.md" -type f 2>/dev/null | head -1)
    if [[ -n "$found" ]]; then
        echo "$found"
        return 0
    fi

    log_error "No tasks.md file found. Create one or specify with --tasks-file"
    return 1
}

# Count unchecked tasks in tasks.md
# Args: $1 - Path to tasks file
# Returns: Number of unchecked tasks
count_unchecked_tasks() {
    local file=$1
    local count
    count=$(grep -c '^\s*- \[ \]' "$file" 2>/dev/null) || count=0
    echo "$count"
}

# Count checked tasks in tasks.md
# Args: $1 - Path to tasks file
# Returns: Number of checked tasks
count_checked_tasks() {
    local file=$1
    local count
    count=$(grep -c '^\s*- \[x\]' "$file" 2>/dev/null) || count=0
    echo "$count"
}

# Compute SHA256 hash of tasks.md file
# Args: $1 - Path to tasks file
# Returns: SHA256 hash or empty string if file not found
compute_tasks_hash() {
    local file=$1
    if [[ ! -f "$file" ]]; then
        echo ""
        return 1
    fi
    sha256sum "$file" | awk '{print $1}'
}
