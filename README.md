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
```

Done! All scripts are now available, and Claude Code will know about them across all projects.

## Contributing

All CodexForge team members can contribute scripts that improve developer workflow and productivity.
