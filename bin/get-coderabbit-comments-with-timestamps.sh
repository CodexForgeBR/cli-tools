#!/bin/bash

# Script to fetch CodeRabbit comments from a GitHub PR with optional timestamp filtering
# Usage: ./get-coderabbit-comments-with-timestamps.sh <PR_NUMBER> [--since TIMESTAMP]
#
# Examples:
#   ./get-coderabbit-comments-with-timestamps.sh 123
#   ./get-coderabbit-comments-with-timestamps.sh 123 --since "2025-10-23T10:30:00Z"
#   ./get-coderabbit-comments-with-timestamps.sh 123 --since "2025-10-23 10:30:00"

set -e

if [ -z "$1" ]; then
    echo "Error: PR number required"
    echo "Usage: $0 <PR_NUMBER> [--since TIMESTAMP]"
    exit 1
fi

PR_NUMBER=$1
SINCE_TIMESTAMP=""

# Parse optional --since parameter
shift
while [[ $# -gt 0 ]]; do
    case $1 in
        --since)
            SINCE_TIMESTAMP="$2"
            shift 2
            ;;
        *)
            echo "Error: Unknown parameter: $1"
            echo "Usage: $0 <PR_NUMBER> [--since TIMESTAMP]"
            exit 1
            ;;
    esac
done

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
if [ -n "$SINCE_TIMESTAMP" ]; then
    echo "Filtering comments since: ${SINCE_TIMESTAMP}"
fi
echo ""

# Build jq filter based on whether we have a timestamp filter
if [ -n "$SINCE_TIMESTAMP" ]; then
    # Convert SINCE_TIMESTAMP to ISO 8601 format if it isn't already
    # This handles formats like "2025-10-23 10:30:00" and converts to "2025-10-23T10:30:00Z"
    SINCE_ISO=$(date -u -j -f "%Y-%m-%d %H:%M:%S" "$SINCE_TIMESTAMP" "+%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "$SINCE_TIMESTAMP")

    JQ_FILTER=".[] | select(.user.login == \"coderabbitai[bot]\") | select(.created_at >= \"${SINCE_ISO}\") | \"FILE: \(.path)\nLINE: \(.line)\nCREATED: \(.created_at)\n\(.body)\n\" + (\"=\" * 80)"
else
    JQ_FILTER='.[] | select(.user.login == "coderabbitai[bot]") | "FILE: \(.path)\nLINE: \(.line)\nCREATED: \(.created_at)\n\(.body)\n" + ("=" * 80)'
fi

# Use --paginate to fetch ALL comments across all pages
gh api --paginate "repos/${REPO}/pulls/${PR_NUMBER}/comments" --jq "$JQ_FILTER"
