#!/bin/bash
# final-plan.prompt.sh - Final plan validation phase prompt generation
# Part of Ralph Loop - Dual-Model Validation Loop for Spec-Driven Development

# Generate final plan validation prompt
_generate_final_plan_validation_prompt() {
    local spec_file="$1"
    local tasks_file="$2"
    local plan_file="$3"
    
    cat << FINAL_PLAN_END
You are validating the final implementation plan before execution begins.

This is the LAST CHECKPOINT before the implementer starts work.

Your job is to ensure:
1. The plan correctly interprets the spec
2. The plan is complete and covers all requirements
3. The plan is actionable and won't cause confusion
4. The plan stays in scope

═══════════════════════════════════════════════════════════════════════════════
VALIDATION CHECKLIST:
═══════════════════════════════════════════════════════════════════════════════

1. Does the plan address all requirements from spec.md?
2. Are there any misinterpretations of the spec?
3. Is the plan adding features not requested in the spec?
4. Are the tasks clear enough to execute without ambiguity?
5. Is there a verification/testing strategy?
6. Are there any obvious gaps or missing steps?

VERDICT OPTIONS:

APPROVE - Plan is ready for execution
REJECT - Plan has issues that must be fixed (list them)

OUTPUT FORMAT:

\`\`\`json
{
  "RALPH_FINAL_PLAN_VALIDATION": {
    "verdict": "APPROVE|REJECT",
    "feedback": "Issues found, or approval confirmation"
  }
}
\`\`\`

SPEC FILE:
$spec_file

TASKS FILE (if different from plan):
$tasks_file

PLAN FILE:
$plan_file

NOW VALIDATE.
FINAL_PLAN_END
}
