package database

import (
	"testing"

	"bus-schedule-lounge/internal/models"
)

func TestTruncateRouteSegmentsAtLoungeClipsAtProjectedPosition(t *testing.T) {
	segments := []models.RouteSegment{
		{
			SegmentOrder:    1,
			StartLatitude:   0,
			StartLongitude:  0,
			EndLatitude:     0,
			EndLongitude:    0.1,
			DistanceKM:      10,
		},
		{
			SegmentOrder:    2,
			StartLatitude:   0,
			StartLongitude:  0.1,
			EndLatitude:     0,
			EndLongitude:    0.2,
			DistanceKM:      20,
		},
		{
			SegmentOrder:    3,
			StartLatitude:   0,
			StartLongitude:  0.2,
			EndLatitude:     0,
			EndLongitude:    0.3,
			DistanceKM:      30,
		},
	}
	loungeLat, loungeLng := 0.0, 0.15

	got := truncateRouteSegmentsAtLounge(segments, &loungeLat, &loungeLng)

	if len(got) != 2 {
		t.Fatalf("got %d route segments, want 2", len(got))
	}
	if got[0].DistanceKM != 10 {
		t.Errorf("first segment distance = %v, want 10", got[0].DistanceKM)
	}
	if got[1].DistanceKM < 9.99 || got[1].DistanceKM > 10.01 {
		t.Errorf("clipped segment distance = %v, want approximately 10", got[1].DistanceKM)
	}
	if got[1].EndLongitude < 0.1499 || got[1].EndLongitude > 0.1501 {
		t.Errorf("clipped segment end longitude = %v, want approximately 0.15", got[1].EndLongitude)
	}
}

func TestTruncateRouteSegmentsAtLoungeRequiresValidLoungeCoordinates(t *testing.T) {
	segments := []models.RouteSegment{{DistanceKM: 1}}

	if got := truncateRouteSegmentsAtLounge(segments, nil, nil); got != nil {
		t.Fatalf("got %v, want nil for missing lounge coordinates", got)
	}
}
