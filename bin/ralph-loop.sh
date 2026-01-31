#!/bin/bash

# Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development
# Based on the Ralph Wiggum technique by Geoffrey Huntley (May 2025)
#
# Usage: ralph-loop.sh [OPTIONS]
#
# Options:
#   -v, --verbose            Pass verbose flag to claude code cli
#   --ai CLI                 AI CLI to use: claude or codex (default: claude)
#   --max-iterations N       Maximum loop iterations (default: 20)
#   --max-inadmissible N     Max inadmissible violations before escalation (default: 5)
#   --implementation-model   Model for implementation (default: opus for claude, config default for codex)
#   --validation-model       Model for validation (default: opus for claude, config default for codex)
#   --tasks-file PATH        Path to tasks.md (auto-detects: ./tasks.md, specs/*/tasks.md)
#
# Exit Codes:
#   0 - All tasks completed successfully
#   1 - Error (no tasks.md, invalid params, etc.)
#   2 - Max iterations reached without completion
#   3 - Escalation requested by validator
#   4 - Tasks blocked - human intervention needed
#   5 - Tasks don't properly implement the plan
#   6 - Repeated inadmissible practices (max violations exceeded)

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
source "${LIB_DIR}/rate-limit.sh"
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
