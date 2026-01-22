# Repository Guidelines

## Project Structure & Module Organization
This repo is a collection of Bash utilities and Claude Code assets.

- `bin/` holds executable scripts and wrappers (e.g., `ralph-loop.sh`, `git`, `rm`, `rmdir`).
- `claude/skills/` and `claude/agents/` store reusable Claude Code skills and sub-agents.
- `CLAUDE.md` defines global Claude instructions used across projects.
- `install-claude.sh` installs skills/agents into `~/.claude/` via symlinks.
- `README.md` documents setup and script usage.

## Build, Test, and Development Commands
There is no build step; scripts run directly.

- `./install-claude.sh` installs skills/agents.
- `./install-claude.sh --dry-run` previews the install without changes.
- `bin/ralph-loop.sh --help` shows the full Ralph Loop CLI usage.
- `bin/get-coderabbit-comments.sh <PR_NUMBER>` fetches CodeRabbit review comments.
- `chmod +x bin/your-script.sh` when adding a new executable.

## Coding Style & Naming Conventions
- Bash is the primary language; keep scripts POSIX-friendly where possible, but `#!/bin/bash` is the norm.
- Use 4-space indentation inside functions (match existing scripts).
- Prefer uppercase for global constants (`STATE_DIR`, `SCRIPT_DIR`) and lowercase for locals.
- Script names in `bin/` use kebab-case for utilities (`get-coderabbit-comments.sh`) and bare names for wrappers (`git`, `rm`, `rmdir`).
- Include a short usage block and examples near the top of user-facing scripts.

## Testing Guidelines
There is no automated test suite in this repository.

- Smoke-test scripts with help or dry-run modes (e.g., `bin/ralph-loop.sh --help`, `./install-claude.sh --dry-run`).
- If a script accepts destructive actions, verify its safe paths before release.

## Commit & Pull Request Guidelines
- Commit messages follow an imperative style seen in history: `Fix <area>: <what>` or `Add <feature> ...`.
- PRs should describe the script or skill change, list commands run (if any), and call out behavioral changes to wrappers like `git`/`rm`.

## Configuration & Safety Notes
- Ensure `~/source/cli-tools/bin` is ahead of system paths so wrappers are active.
- For Claude Code assets, keep `claude/skills/<name>/SKILL.md` and update `README.md` when adding new tools.
