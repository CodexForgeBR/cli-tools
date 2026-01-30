#!/usr/bin/env bash
################################################################################
# RALPH Loop - Recursive Agent Learning & Performance Harness
################################################################################
#
# This is a modular agent orchestration framework that runs continuous
# AI-powered task execution loops with automatic error recovery, state
# persistence, and intelligent model selection.
#
# Key Features:
# - Task queue management with priority scheduling
# - Multi-model support (Claude, OpenAI, Gemini, local models)
# - Automatic error recovery and retry logic
# - State persistence across runs
# - Performance tracking and analytics
# - OpenClaw webhook notifications
# - Configurable via YAML
#
# Usage: ralph-loop.sh [options]
#        See -h/--help for full options
#
# Architecture: Modular design with separated concerns (see lib/ralph/)
################################################################################

set -e

# Resolve script directory (macOS compatible)
if [[ "$OSTYPE" == "darwin"* ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
else
    SCRIPT_DIR="$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")"
fi

# Set library paths
LIB_DIR="${SCRIPT_DIR}/lib/ralph-loop"
PYTHON_DIR="${SCRIPT_DIR}/lib/ralph-loop-python"

# Source all modules in dependency order
source "${LIB_DIR}/globals.sh"
source "${LIB_DIR}/logging.sh"
source "${LIB_DIR}/config.sh"
source "${LIB_DIR}/notifications.sh"
source "${LIB_DIR}/scheduling.sh"
source "${LIB_DIR}/cli.sh"
source "${LIB_DIR}/models.sh"
source "${LIB_DIR}/tasks.sh"
source "${LIB_DIR}/state.sh"
source "${LIB_DIR}/json-parsing.sh"
source "${LIB_DIR}/ai-runners.sh"
source "${LIB_DIR}/phases.sh"
source "${LIB_DIR}/main-loop.sh"

# Set up cleanup trap
trap cleanup INT TERM

# Run main entry point
main "$@"
