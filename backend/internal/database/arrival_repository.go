package database

import (
	"bus-schedule-lounge/internal/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"
)

type ArrivalRepository struct {
	db *sql.DB
}

func NewArrivalRepository(db *sql.DB) *ArrivalRepository {
	return &ArrivalRepository{db: db}
}

// GetAllLoungeArrivals gets today's active trip arrivals for all lounges from actual database tables
func (r *ArrivalRepository) GetAllLoungeArrivals(
	serviceDateStart, serviceDateEnd time.Time,
) ([]models.ArrivalInfo, error) {
	query := `
		SELECT DISTINCT
			l.id as lounge_id,
			l.lounge_name,
			l.latitude as lounge_latitude,
			l.longitude as lounge_longitude,
			at.id as active_trip_id,
			mr.id as master_route_id,
			at.bus_id as bus_id,
			at.driver_id as driver_id,
			b.bus_number,
			mr.route_number,
			mr.origin_city,
			mr.destination_city,
			at.current_latitude,
			at.current_longitude,
			at.current_speed_kmh,
			at.last_location_update,
			COALESCE(
				CASE
					WHEN sb.arrival_time_offset_minutes IS NOT NULL AND sa.arrival_time_offset_minutes IS NOT NULL
						THEN ((sb.arrival_time_offset_minutes + sa.arrival_time_offset_minutes) / 2)
					WHEN sb.arrival_time_offset_minutes IS NOT NULL THEN sb.arrival_time_offset_minutes
					WHEN sa.arrival_time_offset_minutes IS NOT NULL THEN sa.arrival_time_offset_minutes
					ELSE NULL
				END,
				(st.estimated_duration_minutes / 2),
				240
			) as offset_minutes_calculated,
			-- Lounge ETA: schedule departure + midpoint offset between stop_before and stop_after
			st.departure_datetime + 
				(COALESCE(
					CASE
						WHEN sb.arrival_time_offset_minutes IS NOT NULL AND sa.arrival_time_offset_minutes IS NOT NULL
							THEN ((sb.arrival_time_offset_minutes + sa.arrival_time_offset_minutes) / 2)
						WHEN sb.arrival_time_offset_minutes IS NOT NULL THEN sb.arrival_time_offset_minutes
						WHEN sa.arrival_time_offset_minutes IS NOT NULL THEN sa.arrival_time_offset_minutes
						ELSE NULL
					END,
					(st.estimated_duration_minutes / 2),
					240
				) || ' minutes')::INTERVAL as estimated_arrival_time,
			at.actual_arrival_time,
				COALESCE(at.status, 'not_started') as status,
			st.departure_datetime,
			COALESCE((
				SELECT jsonb_agg(jsonb_build_object(
					'segmentOrder', rs.segment_order,
					'startLatitude', rs.start_latitude,
					'startLongitude', rs.start_longitude,
					'endLatitude', rs.end_latitude,
					'endLongitude', rs.end_longitude,
					'distanceKm', rs.distance_km,
					'baselineSpeedKmh', rs.baseline_speed_kmh,
					'roadType', rs.road_type
				) ORDER BY rs.segment_order)
				FROM route_segments rs
				WHERE rs.master_route_id = mr.id
			), '[]'::jsonb) as route_segments
		FROM lounges l
		JOIN lounge_routes lr ON l.id = lr.lounge_id
		JOIN master_routes mr ON lr.master_route_id = mr.id
		JOIN master_route_stops sb ON sb.id = lr.stop_before_id AND sb.master_route_id = lr.master_route_id
		JOIN master_route_stops sa ON sa.id = lr.stop_after_id AND sa.master_route_id = lr.master_route_id
		JOIN bus_owner_routes bor ON bor.master_route_id = mr.id
		JOIN scheduled_trips st ON st.bus_owner_route_id = bor.id
		JOIN active_trips at ON at.scheduled_trip_id = st.id
		JOIN buses b ON at.bus_id = b.id
		WHERE l.is_operational = true
			AND sb.stop_order < sa.stop_order
			AND st.departure_datetime >= $1::timestamp
			AND st.departure_datetime < $2::timestamp
			AND at.status IN ('in_transit', 'at_stop', 'approaching', 'scheduled', 'pre_departure', 'boarding', 'departed')
			AND st.departure_datetime IS NOT NULL
		ORDER BY l.lounge_name, estimated_arrival_time
		LIMIT 50`

	rows, err := r.db.Query(
		query,
		serviceDateStart.Format("2006-01-02 15:04:05.999999999"),
		serviceDateEnd.Format("2006-01-02 15:04:05.999999999"),
	)
	if err != nil {
		return nil, fmt.Errorf("error fetching arrivals: %w", err)
	}
	defer rows.Close()

	var arrivals []models.ArrivalInfo
	for rows.Next() {
		var arrival models.ArrivalInfo
		var routeSegmentsJSON []byte
		err := rows.Scan(
			&arrival.LoungeID,
			&arrival.LoungeName,
			&arrival.LoungeLat,
			&arrival.LoungeLng,
			&arrival.ActiveTripID,
			&arrival.MasterRouteID,
			&arrival.BusID,
			&arrival.DriverID,
			&arrival.BusNumber,
			&arrival.RouteNumber,
			&arrival.Origin,
			&arrival.Destination,
			&arrival.CurrentLat,
			&arrival.CurrentLng,
			&arrival.CurrentSpeedKmh,
			&arrival.LastLocationUpdate,
			&arrival.OffsetMinutes,
			&arrival.ETA,
			&arrival.ActualArrival,
			&arrival.Status,
			&arrival.DepartureTime,
			&routeSegmentsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning arrival: %w", err)
		}
		if err := json.Unmarshal(routeSegmentsJSON, &arrival.RouteSegments); err != nil {
			return nil, fmt.Errorf("error decoding route segments for route %s: %w", arrival.MasterRouteID, err)
		}
		arrival.RouteSegments = truncateRouteSegmentsAtLounge(
			arrival.RouteSegments,
			arrival.LoungeLat,
			arrival.LoungeLng,
		)
		arrivals = append(arrivals, arrival)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating arrivals: %w", err)
	}

	return arrivals, nil
}

// GetArrivalsByLoungeID gets today's active trip arrivals for a specific lounge from actual database tables
func (r *ArrivalRepository) GetArrivalsByLoungeID(
	loungeID string,
	serviceDateStart, serviceDateEnd time.Time,
) ([]models.ArrivalInfo, error) {
	log.Printf("[GetArrivalsByLoungeID] START params: loungeID=%v", loungeID)
	query := `
		WITH lounge_schedule AS (
		SELECT DISTINCT
			l.id as lounge_id,
			l.lounge_name,
			l.latitude as lounge_latitude,
			l.longitude as lounge_longitude,
			COALESCE(at.id::text, '') as active_trip_id,
			mr.id as master_route_id,
			COALESCE(at.bus_id::text, '') as bus_id,
			COALESCE(at.driver_id::text, '') as driver_id,
			COALESCE(b.bus_number, 'Bus TBA') as bus_number,
			mr.route_number,
			mr.origin_city,
			mr.destination_city,
			at.current_latitude,
			at.current_longitude,
			at.current_speed_kmh,
			at.last_location_update,
			COALESCE(
				CASE
					WHEN sb.arrival_time_offset_minutes IS NOT NULL AND sa.arrival_time_offset_minutes IS NOT NULL
						THEN ((sb.arrival_time_offset_minutes + sa.arrival_time_offset_minutes) / 2)
					WHEN sb.arrival_time_offset_minutes IS NOT NULL THEN sb.arrival_time_offset_minutes
					WHEN sa.arrival_time_offset_minutes IS NOT NULL THEN sa.arrival_time_offset_minutes
					ELSE NULL
				END,
				(st.estimated_duration_minutes / 2),
				240
			) as offset_minutes_calculated,
			-- Lounge ETA: schedule departure + midpoint offset between stop_before and stop_after
			st.departure_datetime + 
				(COALESCE(
					CASE
						WHEN sb.arrival_time_offset_minutes IS NOT NULL AND sa.arrival_time_offset_minutes IS NOT NULL
							THEN ((sb.arrival_time_offset_minutes + sa.arrival_time_offset_minutes) / 2)
						WHEN sb.arrival_time_offset_minutes IS NOT NULL THEN sb.arrival_time_offset_minutes
						WHEN sa.arrival_time_offset_minutes IS NOT NULL THEN sa.arrival_time_offset_minutes
						ELSE NULL
					END,
					(st.estimated_duration_minutes / 2),
					240
				) || ' minutes')::INTERVAL as estimated_arrival_time,
			at.actual_arrival_time,
				COALESCE(at.status, 'not_started') as status,
			st.departure_datetime,
			COALESCE((
				SELECT jsonb_agg(jsonb_build_object(
					'segmentOrder', rs.segment_order,
					'startLatitude', rs.start_latitude,
					'startLongitude', rs.start_longitude,
					'endLatitude', rs.end_latitude,
					'endLongitude', rs.end_longitude,
					'distanceKm', rs.distance_km,
					'baselineSpeedKmh', rs.baseline_speed_kmh,
					'roadType', rs.road_type
				) ORDER BY rs.segment_order)
				FROM route_segments rs
				WHERE rs.master_route_id = mr.id
			), '[]'::jsonb) as route_segments
		FROM lounges l
		JOIN lounge_routes lr ON l.id = lr.lounge_id
		JOIN master_routes mr ON lr.master_route_id = mr.id
		JOIN master_route_stops sb ON sb.id = lr.stop_before_id AND sb.master_route_id = lr.master_route_id
		JOIN master_route_stops sa ON sa.id = lr.stop_after_id AND sa.master_route_id = lr.master_route_id
		JOIN bus_owner_routes bor ON bor.master_route_id = mr.id
		JOIN scheduled_trips st ON st.bus_owner_route_id = bor.id
		LEFT JOIN active_trips at ON at.scheduled_trip_id = st.id
		LEFT JOIN buses b ON at.bus_id = b.id
		WHERE l.id = $1
			AND l.is_operational = true
			AND sb.stop_order < sa.stop_order
			AND st.departure_datetime >= $2::timestamp
			AND st.departure_datetime < $3::timestamp
			AND (at.id IS NULL OR at.status IN ('not_started', 'in_transit', 'at_stop'))
			AND st.departure_datetime IS NOT NULL
		)
		SELECT *
		FROM lounge_schedule
		ORDER BY estimated_arrival_time ASC
		LIMIT 10`

	rows, err := r.db.Query(
		query,
		loungeID,
		serviceDateStart.Format("2006-01-02 15:04:05.999999999"),
		serviceDateEnd.Format("2006-01-02 15:04:05.999999999"),
	)
	if err != nil {
		log.Printf("[GetArrivalsByLoungeID] QUERY ERROR: %v", err)
		return nil, fmt.Errorf("error fetching arrivals for lounge %s: %w", loungeID, err)
	}
	defer rows.Close()

	var arrivals []models.ArrivalInfo
	for rows.Next() {
		var arrival models.ArrivalInfo
		var routeSegmentsJSON []byte
		err := rows.Scan(
			&arrival.LoungeID,
			&arrival.LoungeName,
			&arrival.LoungeLat,
			&arrival.LoungeLng,
			&arrival.ActiveTripID,
			&arrival.MasterRouteID,
			&arrival.BusID,
			&arrival.DriverID,
			&arrival.BusNumber,
			&arrival.RouteNumber,
			&arrival.Origin,
			&arrival.Destination,
			&arrival.CurrentLat,
			&arrival.CurrentLng,
			&arrival.CurrentSpeedKmh,
			&arrival.LastLocationUpdate,
			&arrival.OffsetMinutes,
			&arrival.ETA,
			&arrival.ActualArrival,
			&arrival.Status,
			&arrival.DepartureTime,
			&routeSegmentsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning arrival: %w", err)
		}
		if err := json.Unmarshal(routeSegmentsJSON, &arrival.RouteSegments); err != nil {
			return nil, fmt.Errorf("error decoding route segments for route %s: %w", arrival.MasterRouteID, err)
		}
		arrival.RouteSegments = truncateRouteSegmentsAtLounge(
			arrival.RouteSegments,
			arrival.LoungeLat,
			arrival.LoungeLng,
		)
		arrivals = append(arrivals, arrival)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating arrivals: %w", err)
	}

	log.Printf("[GetArrivalsByLoungeID] RESULT: %d rows", len(arrivals))
	return arrivals, nil
}

func truncateRouteSegmentsAtLounge(
	segments []models.RouteSegment,
	loungeLat, loungeLng *float64,
) []models.RouteSegment {
	if len(segments) == 0 || loungeLat == nil || loungeLng == nil ||
		math.IsNaN(*loungeLat) || math.IsInf(*loungeLat, 0) ||
		math.IsNaN(*loungeLng) || math.IsInf(*loungeLng, 0) ||
		*loungeLat < -90 || *loungeLat > 90 || *loungeLng < -180 || *loungeLng > 180 {
		return nil
	}

	cosLatitude := math.Cos(*loungeLat * math.Pi / 180)
	targetX := *loungeLng * 111.32 * cosLatitude
	targetY := *loungeLat * 110.574
	bestDistance := math.Inf(1)
	bestIndex := -1
	bestFraction := 0.0
	bestX, bestY := 0.0, 0.0

	for index, segment := range segments {
		if segment.DistanceKM <= 0 || math.IsNaN(segment.DistanceKM) || math.IsInf(segment.DistanceKM, 0) {
			continue
		}

		startX := segment.StartLongitude * 111.32 * cosLatitude
		startY := segment.StartLatitude * 110.574
		endX := segment.EndLongitude * 111.32 * cosLatitude
		endY := segment.EndLatitude * 110.574
		deltaX := endX - startX
		deltaY := endY - startY
		lengthSquared := deltaX*deltaX + deltaY*deltaY
		if lengthSquared == 0 || math.IsNaN(lengthSquared) || math.IsInf(lengthSquared, 0) {
			continue
		}

		fraction := ((targetX-startX)*deltaX + (targetY-startY)*deltaY) / lengthSquared
		fraction = math.Max(0, math.Min(fraction, 1))
		projectedX := startX + fraction*deltaX
		projectedY := startY + fraction*deltaY
		distance := math.Hypot(targetX-projectedX, targetY-projectedY)
		if distance < bestDistance {
			bestDistance = distance
			bestIndex = index
			bestFraction = fraction
			bestX = projectedX
			bestY = projectedY
		}
	}

	if bestIndex < 0 {
		return nil
	}

	prefix := append([]models.RouteSegment(nil), segments[:bestIndex]...)
	if bestFraction > 0 {
		lastSegment := segments[bestIndex]
		lastSegment.DistanceKM *= bestFraction
		lastSegment.EndLongitude = bestX / (111.32 * cosLatitude)
		lastSegment.EndLatitude = bestY / 110.574
		prefix = append(prefix, lastSegment)
	}
	return prefix
}
