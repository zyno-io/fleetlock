package fleetlock

import (
	"fmt"
	"time"
)

// maintenanceTime represents a time-of-day in HH:MM format.
type maintenanceTime struct {
	hour   int
	minute int
}

// parseMaintenanceTime parses a "HH:MM" string into a maintenanceTime.
func parseMaintenanceTime(s string) (*maintenanceTime, error) {
	var h, m int
	n, err := fmt.Sscanf(s, "%d:%d", &h, &m)
	if err != nil || n != 2 {
		return nil, fmt.Errorf("invalid time format %q, expected HH:MM", s)
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return nil, fmt.Errorf("invalid time %q: hour must be 0-23, minute must be 0-59", s)
	}
	return &maintenanceTime{hour: h, minute: m}, nil
}

// minutesSinceMidnight returns the number of minutes since midnight.
func (mt *maintenanceTime) minutesSinceMidnight() int {
	return mt.hour*60 + mt.minute
}

// inMaintenanceWindow checks if the given time falls within the maintenance window.
// Supports windows that span midnight (e.g. 22:00 - 06:00).
func inMaintenanceWindow(now time.Time, start, end *maintenanceTime) bool {
	nowMinutes := now.Hour()*60 + now.Minute()
	startMinutes := start.minutesSinceMidnight()
	endMinutes := end.minutesSinceMidnight()

	if startMinutes <= endMinutes {
		// same-day window: e.g. 02:00 - 06:00
		return nowMinutes >= startMinutes && nowMinutes < endMinutes
	}
	// cross-midnight window: e.g. 22:00 - 06:00
	return nowMinutes >= startMinutes || nowMinutes < endMinutes
}
