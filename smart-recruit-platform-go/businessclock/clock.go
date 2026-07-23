// Package businessclock defines the platform-wide business timezone contract.
package businessclock

import (
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"
)

const (
	LocationName = "Asia/Shanghai"
	UTCOffset    = 8 * time.Hour
)

// Location is loaded from embedded tzdata, so containers do not depend on the
// host operating system's timezone database.
var Location = mustLoadLocation()

func mustLoadLocation() *time.Location {
	location, err := time.LoadLocation(LocationName)
	if err != nil {
		panic(fmt.Sprintf("load embedded business timezone %s: %v", LocationName, err))
	}
	return location
}

// Configure pins time.Local for libraries that use the process-local timezone.
// Service binaries call it before logging, configuration, or persistence setup.
func Configure() { time.Local = Location }

func Now() time.Time { return time.Now().In(Location) }

func StartOfDay(value time.Time) time.Time {
	local := value.In(Location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, Location)
}

func StartOfMonth(value time.Time) time.Time {
	local := value.In(Location)
	return time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, Location)
}

func MonthBounds(value time.Time) (time.Time, time.Time) {
	start := StartOfMonth(value)
	return start, start.AddDate(0, 1, 0)
}

func FormatRFC3339(value time.Time) string {
	return value.In(Location).Format(time.RFC3339)
}

// Parse accepts absolute RFC3339 values and timezone-free MySQL/API wall-clock
// values. Absolute values retain their instant and are normalized to Shanghai;
// timezone-free values are interpreted as Shanghai business time.
func Parse(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("business time is required")
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.In(Location), nil
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.999999",
		"2006-01-02T15:04:05",
	} {
		if parsed, err := time.ParseInLocation(layout, value, Location); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported business time %q", value)
}
