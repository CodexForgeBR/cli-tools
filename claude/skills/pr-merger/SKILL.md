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
- Applies fixes
- Runs tests before pushing
- Posts response to CodeRabbit
- Waits for CodeRabbit response (polls every 15s, max 3 min)
- Repeats if CodeRabbit raises more issues (max 5 iterations)

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
2. **Max iteration limits** - Prevents infinite loops
3. **Protected branch detection** - Never deletes main/master/develop
4. **Merge conflict detection** - Stops and reports if conflicts exist
5. **Approval requirement check** - Reports if approvals needed
6. **Config file required** - Stops if configuration missing

## Implementation

This skill delegates to the `pr-merger-agent` sub-agent via the Task tool for isolated execution.
