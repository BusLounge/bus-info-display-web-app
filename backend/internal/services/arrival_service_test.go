package services

import (
	"math"
	"testing"

	"bus-schedule-lounge/internal/models"
)

func TestCalculateSegmentBasedETA(t *testing.T) {
	segments := []models.RouteSegment{
		{
			SegmentOrder:     1,
			StartLatitude:    0,
			StartLongitude:   0,
			EndLatitude:      0,
			EndLongitude:     0.1,
			DistanceKM:       10,
			BaselineSpeedKMH: 50,
			RoadType:         "HIGHWAY",
		},
		{
			SegmentOrder:     2,
			StartLatitude:    0,
			StartLongitude:   0.1,
			EndLatitude:      0,
			EndLongitude:     0.2,
			DistanceKM:       20,
			BaselineSpeedKMH: 20,
			RoadType:         "RURAL",
		},
	}

	distance, eta, ok := calculateSegmentBasedETA(0, 0, 0, 0.2, segments, nil)
	if !ok {
		t.Fatal("calculateSegmentBasedETA() did not find a route")
	}
	if math.Abs(distance-30) > 0.001 {
		t.Errorf("distance = %v km, want 30 km", distance)
	}
	if math.Abs(eta-72) > 0.001 {
		t.Errorf("ETA = %v minutes, want 72 minutes", eta)
	}
}

func TestCalculateSegmentBasedETAUsesRoadTypeProfileForLegacyDefaultBaseline(t *testing.T) {
	segments := []models.RouteSegment{
		{
			StartLatitude:  0,
			StartLongitude: 0,
			EndLatitude:    0,
			EndLongitude:   0.1,
			DistanceKM:     10,
			BaselineSpeedKMH: 40,
			RoadType:       "RURAL",
		},
	}

	distance, eta, ok := calculateSegmentBasedETA(0, 0, 0, 0.1, segments, map[string]float64{
		"RURAL": 2,
	})
	if !ok {
		t.Fatal("calculateSegmentBasedETA() did not find a route")
	}
	if math.Abs(distance-10) > 0.001 {
		t.Errorf("distance = %v km, want 10 km", distance)
	}
	if math.Abs(eta-30) > 0.001 {
		t.Errorf("ETA = %v minutes, want 30 minutes", eta)
	}
}

func TestCalculateSegmentBasedETAReturnsNoRouteWhenTargetIsBehindBus(t *testing.T) {
	segments := []models.RouteSegment{
		{
			StartLatitude:    0,
			StartLongitude:   0,
			EndLatitude:      0,
			EndLongitude:     0.1,
			DistanceKM:       10,
			BaselineSpeedKMH: 40,
		},
	}

	if _, _, ok := calculateSegmentBasedETA(0, 0.1, 0, 0, segments, nil); ok {
		t.Fatal("calculateSegmentBasedETA() found a forward route to a target behind the bus")
	}
}
