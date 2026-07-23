package businessclock

import (
	"testing"
	"time"
)

func TestConfigurePinsLocalTimezone(t *testing.T) {
	original := time.Local
	t.Cleanup(func() { time.Local = original })
	time.Local = time.UTC
	Configure()
	if time.Local.String() != LocationName {
		t.Fatalf("time.Local = %q, want %q", time.Local, LocationName)
	}
}

func TestBusinessBoundariesAndRFC3339(t *testing.T) {
	instant := time.Date(2026, 7, 31, 16, 30, 0, 0, time.UTC)
	start, end := MonthBounds(instant)
	if got := start.Format(time.RFC3339); got != "2026-08-01T00:00:00+08:00" {
		t.Fatalf("month start = %s", got)
	}
	if got := end.Format(time.RFC3339); got != "2026-09-01T00:00:00+08:00" {
		t.Fatalf("month end = %s", got)
	}
	if got := FormatRFC3339(instant); got != "2026-08-01T00:30:00+08:00" {
		t.Fatalf("formatted = %s", got)
	}
}

func TestParseNormalizesAbsoluteAndWallClockValues(t *testing.T) {
	absolute, err := Parse("2026-07-21T10:00:00Z")
	if err != nil || absolute.Format(time.RFC3339) != "2026-07-21T18:00:00+08:00" {
		t.Fatalf("absolute = %s, err = %v", absolute, err)
	}
	wallClock, err := Parse("2026-07-21 18:00:00")
	if err != nil || wallClock.Format(time.RFC3339) != "2026-07-21T18:00:00+08:00" {
		t.Fatalf("wall clock = %s, err = %v", wallClock, err)
	}
}
