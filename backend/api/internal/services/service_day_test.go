package services

import (
	"testing"
	"time"
)

func TestCurrentServiceDayBoundsUsesConfiguredTimezone(t *testing.T) {
	location, err := time.LoadLocation("Asia/Colombo")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 10, 20, 0, 0, 0, time.UTC)

	start, end, err := currentServiceDayBounds(now, location)
	if err != nil {
		t.Fatal(err)
	}

	wantStart := time.Date(2026, 11, 11, 0, 0, 0, 0, location)
	wantEnd := time.Date(2026, 11, 12, 0, 0, 0, 0, location)
	if !start.Equal(wantStart) || !end.Equal(wantEnd) {
		t.Fatalf("currentServiceDayBounds() = [%s, %s), want [%s, %s)",
			start, end, wantStart, wantEnd)
	}
	if start.Location() != location || end.Location() != location {
		t.Fatalf("service day bounds must use %s, got %s and %s",
			location, start.Location(), end.Location())
	}
}

func TestCurrentServiceDayBoundsRequiresTimezone(t *testing.T) {
	if _, _, err := currentServiceDayBounds(time.Now(), nil); err == nil {
		t.Fatal("currentServiceDayBounds() with nil timezone should return an error")
	}
}
