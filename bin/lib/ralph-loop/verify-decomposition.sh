#!/usr/bin/env bash
#
# verify-decomposition.sh - Verify main-loop.sh structure
#
# This script checks that all expected functions are defined in main-loop.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MAIN_LOOP="$SCRIPT_DIR/main-loop.sh"

# Expected functions in order
EXPECTED_FUNCTIONS=(
    "cleanup"
    "main_init"
    "main_handle_commands"
    "main_display_banner"
    "main_find_tasks"
    "main_handle_resume"
    "main_validate_setup"
    "main_fetch_github_issue"
    "main_tasks_validation"
    "main_handle_schedule"
    "main_run_post_validation_chain"
    "main_exit_success"
    "main_iteration_loop"
    "main_handle_verdict"
    "main"
)

echo "Verifying main-loop.sh structure..."
echo ""

# Check file exists
if [[ ! -f "$MAIN_LOOP" ]]; then
    echo "ERROR: main-loop.sh not found at: $MAIN_LOOP"
    exit 1
fi

echo "✓ File exists: $MAIN_LOOP"
echo ""

# Check for each expected function
missing_functions=0
for func in "${EXPECTED_FUNCTIONS[@]}"; do
    if grep -q "^${func}() {" "$MAIN_LOOP"; then
        echo "✓ Function defined: $func"
    else
        echo "✗ Function MISSING: $func"
        ((missing_functions++))
    fi
done

echo ""
echo "----------------------------------------"
echo "Total functions expected: ${#EXPECTED_FUNCTIONS[@]}"
echo "Total functions found: $((${#EXPECTED_FUNCTIONS[@]} - missing_functions))"
echo "Total functions missing: $missing_functions"
echo ""

if [[ $missing_functions -eq 0 ]]; then
    echo "✓ All functions present!"
    echo ""
    
    # Check file size
    lines=$(wc -l < "$MAIN_LOOP")
    size=$(ls -lh "$MAIN_LOOP" | awk '{print $5}')
    echo "File statistics:"
    echo "  Lines: $lines"
    echo "  Size: $size"
    echo ""
    
    # Check syntax
    if bash -n "$MAIN_LOOP" 2>/dev/null; then
        echo "✓ Bash syntax valid!"
    else
        echo "✗ Bash syntax errors detected!"
        exit 1
    fi
    
    echo ""
    echo "✓ Decomposition verified successfully!"
    exit 0
else
    echo "✗ Decomposition incomplete - missing functions!"
    exit 1
fi
