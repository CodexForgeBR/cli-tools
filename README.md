# CodexForge CLI Tools

Shared developer tools and scripts for CodexForge team.

## Installation

```bash
# Clone the repo
git clone https://github.com/CodexForgeBR/cli-tools.git ~/source/cli-tools

# Add to PATH in ~/.zshrc
echo 'export PATH="$HOME/source/cli-tools/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

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

Simply clone the repository and add to PATH:

```bash
git clone https://github.com/CodexForgeBR/cli-tools.git ~/source/cli-tools
echo 'export PATH="$HOME/source/cli-tools/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

## Contributing

All CodexForge team members can contribute scripts that improve developer workflow and productivity.
