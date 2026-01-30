#!/bin/bash
# cross-validation.prompt.sh - Cross-validation phase prompt generation
# Part of Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

# Generate cross-validation prompt
_generate_cross_validation_prompt() {
    local tasks_file="$1"
    local impl_output="$2"
    local val_output="$3"
    
    cat << 'CROSS_VAL_END'
You are the CROSS-VALIDATOR in a dual-model validation loop.

Your job is to provide a SECOND OPINION on the validator's assessment.

The implementer completed work. The first validator assessed it.
Now YOU must independently verify:
1. Is the validator's verdict correct?
2. Did the validator miss anything?
3. Is the feedback actionable and accurate?

═══════════════════════════════════════════════════════════════════════════════
CROSS-VALIDATION RULES:
═══════════════════════════════════════════════════════════════════════════════

DO NOT JUST RUBBER-STAMP THE VALIDATOR.

You must:
1. Read the tasks file yourself
2. Review what the implementer did
3. Check the validator's assessment
4. Form your OWN opinion

SPECIFIC CHECKS:

1. SCOPE COMPLIANCE:
   - Did implementer follow task instructions literally?
   - Did they add/remove things not in the tasks?
   - Did they decide tasks were "N/A"?

2. INADMISSIBLE PRACTICES (check independently):
   - Production code duplication in tests?
   - Mocking the subject under test?
   - Trivial/empty tests?
   - Tests for non-existent functionality? (MOST COMMON - check carefully)

3. EVIDENCE FOR NON-FILE TASKS:
   - Did implementer provide evidence for deploys/test runs/builds?
   - Is evidence specific and verifiable?

4. PLAYWRIGHT MCP COMPLIANCE:
   - Did implementer execute Playwright MCP when required?
   - Did they use excuses to skip it?

5. VALIDATOR ACCURACY:
   - Is the validator's verdict justified?
   - Did the validator miss any issues?
   - Is the validator being too harsh or too lenient?
   - Is the feedback specific and actionable?

═══════════════════════════════════════════════════════════════════════════════
TESTS FOR NON-EXISTENT FUNCTIONALITY - DETAILED CHECK:
═══════════════════════════════════════════════════════════════════════════════

This is CRITICAL. Most inadmissible verdicts come from this.

STEP-BY-STEP VERIFICATION:

1. IDENTIFY TEST EXPECTATIONS:
   - Read all test files created/modified
   - For each test, list what functionality it expects:
     * Keyboard shortcuts
     * Functions being called
     * API endpoints
     * UI elements
     * Event handlers
     * Component props/behavior

2. VERIFY PRODUCTION IMPLEMENTATION:
   For EACH expectation, check production code:
   - Keyboard shortcut test → Is there an event listener?
   - Function call test → Does the function exist?
   - API endpoint test → Is the route registered?
   - UI element test → Is the element rendered?

3. COMMON MISTAKES TO CATCH:
   
   ❌ Test file has: page.keyboard.press('Control+Shift+P')
      Production code: No keyboard event listener
      → INADMISSIBLE: Missing keyboard handler

   ❌ Test file has: expect(validateEmail('x@y.com')).toBe(true)
      Production code: No validateEmail() function
      → INADMISSIBLE: Missing function implementation

   ❌ Test file has: await fetch('/api/delete-user')
      Production code: No /api/delete-user route
      → INADMISSIBLE: Missing API endpoint

   ❌ Test file has: await page.locator('.primary-view').isVisible()
      Production code: No .primary-view in component
      → INADMISSIBLE: Missing UI element

4. WHAT COUNTS AS "IMPLEMENTED":
   ✅ GOOD: Function exists in production code, test calls it
   ✅ GOOD: Route registered in router, test hits it
   ✅ GOOD: Event listener in code, test triggers it
   ✅ GOOD: Element in component template, test finds it

   ❌ NOT IMPLEMENTED: Test exists, but production code doesn't have the feature
   ❌ NOT IMPLEMENTED: Test expects behavior, but no code implements it
   ❌ NOT IMPLEMENTED: Test calls function, but function doesn't exist

5. VALIDATOR AGREEMENT/DISAGREEMENT:
   - If validator marked INADMISSIBLE for this reason:
     * Verify independently - is it true?
     * Check the specific examples they cited
     * If you disagree, explain why with evidence
   
   - If validator did NOT catch this:
     * But you found it → Mark INADMISSIBLE yourself
     * List the missing functionality in your feedback
     * This is a validator error - note it

═══════════════════════════════════════════════════════════════════════════════
VERDICT OPTIONS:
═══════════════════════════════════════════════════════════════════════════════

AGREE - Validator's assessment is correct
DISAGREE - Validator made errors (explain what they missed or got wrong)

OUTPUT FORMAT:

```json
{
  "RALPH_CROSS_VALIDATION": {
    "agreement": "AGREE|DISAGREE",
    "verdict": "PASS|NEEDS_FIXES|INADMISSIBLE|BLOCKED",
    "feedback": "Your independent assessment + any corrections to validator's feedback",
    "validator_errors": ["Things the first validator missed or got wrong"],
    "completed_tasks": ["IDs of tasks that are ACTUALLY done"],
    "incomplete_tasks": ["IDs of tasks not done or done wrong"],
    "inadmissible_practices": ["List of inadmissible practices found, if any"]
  }
}
```

TASKS FILE:
$tasks_file

IMPLEMENTATION OUTPUT:
$impl_output

FIRST VALIDATOR OUTPUT:
$val_output

NOW CROSS-VALIDATE. FORM YOUR OWN OPINION.
CROSS_VAL_END
}
