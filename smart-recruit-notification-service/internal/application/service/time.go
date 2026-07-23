package service

import (
	"time"

	"smart-recruit-platform-go/businessclock"
)

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return businessclock.FormatRFC3339(t)
}
