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

**Purpose:** Protective wrapper around the system `rm` command that backs up files instead of deleting them when called from automated CLI tools.

**⚠️ CRITICAL PROTECTION MECHANISM:**
This wrapper is designed to prevent Claude Code and other automated CLI tools from accidentally deleting important files by backing them up to `~/.rm-backup/` instead.

**How It Works:**
1. **Process Tree Analysis**: Recursively checks up to 10 parent processes to detect if called from `claude`, `codex`, or `node` processes
2. **Whitelist Validation**: Checks all file arguments against whitelist patterns
3. **Selective Backup**: Moves non-whitelisted files to `~/.rm-backup/` when called from automated CLIs
4. **Transparent Passthrough**: Works normally when called from regular terminal sessions
5. **Silent Success**: Returns exit code 0 so Claude Code thinks deletion succeeded

**Whitelisted Patterns (Actually Deleted):**
- **Temporary files**: `/tmp/*`, `*.tmp`
- **Hidden files**: `.*` (dotfiles, like `.cache`, `.DS_Store`)
- **Build artifacts**: `bin/`, `obj/`, `node_modules/`, `target/`, `dist/`, `build/`

**Usage:**
```bash
# From normal terminal - works as expected (actually deletes)
rm myfile.txt  # ✅ Deleted

# From Claude Code - backs up non-whitelisted files
rm important.json  # ✅ Moved to ~/.rm-backup/path/to/important.json.2025_11_04_17_00_00

# From Claude Code - actually deletes whitelisted patterns
rm /tmp/test.txt   # ✅ Actually deleted (temporary file)
rm .cache          # ✅ Actually deleted (hidden file)
rm -rf bin/        # ✅ Actually deleted (build artifact)
```

**Backup Location:**
Files are moved to `~/.rm-backup/` preserving the original path structure with timestamp:
```bash
# Original: /Users/bccs/source/project/important.json
# Backup:   ~/.rm-backup/Users/bccs/source/project/important.json.2025_11_04_17_00_00
```

**Restoring Files:**
```bash
# Check what was backed up
ls -la ~/.rm-backup/

# Restore a file (copy it back to original location)
cp ~/.rm-backup/path/to/file.txt.2025_11_04_17_00_00 /original/path/file.txt

# Or use mv to restore and remove backup
mv ~/.rm-backup/path/to/file.txt.2025_11_04_17_00_00 /original/path/file.txt
```

**Cleanup Old Backups:**
```bash
# Review backups before deleting
ls -la ~/.rm-backup/

# Remove all backups (use system rm to bypass wrapper)
/bin/rm -rf ~/.rm-backup

# Remove backups older than 7 days
find ~/.rm-backup -type f -mtime +7 -exec /bin/rm {} \;
```

**Bypass (If Absolutely Needed):**
```bash
# Use absolute path to system rm to actually delete
/bin/rm myfile.txt
```

**Installation:**
The wrapper is automatically active once `~/source/cli-tools/bin` is in your PATH (which appears before `/bin/`, creating an override).

**Why This Exists:**
Automated CLI tools like Claude Code can sometimes suggest destructive `rm` commands that might delete important files. This wrapper provides a safety net by backing up files instead of deleting them, while still allowing common development operations like cleaning build artifacts.

### git (Wrapper Script)

**Purpose:** Protective wrapper around git that blocks destructive commands when called from automated CLI tools (claude, codex, gemini).

**⚠️ CRITICAL PROTECTION MECHANISM:**
This wrapper prevents Claude Code and other automated tools from running destructive git commands that could:
- Bypass pre-commit hooks (`--no-verify`)
- Rewrite git history (force push)
- Permanently delete uncommitted work (`reset --hard`, `clean -f`)

**Key Difference from rm/rmdir Wrappers:**
- **rm/rmdir**: Pretend to succeed (exit 0) + backup files
- **git**: Fail fast (exit 1) + display stern error message

**How It Works:**
1. **Process Tree Analysis**: Recursively checks up to 20 parent processes for claude, codex, gemini
2. **Command Pattern Matching**: Analyzes git command and arguments for destructive patterns
3. **Fail Fast**: Returns exit code 1 with clear error message when blocking commands
4. **Transparent Passthrough**: Works normally when called from regular terminal sessions

**Blocked Commands:**
- `git commit --no-verify` or `git commit -n` - Bypasses Husky.Net pre-commit validation
- `git reset --hard` - Destroys uncommitted changes
- `git clean -f/-d/-x` - Deletes untracked/ignored files
- `git push --force` or `git push -f` - Rewrites remote history
- `git push --force-with-lease` - Still rewrites history

**Usage:**
```bash
# From regular terminal - works as expected
git commit --no-verify -m "emergency fix"  # ✅ Executes (though Husky may still block)

# From Claude Code - blocked with error
git commit --no-verify -m "test"  # ❌ BLOCKED: Bypasses pre-commit hooks
git reset --hard HEAD~1           # ❌ BLOCKED: Destroys uncommitted changes
git push --force origin main      # ❌ BLOCKED: Rewrites remote history

# From Claude Code - safe commands work normally
git status                        # ✅ Works
git commit -m "normal commit"     # ✅ Works (runs pre-commit hooks)
git push origin feature-branch    # ✅ Works
```

**Error Message Example:**
```
========================================
GIT WRAPPER: DESTRUCTIVE COMMAND BLOCKED
========================================

BLOCKED: git commit --no-verify bypasses pre-commit hooks (Husky.Net validation)

Detected automated CLI tool: claude

This protection prevents Claude Code and other automated tools
from running destructive git commands that could:
  - Bypass security hooks (pre-commit validation)
  - Rewrite git history (force push)
  - Permanently delete uncommitted work (reset --hard, clean -f)

To bypass this protection (if absolutely necessary):
  1. Run the command from your regular terminal (not from Claude)
  2. Or use the real git binary: /opt/homebrew/bin/git commit --no-verify

See ~/.git-wrapper-debug.log for details
========================================
```

**Debug Logging:**
Commands are logged to `~/.git-wrapper-debug.log` with:
- Process tree analysis results
- Detected CLI tool (claude, codex, gemini)
- Full command being executed
- Block/allow decision reasoning

**Bypass (If Absolutely Needed):**
```bash
# Use absolute path to real git binary
/opt/homebrew/bin/git commit --no-verify -m "emergency"

# Or run from regular terminal instead of Claude Code
```

**Installation:**
The wrapper is automatically active once `~/source/cli-tools/bin` is in your PATH (which appears before `/opt/homebrew/bin`, creating an override).

**Why This Exists:**
Automated CLI tools can sometimes suggest destructive git commands that might:
- Skip important validation (pre-commit hooks)
- Rewrite shared git history (breaking collaborators)
- Permanently lose uncommitted work

This wrapper provides a safety net while still allowing all normal git operations.

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
