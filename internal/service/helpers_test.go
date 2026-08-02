package service

import (
	"time"
)

// fixedClock returns a clock that always reports the given time.
func fixedClock(t time.Time) Clock {
	return func() time.Time { return t }
}

var (
	clockT1 = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	clockT2 = clockT1.Add(24 * time.Hour)
)
