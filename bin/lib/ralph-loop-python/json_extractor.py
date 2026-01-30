#!/usr/bin/env python3
"""
JSON extractor for ralph-loop.

Extracts JSON objects containing specific keys from files with arbitrary nesting depth.
Supports both markdown code blocks and inline JSON.

Usage: json_extractor.py <file_path> <json_type>
    file_path: Path to file containing JSON
    json_type: Key to search for (e.g., RALPH_STATUS, RALPH_VALIDATION)

Outputs: Pretty-printed JSON to stdout if found
Exit codes: 0 on success, 1 on failure
"""
import sys
import re
import json


def find_json_containing(content, json_type):
    """Find JSON object containing the specified key using bracket matching"""
    search_key = f'"{json_type}"'

    # Method 1: Try markdown code blocks first
    code_block_pattern = r'```json\s*(.*?)```'
    for match in re.finditer(code_block_pattern, content, re.DOTALL):
        block = match.group(1).strip()
        if json_type in block:
            try:
                parsed = json.loads(block)
                if json_type in parsed:
                    return block
            except:
                pass

    # Method 2: Bracket-matching for arbitrary nesting depth
    key_pos = content.find(search_key)
    if key_pos == -1:
        return None

    # Find the opening brace before the key
    start = key_pos
    while start > 0 and content[start] != '{':
        start -= 1

    if start < 0 or content[start] != '{':
        return None

    # Match brackets with proper depth tracking
    depth = 0
    in_string = False
    escape_next = False
    end = start

    for i, char in enumerate(content[start:], start):
        if escape_next:
            escape_next = False
            continue
        if char == '\\' and in_string:
            escape_next = True
            continue
        if char == '"' and not escape_next:
            in_string = not in_string
            continue
        if in_string:
            continue
        if char == '{':
            depth += 1
        elif char == '}':
            depth -= 1
            if depth == 0:
                end = i + 1
                break

    if depth != 0:
        return None

    candidate = content[start:end]
    try:
        parsed = json.loads(candidate)
        if json_type in parsed:
            return candidate
    except:
        pass

    return None


def main():
    if len(sys.argv) != 3:
        print("Usage: json_extractor.py <file_path> <json_type>", file=sys.stderr)
        sys.exit(1)

    file_path = sys.argv[1]
    json_type = sys.argv[2]

    try:
        with open(file_path, 'r') as f:
            content = f.read()
    except Exception as e:
        print(f"Error reading file: {e}", file=sys.stderr)
        sys.exit(1)

    # Try to find and parse JSON containing the specified key
    result = find_json_containing(content, json_type)
    if result:
        try:
            parsed = json.loads(result)
            print(json.dumps(parsed))
            sys.exit(0)
        except Exception as e:
            print(f"Error parsing JSON: {e}", file=sys.stderr)

    # Nothing found
    sys.exit(1)


if __name__ == '__main__':
    main()
