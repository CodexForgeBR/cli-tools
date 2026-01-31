#!/usr/bin/env python3
"""
Rate limit detection and reset time parser.

Reads a file, searches for rate limit patterns, parses the reset time and timezone,
and outputs the epoch timestamp.

Exit codes:
    0 - Rate limit found and parsed successfully
    1 - No rate limit detected
    2 - Rate limit detected but time unparseable (fallback needed)
"""

import re
import sys
from datetime import datetime, timedelta
from zoneinfo import ZoneInfo

# Buffer to add to reset time to avoid retrying too early
RATE_LIMIT_BUFFER_SECONDS = 60

# Maximum content size for bare pattern matching to avoid false positives
# from AI discussing rate limits in its analysis text
BARE_PATTERN_MAX_CONTENT_SIZE = 500


def find_rate_limit_pattern(content):
    """
    Search for rate limit patterns in content.

    Returns:
        tuple: (reset_time_str, timezone_str) or (None, None) if no parseable pattern found
        bool: True if rate limit detected (even if unparseable)
    """
    # Pattern 1: "resets 6pm (America/Bahia)" or "reset 6pm (America/Bahia)"
    pattern1 = re.compile(
        r'resets?\s+(\d{1,2}\s*(?:am|pm))\s*\(([^)]+)\)',
        re.IGNORECASE
    )

    # Pattern 2: "resets 6:30pm (America/Sao_Paulo)"
    pattern2 = re.compile(
        r'resets?\s+(\d{1,2}:\d{2}\s*(?:am|pm))\s*\(([^)]+)\)',
        re.IGNORECASE
    )

    # Pattern 3: "resets 18:00 (UTC)"
    pattern3 = re.compile(
        r'resets?\s+(\d{1,2}:\d{2})\s*\(([^)]+)\)',
        re.IGNORECASE
    )

    # Pattern 4: "resets Jan 1, 2026, 9am (UTC)" or "resets January 15, 2026, 3:30pm (America/Bahia)"
    pattern4 = re.compile(
        r'resets?\s+[A-Za-z]+\s+\d{1,2},?\s+\d{4},?\s+(\d{1,2}(?::\d{2})?\s*(?:am|pm))\s*\(([^)]+)\)',
        re.IGNORECASE
    )

    # Bare detection patterns (no parseable time)
    # More specific patterns to avoid false positives
    bare_patterns = [
        r"you'?ve hit your limit",
        r'rate limit exceeded',
        r'rate limited',
        r'too many requests'
    ]

    # Try parseable patterns first
    for pattern in [pattern2, pattern1, pattern3, pattern4]:
        match = pattern.search(content)
        if match:
            return (match.group(1).strip(), match.group(2).strip()), True

    # Only check bare patterns for short content to avoid false positives
    # from AI discussing rate limits in its analysis text.
    if len(content) <= BARE_PATTERN_MAX_CONTENT_SIZE:
        for bare_pattern in bare_patterns:
            if re.search(bare_pattern, content, re.IGNORECASE):
                return (None, None), True

    return (None, None), False


def parse_time_with_timezone(time_str, tz_str):
    """
    Parse time string in given timezone and convert to epoch.

    Args:
        time_str: Time like "6pm", "6:30pm", or "18:00"
        tz_str: IANA timezone like "America/Bahia"

    Returns:
        tuple: (epoch_timestamp, human_readable_time, timezone_str)
    """
    try:
        tz = ZoneInfo(tz_str)
    except Exception as e:
        print(f"Error: Invalid timezone '{tz_str}': {e}", file=sys.stderr)
        return None

    # Get current time in the specified timezone
    now = datetime.now(tz)

    # Parse the time string
    time_str_lower = time_str.lower().strip()

    # Handle 24-hour format (e.g., "18:00")
    if 'am' not in time_str_lower and 'pm' not in time_str_lower:
        if ':' in time_str_lower:
            try:
                hour, minute = time_str_lower.split(':')
                hour = int(hour)
                minute = int(minute)
            except ValueError:
                return None
        else:
            return None
    else:
        # Handle 12-hour format with am/pm
        # Remove spaces between time and am/pm
        time_str_lower = re.sub(r'\s+', '', time_str_lower)

        try:
            if ':' in time_str_lower:
                # Format: "6:30pm"
                time_part = time_str_lower.rstrip('apm')
                hour, minute = time_part.split(':')
                hour = int(hour)
                minute = int(minute)
            else:
                # Format: "6pm"
                hour = int(time_str_lower.rstrip('apm'))
                minute = 0

            # Convert to 24-hour format
            if 'pm' in time_str_lower and hour != 12:
                hour += 12
            elif 'am' in time_str_lower and hour == 12:
                hour = 0
        except ValueError:
            return None

    # Create reset datetime for today
    reset_time = now.replace(hour=hour, minute=minute, second=0, microsecond=0)

    # If the time is in the past, assume it's tomorrow
    if reset_time <= now:
        reset_time += timedelta(days=1)

    # Add buffer
    reset_time += timedelta(seconds=RATE_LIMIT_BUFFER_SECONDS)

    # Convert to epoch
    epoch = int(reset_time.timestamp())

    # Format human-readable time
    human = reset_time.strftime('%Y-%m-%d %H:%M:%S %Z')

    return epoch, human, tz_str


def main():
    if len(sys.argv) != 2:
        print("Usage: rate_limit_parser.py <file_path>", file=sys.stderr)
        sys.exit(1)

    file_path = sys.argv[1]

    try:
        with open(file_path, 'r', encoding='utf-8', errors='replace') as f:
            content = f.read()
    except Exception as e:
        print(f"Error reading file: {e}", file=sys.stderr)
        sys.exit(1)

    (time_str, tz_str), detected = find_rate_limit_pattern(content)

    if not detected:
        # No rate limit detected
        sys.exit(1)

    if time_str is None or tz_str is None:
        # Rate limit detected but time is unparseable
        print("Rate limit detected but reset time could not be parsed", file=sys.stderr)
        sys.exit(2)

    result = parse_time_with_timezone(time_str, tz_str)
    if result is None:
        # Rate limit detected but time is unparseable
        print(f"Rate limit detected but could not parse time '{time_str}' in timezone '{tz_str}'", file=sys.stderr)
        sys.exit(2)

    epoch, human, tz = result

    # Output the results
    print(epoch)
    print(human)
    print(tz)

    sys.exit(0)


if __name__ == '__main__':
    main()
