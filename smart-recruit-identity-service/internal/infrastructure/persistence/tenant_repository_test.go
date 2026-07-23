package persistence

import (
	"testing"
	"time"
)

func TestUsageWindowUsesCalendarMonthForMonthlyApplications(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	measuredAt := time.Date(2026, time.July, 19, 16, 30, 0, 0, location)
	start, end := usageWindow("applications.monthly.max", measuredAt)
	if want := time.Date(2026, time.July, 1, 0, 0, 0, 0, location); !start.Equal(want) {
		t.Fatalf("start = %v, want %v", start, want)
	}
	if want := time.Date(2026, time.August, 1, 0, 0, 0, 0, location); !end.Equal(want) {
		t.Fatalf("end = %v, want %v", end, want)
	}
}

func TestUsageWindowUsesDailyBucketsForPointInTimeMetrics(t *testing.T) {
	measuredAt := time.Date(2026, time.July, 19, 16, 30, 0, 0, time.UTC)
	start, end := usageWindow("members.max", measuredAt)
	if want := time.Date(2026, time.July, 19, 0, 0, 0, 0, time.UTC); !start.Equal(want) {
		t.Fatalf("start = %v, want %v", start, want)
	}
	if want := time.Date(2026, time.July, 20, 0, 0, 0, 0, time.UTC); !end.Equal(want) {
		t.Fatalf("end = %v, want %v", end, want)
	}
}
