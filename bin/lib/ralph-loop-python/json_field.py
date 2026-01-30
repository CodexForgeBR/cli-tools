#!/usr/bin/env python3
"""
JSON field extractor for ralph-loop.

Generic utility for extracting specific fields from JSON data.
Consolidates all inline python3 -c JSON extraction calls.

Usage:
    json_field.py verdict           - Extract RALPH_VALIDATION.verdict
    json_field.py feedback          - Extract RALPH_VALIDATION.feedback
    json_field.py remaining         - Extract RALPH_VALIDATION.tasks_analysis.remaining_unchecked
    json_field.py blocked_count     - Extract RALPH_VALIDATION.tasks_analysis.confirmed_blocked
    json_field.py blocked_tasks     - Extract and format RALPH_VALIDATION.blocked_tasks list
    json_field.py escape            - Escape string for JSON (reads from stdin)

Reads JSON from stdin, writes result to stdout.
Exit codes: 0 on success, 1 on failure
"""
import sys
import json


def extract_verdict(data):
    """Extract validation verdict"""
    try:
        validation = data.get('RALPH_VALIDATION', {})
        print(validation.get('verdict', 'UNKNOWN'))
    except:
        print('PARSE_ERROR')


def extract_feedback(data):
    """Extract validation feedback"""
    try:
        validation = data.get('RALPH_VALIDATION', {})
        print(validation.get('feedback', 'No feedback provided'))
    except Exception as e:
        print(f'Error parsing feedback: {e}')


def extract_remaining(data):
    """Extract remaining unchecked count"""
    try:
        validation = data.get('RALPH_VALIDATION', {})
        analysis = validation.get('tasks_analysis', {})
        print(analysis.get('remaining_unchecked', -1))
    except:
        print(-1)


def extract_blocked_count(data):
    """Extract confirmed blocked count"""
    try:
        validation = data.get('RALPH_VALIDATION', {})
        analysis = validation.get('tasks_analysis', {})
        print(analysis.get('confirmed_blocked', 0))
    except:
        print(0)


def extract_blocked_tasks(data):
    """Extract and format blocked tasks list"""
    try:
        validation = data.get('RALPH_VALIDATION', {})
        blocked = validation.get('blocked_tasks', [])

        if not blocked:
            print('No blocked tasks reported')
        else:
            for task in blocked:
                task_id = task.get('task_id', 'Unknown')
                desc = task.get('description', '')
                reason = task.get('reason', 'No reason given')
                print(f'  - {task_id}: {desc}')
                print(f'    Reason: {reason}')
    except Exception as e:
        print(f'Error parsing blocked tasks: {e}')


def escape_for_json():
    """Escape stdin content for JSON"""
    try:
        content = sys.stdin.read()
        print(json.dumps(content))
    except Exception as e:
        print(f'Error escaping content: {e}', file=sys.stderr)
        sys.exit(1)


def main():
    if len(sys.argv) != 2:
        print("Usage: json_field.py {verdict|feedback|remaining|blocked_count|blocked_tasks|escape}", file=sys.stderr)
        sys.exit(1)

    field = sys.argv[1]

    # Special case: escape doesn't need JSON input
    if field == 'escape':
        escape_for_json()
        return

    # Read and parse JSON from stdin
    try:
        data = json.load(sys.stdin)
    except json.JSONDecodeError as e:
        print(f'JSON parse error: {e}', file=sys.stderr)
        if field == 'verdict':
            print('PARSE_ERROR')
        elif field == 'feedback':
            print('Could not parse feedback')
        elif field == 'remaining':
            print(-1)
        elif field == 'blocked_count':
            print(0)
        elif field == 'blocked_tasks':
            print('Could not parse blocked tasks')
        sys.exit(1)

    # Extract requested field
    if field == 'verdict':
        extract_verdict(data)
    elif field == 'feedback':
        extract_feedback(data)
    elif field == 'remaining':
        extract_remaining(data)
    elif field == 'blocked_count':
        extract_blocked_count(data)
    elif field == 'blocked_tasks':
        extract_blocked_tasks(data)
    else:
        print(f"Unknown field: {field}", file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()
