package middleware

import (
	"testing"
	"time"
)

func TestMidnightUsesShanghaiCivilDay(t *testing.T) {
	instant := time.Date(2026, 7, 21, 16, 30, 0, 0, time.UTC)
	if got := midnight(instant).Format(time.RFC3339); got != "2026-07-22T00:00:00+08:00" {
		t.Fatalf("midnight = %s", got)
	}
	if ttl := secondsUntilMidnight(instant); ttl != 23*60*60+30*60+1 {
		t.Fatalf("seconds until midnight = %d", ttl)
	}
}

func TestNextHourUsesShanghaiClock(t *testing.T) {
	instant := time.Date(2026, 7, 21, 16, 30, 0, 0, time.UTC)
	if got := nextHour(instant).Format(time.RFC3339); got != "2026-07-22T01:00:00+08:00" {
		t.Fatalf("next hour = %s", got)
	}
}
