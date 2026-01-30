#!/bin/bash
# impl-continue.prompt.sh - Continuation iteration implementation prompt
# Part of Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

# Source shared sections
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/impl-shared.sh"

# Generate continuation implementation prompt
_generate_impl_continue_prompt() {
    local tasks_file="$1"
    local feedback="$2"
    
    cat << CONTINUE_END
Continue implementing tasks from: $tasks_file

VALIDATION CAUGHT YOUR LIES:
$feedback

YOU MUST FIX YOUR LIES NOW.

REMEMBER:
- YOU CANNOT CHANGE SCOPE
- YOU CANNOT DECIDE TASKS ARE N/A
- YOU CANNOT REWRITE TASKS
- IF TASK SAYS REMOVE → REMOVE IT
- NO EXCUSES. NO OPINIONS. JUST DO IT.

CRITICAL - DO NOT WRITE TESTS FOR NON-EXISTENT FUNCTIONALITY:
- If you write a test that expects a keyboard shortcut → IMPLEMENT THE HANDLER FIRST
- If you write a test that calls a function → CREATE THE FUNCTION FIRST
- If you write a test that hits an API endpoint → REGISTER THE ROUTE FIRST
- If you write a test that expects a UI element → RENDER THE ELEMENT FIRST
- Implementation FIRST, then tests. Not tests INSTEAD OF implementation.
- Tests for features you didn't implement = INADMISSIBLE = Automatic failure

$(_get_evidence_rules)

$(_get_playwright_rules)

When done, output:
\`\`\`json
{
  "RALPH_STATUS": {
    "completed_tasks": ["task IDs you ACTUALLY completed"],
    "blocked_tasks": ["tasks with REAL blockers only"],
    "notes": "what you did"
  }
}
\`\`\`

FIX YOUR MISTAKES NOW.
CONTINUE_END
}
