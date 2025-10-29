#!/bin/bash

# Script to fetch CodeRabbit comments from a GitHub PR
# Usage: ./get-coderabbit-comments.sh <PR_NUMBER>

set -e

if [ -z "$1" ]; then
    echo "Error: PR number required"
    echo "Usage: $0 <PR_NUMBER>"
    exit 1
fi

PR_NUMBER=$1

# Detect repository from git remote
REPO_URL=$(git remote get-url origin 2>/dev/null)
if [ -z "$REPO_URL" ]; then
    echo "Error: Not in a git repository or no remote 'origin' found"
    exit 1
fi

# Extract owner/repo from various URL formats
# Supports: git@github.com:owner/repo.git, https://github.com/owner/repo.git, etc.
REPO=$(echo "$REPO_URL" | sed -E 's|.*github\.com[:/]([^/]+/[^/]+)(\.git)?$|\1|' | sed 's/\.git$//')

if [ -z "$REPO" ]; then
    echo "Error: Could not extract repository from remote URL: $REPO_URL"
    exit 1
fi

echo "Repository: ${REPO}"
echo "Fetching CodeRabbit comments from PR #${PR_NUMBER}..."
echo ""

# Use --paginate to fetch ALL comments across all pages
gh api --paginate "repos/${REPO}/pulls/${PR_NUMBER}/comments" --jq '.[] | select(.user.login == "coderabbitai[bot]") | "FILE: \(.path)\nLINE: \(.line)\n\(.body)\n" + ("=" * 80)'
