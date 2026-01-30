#!/usr/bin/env python3
"""
Learnings extractor for ralph-loop.

Extracts content from RALPH_LEARNINGS sections in implementation output files.
Used to capture insights and patterns discovered during implementation iterations.

Usage: learnings_extractor.py <output_file>

Outputs: Extracted learnings text to stdout (only if non-empty)
Exit codes: 0 always (learnings are optional)
"""
import sys
import re


def extract_learnings(output_file):
    """Extract content between RALPH_LEARNINGS markers"""
    try:
        with open(output_file, 'r') as f:
            content = f.read()

        # Look for RALPH_LEARNINGS block
        pattern = r'RALPH_LEARNINGS:\s*(.*?)(?:\n```|$)'
        match = re.search(pattern, content, re.DOTALL)

        if match:
            learnings = match.group(1).strip()
            # Only output if there's actual content
            if learnings and learnings != '-':
                print(learnings)

    except Exception:
        pass  # Silently fail - learnings are optional


def main():
    if len(sys.argv) != 2:
        print("Usage: learnings_extractor.py <output_file>", file=sys.stderr)
        sys.exit(1)

    output_file = sys.argv[1]
    extract_learnings(output_file)


if __name__ == '__main__':
    main()
