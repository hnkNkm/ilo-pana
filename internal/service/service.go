// Package service implements the domain logic behind the Wails App facade:
// HTTP request execution, collection/environment persistence and OpenAPI
// imports. All services are plain Go (no Wails types) and take their
// storage and clock as dependencies so they can be unit-tested directly.
package service

import "time"

// Clock returns the current time. It is injectable so tests can control
// the timestamps written to persisted collections and environments.
type Clock func() time.Time

// RealClock returns the system clock.
func RealClock() Clock {
	return time.Now
}
