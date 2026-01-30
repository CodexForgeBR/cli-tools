#!/bin/bash
# impl-shared.sh - Shared prompt sections for implementation phase
# Part of Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

# Inadmissible practices section - used in first iteration
_get_inadmissible_rules() {
    cat << 'INADMISSIBLE_END'
═══════════════════════════════════════════════════════════════════════════════
INADMISSIBLE PRACTICES - AUTOMATIC FAILURE
═══════════════════════════════════════════════════════════════════════════════

These practices will result in IMMEDIATE ESCALATION with INADMISSIBLE verdict.
Do NOT do any of these under any circumstances:

1. PRODUCTION CODE DUPLICATION IN TESTS:
   - DO NOT copy production logic into test files
   - DO NOT create "test helpers" that re-implement production algorithms
   - DO NOT create "test harnesses" that duplicate production code
   - Tests MUST import and call ACTUAL production code

   WRONG: class TestHelper { SameMethodAsProduction() { /* copied logic */ } }
   RIGHT: import { ProductionClass } from '@app/production';
          productionInstance.methodUnderTest();

2. MOCK THE SUBJECT UNDER TEST:
   - DO NOT mock the exact code you're supposed to be testing
   - Mocking dependencies is fine; mocking the subject = FAILURE

3. TRIVIAL/EMPTY TESTS:
   - DO NOT write tests that don't invoke production code
   - DO NOT write expect(true).toBe(true) style tests

4. TESTS FOR NON-EXISTENT FUNCTIONALITY - CRITICAL:
   - DO NOT write tests for functionality that doesn't exist in production code
   - If you write a test that expects functionality, that functionality MUST EXIST
   - Tests verify EXISTING features or NEW features you IMPLEMENT
   - Tests come AFTER implementation, not INSTEAD OF implementation

   EXAMPLES OF INADMISSIBLE TEST-WRITING:
   ❌ Write E2E test: page.keyboard.press('Control+Shift+P')
      But NEVER implement the keyboard event handler for Ctrl+Shift+P
      → INADMISSIBLE: Test for non-existent shortcut

   ❌ Write unit test: expect(validateEmail('test@test.com')).toBe(true)
      But NEVER create the validateEmail() function
      → INADMISSIBLE: Test for non-existent function

   ❌ Write integration test: await fetch('/api/delete-user')
      But NEVER register the /api/delete-user route
      → INADMISSIBLE: Test for non-existent endpoint

   ❌ Write E2E test: await page.locator('.primary-view').isVisible()
      But NEVER render a .primary-view element in the component
      → INADMISSIBLE: Test for non-existent UI element

   THE ONLY VALID PATTERN - TWO-STEP PROCESS:
   ✅ STEP 1: Implement the functionality in production code
      - Add keyboard event handler for Ctrl+Shift+P
      - Create validateEmail() function
      - Register /api/delete-user route
      - Render .primary-view element
   ✅ STEP 2: Write tests that verify the functionality you just implemented
      - Test that Ctrl+Shift+P calls the handler
      - Test that validateEmail() works correctly
      - Test that /api/delete-user responds
      - Test that .primary-view is visible

   DETECTION - VALIDATOR WILL CHECK:
   - Read your test files - what functionality do they expect?
   - Search production code - does that functionality exist?
   - If NOT FOUND → INADMISSIBLE verdict → You must fix it

   WHY THIS IS INADMISSIBLE:
   - You wrote tests but FORGOT to implement the actual feature
   - Tests will ALWAYS FAIL because the feature doesn't exist
   - This is not a minor bug - it's forgetting half the work
   - Cannot be fixed by tweaking tests - requires implementing missing features

   REMEMBER: Implementation first, then tests. Not tests instead of implementation.

If you violate these rules, the entire implementation will be marked INADMISSIBLE.
You will get explicit feedback on how to fix it, but repeated violations will
escalate to human intervention. Fix inadmissible practices IMMEDIATELY.
═══════════════════════════════════════════════════════════════════════════════
INADMISSIBLE_END
}

# Evidence capture rules - shared between both branches
_get_evidence_rules() {
    cat << 'EVIDENCE_END'
EVIDENCE CAPTURE FOR NON-FILE TASKS:
For tasks that don't just create/modify files, capture evidence in RALPH_STATUS.notes:

| Task Type | What to Record |
|-----------|----------------|
| Deploy X | Version deployed (e.g., "BCL 2026.1.23.4-servidor deployed") |
| Run tests | Results (e.g., "4238 passed, 3 skipped, 0 failed") |
| Build X | Result (e.g., "Build succeeded: 0 errors, 0 warnings") |
| Verify X | What you verified (e.g., "Packages exist on BaGet: curl confirmed") |
| Run/Execute X | Outcome (e.g., "Quickstart scenarios: all error messages match") |
| Playwright MCP | Screenshot path OR what was verified (e.g., "Navigated to localhost:4200/banks, verified no Back button, screenshot at validation/us1-banks.png") |

This evidence helps validation verify your work without re-running everything.
EVIDENCE_END
}

# Playwright MCP rules - shared between both branches
_get_playwright_rules() {
    cat << 'PLAYWRIGHT_END'
═══════════════════════════════════════════════════════════════════════════════
PLAYWRIGHT MCP VALIDATION - MANDATORY EXECUTION
═══════════════════════════════════════════════════════════════════════════════

When tasks.md contains tasks with "Playwright MCP" or "via Playwright MCP":

1. "APP NOT RUNNING" IS NOT A BLOCKER - START IT YOURSELF:
   - If the app isn't running → START IT using the command in the task
   - Wait for the server to respond before proceeding
   - If the build fails → FIX the build error, then start again
   - NEVER skip Playwright MCP tasks because "the app isn't running"

2. EXECUTION SEQUENCE:
   a. Start the application(s) per the task instructions
   b. Wait for HTTP response on the expected port
   c. Use Playwright MCP to navigate to the specified URL
   d. Perform the interactions described in the task
   e. Verify the expected elements/results
   f. Capture screenshots if a storage path is specified
   g. Record evidence in RALPH_STATUS.notes

3. FORBIDDEN EXCUSES (all result in INADMISSIBLE verdict):
   - "App not running" → START IT
   - "Server not started" → START IT
   - "Frontend not available" → START IT
   - "Can't use Playwright because app isn't running" → START THE APP
   - "Blocked by infrastructure" → FIX IT OR START IT
   - "Validated via code review instead" → WRONG METHOD, USE PLAYWRIGHT MCP

═══════════════════════════════════════════════════════════════════════════════
PLAYWRIGHT_END
}

# Learnings section - appended to all prompts
_get_learnings_section() {
    local learnings="$1"
    
    if [[ -n "$learnings" ]]; then
        cat << LEARNINGS_END

═══════════════════════════════════════════════════════════════════════════════
LEARNINGS FROM PREVIOUS ITERATIONS:
Read these FIRST before starting work. They contain important patterns and gotchas.
═══════════════════════════════════════════════════════════════════════════════

$learnings

Pay special attention to the 'Codebase Patterns' section at the top.
LEARNINGS_END
    fi
}

# Learnings output instruction - appended to all prompts
_get_learnings_output() {
    cat << 'LEARNINGS_OUTPUT_END'

═══════════════════════════════════════════════════════════════════════════════
LEARNINGS OUTPUT:
═══════════════════════════════════════════════════════════════════════════════

At the end of your work, output any NEW learnings in this format:
```
RALPH_LEARNINGS:
- Pattern: [describe any reusable pattern you discovered]
- Gotcha: [describe any gotcha or non-obvious requirement]
- Context: [describe any useful context for future iterations]
```

Only include GENERAL learnings that would help future iterations.
Do NOT include task-specific details.
LEARNINGS_OUTPUT_END
}
