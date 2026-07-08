package database

import (
	"bus-schedule-lounge/internal/models"
	"database/sql"
	"fmt"
)

type DepartureRepository struct {
	db *sql.DB
}

func NewDepartureRepository(db *sql.DB) *DepartureRepository {
	return &DepartureRepository{db: db}
}

// GetAllLoungeDepartures gets today's active trip departures for all lounges from actual database tables
func (r *DepartureRepository) GetAllLoungeDepartures() ([]models.DepartureInfo, error) {
	query := `
		SELECT DISTINCT
			l.id as lounge_id,
			l.lounge_name,
			l.latitude as lounge_latitude,
			l.longitude as lounge_longitude,
			at.id as active_trip_id,
			b.bus_number,
			mr.route_number,
			mr.origin_city,
			mr.destination_city,
			at.current_latitude,
			at.current_longitude,
			st.departure_datetime,
			at.actual_departure_time,
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
			-- Arrival ETA at lounge
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
				) || ' minutes')::INTERVAL as arrival_eta,
			-- Departure ETA: arrival ETA + fixed stopover
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
				) + 20 || ' minutes')::INTERVAL as departure_eta,
		20 as stopover_minutes,
			at.status,
			st.departure_datetime as trip_departure_datetime
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
			AND at.status IN ('scheduled', 'pre_departure', 'boarding', 'departed', 'in_transit')
			AND st.departure_datetime IS NOT NULL
		ORDER BY l.lounge_name, departure_eta
		LIMIT 50`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error fetching departures: %w", err)
	}
	defer rows.Close()

	var departures []models.DepartureInfo
	for rows.Next() {
		var departure models.DepartureInfo
		err := rows.Scan(
			&departure.LoungeID,
			&departure.LoungeName,
			&departure.LoungeLat,
			&departure.LoungeLng,
			&departure.ActiveTripID,
			&departure.BusNumber,
			&departure.RouteNumber,
			&departure.Origin,
			&departure.Destination,
			&departure.CurrentLat,
			&departure.CurrentLng,
			&departure.ScheduledDepartureTime,
			&departure.ActualDeparture,
			&departure.OffsetMinutes,
			&departure.ArrivalETA,
			&departure.DepartureETA,
			&departure.StopoverMinutes,
			&departure.Status,
			&departure.DepartureTime,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning departure: %w", err)
		}
		departures = append(departures, departure)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating departures: %w", err)
	}

	return departures, nil
}

// GetDeparturesByLoungeID gets today's active trip departures for a specific lounge from actual database tables
func (r *DepartureRepository) GetDeparturesByLoungeID(loungeID string) ([]models.DepartureInfo, error) {
	query := `
		SELECT DISTINCT
			l.id as lounge_id,
			l.lounge_name,
			l.latitude as lounge_latitude,
			l.longitude as lounge_longitude,
			at.id as active_trip_id,
			b.bus_number,
			mr.route_number,
			mr.origin_city,
			mr.destination_city,
			at.current_latitude,
			at.current_longitude,
			st.departure_datetime,
			at.actual_departure_time,
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
			-- Arrival ETA at lounge
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
				) || ' minutes')::INTERVAL as arrival_eta,
			-- Departure ETA: arrival ETA + fixed stopover
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
				) + 20 || ' minutes')::INTERVAL as departure_eta,
			20 as stopover_minutes,
			at.status,
			st.departure_datetime as trip_departure_datetime
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
			AND at.status IN ('scheduled', 'pre_departure', 'boarding', 'departed', 'in_transit')
			AND st.departure_datetime IS NOT NULL
		ORDER BY departure_eta
		LIMIT 10`

	rows, err := r.db.Query(query, loungeID)
	if err != nil {
		return nil, fmt.Errorf("error fetching departures for lounge: %w", err)
	}
	defer rows.Close()

	var departures []models.DepartureInfo
	for rows.Next() {
		var departure models.DepartureInfo
		err := rows.Scan(
			&departure.LoungeID,
			&departure.LoungeName,
			&departure.LoungeLat,
			&departure.LoungeLng,
			&departure.ActiveTripID,
			&departure.BusNumber,
			&departure.RouteNumber,
			&departure.Origin,
			&departure.Destination,
			&departure.CurrentLat,
			&departure.CurrentLng,
			&departure.ScheduledDepartureTime,
			&departure.ActualDeparture,
			&departure.OffsetMinutes,
			&departure.ArrivalETA,
			&departure.DepartureETA,
			&departure.StopoverMinutes,
			&departure.Status,
			&departure.DepartureTime,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning departure: %w", err)
		}
		departures = append(departures, departure)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating departures: %w", err)
	}

	return departures, nil
}
