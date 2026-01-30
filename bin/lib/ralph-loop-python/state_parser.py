#!/usr/bin/env python3
"""
State parser for ralph-loop.

Parses ralph-loop state JSON files and outputs shell variable assignments
or displays formatted status information.

Usage:
    state_parser.py load <state_file>     - Output shell variables for eval
    state_parser.py status <state_file>   - Display formatted status
    state_parser.py check <state_file>    - Get stored status value

Exit codes: 0 on success, 1 on failure
"""
import sys
import json
import base64
from datetime import datetime


def load_state(state_file):
    """Load state and output shell variable assignments"""
    try:
        with open(state_file, 'r') as f:
            state = json.load(f)

        # Export variables safely
        print(f"SCRIPT_START_TIME='{state.get('started_at', '')}'")
        print(f"ITERATION={state.get('iteration', 0)}")
        print(f"CURRENT_PHASE='{state.get('phase', '')}'")

        # Encode feedback as base64 to avoid quote escaping issues
        feedback = state.get('last_feedback', '')
        feedback_b64 = base64.b64encode(feedback.encode('utf-8')).decode('ascii')
        print(f"LAST_FEEDBACK_B64='{feedback_b64}'")

        print(f"SESSION_ID='{state.get('session_id', '')}'")
        print(f"STORED_AI_CLI='{state.get('ai_cli', '')}'")

        # Store tasks file hash for validation
        print(f"STORED_TASKS_HASH='{state.get('tasks_file_hash', '')}'")
        print(f"STORED_TASKS_FILE='{state.get('tasks_file', '')}'")
        print(f"STORED_IMPL_MODEL='{state.get('implementation_model', '')}'")
        print(f"STORED_VAL_MODEL='{state.get('validation_model', '')}'")

        # Restore plan validation settings
        print(f"STORED_ORIGINAL_PLAN_FILE='{state.get('original_plan_file', '')}'")
        print(f"STORED_GITHUB_ISSUE='{state.get('github_issue', '')}'")
        print(f"STORED_MAX_ITERATIONS={state.get('max_iterations', 20)}")
        print(f"STORED_MAX_INADMISSIBLE={state.get('max_inadmissible', 5)}")

        # Restore learnings settings (defaults for backward compatibility)
        learnings = state.get('learnings', {})
        print(f"STORED_LEARNINGS_ENABLED={learnings.get('enabled', 1)}")
        print(f"STORED_LEARNINGS_FILE='{learnings.get('file', '')}'")

        # Retry state for resume (defaults for backward compatibility)
        retry_state = state.get('retry_state', {})
        print(f"CURRENT_RETRY_ATTEMPT={retry_state.get('attempt', 1)}")
        print(f"CURRENT_RETRY_DELAY={retry_state.get('delay', 5)}")

        # Inadmissible count (defaults to 0 for backward compatibility)
        print(f"INADMISSIBLE_COUNT={state.get('inadmissible_count', 0)}")

        # Schedule data (defaults for backward compatibility)
        schedule = state.get('schedule', {})
        print(f"STORED_SCHEDULE_ENABLED={1 if schedule.get('enabled', False) else 0}")
        print(f"STORED_SCHEDULE_TARGET_EPOCH={schedule.get('target_epoch', 0)}")
        print(f"STORED_SCHEDULE_TARGET_HUMAN='{schedule.get('target_human', '')}'")

        sys.exit(0)
    except Exception as e:
        print(f"# Error loading state: {e}", file=sys.stderr)
        sys.exit(1)


def show_status(state_file):
    """Display formatted status information"""
    try:
        with open(state_file, 'r') as f:
            state = json.load(f)

        print(f"Session ID:           {state.get('session_id', 'N/A')}")
        print(f"Status:               {state.get('status', 'UNKNOWN')}")
        print(f"Iteration:            {state.get('iteration', 0)}")
        print(f"Phase:                {state.get('phase', 'N/A')}")
        print(f"Started:              {state.get('started_at', 'N/A')}")
        print(f"Last Updated:         {state.get('last_updated', 'N/A')}")
        print(f"Tasks File:           {state.get('tasks_file', 'N/A')}")
        print(f"AI CLI:               {state.get('ai_cli', 'N/A')}")
        print(f"Implementation Model: {state.get('implementation_model', 'N/A')}")
        print(f"Validation Model:     {state.get('validation_model', 'N/A')}")
        print(f"Max Iterations:       {state.get('max_iterations', 'N/A')}")

        circuit = state.get('circuit_breaker', {})
        if circuit:
            print(f"\nCircuit Breaker:")
            print(f"  No Progress Count:  {circuit.get('no_progress_count', 0)}")
            print(f"  Last Unchecked:     {circuit.get('last_unchecked_count', 0)}")

        retry = state.get('retry_state', {})
        if retry and retry.get('attempt', 1) > 1:
            print(f"\nRetry State (mid-retry when interrupted):")
            print(f"  Next Attempt:       {retry.get('attempt', 1)}")
            print(f"  Next Delay:         {retry.get('delay', 5)}s")

        feedback = state.get('last_feedback', '')
        if feedback:
            print(f"\nLast Feedback:")
            print(f"  {feedback[:100]}{'...' if len(feedback) > 100 else ''}")

    except Exception as e:
        print(f"Error reading state: {e}")
        sys.exit(1)


def check_status(state_file):
    """Get stored status value"""
    try:
        with open(state_file, 'r') as f:
            state = json.load(f)
        print(state.get('status', 'UNKNOWN'))
    except:
        print('ERROR')


def main():
    if len(sys.argv) < 3:
        print("Usage: state_parser.py {load|status|check} <state_file>", file=sys.stderr)
        sys.exit(1)

    command = sys.argv[1]
    state_file = sys.argv[2]

    if command == 'load':
        load_state(state_file)
    elif command == 'status':
        show_status(state_file)
    elif command == 'check':
        check_status(state_file)
    else:
        print(f"Unknown command: {command}", file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()
