---
name: pr-merger
description: Use when the user wants to shepherd a PR to merge. Monitors CI, fixes failures, addresses CodeRabbit feedback iteratively until satisfied, merges the PR, and cleans up branches. Trigger phrases: "merge PR", "shepherd PR", "get PR ready to merge", "finish PR".
allowed-tools: ["Task"]
---

# PR Merger Skill

Shepherds a Pull Request through the entire merge lifecycle:
1. Monitors CI/CD and fixes failures
2. Addresses CodeRabbit feedback iteratively until satisfied
3. Merges the PR when ready
4. Cleans up local and remote branches

## Usage

Activate this skill when you want to:
- Merge a PR end-to-end
- Shepherd a PR to completion
- Get a PR ready for merge

### Example Commands

```bash
# Merge a PR by number
/pr-merger 123

# User can also say:
"Merge PR #456"
"Shepherd PR #789 to completion"
"Get PR #100 ready to merge"
"Finish up PR #42"
"Monitor and merge PR #200"
```

## What This Skill Does

### Phase 1: CI/CD Monitoring & Fixing
- Checks GitHub Actions status
- Identifies failing checks
- Attempts to fix failures
- Re-runs checks until all pass (max 5 iterations)

### Phase 2: CodeRabbit Feedback Loop
- Fetches CodeRabbit review comments
- **MUST fix ALL comments - NO exceptions, NO skipping, NO judgment calls**
- Applies fixes for EVERY comment regardless of severity (Trivial, Minor, Major, Critical)
- Runs tests before pushing
- Posts response to CodeRabbit
- Waits for CodeRabbit response (polls every 15s, max 3 min)
- Repeats if CodeRabbit raises more issues (max 5 iterations)
- **If max iterations reached with unresolved comments: FAIL the merge - do NOT proceed**

### Phase 3: Merge PR
- Validates all checks pass
- Verifies no unresolved comments
- Executes rebase merge (`gh pr merge --rebase`)
- Auto-deletes remote branch

### Phase 4: Post-Merge Cleanup
- Switches to main branch
- Pulls latest changes
- Deletes local feature branch
- Verifies remote branch deletion

## Configuration Required

Each repository must have `.claude/pr-merger.json`:

```json
{
  "testCommand": "dotnet test --no-build --verbosity normal",
  "buildCommand": "dotnet build",
  "preMergeChecks": ["build", "test"],
  "requiredChecks": ["build", "test", "coderabbit"],
  "maxCIFixIterations": 5,
  "maxCodeRabbitIterations": 5
}
```

**Required fields:**
- `testCommand` - Command to run tests (optional if `skipTests: true`)

**Optional fields:**
- `skipTests` - Skip test execution (default: false) - allows projects without tests
- `buildCommand` - Build command to run before tests
- `preMergeChecks` - Checks to run before merging (default: `["test"]`)
- `requiredChecks` - GitHub check names that must pass
- `maxCIFixIterations` - Max CI fix attempts (default: 5)
- `maxCodeRabbitIterations` - Max CodeRabbit loops (default: 5)

**Example for projects without tests:**
```json
{
  "skipTests": true,
  "buildCommand": "npm run build"
}
```

## Dependencies

- `gh` CLI - GitHub API operations
- `get-coderabbit-comments-with-timestamps.sh` - Fetch CodeRabbit comments
- `.claude/pr-merger.json` - Per-repo configuration
- Git - Version control operations

## Safety Features

1. **Never push failing tests** - All tests must pass before pushing
2. **NEVER skip CodeRabbit feedback** - ALL comments must be addressed, regardless of severity
3. **Max iteration limits** - Prevents infinite loops; if reached with unresolved comments, merge FAILS
4. **Protected branch detection** - Never deletes main/master/develop
5. **Merge conflict detection** - Stops and reports if conflicts exist
6. **Approval requirement check** - Reports if approvals needed
7. **Config file required** - Stops if configuration missing

**CRITICAL RULE**: The agent has **ZERO authority** to decide which CodeRabbit comments to skip. Every comment (Trivial, Minor, Major, Critical) MUST be addressed in each iteration. If the agent cannot resolve all comments within max iterations, it MUST fail the merge and report to the user.

## Implementation

This skill delegates to a general-purpose agent via the Task tool for isolated execution.

### Agent Prompt Template

When invoking this skill, use this EXACT prompt structure:

```
You are the pr-merger-agent. Your job is to shepherd Pull Request #[PR_NUMBER] through the complete merge lifecycle:

**CRITICAL RULES - DO NOT VIOLATE:**
- You have ZERO authority to skip CodeRabbit feedback
- You MUST fix ALL comments regardless of severity (Trivial, Minor, Major, Critical)
- You MUST NOT make judgment calls about which comments are "important enough" to fix
- If max iterations is reached with unresolved comments, you MUST FAIL the merge
- NEVER proceed to merge if any CodeRabbit comments remain unaddressed

1. **Phase 1: CI/CD Monitoring & Fixing**
   - Check GitHub Actions status for PR #[PR_NUMBER]
   - Identify any failing checks
   - Attempt to fix failures
   - Re-run checks until all pass (max [maxCIFixIterations] iterations)
   - If max iterations reached with failing checks, FAIL and report to user

2. **Phase 2: CodeRabbit Feedback Loop**
   - Fetch CodeRabbit review comments using `get-coderabbit-comments-with-timestamps.sh [PR_NUMBER]`
   - **CRITICAL**: Address EVERY comment - no exceptions, no skipping
   - Apply fixes for ALL comments regardless of severity level
   - Run tests before pushing ([testCommand])
   - Push fixes ONLY if tests pass
   - Post response to CodeRabbit explaining what was fixed
   - Wait for CodeRabbit response (poll every 15s, max 3 min)
   - Repeat if CodeRabbit raises more issues (max [maxCodeRabbitIterations] iterations)
   - If max iterations reached with unresolved comments, FAIL the merge and report to user

3. **Phase 3: Merge PR**
   - Validate all checks pass
   - Validate no unresolved CodeRabbit comments exist
   - Verify no merge conflicts
   - Execute rebase merge: `gh pr merge [PR_NUMBER] --rebase --delete-branch`
   - Report if approvals are required but missing

4. **Phase 4: Post-Merge Cleanup**
   - Switch to main branch
   - Pull latest changes
   - Delete local feature branch (if not already deleted)
   - Verify remote branch deletion

**Configuration** (from `.claude/pr-merger.json`):
- testCommand: "[testCommand]"
- buildCommand: "[buildCommand]"
- maxCIFixIterations: [maxCIFixIterations]
- maxCodeRabbitIterations: [maxCodeRabbitIterations]

**Safety Rules:**
- NEVER push failing tests
- NEVER skip CodeRabbit feedback
- Stop if merge conflicts exist
- Stop if approvals are needed
- Respect max iteration limits
- Never delete protected branches (main/master/develop)

**Failure Conditions**:
- Tests fail after applying fixes
- Max CI iterations reached with failing checks
- Max CodeRabbit iterations reached with unresolved comments
- Merge conflicts exist
- PR is not in mergeable state

Start with Phase 1 and work through each phase sequentially. Report clear success/failure status at each phase.
```
