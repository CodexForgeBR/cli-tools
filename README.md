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
