#!/bin/bash
# tasks-validation.prompt.sh - Tasks validation phase prompt generation
# Part of Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

# Generate tasks validation prompt
_generate_tasks_validation_prompt() {
    local spec_file="$1"
    local tasks_file="$2"
    
    cat << TASKS_VAL_END
You are validating that a tasks.md file correctly implements a spec.md file.

Your job is to ensure the tasks are:
1. COMPLETE - Cover all requirements from the spec
2. ACCURATE - Match the spec's intent
3. ACTIONABLE - Clear, specific, testable
4. IN SCOPE - Don't add things not in the spec

═══════════════════════════════════════════════════════════════════════════════
VALIDATION RULES:
═══════════════════════════════════════════════════════════════════════════════

1. COMPLETENESS CHECK:
   - Read the spec file completely
   - Identify all functional requirements
   - Check if tasks.md covers each requirement
   - Missing requirements → INVALID

2. ACCURACY CHECK:
   - Do tasks correctly interpret the spec?
   - Are there misunderstandings or scope changes?
   - Do tasks add features not in the spec?
   - Inaccurate tasks → INVALID

3. ACTIONABILITY CHECK:
   - Is each task clear and specific?
   - Can someone implement it without guessing?
   - Are test criteria provided where needed?
   - Vague tasks → INVALID

4. SCOPE CONTROL:
   - Do tasks stay within the spec boundaries?
   - Are there "bonus" features not requested?
   - Out-of-scope additions → INVALID

═══════════════════════════════════════════════════════════════════════════════
COMMON ISSUES TO CATCH:
═══════════════════════════════════════════════════════════════════════════════

1. MISSING REQUIREMENTS:
   - Spec says "Add X feature" → No task for X
   - Spec says "Remove Y" → No task to remove Y
   - Spec has acceptance criteria → No task to verify them

2. MISINTERPRETATION:
   - Spec says "Remove X" → Task says "Keep X but..."
   - Spec says "Create A" → Task creates B instead
   - Spec says "Modify C" → Task rewrites C entirely

3. VAGUE TASKS:
   - Task: "Improve performance" → Too vague
   - Task: "Fix bugs" → Which bugs?
   - Task: "Update tests" → Update how?

4. SCOPE CREEP:
   - Spec never mentions Z → Task adds Z
   - Spec says minimal change → Task does major refactor
   - Spec focuses on X → Tasks include Y and Z too

5. MISSING VERIFICATION:
   - Task creates feature → No task to test it
   - Task modifies code → No task to verify it works
   - Task removes feature → No task to verify removal

═══════════════════════════════════════════════════════════════════════════════
TASK QUALITY STANDARDS:
═══════════════════════════════════════════════════════════════════════════════

GOOD TASKS:
✅ "Remove the Back button from the Banks view (file: src/app/banks/banks.component.html)"
✅ "Add keyboard shortcut Ctrl+Shift+P to open command palette - implement event handler in app.component.ts"
✅ "Run E2E tests with 'npm run e2e' and verify all pass (record results in notes)"
✅ "Deploy BCL to servidor environment using bcl/deploy.ps1 -env servidor (record version)"

BAD TASKS:
❌ "Improve the Banks view" - Too vague
❌ "Add some shortcuts" - Not specific
❌ "Make it better" - Meaningless
❌ "Fix issues" - Which issues?
❌ "Refactor code" - Why? What spec requirement?

═══════════════════════════════════════════════════════════════════════════════
VERDICT OPTIONS:
═══════════════════════════════════════════════════════════════════════════════

VALID - Tasks correctly implement the spec
INVALID - Issues found (list them specifically)

OUTPUT FORMAT:

\`\`\`json
{
  "RALPH_TASKS_VALIDATION": {
    "verdict": "VALID|INVALID",
    "feedback": "Specific issues and how to fix them",
    "missing_requirements": ["Requirements from spec not covered in tasks"],
    "out_of_scope_tasks": ["Tasks that add things not in spec"],
    "vague_tasks": ["Task IDs that need more clarity"],
    "quality_score": "Brief overall assessment"
  }
}
\`\`\`

SPEC FILE TO VALIDATE AGAINST:
$spec_file

TASKS FILE TO VALIDATE:
$tasks_file

NOW VALIDATE. BE THOROUGH.
TASKS_VAL_END
}
