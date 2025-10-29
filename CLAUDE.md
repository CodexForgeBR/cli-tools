# CodexForge Global Developer Configuration

This file provides global guidance to Claude Code across all CodexForge projects.

## CodexForge CLI Tools

This configuration applies to shared CLI tools available to all CodexForge developers.

### Repository Information
- **GitHub Repository**: https://github.com/CodexForgeBR/cli-tools
- **Local Path**: `~/source/cli-tools`
- **In PATH**: Yes (via `~/.zshrc`)

## Available Scripts

### get-coderabbit-comments.sh

**Purpose:** Fetches CodeRabbit review comments from GitHub Pull Requests.

**Usage:**
```bash
# Preferred method - just use the command name
get-coderabbit-comments.sh <PR_NUMBER>

# Or with full path if not in PATH
~/source/cli-tools/bin/get-coderabbit-comments.sh <PR_NUMBER>
```

**Example:**
```bash
# Fetch all CodeRabbit comments from PR #72
get-coderabbit-comments.sh 72
```

**Output:** Displays all inline CodeRabbit review comments with:
- File path where comment was made
- Line number
- Full comment body with severity level and suggestions

**IMPORTANT:** This script is the **preferred method** for analyzing CodeRabbit feedback. Always use this script instead of manually running `gh api` commands when the user asks to "read", "analyze", or "fetch" CodeRabbit comments.

### get-coderabbit-comments-with-timestamps.sh

**Purpose:** Fetches CodeRabbit review comments with optional timestamp filtering to get only NEW comments since a specific time.

**Usage:**
```bash
# Fetch all comments (same as original script)
get-coderabbit-comments-with-timestamps.sh <PR_NUMBER>

# Fetch only comments created after a specific timestamp
get-coderabbit-comments-with-timestamps.sh <PR_NUMBER> --since "<TIMESTAMP>"

# Or with full path if not in PATH
~/source/cli-tools/bin/get-coderabbit-comments-with-timestamps.sh <PR_NUMBER> --since "<TIMESTAMP>"
```

**Examples:**
```bash
# Fetch all comments from PR #72
get-coderabbit-comments-with-timestamps.sh 72

# Fetch only comments after a specific ISO 8601 timestamp
get-coderabbit-comments-with-timestamps.sh 72 --since "2025-10-23T10:30:00Z"

# Fetch only comments after last commit
LAST_COMMIT=$(git log -1 --format="%aI")
get-coderabbit-comments-with-timestamps.sh 72 --since "$LAST_COMMIT"
```

**Output:** Displays CodeRabbit review comments with:
- File path where comment was made
- Line number
- **Creation timestamp** (NEW)
- Full comment body with severity level and suggestions

**Use Case:** This script is essential for iterative review cycles where you want to address only the LATEST feedback without re-processing already-fixed issues. It's used by the `coderabbit-workflow` skill and `coderabbit-fixer` sub-agent for automated review handling.

**Timestamp Formats Supported:**
- ISO 8601: `2025-10-23T10:30:00Z`
- ISO 8601 with timezone: `2025-10-23T10:30:00-07:00`
- Readable format: `2025-10-23 10:30:00` (automatically converted)

## Global Development Guidelines

### Working with CodeRabbit

#### Local CodeRabbit Reviews (Pre-PR Workflow)

**Purpose:** Run comprehensive local CodeRabbit scans before creating pull requests.

**Activation:** When the user asks to run a local CodeRabbit scan, the `pre-pr-review` skill automatically activates and delegates to the `local-coderabbit-reviewer` sub-agent.

**Trigger Phrases:**
- "Run local coderabbit scan"
- "Do a local coderabbit review"
- "Review my code before PR"
- "Local CodeRabbit scan"

**What Happens Automatically:**
1. Runs scan: `coderabbit review --plain --base main --config CLAUDE.md`
2. Saves results to `.coderabbit/review.txt`
3. Analyzes all findings
4. **Presents a plan** for implementing fixes (always, even if not in plan mode)
5. Applies fixes systematically
6. Runs full test suite
7. **Only if all tests pass**: commits with descriptive message
8. **If tests fail**: stops and reports (does NOT commit)

**Components:**
- **CodeRabbit CLI**: `~/.local/bin/coderabbit` (must be installed and authenticated)
- **Sub-agent**: `~/.claude/agents/local-coderabbit-reviewer.md` (executes workflow)
- **Skill**: `~/.claude/skills/pre-pr-review/SKILL.md` (auto-activation)

**Safety Features:**
- Never commits if tests fail
- Always presents plan before making changes
- Respects project conventions and patterns
- Comprehensive scan (all changes vs main)

**Documentation:** See `~/.claude/skills/pre-pr-review/README.md` for complete workflow details.

**Manual Usage** (if needed):
```bash
# Review all changes (committed + uncommitted)
coderabbit review --plain --base main --config CLAUDE.md > .coderabbit/review.txt 2>&1

# Review only uncommitted changes
coderabbit review --plain --type uncommitted --config CLAUDE.md > .coderabbit/review.txt 2>&1

# Review only committed changes
coderabbit review --plain --type committed --base main --config CLAUDE.md > .coderabbit/review.txt 2>&1
```

**Installation:**
```bash
# Install CodeRabbit CLI
curl -fsSL https://cli.coderabbit.ai/install.sh | sh

# Authenticate
coderabbit auth login
```

#### GitHub PR CodeRabbit Reviews

When the user asks to analyze, read, or fetch CodeRabbit comments from PRs:
1. **Always use** `get-coderabbit-comments.sh <PR_NUMBER>`
2. Do NOT manually construct `gh api` commands
3. The script handles all the GitHub API complexity
4. Output is formatted for easy reading and analysis

#### Automated CodeRabbit Workflow (NEW)

**Purpose:** Automatically address CodeRabbit feedback in iterative review cycles.

**Activation:** When the user asks to "address", "fix", or "handle" CodeRabbit issues on a PR, the `coderabbit-workflow` skill automatically activates and delegates to the `coderabbit-fixer` sub-agent.

**Trigger Phrases:**
- "Address latest issues raised by coderabbit on PR #123"
- "Fix coderabbit comments on pull request 456"
- "Handle coderabbit feedback for PR #789"

**What Happens Automatically:**
1. Gets timestamp of last commit on current branch
2. Fetches only NEW CodeRabbit comments (using `get-coderabbit-comments-with-timestamps.sh`)
3. Analyzes and applies fixes systematically
4. Runs full test suite
5. **Only if all tests pass**: commits, pushes, and creates detailed GitHub comment
6. **If tests fail**: stops and reports (does NOT push)

**Components:**
- **Script**: `get-coderabbit-comments-with-timestamps.sh` (filters by timestamp)
- **Sub-agent**: `~/.claude/agents/coderabbit-fixer.md` (executes workflow)
- **Skill**: `~/.claude/skills/coderabbit-workflow/SKILL.md` (auto-activation)

**Safety Features:**
- Never pushes if tests fail
- Only processes new comments (avoids re-fixing)
- Creates detailed GitHub comments for transparency
- Respects project conventions and patterns

**Documentation:** See `~/.claude/skills/coderabbit-workflow/README.md` for complete workflow details.

#### Post-Merge Branch Cleanup (NEW)

**Purpose:** Safely delete merged feature branches with comprehensive validation to prevent data loss.

**Activation:** When the user asks to "clean up", "delete", or "remove" a merged branch, the `post-merge-cleanup` skill automatically activates and delegates to the `branch-cleanup` sub-agent.

**Trigger Phrases:**
- "Clean up after merge"
- "Delete merged branch"
- "Remove feature branch"
- "Post-merge cleanup"

**What Happens Automatically:**
1. Detects current feature branch (refuses if on protected branch like main/master)
2. Runs ALL safety validations (must ALL pass):
   - ✅ Check for uncommitted changes
   - ✅ Verify branch is merged to main (git)
   - ✅ Confirm GitHub PR is merged (via `gh` CLI)
   - ✅ Ensure no unpushed commits exist
3. **Only if ALL validations pass**: switches to main, pulls latest, deletes local and remote branches
4. **If ANY validation fails**: stops immediately and reports why (does NOT delete)

**Components:**
- **Sub-agent**: `~/.claude/agents/branch-cleanup.md` (executes workflow)
- **Skill**: `~/.claude/skills/post-merge-cleanup/SKILL.md` (auto-activation)

**Safety Features:**
- Never deletes unmerged branches (dual validation: git + GitHub)
- Prevents data loss from uncommitted or unpushed work
- Protected branch detection (never deletes main, master, develop, etc.)
- Clear failure reporting with remediation steps
- Fail-fast behavior (stops at first validation failure)

**Edge Cases Handled:**
- Squash/rebase merges (detects and provides manual cleanup instructions)
- Remote branch already deleted by GitHub (gracefully handles)
- GitHub CLI unavailable (falls back to git-only validation)

**Documentation:** See `~/.claude/skills/post-merge-cleanup/README.md` for complete workflow details.

#### Performance Testing Automation (NEW)

**Purpose:** Intelligently run performance tests, detect regressions, compare with baselines, and generate comprehensive reports.

**Activation:** When the user asks to "run performance tests", "check performance", or "test performance", the `performance-testing` skill automatically activates and delegates to the `perf-test-runner` sub-agent.

**Trigger Phrases:**
- "Run performance tests"
- "Check performance"
- "Run perf tests"
- "Test performance"
- "Check for performance regressions"

**What Happens Automatically:**
1. Intelligent script discovery (searches common locations and patterns):
   - `./run-performance-tests.sh`
   - `./scripts/run-performance-tests.sh`
   - `./test/performance/run-tests.sh`
   - `*performance*.sh`, `*perf*.sh` (project-wide search)
2. Executes script with real-time output monitoring
3. Analyzes results for regressions (keyword and metric-based)
4. Compares with baseline if available (percentage-based thresholds)
5. Generates comprehensive report highlighting regressions, improvements, and stable metrics
6. Saves timestamped results to `.perf-results/` for history tracking

**Components:**
- **Sub-agent**: `~/.claude/agents/perf-test-runner.md` (executes workflow)
- **Skill**: `~/.claude/skills/performance-testing/SKILL.md` (auto-activation)

**Key Features:**
- Smart script discovery with fallbacks
- Real-time test execution monitoring
- Dual regression detection (keywords + metric comparison)
- Baseline management (create, update, compare)
- Comprehensive reporting with actionable recommendations
- Result persistence and trend analysis

**Report Example:**
```
📊 Performance Test Results

Status: ⚠️ PASSED (with 2 regressions)

Regressions:
⚠️ API response time: 250ms → 320ms (+28%)
⚠️ Memory usage: 512MB → 580MB (+13%)

Improvements:
✅ Database queries: 45ms → 38ms (-15%)

Results: .perf-results/2025-10-23-145230.json
```

**Documentation:** See `~/.claude/skills/performance-testing/README.md` for complete workflow details.

#### Automated Testing Workflow (NEW)

**Purpose:** Execute the complete clean-build-test workflow for .NET solutions with automatic performance test exclusion.

**Activation:** When the user asks to "run tests", "test", or "run integration tests", the `run-tests` skill automatically activates and delegates to the `test-runner` sub-agent.

**Trigger Phrases:**
- "Run tests"
- "Test"
- "Run integration tests"
- "Clean, build and test"
- "Execute tests"

**What Happens Automatically:**
1. Discovers .NET solution file in current or parent directory
2. Cleans solution (both Debug and Release configurations)
3. Builds solution (Release configuration)
4. Discovers all test projects in solution
5. Automatically excludes performance test projects (containing: Performance, Perf, Benchmark, LoadTest, StressTest)
6. Runs all test projects (continues even if some fail to get complete picture)
7. Generates comprehensive report with pass/fail details and exact failure locations

**Components:**
- **Sub-agent**: `~/.claude/agents/test-runner.md` (executes workflow)
- **Skill**: `~/.claude/skills/run-tests/SKILL.md` (auto-activation)

**Key Features:**
- Complete clean-build-test workflow in one command
- Automatic performance test exclusion
- Continues running all tests even if some fail
- Detailed failure reporting with file locations and line numbers
- Summary statistics (total, passed, failed, percentage)
- Integration tests included by default

**Report Example:**
```
🧪 Test Execution Summary

Status: ❌ FAILED

✅ Infrastructure.Tests (247 passed)
✅ Domain.Tests (89 passed)
❌ Application.Tests (2 failed, 154 passed)

Failed Tests:
❌ UserService_Should_Validate_Email
   Error: Expected true, got false
   Location: UserServiceTests.cs:45

Overall: 602/604 passed (99.7%)
Action: Fix 2 failing tests before pushing
```

**Exclusions:** Performance test projects are automatically excluded. To run performance tests, use: "Run performance tests" (triggers the `performance-testing` skill instead).

**Documentation:** See `~/.claude/skills/run-tests/README.md` for complete workflow details.

### Adding New Global Tools

To add new scripts to this repository:
1. Place the script in `bin/` directory
2. Make it executable: `chmod +x bin/script-name.sh`
3. Document it in this CLAUDE.md file
4. Update the repository README.md
5. Commit and push to GitHub

## Installation on New Machine

See the README.md in this repository for complete setup instructions including:
- Cloning the repository
- Adding to PATH
- Creating the global CLAUDE.md symlink
