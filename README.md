# CodexForge CLI Tools

Shared developer tools and scripts for CodexForge team.

## Installation

```bash
# Clone the repo
git clone https://github.com/CodexForgeBR/cli-tools.git ~/source/cli-tools

# Add to PATH in ~/.zshrc
echo 'export PATH="$HOME/source/cli-tools/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

# Set up global Claude Code configuration (optional but recommended)
mkdir -p ~/.claude
ln -s ~/source/cli-tools/CLAUDE.md ~/.claude/CLAUDE.md
```

**What does the global CLAUDE.md do?**
- Provides instructions to Claude Code across ALL projects
- Automatically tells Claude about available CLI tools
- Makes Claude use these scripts instead of reinventing solutions
- Syncs via Git when you update the repository

## Available Scripts

### `get-coderabbit-comments.sh`

Fetches CodeRabbit review comments from a GitHub Pull Request.

**Usage:**
```bash
get-coderabbit-comments.sh <PR_NUMBER>
```

**Example:**
```bash
get-coderabbit-comments.sh 72
```

**Output:** Displays all inline CodeRabbit comments with file path, line number, and comment body.

### `rm` (Wrapper Script)

**⚠️ IMPORTANT**: This is a protective wrapper around the system `rm` command that backs up files instead of deleting them when called from automated CLI tools (Claude Code, Codex CLI).

**How It Works:**
- Analyzes process tree to detect if called from automated CLI tools
- **Non-whitelisted files**: Moved to `~/.rm-backup/` with full path structure preserved
- **Whitelisted files**: Actually deleted (temp files, build artifacts, hidden files)
- Transparent passthrough for normal terminal usage
- **Claude Code thinks the deletion succeeded** (exit code 0)

**Whitelisted Patterns (Actually Deleted):**
- **Temporary files**: `/tmp/*`, `*.tmp`
- **Hidden files**: `.*` (dotfiles)
- **Build artifacts**: `bin/`, `obj/`, `node_modules/`, `target/`, `dist/`, `build/`

**Examples:**

```bash
# From normal terminal - works as usual (actually deletes)
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
# Original: /Users/you/project/important.json
# Backup:   ~/.rm-backup/Users/you/project/important.json.2025_11_04_17_00_00
```

**Restoring Files:**
```bash
# Check what was backed up
ls -la ~/.rm-backup/

# Restore a file
cp ~/.rm-backup/path/to/file.txt.2025_11_04_17_00_00 /original/path/file.txt
```

**Cleanup Old Backups:**
```bash
# Remove all backups (use with caution)
/bin/rm -rf ~/.rm-backup
```

**Installation Note:**
The wrapper works automatically once `~/source/cli-tools/bin` is added to PATH (it appears before `/bin` in the PATH, creating an override).

### `git` (Wrapper Script)

**⚠️ IMPORTANT**: This is a protective wrapper around git that blocks destructive commands when called from automated CLI tools (Claude Code, Codex CLI, Gemini CLI).

**Key Difference from rm/rmdir Wrappers:**
- **rm/rmdir**: Pretend to succeed (exit 0) + backup files
- **git**: Fail fast (exit 1) + display stern error message

**How It Works:**
- Analyzes process tree to detect if called from automated CLI tools
- Checks git command for destructive patterns
- **Blocks with error** if destructive command detected
- Transparent passthrough for safe git commands and normal terminal usage

**Blocked Commands:**
- `git commit --no-verify` or `-n` - Bypasses pre-commit hooks
- `git reset --hard` - Destroys uncommitted changes
- `git clean -f/-d/-x` - Deletes untracked/ignored files
- `git push --force` or `-f` - Rewrites remote history
- `git push --force-with-lease` - Still rewrites history

**Examples:**

```bash
# From normal terminal - works as usual
git commit --no-verify -m "emergency"  # ✅ Executes

# From Claude Code - blocked with error
git commit --no-verify -m "test"  # ❌ BLOCKED: Bypasses pre-commit hooks
git reset --hard HEAD             # ❌ BLOCKED: Destroys uncommitted changes
git push --force origin main      # ❌ BLOCKED: Rewrites remote history

# From Claude Code - safe commands work normally
git status                        # ✅ Works
git commit -m "normal commit"     # ✅ Works
git push origin feature-branch    # ✅ Works
```

**Error Message Example:**
```
========================================
GIT WRAPPER: DESTRUCTIVE COMMAND BLOCKED
========================================

BLOCKED: git commit --no-verify bypasses pre-commit hooks (Husky.Net validation)

Detected automated CLI tool: claude

To bypass this protection (if absolutely necessary):
  1. Run the command from your regular terminal (not from Claude)
  2. Or use the real git binary: /opt/homebrew/bin/git commit --no-verify

See ~/.git-wrapper-debug.log for details
========================================
```

**Bypass Options:**
```bash
# Use absolute path to real git binary
/opt/homebrew/bin/git commit --no-verify -m "emergency"

# Or run from regular terminal instead of Claude Code
```

**Debug Log:**
Commands are logged to `~/.git-wrapper-debug.log` with process tree analysis and block/allow decisions.

**Installation Note:**
The wrapper works automatically once `~/source/cli-tools/bin` is added to PATH (it appears before `/opt/homebrew/bin`, creating an override).

### `ralph-loop.sh`

**Dual-Model Validation Loop** for Spec-Driven Development based on the Ralph Wiggum technique by Geoffrey Huntley (May 2025).

**What It Does:**
- Implements tasks from a `tasks.md` specification file using one Claude model
- Validates the work using a different Claude model to catch "lies" and incomplete work
- Loops until all tasks are completed or max iterations reached
- Supports intelligent session resumption if interrupted

**Basic Usage:**
```bash
# Auto-detect tasks.md and run with defaults
ralph-loop.sh

# Specify tasks file
ralph-loop.sh --tasks-file specs/feature/tasks.md

# Use different models
ralph-loop.sh --implementation-model opus --validation-model sonnet

# Limit iterations
ralph-loop.sh --max-iterations 10
```

**Session Management:**

When interrupted (Ctrl+C), ralph-loop.sh automatically saves its state. Running the script again will detect the interrupted session:

```bash
# After interrupting with Ctrl+C, run again:
ralph-loop.sh

# Output:
╔═══════════════════════════════════════════════════════════════╗
║              PREVIOUS SESSION DETECTED                        ║
╚═══════════════════════════════════════════════════════════════╝

A previous Ralph Loop session was interrupted.
  Status:    INTERRUPTED
  Iteration: 3
  Phase:     validation

Options:
  ralph-loop.sh --resume        Resume from where you left off
  ralph-loop.sh --clean         Start fresh (discards previous state)
  ralph-loop.sh --status        View detailed session status
```

**Resume Options:**
```bash
# Resume from last saved state
ralph-loop.sh --resume

# Resume even if tasks.md was modified
ralph-loop.sh --resume-force

# Start fresh, discarding previous state
ralph-loop.sh --clean

# View session status without running
ralph-loop.sh --status
```

**All Options:**
```bash
ralph-loop.sh [OPTIONS]

Options:
  -v, --verbose              Pass verbose flag to claude code cli
  --max-iterations N         Maximum loop iterations (default: 20)
  --implementation-model M   Model for implementation (default: opus)
  --validation-model M       Model for validation (default: opus)
  --tasks-file PATH          Path to tasks.md (auto-detects if not specified)
  --resume                   Resume from last interrupted session
  --resume-force             Resume even if tasks.md has changed
  --clean                    Start fresh, delete existing .ralph-loop state
  --status                   Show current session status without running
  -h, --help                 Show this help message
```

**Exit Codes:**
- `0` - All tasks completed successfully
- `1` - Error (no tasks.md, invalid params, etc.)
- `2` - Max iterations reached without completion
- `3` - Escalation requested by validator

**State Management:**

The script saves state to `.ralph-loop/` directory including:
- Current iteration and phase (implementation/validation)
- Circuit breaker counters (no-progress detection)
- Last validation feedback
- Tasks file hash (for detecting modifications)
- Iteration snapshots with output logs

**Phase-Aware Resumption:**

If interrupted during:
- **Implementation phase**: Resumes by re-running implementation with previous feedback
- **Validation phase**: Skips to validation using existing implementation output

**Tasks File Change Detection:**

If you modify `tasks.md` after interrupting a session:
```bash
ralph-loop.sh --resume

# Output:
╔═══════════════════════════════════════════════════════════════╗
║              TASKS FILE MODIFIED                              ║
╚═══════════════════════════════════════════════════════════════╝

The tasks.md file has changed since the session was interrupted.

Options:
  ralph-loop.sh --resume-force   Resume with modified file
  ralph-loop.sh --clean          Start fresh with new file
```

**Examples:**
```bash
# Standard usage with auto-detection
ralph-loop.sh

# Verbose mode
ralph-loop.sh -v

# Custom models and iterations
ralph-loop.sh --implementation-model opus --validation-model haiku --max-iterations 15

# Resume after Ctrl+C
ralph-loop.sh --resume

# Check status without running
ralph-loop.sh --status

# Start fresh after modifying tasks
ralph-loop.sh --clean
```

## Claude Code Skills & Agents

This repository includes custom Claude Code skills and sub-agents that can be shared across machines and team members.

### What's Included

**Skills** (workflow automation):
| Skill | Description |
|-------|-------------|
| `run-tests` | Clean, build, and test workflow for .NET solutions |
| `coderabbit-workflow` | Auto-fix CodeRabbit PR review feedback |
| `pre-pr-review` | Run local CodeRabbit scan before creating PR |
| `post-merge-cleanup` | Safe branch deletion after merge |
| `qodana-local-review` | Local Qodana static code analysis |
| `calculate-coverage` | Code coverage metrics for .NET projects |
| `performance-testing` | Performance regression detection |

**Sub-agents** (specialized task handlers):
| Agent | Description |
|-------|-------------|
| `test-runner` | Executes clean-build-test workflow |
| `coderabbit-fixer` | Addresses CodeRabbit review feedback |
| `local-coderabbit-reviewer` | Runs local CodeRabbit scans |
| `branch-cleanup` | Safely cleans up merged branches |
| `local-qodana-reviewer` | Runs local Qodana analysis |
| `perf-test-runner` | Runs performance tests with regression detection |

### Installing Skills & Agents

```bash
# Install all skills and agents via symlinks
./install-claude.sh

# Preview what would be installed (dry run)
./install-claude.sh --dry-run

# Overwrite existing files (creates backups)
./install-claude.sh --force

# Uninstall (remove symlinks)
./install-claude.sh --remove
```

**How it works:**
- Creates symlinks from `~/.claude/skills/` and `~/.claude/agents/` to this repository
- Updates to the repo automatically propagate to Claude Code
- Existing files are backed up when using `--force`

### Using Skills

Skills auto-activate based on your requests:
- "run tests" → triggers `run-tests` skill
- "address coderabbit feedback" → triggers `coderabbit-workflow` skill
- "clean up this branch" → triggers `post-merge-cleanup` skill

You can also invoke skills directly:
- `/pre-pr-review` - Run local CodeRabbit scan
- `/qodana-local-review` - Run local Qodana scan

## Adding New Scripts

1. Add your script to the `bin/` directory
2. Make it executable: `chmod +x bin/your-script.sh`
3. Update this README with usage instructions
4. Commit and push

## On a New Machine

Simply clone the repository, add to PATH, and set up global config:

```bash
# Clone the repository
git clone https://github.com/CodexForgeBR/cli-tools.git ~/source/cli-tools

# Add to PATH
echo 'export PATH="$HOME/source/cli-tools/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

# Set up global Claude Code configuration
mkdir -p ~/.claude
ln -s ~/source/cli-tools/CLAUDE.md ~/.claude/CLAUDE.md

# Install Claude Code skills and agents
./install-claude.sh
```

Done! All scripts are now available, and Claude Code will know about them across all projects. Skills and agents are ready to use.

## Contributing

All CodexForge team members can contribute scripts that improve developer workflow and productivity.
