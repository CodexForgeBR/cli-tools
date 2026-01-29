# PR Merger Skill

Fully automated Pull Request merge orchestration that handles CI failures, CodeRabbit feedback, merging, and cleanup.

## Overview

The PR Merger skill shepherds a Pull Request through the complete merge lifecycle:

1. **CI/CD Monitoring** - Identifies and fixes failing GitHub Actions checks
2. **CodeRabbit Feedback Loop** - Iteratively addresses review comments until satisfied
3. **Merge Execution** - Rebases and merges when ready
4. **Branch Cleanup** - Deletes local and remote branches

## Quick Start

### 1. Configure Your Repository

Create `.claude/pr-merger.json` in your repository root:

```json
{
  "testCommand": "dotnet test --no-build --verbosity normal",
  "buildCommand": "dotnet build",
  "maxCIFixIterations": 5,
  "maxCodeRabbitIterations": 5
}
```

**Required:**
- `testCommand` - Command to run tests (e.g., `npm test`, `cargo test`, `pytest`)
  - **Note:** Optional if `skipTests: true` is set

**Optional:**
- `skipTests` - Set to `true` to skip test execution (default: `false`)
  - Use for projects without tests
  - Build command will still run if specified
- `buildCommand` - Build command to run before tests
- `preMergeChecks` - Array of check names required before merge
- `requiredChecks` - Array of GitHub check names that must pass
- `maxCIFixIterations` - Max attempts to fix CI (default: 5)
- `maxCodeRabbitIterations` - Max feedback loops (default: 5)

### 2. Use the Skill

```bash
# In Claude Code CLI
/pr-merger 123

# Or just ask naturally
"Merge PR #456"
"Shepherd PR #789 to completion"
"Get PR #100 ready to merge"
```

## How It Works

### Phase 1: CI/CD Monitoring

```
Check GitHub Actions → Identify failures → Fix issues → Run tests → Push → Repeat
```

- Automatically identifies failing checks
- Analyzes failure logs
- Applies fixes
- Runs tests locally before pushing
- Polls for check completion
- Max 5 iterations to prevent infinite loops

### Phase 2: CodeRabbit Feedback

```
Fetch comments → Apply ALL fixes (NO SKIPPING) → Test → Push → Post response → Wait for CodeRabbit → Repeat
```

- Fetches only NEW comments since last commit (uses timestamps)
- **MUST apply fixes for ALL comments - NO exceptions, NO skipping**
- **NO authority to judge which comments are "important enough" to fix**
- Fixes EVERY comment regardless of severity (Trivial, Minor, Major, Critical)
- Runs tests before pushing (safety check)
- Posts response tagging @coderabbitai
- Polls for CodeRabbit response (15s intervals, 3 min max)
- Parses response to check if satisfied or more issues
- Loops until CodeRabbit confirms resolution
- Max 5 iterations
- **If max iterations reached with unresolved comments: merge FAILS**

### Phase 3: Merge PR

```
Validate checks → Validate mergeable → Execute rebase merge → Auto-delete remote branch
```

- Verifies all CI checks pass
- Checks for merge conflicts
- Validates PR is mergeable
- Executes: `gh pr merge --rebase --delete-branch`
- Reports if approval required but missing

### Phase 4: Branch Cleanup

```
Switch to main → Pull latest → Delete local branch → Delete remote branch
```

- Switches to main/master branch
- Pulls latest changes
- Safely deletes local feature branch
- Deletes remote branch (if not already auto-deleted)
- Never deletes protected branches

## Configuration Examples

### Node.js Project

```json
{
  "testCommand": "npm test",
  "buildCommand": "npm run build",
  "preMergeChecks": ["build", "test", "lint"],
  "maxCIFixIterations": 5,
  "maxCodeRabbitIterations": 5
}
```

### Python Project

```json
{
  "testCommand": "pytest tests/",
  "buildCommand": "",
  "preMergeChecks": ["test"],
  "maxCIFixIterations": 3,
  "maxCodeRabbitIterations": 3
}
```

### Rust Project

```json
{
  "testCommand": "cargo test",
  "buildCommand": "cargo build --release",
  "preMergeChecks": ["build", "test", "clippy"],
  "maxCIFixIterations": 5,
  "maxCodeRabbitIterations": 5
}
```

### .NET Project

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

### Project Without Tests

```json
{
  "skipTests": true,
  "buildCommand": "npm run build",
  "maxCIFixIterations": 5,
  "maxCodeRabbitIterations": 5
}
```

## Safety Features

### 🛡️ Never Pushes Failing Tests
All tests must pass locally before any push. If tests fail, the agent STOPS and reports to you.

### 🚫 NEVER Skips CodeRabbit Feedback
**CRITICAL**: The agent has **ZERO authority** to skip ANY CodeRabbit comments.
- ALL comments must be addressed (Trivial, Minor, Major, Critical)
- NO judgment calls about which comments are "important enough"
- If max iterations reached with unresolved comments: **merge FAILS**

### 🔄 Iteration Limits
- Max 5 CI fix attempts (configurable)
- Max 5 CodeRabbit feedback loops (configurable)
- Prevents infinite loops
- **If limits reached with unresolved issues: FAILS instead of proceeding**

### 🔒 Protected Branch Detection
Never deletes main, master, develop, staging, or production branches.

### ✅ Pre-Merge Validation
- All CI checks must pass
- All CodeRabbit comments must be resolved
- No merge conflicts
- PR must be in mergeable state
- Reports if approvals required

### 📊 Clear Reporting
Every step is logged with clear success/failure indicators.

## Dependencies

### Required Tools

1. **GitHub CLI (`gh`)**
   ```bash
   # macOS
   brew install gh

   # Authenticate
   gh auth login
   ```

2. **CodeRabbit Comments Script**

   Already available in CodexForge CLI tools:
   ```bash
   get-coderabbit-comments-with-timestamps.sh
   ```

3. **Git**

   Standard Git installation with push access to repository.

4. **Configuration File**

   `.claude/pr-merger.json` in repository root.

## Troubleshooting

### "Configuration file not found"

Create `.claude/pr-merger.json` in your repository root.

For projects with tests:
```json
{
  "testCommand": "your test command here"
}
```

For projects without tests:
```json
{
  "skipTests": true
}
```

### "Tests failed after applying fixes"

The agent stops when tests fail to prevent pushing broken code. Review the test output, fix manually, then re-run the agent.

### "Max iterations reached"

The agent tried 5 times but couldn't resolve all issues automatically. Review remaining issues manually.

### "PR is not mergeable"

Possible causes:
- Merge conflicts exist (requires manual resolution)
- Required approvals missing
- Branch protection rules not satisfied

### "CodeRabbit did not respond within 3 minutes"

The agent waited but CodeRabbit didn't reply. It will still check for new comments via timestamp queries and continue if none found.

## Examples

### Successful Merge

```bash
$ /pr-merger 123

🔍 PR Merger Agent Starting...
✅ All prerequisites met

Phase 1: CI/CD Monitoring
✅ All checks passing

Phase 2: CodeRabbit Feedback
Found 2 comments, applying fixes...
✅ CodeRabbit confirmed: "LGTM!"

Phase 3: Merge PR
✅ Merged using rebase

Phase 4: Cleanup
✅ Branches cleaned up

✅ SUCCESS: PR #123 merged!
```

### With CI Fixes

```bash
$ /pr-merger 456

Phase 1: CI/CD Monitoring
❌ Found failing check: test
Fixing NullReferenceException...
✅ Fixed and pushed
⏳ Waiting for checks...
✅ All checks now passing

Phase 2: CodeRabbit Feedback
No comments found, skipping

Phase 3: Merge PR
✅ Merged successfully

✅ SUCCESS: PR #456 merged!
```

### With Multiple CodeRabbit Iterations

```bash
$ /pr-merger 789

Phase 2: CodeRabbit Feedback (Iteration 1)
Found 3 comments, applying fixes...
Posted response to CodeRabbit...
⏳ Waiting for response...
✅ CodeRabbit replied with 1 more issue

Phase 2: CodeRabbit Feedback (Iteration 2)
Found 1 comment, applying fix...
Posted response to CodeRabbit...
⏳ Waiting for response...
✅ CodeRabbit confirmed: "All addressed!"

Phase 3: Merge PR
✅ Merged successfully
```

## Advanced Usage

### Custom Max Iterations

Adjust per-repository based on complexity:

```json
{
  "testCommand": "npm test",
  "maxCIFixIterations": 3,
  "maxCodeRabbitIterations": 10
}
```

### Multiple Test Commands

Chain commands in `testCommand`:

```json
{
  "testCommand": "npm run lint && npm test && npm run integration-test"
}
```

### Skip Build Command

If your tests auto-build, omit `buildCommand`:

```json
{
  "testCommand": "cargo test"
}
```

## Integration with Other Skills

This skill works alongside:
- `/pre-pr-review` - Run local CodeRabbit scan before creating PR
- `/post-merge-cleanup` - Standalone branch cleanup (already included)
- `/coderabbit-workflow` - Legacy skill (superseded by this one)

## Contributing

This skill is part of the CodexForge CLI tools repository.

- **Repository**: https://github.com/CodexForgeBR/cli-tools
- **Issues**: https://github.com/CodexForgeBR/cli-tools/issues

## License

Part of CodexForge CLI Tools - Internal Developer Tooling
