#!/usr/bin/env bash
#
# install-claude.sh - Install Claude Code skills and sub-agents via symlinks
#
# This script creates symlinks from ~/.claude/ to the skills and agents
# in this repository. Updates to the repo automatically propagate to Claude.
#
# Usage:
#   ./install-claude.sh           # Install all skills and agents
#   ./install-claude.sh --dry-run # Show what would be done without doing it
#   ./install-claude.sh --force   # Overwrite existing files (creates backups)
#   ./install-claude.sh --remove  # Remove symlinks (uninstall)
#

set -euo pipefail

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLAUDE_DIR="$HOME/.claude"
BACKUP_DIR="$HOME/.claude-backup-$(date +%Y%m%d_%H%M%S)"

# Skills and agents to install
SKILLS=(
    "run-tests"
    "coderabbit-workflow"
    "pre-pr-review"
    "post-merge-cleanup"
    "qodana-local-review"
    "calculate-coverage"
    "performance-testing"
)

AGENTS=(
    "test-runner.md"
    "coderabbit-fixer.md"
    "local-coderabbit-reviewer.md"
    "branch-cleanup.md"
    "local-qodana-reviewer.md"
    "perf-test-runner.md"
)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse arguments
DRY_RUN=false
FORCE=false
REMOVE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --force)
            FORCE=true
            shift
            ;;
        --remove)
            REMOVE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Install Claude Code skills and sub-agents via symlinks."
            echo ""
            echo "Options:"
            echo "  --dry-run    Show what would be done without doing it"
            echo "  --force      Overwrite existing files (creates backups)"
            echo "  --remove     Remove symlinks (uninstall)"
            echo "  -h, --help   Show this help message"
            echo ""
            echo "Skills installed:"
            for skill in "${SKILLS[@]}"; do
                echo "  - $skill"
            done
            echo ""
            echo "Agents installed:"
            for agent in "${AGENTS[@]}"; do
                echo "  - ${agent%.md}"
            done
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Helper function to create a symlink
create_symlink() {
    local source="$1"
    local target="$2"
    local name="$3"
    local type="$4"  # "skill" or "agent"

    # Check if source exists
    if [[ ! -e "$source" ]]; then
        echo -e "${RED}  [ERROR]${NC} Source not found: $source"
        return 1
    fi

    # Check if target already exists
    if [[ -L "$target" ]]; then
        # It's a symlink - check if it points to the right place
        local current_target
        current_target=$(readlink "$target")
        if [[ "$current_target" == "$source" ]]; then
            echo -e "${BLUE}  [SKIP]${NC} $name - already correctly linked"
            return 0
        else
            # Symlink points elsewhere
            if [[ "$FORCE" == true ]]; then
                if [[ "$DRY_RUN" == true ]]; then
                    echo -e "${YELLOW}  [WOULD REPLACE]${NC} $name - symlink points to $current_target"
                else
                    rm "$target"
                    ln -s "$source" "$target"
                    echo -e "${GREEN}  [REPLACED]${NC} $name - was pointing to $current_target"
                fi
            else
                echo -e "${YELLOW}  [CONFLICT]${NC} $name - symlink exists pointing to $current_target"
                echo "            Use --force to replace"
                return 1
            fi
        fi
    elif [[ -e "$target" ]]; then
        # It's a regular file or directory
        if [[ "$FORCE" == true ]]; then
            if [[ "$DRY_RUN" == true ]]; then
                echo -e "${YELLOW}  [WOULD BACKUP]${NC} $name - existing $type will be moved to $BACKUP_DIR"
            else
                # Create backup directory if needed
                mkdir -p "$BACKUP_DIR/${type}s"
                mv "$target" "$BACKUP_DIR/${type}s/"
                ln -s "$source" "$target"
                echo -e "${GREEN}  [INSTALLED]${NC} $name - existing backed up to $BACKUP_DIR/${type}s/"
            fi
        else
            echo -e "${YELLOW}  [CONFLICT]${NC} $name - file/directory already exists"
            echo "            Use --force to backup and replace"
            return 1
        fi
    else
        # Target doesn't exist - create symlink
        if [[ "$DRY_RUN" == true ]]; then
            echo -e "${GREEN}  [WOULD INSTALL]${NC} $name"
        else
            ln -s "$source" "$target"
            echo -e "${GREEN}  [INSTALLED]${NC} $name"
        fi
    fi

    return 0
}

# Helper function to remove a symlink
remove_symlink() {
    local target="$1"
    local name="$2"

    if [[ -L "$target" ]]; then
        if [[ "$DRY_RUN" == true ]]; then
            echo -e "${YELLOW}  [WOULD REMOVE]${NC} $name"
        else
            rm "$target"
            echo -e "${GREEN}  [REMOVED]${NC} $name"
        fi
    elif [[ -e "$target" ]]; then
        echo -e "${BLUE}  [SKIP]${NC} $name - not a symlink (won't remove)"
    else
        echo -e "${BLUE}  [SKIP]${NC} $name - doesn't exist"
    fi
}

# Main installation/removal logic
main() {
    local errors=0

    if [[ "$REMOVE" == true ]]; then
        echo ""
        echo -e "${BLUE}Removing Claude Code skills and agents...${NC}"
        if [[ "$DRY_RUN" == true ]]; then
            echo -e "${YELLOW}(Dry run - no changes will be made)${NC}"
        fi
        echo ""

        echo "Skills:"
        for skill in "${SKILLS[@]}"; do
            remove_symlink "$CLAUDE_DIR/skills/$skill" "$skill"
        done

        echo ""
        echo "Agents:"
        for agent in "${AGENTS[@]}"; do
            remove_symlink "$CLAUDE_DIR/agents/$agent" "${agent%.md}"
        done
    else
        echo ""
        echo -e "${BLUE}Installing Claude Code skills and agents...${NC}"
        if [[ "$DRY_RUN" == true ]]; then
            echo -e "${YELLOW}(Dry run - no changes will be made)${NC}"
        fi
        echo ""

        # Create directories if they don't exist
        if [[ "$DRY_RUN" == false ]]; then
            mkdir -p "$CLAUDE_DIR/skills"
            mkdir -p "$CLAUDE_DIR/agents"
        fi

        echo "Skills:"
        for skill in "${SKILLS[@]}"; do
            if ! create_symlink \
                "$SCRIPT_DIR/claude/skills/$skill" \
                "$CLAUDE_DIR/skills/$skill" \
                "$skill" \
                "skill"; then
                ((errors++))
            fi
        done

        echo ""
        echo "Agents:"
        for agent in "${AGENTS[@]}"; do
            if ! create_symlink \
                "$SCRIPT_DIR/claude/agents/$agent" \
                "$CLAUDE_DIR/agents/$agent" \
                "${agent%.md}" \
                "agent"; then
                ((errors++))
            fi
        done
    fi

    echo ""
    if [[ "$errors" -gt 0 ]]; then
        echo -e "${YELLOW}Completed with $errors conflict(s). Use --force to resolve.${NC}"
        exit 1
    else
        if [[ "$DRY_RUN" == true ]]; then
            echo -e "${GREEN}Dry run complete. Run without --dry-run to apply changes.${NC}"
        elif [[ "$REMOVE" == true ]]; then
            echo -e "${GREEN}Uninstall complete!${NC}"
        else
            echo -e "${GREEN}Installation complete!${NC}"
            echo ""
            echo "Skills and agents are now available in Claude Code."
            echo "Run 'claude' and try commands like 'run tests' or '/pre-pr-review'."
        fi
    fi
}

main
