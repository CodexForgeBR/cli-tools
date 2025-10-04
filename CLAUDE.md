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

## Global Development Guidelines

### Working with CodeRabbit PR Reviews

When the user asks to analyze, read, or fetch CodeRabbit comments:
1. **Always use** `get-coderabbit-comments.sh <PR_NUMBER>`
2. Do NOT manually construct `gh api` commands
3. The script handles all the GitHub API complexity
4. Output is formatted for easy reading and analysis

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
