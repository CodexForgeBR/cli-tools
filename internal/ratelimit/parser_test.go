package ratelimit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// FindRateLimitPattern tests
// ---------------------------------------------------------------------------

func TestFindRateLimitPattern_Pattern1(t *testing.T) {
	content := `Error: rate limit hit, resets 6pm (America/Bahia)`
	timeStr, tzStr, detected := FindRateLimitPattern(content)
	assert.True(t, detected)
	assert.Equal(t, "6pm", timeStr)
	assert.Equal(t, "America/Bahia", tzStr)
}

func TestFindRateLimitPattern_Pattern2(t *testing.T) {
	content := `Error: rate limit, resets 6:30pm (America/Sao_Paulo)`
	timeStr, tzStr, detected := FindRateLimitPattern(content)
	assert.True(t, detected)
	assert.Equal(t, "6:30pm", timeStr)
	assert.Equal(t, "America/Sao_Paulo", tzStr)
}

func TestFindRateLimitPattern_Pattern3(t *testing.T) {
	content := `Rate limit resets 18:00 (UTC)`
	timeStr, tzStr, detected := FindRateLimitPattern(content)
	assert.True(t, detected)
	assert.Equal(t, "18:00", timeStr)
	assert.Equal(t, "UTC", tzStr)
}

func TestFindRateLimitPattern_Pattern4(t *testing.T) {
	content := `Rate limit resets Jan 15, 2026, 9am (UTC)`
	timeStr, tzStr, detected := FindRateLimitPattern(content)
	assert.True(t, detected)
	assert.Equal(t, "9am", timeStr)
	assert.Equal(t, "UTC", tzStr)
}

func TestFindRateLimitPattern_Pattern2Priority(t *testing.T) {
	// Pattern 2 (time with minutes) should take priority over pattern 1
	content := `resets 3:45pm (Europe/London)`
	timeStr, tzStr, detected := FindRateLimitPattern(content)
	assert.True(t, detected)
	assert.Equal(t, "3:45pm", timeStr)
	assert.Equal(t, "Europe/London", tzStr)
}

func TestFindRateLimitPattern_BarePatterns(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"rate limit exceeded", "rate limit exceeded"},
		{"rate limited", "You are rate limited"},
		{"too many requests", "too many requests, try again later"},
		{"hit your limit", "you've hit your limit"},
		{"hit limit no apostrophe", "youve hit your limit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeStr, tzStr, detected := FindRateLimitPattern(tt.content)
			assert.True(t, detected, "should detect bare pattern")
			assert.Empty(t, timeStr)
			assert.Empty(t, tzStr)
		})
	}
}

func TestFindRateLimitPattern_BarePatternSkippedForLargeContent(t *testing.T) {
	// Content larger than BarePatternMaxContentSize should not trigger bare patterns
	content := "rate limit exceeded" + strings.Repeat(" ", BarePatternMaxContentSize)
	_, _, detected := FindRateLimitPattern(content)
	assert.False(t, detected, "should not trigger bare pattern on large content")
}

func TestFindRateLimitPattern_NoMatch(t *testing.T) {
	content := "Everything is working fine, no issues here."
	_, _, detected := FindRateLimitPattern(content)
	assert.False(t, detected)
}

func TestFindRateLimitPattern_CaseInsensitive(t *testing.T) {
	content := `RESETS 10AM (UTC)`
	timeStr, tzStr, detected := FindRateLimitPattern(content)
	assert.True(t, detected)
	assert.Equal(t, "10AM", timeStr)
	assert.Equal(t, "UTC", tzStr)
}

func TestFindRateLimitPattern_ResetWithoutS(t *testing.T) {
	content := `reset 5pm (America/New_York)`
	timeStr, tzStr, detected := FindRateLimitPattern(content)
	assert.True(t, detected)
	assert.Equal(t, "5pm", timeStr)
	assert.Equal(t, "America/New_York", tzStr)
}

// ---------------------------------------------------------------------------
// ParseTimeWithTimezone tests
// ---------------------------------------------------------------------------

func TestParseTimeWithTimezone_SimpleHour(t *testing.T) {
	epoch, human, tz, err := ParseTimeWithTimezone("6pm", "UTC")
	require.NoError(t, err)
	assert.NotZero(t, epoch)
	assert.NotEmpty(t, human)
	assert.Equal(t, "UTC", tz)
}

func TestParseTimeWithTimezone_HourMinute(t *testing.T) {
	epoch, human, tz, err := ParseTimeWithTimezone("6:30pm", "UTC")
	require.NoError(t, err)
	assert.NotZero(t, epoch)
	assert.NotEmpty(t, human)
	assert.Equal(t, "UTC", tz)
}

func TestParseTimeWithTimezone_24Hour(t *testing.T) {
	epoch, human, tz, err := ParseTimeWithTimezone("18:00", "UTC")
	require.NoError(t, err)
	assert.NotZero(t, epoch)
	assert.NotEmpty(t, human)
	assert.Equal(t, "UTC", tz)
}

func TestParseTimeWithTimezone_AM(t *testing.T) {
	epoch, _, _, err := ParseTimeWithTimezone("8am", "UTC")
	require.NoError(t, err)
	assert.NotZero(t, epoch)
}

func TestParseTimeWithTimezone_Midnight(t *testing.T) {
	epoch, _, _, err := ParseTimeWithTimezone("12am", "UTC")
	require.NoError(t, err)
	assert.NotZero(t, epoch)
}

func TestParseTimeWithTimezone_Noon(t *testing.T) {
	epoch, _, _, err := ParseTimeWithTimezone("12pm", "UTC")
	require.NoError(t, err)
	assert.NotZero(t, epoch)
}

func TestParseTimeWithTimezone_FutureTime(t *testing.T) {
	// A time far in the future should resolve to today
	now := time.Now().UTC()
	futureHour := (now.Hour() + 2) % 24
	timeStr := time.Date(2000, 1, 1, futureHour, 0, 0, 0, time.UTC).Format("15:04")

	epoch, _, _, err := ParseTimeWithTimezone(timeStr, "UTC")
	require.NoError(t, err)

	resetTime := time.Unix(epoch, 0).UTC()
	// Should be today (with buffer), so same day or tomorrow at most
	assert.GreaterOrEqual(t, resetTime.Day(), now.Day())
}

func TestParseTimeWithTimezone_PastTimeWrapsToTomorrow(t *testing.T) {
	// A time in the past should wrap to tomorrow
	now := time.Now().UTC()
	pastHour := (now.Hour() + 23) % 24 // Effectively 1 hour ago
	timeStr := time.Date(2000, 1, 1, pastHour, 0, 0, 0, time.UTC).Format("15:04")

	epoch, _, _, err := ParseTimeWithTimezone(timeStr, "UTC")
	require.NoError(t, err)

	resetTime := time.Unix(epoch, 0).UTC()
	// Reset time should be in the future (includes buffer)
	assert.True(t, resetTime.After(now))
}

func TestParseTimeWithTimezone_InvalidTimezone(t *testing.T) {
	_, _, _, err := ParseTimeWithTimezone("6pm", "Invalid/Timezone")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timezone")
}

func TestParseTimeWithTimezone_InvalidFormat(t *testing.T) {
	_, _, _, err := ParseTimeWithTimezone("not-a-time", "UTC")
	require.Error(t, err)
}

func TestParseTimeWithTimezone_BufferAdded(t *testing.T) {
	// Verify that the buffer (60 seconds) is added to the target time
	now := time.Now().UTC()
	futureHour := (now.Hour() + 3) % 24
	targetTime := time.Date(now.Year(), now.Month(), now.Day(), futureHour, 0, 0, 0, time.UTC)
	if targetTime.Before(now) {
		targetTime = targetTime.Add(24 * time.Hour)
	}

	timeStr := targetTime.Format("15:04")

	epoch, _, _, err := ParseTimeWithTimezone(timeStr, "UTC")
	require.NoError(t, err)

	resetTime := time.Unix(epoch, 0).UTC()
	expectedWithBuffer := targetTime.Add(RateLimitBufferSeconds * time.Second)
	// Allow 1 second tolerance for test execution time
	diff := resetTime.Sub(expectedWithBuffer)
	assert.LessOrEqual(t, diff.Abs(), 1*time.Second,
		"reset time should be target + %ds buffer", RateLimitBufferSeconds)
}

func TestParseTimeWithTimezone_SpaceBetweenTimeAndAMPM(t *testing.T) {
	epoch, _, _, err := ParseTimeWithTimezone("6 pm", "UTC")
	require.NoError(t, err)
	assert.NotZero(t, epoch)
}

func TestParseTimeWithTimezone_24HourNoColon(t *testing.T) {
	_, _, _, err := ParseTimeWithTimezone("1800", "UTC")
	require.Error(t, err, "24-hour format without colon should fail")
}

// ---------------------------------------------------------------------------
// CheckRateLimit tests
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// ParseTimeWithTimezone error-branch tests
// ---------------------------------------------------------------------------

func TestParseTimeWithTimezone_24Hour_InvalidParts(t *testing.T) {
	// "18:00:00" splits into 3 parts on ":", triggering len(parts) != 2
	_, _, _, err := ParseTimeWithTimezone("18:00:00", "UTC")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid time format")
}

func TestParseTimeWithTimezone_24Hour_InvalidHour(t *testing.T) {
	// "abc:00" — first part is not a number
	_, _, _, err := ParseTimeWithTimezone("abc:00", "UTC")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid hour")
}

func TestParseTimeWithTimezone_24Hour_InvalidMinute(t *testing.T) {
	// "18:def" — second part is not a number
	_, _, _, err := ParseTimeWithTimezone("18:def", "UTC")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid minute")
}

func TestParseTimeWithTimezone_12Hour_InvalidParts(t *testing.T) {
	// "6:30:15pm" — splits into 3 parts on ":", triggering len(parts) != 2
	_, _, _, err := ParseTimeWithTimezone("6:30:15pm", "UTC")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid time format")
}

func TestParseTimeWithTimezone_12Hour_InvalidHour(t *testing.T) {
	// "abc:30pm" — first part is not a number
	_, _, _, err := ParseTimeWithTimezone("abc:30pm", "UTC")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid hour")
}

func TestParseTimeWithTimezone_12Hour_InvalidMinute(t *testing.T) {
	// "6:defpm" — second part is not a number
	_, _, _, err := ParseTimeWithTimezone("6:defpm", "UTC")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid minute")
}

func TestParseTimeWithTimezone_SimpleHour_InvalidHour(t *testing.T) {
	// "abcpm" — hour part is not a number
	_, _, _, err := ParseTimeWithTimezone("abcpm", "UTC")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid hour")
}

// ---------------------------------------------------------------------------
// CheckRateLimit tests
// ---------------------------------------------------------------------------

func TestCheckRateLimit_FileWithRateLimit(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "output.txt")
	err := os.WriteFile(filePath, []byte("Error: resets 6pm (UTC)"), 0644)
	require.NoError(t, err)

	info, err := CheckRateLimit(filePath)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.True(t, info.Detected)
	assert.True(t, info.Parseable)
	assert.NotZero(t, info.ResetEpoch)
}

func TestCheckRateLimit_FileWithoutRateLimit(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "output.txt")
	err := os.WriteFile(filePath, []byte("All good, no errors!"), 0644)
	require.NoError(t, err)

	info, err := CheckRateLimit(filePath)
	require.NoError(t, err)
	assert.Nil(t, info)
}

func TestCheckRateLimit_FileNotFound(t *testing.T) {
	info, err := CheckRateLimit("/nonexistent/path/file.txt")
	require.Error(t, err)
	assert.Nil(t, info)
}

func TestCheckRateLimit_BarePatternDetected(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "output.txt")
	err := os.WriteFile(filePath, []byte("rate limit exceeded"), 0644)
	require.NoError(t, err)

	info, err := CheckRateLimit(filePath)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.True(t, info.Detected)
	assert.False(t, info.Parseable, "bare pattern should not be parseable")
}

func TestCheckRateLimit_UnparseableTimezone(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "output.txt")
	// Time with invalid timezone: detected but not parseable
	err := os.WriteFile(filePath, []byte("resets 6pm (Fake/Zone)"), 0644)
	require.NoError(t, err)

	info, err := CheckRateLimit(filePath)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.True(t, info.Detected)
	assert.False(t, info.Parseable, "invalid timezone should yield not parseable")
}
