package database

import (
	"bus-schedule-lounge/internal/models"
	"database/sql"
	"fmt"
)

type ArrivalRepository struct {
	db *sql.DB
}

func NewArrivalRepository(db *sql.DB) *ArrivalRepository {
	return &ArrivalRepository{db: db}
}

// GetAllLoungeArrivals gets today's active trip arrivals for all lounges from actual database tables
func (r *ArrivalRepository) GetAllLoungeArrivals() ([]models.ArrivalInfo, error) {
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
			at.status,
			st.departure_datetime
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
			AND DATE(st.departure_datetime) = CURRENT_DATE
			AND at.status IN ('in_transit', 'at_stop', 'approaching', 'scheduled', 'pre_departure', 'boarding', 'departed')
			AND st.departure_datetime IS NOT NULL
		ORDER BY l.lounge_name, estimated_arrival_time
		LIMIT 50`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error fetching arrivals: %w", err)
	}
	defer rows.Close()

	var arrivals []models.ArrivalInfo
	for rows.Next() {
		var arrival models.ArrivalInfo
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
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning arrival: %w", err)
		}
		arrivals = append(arrivals, arrival)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating arrivals: %w", err)
	}

	return arrivals, nil
}

// GetArrivalsByLoungeID gets today's active trip arrivals for a specific lounge from actual database tables
func (r *ArrivalRepository) GetArrivalsByLoungeID(loungeID string) ([]models.ArrivalInfo, error) {
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
			at.status,
			st.departure_datetime
		FROM lounges l
		JOIN lounge_routes lr ON l.id = lr.lounge_id
		JOIN master_routes mr ON lr.master_route_id = mr.id
		JOIN master_route_stops sb ON sb.id = lr.stop_before_id AND sb.master_route_id = lr.master_route_id
		JOIN master_route_stops sa ON sa.id = lr.stop_after_id AND sa.master_route_id = lr.master_route_id
		JOIN bus_owner_routes bor ON bor.master_route_id = mr.id
		JOIN scheduled_trips st ON st.bus_owner_route_id = bor.id
		JOIN active_trips at ON at.scheduled_trip_id = st.id
		JOIN buses b ON at.bus_id = b.id
		WHERE l.id = $1
			AND l.is_operational = true
			AND sb.stop_order < sa.stop_order
			AND DATE(st.departure_datetime) = CURRENT_DATE
			AND at.status IN ('in_transit', 'at_stop', 'approaching', 'scheduled', 'pre_departure', 'boarding', 'departed')
			AND st.departure_datetime IS NOT NULL
		ORDER BY estimated_arrival_time
		LIMIT 10`

	rows, err := r.db.Query(query, loungeID)
	if err != nil {
		return nil, fmt.Errorf("error fetching arrivals for lounge %s: %w", loungeID, err)
	}
	defer rows.Close()

	var arrivals []models.ArrivalInfo
	for rows.Next() {
		var arrival models.ArrivalInfo
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
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning arrival: %w", err)
		}
		arrivals = append(arrivals, arrival)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating arrivals: %w", err)
	}

	return arrivals, nil
}
