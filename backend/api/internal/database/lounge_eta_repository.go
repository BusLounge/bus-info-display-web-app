package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"bus-schedule-lounge/internal/models"
)

// LoungeETARepository loads the data needed to estimate when buses reach a lounge.
type LoungeETARepository struct {
	db *sql.DB
}

func NewLoungeETARepository(db *sql.DB) *LoungeETARepository {
	return &LoungeETARepository{db: db}
}

// LoungeTripRow is one scheduled trip that passes a lounge.
type LoungeTripRow struct {
	ScheduledTripID string
	LoungeID        string
	LoungeName      string
	LoungeLat       *float64
	LoungeLng       *float64

	MasterRouteID        string
	RouteNumber          string
	Origin               string
	Destination          string
	RouteDurationMinutes *float64

	BusCompanyName      string
	DepartureTime       time.Time
	TripDurationMinutes *float64

	StopBeforeID string
	StopAfterID  string

	// Active trip, written by the conductor app (empty when the trip has not been started).
	ActiveTripID       string
	ActiveTripStatus   string
	BusID              string
	DriverID           string
	BusNumber          string
	CurrentLat         *float64
	CurrentLng         *float64
	CurrentSpeedKmh    *float64
	LastLocationUpdate *time.Time
}

// RouteStop is one stop of a master route.
type RouteStop struct {
	ID            string
	Order         int
	Lat           *float64
	Lng           *float64
	OffsetMinutes *float64
}

// RouteGeometry is a master route's road segments (in order) and stops.
type RouteGeometry struct {
	Segments []models.RouteSegment
	Stops    []RouteStop
}

// GetLoungeTrips returns the trips that pass a lounge and depart from the start of
// their route between departedFrom and departedTo. Only trips that:
//   - are published by the operator and not cancelled,
//   - run in the route's own direction (UP), the direction the lounge is defined in,
//   - stop at both stops either side of the lounge (when the operator lists its stops),
//   - have not been completed or cancelled by the conductor.
func (r *LoungeETARepository) GetLoungeTrips(
	ctx context.Context,
	loungeID string,
	departedFrom, departedTo time.Time,
) ([]LoungeTripRow, error) {
	query := `
		SELECT
			st.id::text,
			l.id::text,
			l.lounge_name,
			l.latitude,
			l.longitude,
			mr.id::text,
			COALESCE(mr.route_number, ''),
			COALESCE(mr.origin_city, ''),
			COALESCE(mr.destination_city, ''),
			mr.estimated_duration_minutes,
			COALESCE(bo.company_name, ''),
			st.departure_datetime,
			st.estimated_duration_minutes,
			lr.stop_before_id::text,
			lr.stop_after_id::text,
			COALESCE(at.id::text, ''),
			COALESCE(at.status, ''),
			COALESCE(at.bus_id::text, ''),
			COALESCE(at.driver_id::text, ''),
			COALESCE(b.bus_number, ''),
			at.current_latitude,
			at.current_longitude,
			at.current_speed_kmh,
			at.last_location_update
		FROM lounges l
		JOIN lounge_routes lr ON lr.lounge_id = l.id
		JOIN master_route_stops sb ON sb.id = lr.stop_before_id AND sb.master_route_id = lr.master_route_id
		JOIN master_route_stops sa ON sa.id = lr.stop_after_id AND sa.master_route_id = lr.master_route_id
		JOIN master_routes mr ON mr.id = lr.master_route_id
		JOIN bus_owner_routes bor ON bor.master_route_id = mr.id
		LEFT JOIN bus_owners bo ON bo.id = bor.bus_owner_id
		JOIN scheduled_trips st ON st.bus_owner_route_id = bor.id
		LEFT JOIN LATERAL (
			SELECT a.*
			FROM active_trips a
			WHERE a.scheduled_trip_id = st.id
			ORDER BY a.updated_at DESC NULLS LAST
			LIMIT 1
		) at ON true
		LEFT JOIN buses b ON b.id = at.bus_id
		WHERE l.id = $1::uuid
			AND l.is_operational = true
			AND sb.stop_order < sa.stop_order
			AND COALESCE(bor.direction, 'UP') = 'UP'
			AND (
				bor.selected_stop_ids IS NULL
				OR cardinality(bor.selected_stop_ids) = 0
				OR (lr.stop_before_id = ANY(bor.selected_stop_ids) AND lr.stop_after_id = ANY(bor.selected_stop_ids))
			)
			AND st.status != 'cancelled'
			AND st.ever_published = true
			AND st.departure_datetime IS NOT NULL
			AND st.departure_datetime >= $2::timestamptz
			AND st.departure_datetime < $3::timestamptz
			AND (at.id IS NULL OR at.status IN ('not_started', 'in_transit', 'at_stop'))
		ORDER BY st.departure_datetime`

	rows, err := r.db.QueryContext(ctx, query, loungeID, departedFrom, departedTo)
	if err != nil {
		return nil, fmt.Errorf("fetch trips for lounge %s: %w", loungeID, err)
	}
	defer rows.Close()

	var trips []LoungeTripRow
	seen := make(map[string]bool)
	for rows.Next() {
		var (
			trip                        LoungeTripRow
			loungeLat, loungeLng        sql.NullFloat64
			routeDuration, tripDuration sql.NullFloat64
			currentLat, currentLng      sql.NullFloat64
			currentSpeed                sql.NullFloat64
			lastLocationUpdate          sql.NullTime
		)
		if err := rows.Scan(
			&trip.ScheduledTripID,
			&trip.LoungeID,
			&trip.LoungeName,
			&loungeLat,
			&loungeLng,
			&trip.MasterRouteID,
			&trip.RouteNumber,
			&trip.Origin,
			&trip.Destination,
			&routeDuration,
			&trip.BusCompanyName,
			&trip.DepartureTime,
			&tripDuration,
			&trip.StopBeforeID,
			&trip.StopAfterID,
			&trip.ActiveTripID,
			&trip.ActiveTripStatus,
			&trip.BusID,
			&trip.DriverID,
			&trip.BusNumber,
			&currentLat,
			&currentLng,
			&currentSpeed,
			&lastLocationUpdate,
		); err != nil {
			return nil, fmt.Errorf("scan lounge trip: %w", err)
		}

		// A lounge can be linked to the same route more than once; keep one row per trip.
		if seen[trip.ScheduledTripID] {
			continue
		}
		seen[trip.ScheduledTripID] = true

		trip.LoungeLat = nullFloatPtr(loungeLat)
		trip.LoungeLng = nullFloatPtr(loungeLng)
		trip.RouteDurationMinutes = nullFloatPtr(routeDuration)
		trip.TripDurationMinutes = nullFloatPtr(tripDuration)
		trip.CurrentLat = nullFloatPtr(currentLat)
		trip.CurrentLng = nullFloatPtr(currentLng)
		trip.CurrentSpeedKmh = nullFloatPtr(currentSpeed)
		if lastLocationUpdate.Valid {
			value := lastLocationUpdate.Time
			trip.LastLocationUpdate = &value
		}
		trips = append(trips, trip)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read lounge trips: %w", err)
	}
	return trips, nil
}

// GetRouteGeometry returns the road segments (in order) and stops of a master route.
func (r *LoungeETARepository) GetRouteGeometry(ctx context.Context, masterRouteID string) (RouteGeometry, error) {
	var geometry RouteGeometry

	segmentRows, err := r.db.QueryContext(ctx, `
		SELECT
			segment_order,
			COALESCE(start_latitude, 0),
			COALESCE(start_longitude, 0),
			COALESCE(end_latitude, 0),
			COALESCE(end_longitude, 0),
			COALESCE(distance_km, 0),
			COALESCE(baseline_speed_kmh, 0),
			COALESCE(road_type, '')
		FROM route_segments
		WHERE master_route_id = $1::uuid
		ORDER BY segment_order`, masterRouteID)
	if err != nil {
		return geometry, fmt.Errorf("fetch route segments for %s: %w", masterRouteID, err)
	}
	defer segmentRows.Close()
	for segmentRows.Next() {
		segment := models.RouteSegment{MasterRouteID: masterRouteID}
		if err := segmentRows.Scan(
			&segment.SegmentOrder,
			&segment.StartLatitude,
			&segment.StartLongitude,
			&segment.EndLatitude,
			&segment.EndLongitude,
			&segment.DistanceKM,
			&segment.BaselineSpeedKMH,
			&segment.RoadType,
		); err != nil {
			return geometry, fmt.Errorf("scan route segment: %w", err)
		}
		geometry.Segments = append(geometry.Segments, segment)
	}
	if err := segmentRows.Err(); err != nil {
		return geometry, fmt.Errorf("read route segments: %w", err)
	}

	stopRows, err := r.db.QueryContext(ctx, `
		SELECT id::text, stop_order, latitude, longitude, arrival_time_offset_minutes
		FROM master_route_stops
		WHERE master_route_id = $1::uuid
		ORDER BY stop_order`, masterRouteID)
	if err != nil {
		return geometry, fmt.Errorf("fetch route stops for %s: %w", masterRouteID, err)
	}
	defer stopRows.Close()
	for stopRows.Next() {
		var (
			stop          RouteStop
			lat, lng, off sql.NullFloat64
		)
		if err := stopRows.Scan(&stop.ID, &stop.Order, &lat, &lng, &off); err != nil {
			return geometry, fmt.Errorf("scan route stop: %w", err)
		}
		stop.Lat = nullFloatPtr(lat)
		stop.Lng = nullFloatPtr(lng)
		stop.OffsetMinutes = nullFloatPtr(off)
		geometry.Stops = append(geometry.Stops, stop)
	}
	if err := stopRows.Err(); err != nil {
		return geometry, fmt.Errorf("read route stops: %w", err)
	}

	return geometry, nil
}

func nullFloatPtr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	result := value.Float64
	return &result
}
