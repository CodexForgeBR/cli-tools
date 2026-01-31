package schedule

import (
	"fmt"
	"time"
)

// ParseSchedule parses a schedule string into a time.Time.
// Supports 4 formats:
// - YYYY-MM-DD → midnight of that date
// - HH:MM → today if future, tomorrow if past
// - "YYYY-MM-DD HH:MM" → exact datetime
// - YYYY-MM-DDTHH:MM → ISO 8601 format
func ParseSchedule(input string) (time.Time, error) {
	now := time.Now()
	local := now.Location()

	// Try YYYY-MM-DDTHH:MM (ISO 8601)
	if t, err := time.ParseInLocation("2006-01-02T15:04", input, local); err == nil {
		return t, nil
	}

	// Try "YYYY-MM-DD HH:MM"
	if t, err := time.ParseInLocation("2006-01-02 15:04", input, local); err == nil {
		return t, nil
	}

	// Try YYYY-MM-DD
	if t, err := time.ParseInLocation("2006-01-02", input, local); err == nil {
		return t, nil
	}

	// Try HH:MM
	if t, err := time.ParseInLocation("15:04", input, local); err == nil {
		// Set to today's date
		scheduled := time.Date(now.Year(), now.Month(), now.Day(),
			t.Hour(), t.Minute(), 0, 0, local)
		// If past, move to tomorrow
		if scheduled.Before(now) {
			scheduled = scheduled.AddDate(0, 0, 1)
		}
		return scheduled, nil
	}

	return time.Time{}, fmt.Errorf("invalid schedule format: %q (supported: YYYY-MM-DD, HH:MM, \"YYYY-MM-DD HH:MM\", YYYY-MM-DDTHH:MM)", input)
}
