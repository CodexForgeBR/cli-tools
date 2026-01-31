// Package ratelimit provides rate limit detection and reset time parsing.
package ratelimit

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// RateLimitBufferSeconds is the buffer added to reset time to avoid retrying too early
	RateLimitBufferSeconds = 60

	// BarePatternMaxContentSize is the maximum content size for bare pattern matching
	// to avoid false positives from AI discussing rate limits in its analysis text
	BarePatternMaxContentSize = 500
)

// RateLimitInfo contains parsed rate limit information
type RateLimitInfo struct {
	// Detected indicates if a rate limit was found
	Detected bool

	// Parseable indicates if the reset time could be parsed
	Parseable bool

	// ResetEpoch is the Unix timestamp when rate limit resets (with buffer)
	ResetEpoch int64

	// ResetHuman is the human-readable reset time
	ResetHuman string

	// Timezone is the IANA timezone string
	Timezone string
}

var (
	// Pattern 1: "resets 6pm (America/Bahia)" or "reset 6pm (America/Bahia)"
	pattern1 = regexp.MustCompile(`(?i)resets?\s+(\d{1,2}\s*(?:am|pm))\s*\(([^)]+)\)`)

	// Pattern 2: "resets 6:30pm (America/Sao_Paulo)"
	pattern2 = regexp.MustCompile(`(?i)resets?\s+(\d{1,2}:\d{2}\s*(?:am|pm))\s*\(([^)]+)\)`)

	// Pattern 3: "resets 18:00 (UTC)"
	pattern3 = regexp.MustCompile(`(?i)resets?\s+(\d{1,2}:\d{2})\s*\(([^)]+)\)`)

	// Pattern 4: "resets Jan 1, 2026, 9am (UTC)" or "resets January 15, 2026, 3:30pm (America/Bahia)"
	pattern4 = regexp.MustCompile(`(?i)resets?\s+[A-Za-z]+\s+\d{1,2},?\s+\d{4},?\s+(\d{1,2}(?::\d{2})?\s*(?:am|pm))\s*\(([^)]+)\)`)

	// Bare detection patterns (no parseable time)
	barePatterns = []string{
		`you'?ve hit your limit`,
		`rate limit exceeded`,
		`rate limited`,
		`too many requests`,
	}
)

// FindRateLimitPattern searches for rate limit patterns in content.
// Returns (timeStr, tzStr, detected) where detected indicates if any rate limit pattern was found.
// If detected is true but timeStr/tzStr are empty, the rate limit was detected but not parseable.
func FindRateLimitPattern(content string) (timeStr, tzStr string, detected bool) {
	// Try parseable patterns first (pattern2 has highest priority)
	patterns := []*regexp.Regexp{pattern2, pattern1, pattern3, pattern4}
	for _, pattern := range patterns {
		match := pattern.FindStringSubmatch(content)
		if match != nil {
			return strings.TrimSpace(match[1]), strings.TrimSpace(match[2]), true
		}
	}

	// Only check bare patterns for short content to avoid false positives
	if len(content) <= BarePatternMaxContentSize {
		for _, barePattern := range barePatterns {
			matched, _ := regexp.MatchString("(?i)"+barePattern, content)
			if matched {
				return "", "", true
			}
		}
	}

	return "", "", false
}

// ParseTimeWithTimezone parses a time string in the given timezone and converts to epoch.
// Returns (epoch, human, tz, error).
func ParseTimeWithTimezone(timeStr, tzStr string) (epoch int64, human, tz string, err error) {
	// Load timezone
	loc, err := time.LoadLocation(tzStr)
	if err != nil {
		return 0, "", "", fmt.Errorf("invalid timezone '%s': %w", tzStr, err)
	}

	// Get current time in the specified timezone
	now := time.Now().In(loc)

	// Parse the time string
	timeStrLower := strings.ToLower(strings.TrimSpace(timeStr))

	var hour, minute int

	// Handle 24-hour format (e.g., "18:00")
	if !strings.Contains(timeStrLower, "am") && !strings.Contains(timeStrLower, "pm") {
		if strings.Contains(timeStrLower, ":") {
			parts := strings.Split(timeStrLower, ":")
			if len(parts) != 2 {
				return 0, "", "", fmt.Errorf("invalid time format: %s", timeStr)
			}
			hour, err = strconv.Atoi(parts[0])
			if err != nil {
				return 0, "", "", fmt.Errorf("invalid hour: %s", parts[0])
			}
			minute, err = strconv.Atoi(parts[1])
			if err != nil {
				return 0, "", "", fmt.Errorf("invalid minute: %s", parts[1])
			}
		} else {
			return 0, "", "", fmt.Errorf("24-hour format requires colon: %s", timeStr)
		}
	} else {
		// Handle 12-hour format with am/pm
		// Remove spaces between time and am/pm
		timeStrLower = regexp.MustCompile(`\s+`).ReplaceAllString(timeStrLower, "")

		if strings.Contains(timeStrLower, ":") {
			// Format: "6:30pm"
			timePart := strings.TrimSuffix(strings.TrimSuffix(timeStrLower, "pm"), "am")
			parts := strings.Split(timePart, ":")
			if len(parts) != 2 {
				return 0, "", "", fmt.Errorf("invalid time format: %s", timeStr)
			}
			hour, err = strconv.Atoi(parts[0])
			if err != nil {
				return 0, "", "", fmt.Errorf("invalid hour: %s", parts[0])
			}
			minute, err = strconv.Atoi(parts[1])
			if err != nil {
				return 0, "", "", fmt.Errorf("invalid minute: %s", parts[1])
			}
		} else {
			// Format: "6pm"
			timePart := strings.TrimSuffix(strings.TrimSuffix(timeStrLower, "pm"), "am")
			hour, err = strconv.Atoi(timePart)
			if err != nil {
				return 0, "", "", fmt.Errorf("invalid hour: %s", timePart)
			}
			minute = 0
		}

		// Convert to 24-hour format
		if strings.Contains(timeStrLower, "pm") && hour != 12 {
			hour += 12
		} else if strings.Contains(timeStrLower, "am") && hour == 12 {
			hour = 0
		}
	}

	// Create reset datetime for today
	resetTime := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)

	// If the time is in the past, assume it's tomorrow
	if resetTime.Before(now) || resetTime.Equal(now) {
		resetTime = resetTime.Add(24 * time.Hour)
	}

	// Add buffer
	resetTime = resetTime.Add(RateLimitBufferSeconds * time.Second)

	// Convert to epoch
	epoch = resetTime.Unix()

	// Format human-readable time
	human = resetTime.Format("2006-01-02 15:04:05 MST")

	return epoch, human, tzStr, nil
}

// CheckRateLimit reads a file and checks for rate limit patterns.
// Returns nil if no rate limit detected.
// Returns RateLimitInfo with Detected=true, Parseable=false if rate limit found but unparseable.
// Returns RateLimitInfo with Detected=true, Parseable=true and timing info if fully parsed.
func CheckRateLimit(filePath string) (*RateLimitInfo, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	timeStr, tzStr, detected := FindRateLimitPattern(string(content))

	if !detected {
		return nil, nil
	}

	if timeStr == "" || tzStr == "" {
		// Rate limit detected but not parseable
		return &RateLimitInfo{
			Detected:  true,
			Parseable: false,
		}, nil
	}

	epoch, human, tz, err := ParseTimeWithTimezone(timeStr, tzStr)
	if err != nil {
		// Rate limit detected but time parsing failed
		return &RateLimitInfo{
			Detected:  true,
			Parseable: false,
		}, nil
	}

	return &RateLimitInfo{
		Detected:   true,
		Parseable:  true,
		ResetEpoch: epoch,
		ResetHuman: human,
		Timezone:   tz,
	}, nil
}
