package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSchedule_YYYY_MM_DD(t *testing.T) {
	result, err := ParseSchedule("2026-03-15")
	require.NoError(t, err)

	assert.Equal(t, 2026, result.Year())
	assert.Equal(t, time.March, result.Month())
	assert.Equal(t, 15, result.Day())
	assert.Equal(t, 0, result.Hour())
	assert.Equal(t, 0, result.Minute())
}

func TestParseSchedule_HH_MM_Future(t *testing.T) {
	now := time.Now()

	// Create a time that's definitely in the future (1 hour from now)
	futureTime := now.Add(1 * time.Hour)
	input := futureTime.Format("15:04")

	result, err := ParseSchedule(input)
	require.NoError(t, err)

	// Should be today
	assert.Equal(t, now.Year(), result.Year())
	assert.Equal(t, now.Month(), result.Month())
	assert.Equal(t, now.Day(), result.Day())
	assert.Equal(t, futureTime.Hour(), result.Hour())
	assert.Equal(t, futureTime.Minute(), result.Minute())

	// Should be in the future
	assert.True(t, result.After(now), "parsed time should be in the future")
}

func TestParseSchedule_HH_MM_Past(t *testing.T) {
	now := time.Now()

	// Create a time that's definitely in the past (1 hour ago)
	pastTime := now.Add(-1 * time.Hour)
	input := pastTime.Format("15:04")

	result, err := ParseSchedule(input)
	require.NoError(t, err)

	// Should be tomorrow
	tomorrow := now.AddDate(0, 0, 1)
	assert.Equal(t, tomorrow.Year(), result.Year())
	assert.Equal(t, tomorrow.Month(), result.Month())
	assert.Equal(t, tomorrow.Day(), result.Day())
	assert.Equal(t, pastTime.Hour(), result.Hour())
	assert.Equal(t, pastTime.Minute(), result.Minute())

	// Should be in the future
	assert.True(t, result.After(now), "parsed time should be tomorrow (in the future)")
}

func TestParseSchedule_DateTimeWithSpace(t *testing.T) {
	result, err := ParseSchedule("2026-03-15 14:30")
	require.NoError(t, err)

	assert.Equal(t, 2026, result.Year())
	assert.Equal(t, time.March, result.Month())
	assert.Equal(t, 15, result.Day())
	assert.Equal(t, 14, result.Hour())
	assert.Equal(t, 30, result.Minute())
}

func TestParseSchedule_ISO8601(t *testing.T) {
	result, err := ParseSchedule("2026-03-15T14:30")
	require.NoError(t, err)

	assert.Equal(t, 2026, result.Year())
	assert.Equal(t, time.March, result.Month())
	assert.Equal(t, 15, result.Day())
	assert.Equal(t, 14, result.Hour())
	assert.Equal(t, 30, result.Minute())
}

func TestParseSchedule_InvalidFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"invalid date", "2026-13-45"},
		{"invalid time", "25:99"},
		{"random text", "not a date"},
		{"partial date", "2026-03"},
		{"partial time", "14"},
		{"wrong separator", "2026/03/15"},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSchedule(tt.input)
			assert.Error(t, err, "should error for input: %q", tt.input)
			assert.Contains(t, err.Error(), "invalid schedule format", "error should mention invalid format")
		})
	}
}

func TestParseSchedule_AllFormats(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{"YYYY-MM-DD", "2026-06-15", false},
		{"HH:MM", "14:30", false},
		{"YYYY-MM-DD HH:MM", "2026-06-15 14:30", false},
		{"ISO 8601", "2026-06-15T14:30", false},
		{"invalid", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSchedule(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.False(t, result.IsZero(), "result should not be zero time")
			}
		})
	}
}

func TestParseSchedule_EdgeCases(t *testing.T) {
	t.Run("midnight", func(t *testing.T) {
		result, err := ParseSchedule("2026-03-15 00:00")
		require.NoError(t, err)
		assert.Equal(t, 0, result.Hour())
		assert.Equal(t, 0, result.Minute())
	})

	t.Run("end of day", func(t *testing.T) {
		result, err := ParseSchedule("2026-03-15 23:59")
		require.NoError(t, err)
		assert.Equal(t, 23, result.Hour())
		assert.Equal(t, 59, result.Minute())
	})

	t.Run("leap day", func(t *testing.T) {
		result, err := ParseSchedule("2024-02-29")
		require.NoError(t, err)
		assert.Equal(t, 2024, result.Year())
		assert.Equal(t, time.February, result.Month())
		assert.Equal(t, 29, result.Day())
	})
}

func TestParseSchedule_Timezone(t *testing.T) {
	// All parsed times should be in local timezone
	result, err := ParseSchedule("2026-03-15 14:30")
	require.NoError(t, err)

	localZone := time.Now().Location()
	assert.Equal(t, localZone, result.Location(), "should use local timezone")
}
