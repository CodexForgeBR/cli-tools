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
REPO="CodexForgeBR/MDA"

echo "Fetching CodeRabbit comments from PR #${PR_NUMBER}..."
echo ""

gh api "repos/${REPO}/pulls/${PR_NUMBER}/comments" --jq '.[] | select(.user.login == "coderabbitai[bot]") | "FILE: \(.path)\nLINE: \(.line)\n\(.body)\n" + ("=" * 80)'
