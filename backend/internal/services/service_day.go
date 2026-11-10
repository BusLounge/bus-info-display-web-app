package services

import (
	"fmt"
	"time"
)

func currentServiceDayBounds(now time.Time, location *time.Location) (time.Time, time.Time, error) {
	if location == nil {
		return time.Time{}, time.Time{}, fmt.Errorf("service timezone is not configured")
	}

	localNow := now.In(location)
	start := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location)
	return start, start.AddDate(0, 0, 1), nil
}
