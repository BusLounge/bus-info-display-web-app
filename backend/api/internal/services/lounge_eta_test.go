package services

import (
	"math"
	"testing"
	"time"

	"bus-schedule-lounge/internal/database"
	"bus-schedule-lounge/internal/models"
)

// Test route: a straight line along the equator from longitude 0 to 1.0, made of
// ten 10 km segments (100 km in total). Stops at km 0, 40, 60 and 100. The lounge
// is at km 50, between stop-40 and stop-60. All roads are URBAN = 40 km/h.

var testProfiles = map[string]float64{"URBAN": 1.0}

func floatPtr(v float64) *float64 { return &v }

func testGeometry(withSegments bool, offsetBefore, offsetAfter *float64) database.RouteGeometry {
	var geometry database.RouteGeometry
	if withSegments {
		for i := 0; i < 10; i++ {
			geometry.Segments = append(geometry.Segments, models.RouteSegment{
				SegmentOrder:   i + 1,
				StartLatitude:  0,
				StartLongitude: float64(i) / 10,
				EndLatitude:    0,
				EndLongitude:   float64(i+1) / 10,
				DistanceKM:     10,
				RoadType:       "URBAN",
			})
		}
	}
	geometry.Stops = []database.RouteStop{
		{ID: "stop-0", Order: 1, Lat: floatPtr(0), Lng: floatPtr(0)},
		{ID: "stop-40", Order: 2, Lat: floatPtr(0), Lng: floatPtr(0.4), OffsetMinutes: offsetBefore},
		{ID: "stop-60", Order: 3, Lat: floatPtr(0), Lng: floatPtr(0.6), OffsetMinutes: offsetAfter},
		{ID: "stop-100", Order: 4, Lat: floatPtr(0), Lng: floatPtr(1.0)},
	}
	return geometry
}

var testDeparture = time.Date(2026, 11, 17, 6, 0, 0, 0, time.UTC)

func testTrip() database.LoungeTripRow {
	return database.LoungeTripRow{
		ScheduledTripID: "trip-1",
		LoungeID:        "lounge-1",
		LoungeName:      "Test Lounge",
		LoungeLat:       floatPtr(0),
		LoungeLng:       floatPtr(0.5),
		MasterRouteID:   "route-1",
		DepartureTime:   testDeparture,
		StopBeforeID:    "stop-40",
		StopAfterID:     "stop-60",
	}
}

func approx(t *testing.T, name string, got, want, tolerance float64) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Errorf("%s = %v, want %v (±%v)", name, got, want, tolerance)
	}
}

func TestLoungePositionAlongRoute(t *testing.T) {
	model := newRouteModel(testGeometry(true, nil, nil))
	km, ok := model.loungePosition(testTrip())
	if !ok {
		t.Fatal("lounge position not found")
	}
	approx(t, "lounge km", km, 50, 0.01)
}

func TestLoungePositionIsKeptBetweenItsStops(t *testing.T) {
	model := newRouteModel(testGeometry(true, nil, nil))
	trip := testTrip()
	trip.LoungeLng = floatPtr(0.05) // coordinates wrongly near the start of the route
	km, ok := model.loungePosition(trip)
	if !ok {
		t.Fatal("lounge position not found")
	}
	approx(t, "lounge km", km, 40, 0.01)
}

func TestScheduleEstimateUsesStopOffsetsInterpolated(t *testing.T) {
	model := newRouteModel(testGeometry(true, floatPtr(80), floatPtr(120)))
	estimate, ok := estimateFromSchedule(testTrip(), model, testProfiles)
	if !ok {
		t.Fatal("no estimate")
	}
	if estimate.source != ETASourceStopOffsets {
		t.Errorf("source = %s, want %s", estimate.source, ETASourceStopOffsets)
	}
	approx(t, "minutes", estimate.minutes, 100, 0.01) // halfway between 80 and 120
}

func TestScheduleEstimateRejectsImpossibleSpeed(t *testing.T) {
	model := newRouteModel(testGeometry(true, nil, nil))
	trip := testTrip()
	trip.TripDurationMinutes = floatPtr(60)   // 50 km in 30 min = 100 km/h: impossible
	trip.RouteDurationMinutes = floatPtr(200) // 50 km in 100 min = 30 km/h: plausible
	estimate, ok := estimateFromSchedule(trip, model, testProfiles)
	if !ok {
		t.Fatal("no estimate")
	}
	if estimate.source != ETASourceRouteDuration {
		t.Errorf("source = %s, want %s", estimate.source, ETASourceRouteDuration)
	}
	approx(t, "minutes", estimate.minutes, 100, 0.01)
}

func TestScheduleEstimateFallsBackToRoadTypeSpeeds(t *testing.T) {
	model := newRouteModel(testGeometry(true, nil, nil))
	trip := testTrip()
	trip.TripDurationMinutes = floatPtr(60)
	trip.RouteDurationMinutes = floatPtr(70)
	estimate, ok := estimateFromSchedule(trip, model, testProfiles)
	if !ok {
		t.Fatal("no estimate")
	}
	if estimate.source != ETASourceRoadType {
		t.Errorf("source = %s, want %s", estimate.source, ETASourceRoadType)
	}
	approx(t, "minutes", estimate.minutes, 75, 0.01) // 50 km at 40 km/h
}

func TestScheduleEstimateWithoutRoadSegmentsUsesStopOrder(t *testing.T) {
	model := newRouteModel(testGeometry(false, nil, nil))
	trip := testTrip()
	trip.TripDurationMinutes = floatPtr(120)
	estimate, ok := estimateFromSchedule(trip, model, testProfiles)
	if !ok {
		t.Fatal("no estimate")
	}
	if estimate.source != ETASourceTripDuration {
		t.Errorf("source = %s, want %s", estimate.source, ETASourceTripDuration)
	}
	// Lounge between stop 2 and 3 of 4 (ranks 1 and 2 of 0-3): 1.5 / 3 = half way.
	approx(t, "minutes", estimate.minutes, 60, 0.01)
}

func TestNoEstimateWithoutAnyData(t *testing.T) {
	model := newRouteModel(database.RouteGeometry{})
	if _, ok := estimateFromSchedule(testTrip(), model, testProfiles); ok {
		t.Fatal("expected no estimate without stops, segments or durations")
	}
}

func liveTrip(lng float64, updatedAgo time.Duration, now time.Time) database.LoungeTripRow {
	trip := testTrip()
	trip.TripDurationMinutes = floatPtr(150) // 75 min to the lounge = 40 km/h
	trip.ActiveTripStatus = "in_transit"
	trip.CurrentLat = floatPtr(0)
	trip.CurrentLng = floatPtr(lng)
	updated := now.Add(-updatedAgo)
	trip.LastLocationUpdate = &updated
	return trip
}

func computeOne(t *testing.T, trip database.LoungeTripRow, now time.Time) []LoungeTripETA {
	t.Helper()
	routes := map[string]*routeModel{"route-1": newRouteModel(testGeometry(true, nil, nil))}
	return computeLoungeETAs([]database.LoungeTripRow{trip}, routes, now, testProfiles, 3*time.Minute)
}

func TestLiveGPSGivesArrivalFromBusPosition(t *testing.T) {
	now := testDeparture.Add(40 * time.Minute)
	entries := computeOne(t, liveTrip(0.3, time.Minute, now), now) // bus at km 30, updated 1 min ago
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	entry := entries[0]
	if entry.Source != ETASourceLiveGPS {
		t.Errorf("source = %s, want %s", entry.Source, ETASourceLiveGPS)
	}
	// 20 km left at 40 km/h = 30 min after the position was reported.
	want := now.Add(-time.Minute).Add(30 * time.Minute)
	if !entry.ArrivalETA.Equal(want) {
		t.Errorf("arrival = %v, want %v", entry.ArrivalETA, want)
	}
	approx(t, "remaining km", entry.DistanceKm, 20, 0.01)
}

func TestLiveGPSDropsBusThatHasPassedTheLounge(t *testing.T) {
	now := testDeparture.Add(80 * time.Minute)
	entries := computeOne(t, liveTrip(0.55, time.Minute, now), now) // bus at km 55, lounge at km 50
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want the passed bus to be dropped", len(entries))
	}
}

func TestStaleGPSIsIgnored(t *testing.T) {
	now := testDeparture.Add(40 * time.Minute)
	entries := computeOne(t, liveTrip(0.55, 10*time.Minute, now), now) // position 10 min old
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Source == ETASourceLiveGPS {
		t.Error("used a 10 minute old GPS position")
	}
	if want := testDeparture.Add(75 * time.Minute); !entries[0].ArrivalETA.Equal(want) {
		t.Errorf("arrival = %v, want schedule estimate %v", entries[0].ArrivalETA, want)
	}
}

func TestGPSFarFromTheRouteIsIgnored(t *testing.T) {
	now := testDeparture.Add(40 * time.Minute)
	trip := liveTrip(0.3, time.Minute, now)
	trip.CurrentLat = floatPtr(0.1) // about 11 km off the route
	entries := computeOne(t, trip, now)
	if len(entries) != 1 || entries[0].Source == ETASourceLiveGPS {
		t.Fatalf("expected the schedule estimate, got %+v", entries)
	}
}

func TestGPSIgnoredUntilTheConductorStartsTheTrip(t *testing.T) {
	now := testDeparture.Add(40 * time.Minute)
	trip := liveTrip(0.3, time.Minute, now)
	trip.ActiveTripStatus = "not_started"
	entries := computeOne(t, trip, now)
	if len(entries) != 1 || entries[0].Source == ETASourceLiveGPS {
		t.Fatalf("expected the schedule estimate, got %+v", entries)
	}
}

func TestDepartureIsArrivalPlusStopover(t *testing.T) {
	now := testDeparture
	trip := testTrip()
	trip.TripDurationMinutes = floatPtr(150)
	entries := computeOne(t, trip, now)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if got := entries[0].DepartureETA.Sub(entries[0].ArrivalETA); got != 3*time.Minute {
		t.Errorf("departure - arrival = %v, want 3m", got)
	}
}

func TestSelectForBoardWindowSortAndLimit(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	at := func(offset time.Duration) LoungeTripETA { return LoungeTripETA{ArrivalETA: now.Add(offset)} }
	entries := []LoungeTripETA{
		at(-20 * time.Minute), // more than 15 min ago: dropped
		at(2 * time.Hour),
		at(-10 * time.Minute), // within the 15 min grace: kept
		at(7 * time.Hour),     // beyond 6 hours: dropped
		at(30 * time.Minute),
	}
	got := selectForBoard(entries, now, func(e LoungeTripETA) time.Time { return e.ArrivalETA })
	want := []time.Duration{-10 * time.Minute, 30 * time.Minute, 2 * time.Hour}
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d", len(got), len(want))
	}
	for i, offset := range want {
		if !got[i].ArrivalETA.Equal(now.Add(offset)) {
			t.Errorf("entry %d = %v, want %v", i, got[i].ArrivalETA, now.Add(offset))
		}
	}

	var many []LoungeTripETA
	for i := 0; i < 15; i++ {
		many = append(many, at(time.Duration(i)*time.Minute))
	}
	if n := len(selectForBoard(many, now, func(e LoungeTripETA) time.Time { return e.ArrivalETA })); n != boardMaxRows {
		t.Errorf("got %d entries, want at most %d", n, boardMaxRows)
	}
}

func TestRemarksAndClockTime(t *testing.T) {
	colombo, err := time.LoadLocation("Asia/Colombo")
	if err != nil {
		t.Skip("Asia/Colombo timezone data not available")
	}
	now := time.Date(2026, 11, 17, 6, 30, 0, 0, time.UTC) // 12:00 in Colombo
	cases := []struct {
		got, want string
	}{
		{arrivalRemark(now.Add(-10*time.Minute), now, colombo), "Delayed 10 min"},
		{arrivalRemark(now.Add(3*time.Minute), now, colombo), "Arriving in 3 min"},
		{arrivalRemark(now.Add(2*time.Hour), now, colombo), "Expected at 14:00"},
		{departureRemark(now.Add(4*time.Minute), now, colombo), "Boarding Now"},
		{departureRemark(now.Add(45*time.Minute), now, colombo), "Departs at 12:45"},
		{clockTime(now, colombo), "12:00"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}
