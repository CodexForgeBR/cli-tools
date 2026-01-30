#!/bin/bash
# logging.sh - Logging functions for ralph-loop
# Part of Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_phase() {
    echo -e "\n${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}  $1${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

log_debug() {
    [[ -n "$VERBOSE" ]] && echo -e "${BLUE}[DEBUG]${NC} $1"
    return 0
}

# Format seconds into human readable time
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

# Get current timestamp in seconds
get_timestamp() {
    date +%s
}

# Get current git commit hash (short)
get_current_commit() {
    git rev-parse --short HEAD 2>/dev/null || echo ""
}

# Detect date command flavor (gnu or bsd)
detect_date_flavor() {
    if date --version >/dev/null 2>&1; then
        echo "gnu"
    else
        echo "bsd"
    fi
}
