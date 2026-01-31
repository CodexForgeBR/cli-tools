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

### rm (Wrapper Script)

**Purpose:** Backs up files to `~/.rm-backup/` instead of deleting when called from Claude/Codex/Node.

**⚠️ CRITICAL**: Prevents accidental deletion of important files by automated CLI tools.

**Whitelisted Patterns (Actually Deleted):**
- `/tmp/*`, `*.tmp`, `.*` (dotfiles), `bin/`, `obj/`, `node_modules/`, `target/`, `dist/`, `build/`

**Behavior:**
- Normal terminal: works as expected
- Claude Code: non-whitelisted files → `~/.rm-backup/`, whitelisted → deleted

**Restore**: `cp ~/.rm-backup/path/to/file.timestamp /original/path/file`

**Bypass**: `/bin/rm myfile.txt`

### git (Wrapper Script)

**Purpose:** Blocks destructive git commands when called from Claude/Codex/Gemini. **Fails fast** (exit 1) unlike rm wrapper.

**⚠️ CRITICAL**: Prevents bypassing pre-commit hooks, force pushing, and losing uncommitted work.

**Blocked Commands:**
- `git commit --no-verify` / `-n` - Bypasses Husky.Net validation
- `git reset --hard` - Destroys uncommitted changes
- `git clean -f/-d/-x` - Deletes untracked files
- `git push --force` / `-f` / `--force-with-lease` - Rewrites history

**Behavior:**
- Normal terminal: all commands work
- Claude Code: blocked commands → error, safe commands → work normally

**Debug log**: `~/.git-wrapper-debug.log`

**Bypass**: `/opt/homebrew/bin/git commit --no-verify` or run from regular terminal.

## Available Workflow Skills

Specialized workflow automation skills are available in `~/.claude/skills/`:
- `run-tests` - Clean, build, and test workflow
- `performance-testing` - Performance regression detection
- `pre-pr-review` - Local CodeRabbit scans before creating PRs
- `coderabbit-workflow` - Auto-fix PR review feedback
- `post-merge-cleanup` - Safe branch deletion

Skills auto-activate based on your requests. See individual skill files for detailed documentation.

## Global Development Guidelines

### Working with CodeRabbit

When the user asks to analyze, read, or fetch CodeRabbit comments from PRs:
1. **Always use** `get-coderabbit-comments.sh <PR_NUMBER>`
2. Do NOT manually construct `gh api` commands
3. The script handles all the GitHub API complexity
4. Output is formatted for easy reading and analysis

**Installation (if needed):**
```bash
# Install CodeRabbit CLI
curl -fsSL https://cli.coderabbit.ai/install.sh | sh

# Authenticate
coderabbit auth login
```

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

## Active Technologies
- Go 1.22+ + `github.com/spf13/cobra` (CLI framework), `github.com/fatih/color` (ANSI terminal colors), `github.com/stretchr/testify` (test assertions). All else is stdlib. (001-ralph-loop-go-cli)
- Filesystem — JSON state files in `.ralph-loop/`, config files in `~/.config/ralph-loop/` and `.ralph-loop/config`, markdown learnings file. (001-ralph-loop-go-cli)

## Recent Changes
- 001-ralph-loop-go-cli: Added Go 1.22+ + `github.com/spf13/cobra` (CLI framework), `github.com/fatih/color` (ANSI terminal colors), `github.com/stretchr/testify` (test assertions). All else is stdlib.
