package database

import (
	"context"
	"database/sql"
	"fmt"

	"bus-schedule-lounge/internal/models"
)

type RouteRepository struct {
	db *sql.DB
}

func NewRouteRepository(db *sql.DB) *RouteRepository {
	return &RouteRepository{db: db}
}

func (r *RouteRepository) GetAll(ctx context.Context) ([]*models.MasterRoute, error) {
	query := `
		SELECT id, route_number, route_name, origin_city, destination_city,
		       total_distance_km, estimated_duration_minutes, encoded_polyline,
		       is_active, created_at, updated_at
		FROM master_routes
		ORDER BY route_number
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query routes: %w", err)
	}
	defer rows.Close()

	var routes []*models.MasterRoute
	for rows.Next() {
		route := &models.MasterRoute{}
		var totalDistance sql.NullString
		var estimatedDuration sql.NullInt32
		err := rows.Scan(
			&route.ID,
			&route.RouteNumber,
			&route.RouteName,
			&route.OriginCity,
			&route.DestinationCity,
			&totalDistance,
			&estimatedDuration,
			&route.EncodedPolyline,
			&route.IsActive,
			&route.CreatedAt,
			&route.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan route: %w", err)
		}

		if totalDistance.Valid {
			route.TotalDistanceKM = totalDistance.String
		} else {
			route.TotalDistanceKM = ""
		}

		if estimatedDuration.Valid {
			route.EstimatedDurationMinutes = int(estimatedDuration.Int32)
		} else {
			route.EstimatedDurationMinutes = 0
		}

		routes = append(routes, route)
	}

	return routes, nil
}

func (r *RouteRepository) GetActiveWithDetails(ctx context.Context) ([]*models.ActiveRouteDetails, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			mr.id, mr.route_number, mr.route_name, mr.origin_city, mr.destination_city,
			mr.total_distance_km, mr.estimated_duration_minutes, mr.encoded_polyline,
			mr.is_active, mr.created_at, mr.updated_at,
			rs.id, rs.master_route_id, rs.segment_order,
			rs.start_latitude, rs.start_longitude, rs.end_latitude, rs.end_longitude,
			rs.distance_km, rs.baseline_duration_minutes, rs.baseline_speed_kmh,
			rs.road_type, rs.traffic_sensitivity_factor,
			l.id, l.lounge_name, l.address, lo.district, l.latitude, l.longitude,
			lr.stop_before_id, lr.stop_after_id
		FROM master_routes mr
		LEFT JOIN route_segments rs ON rs.master_route_id = mr.id
		LEFT JOIN lounge_routes lr ON lr.master_route_id = mr.id
		LEFT JOIN lounges l ON l.id = lr.lounge_id
		LEFT JOIN lounge_owners lo ON lo.id = l.lounge_owner_id
		WHERE mr.is_active = TRUE
		ORDER BY mr.route_number, rs.segment_order, l.lounge_name`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query active route details: %w", err)
	}
	defer rows.Close()

	routeByID := make(map[string]*models.ActiveRouteDetails)
	segmentSeen := make(map[string]map[int]bool)
	loungeSeen := make(map[string]map[string]bool)
	orderedRoutes := make([]*models.ActiveRouteDetails, 0)

	for rows.Next() {
		var scannedRoute models.ActiveRouteDetails
		var totalDistance sql.NullString
		var estimatedDuration sql.NullInt32
		var encodedPolyline sql.NullString
		var segment models.RouteSegment
		var segmentID, segmentRouteID sql.NullString
		var segmentOrder sql.NullInt32
		var startLat, startLng, endLat, endLng, distance, baselineSpeed, trafficFactor sql.NullFloat64
		var baselineDuration sql.NullInt32
		var roadType sql.NullString
		var loungeID, loungeName, loungeAddress, loungeDistrict, stopBeforeID, stopAfterID sql.NullString
		var loungeLatitude, loungeLongitude sql.NullFloat64

		if err := rows.Scan(
			&scannedRoute.ID, &scannedRoute.RouteNumber, &scannedRoute.RouteName, &scannedRoute.OriginCity, &scannedRoute.DestinationCity,
			&totalDistance, &estimatedDuration, &encodedPolyline, &scannedRoute.IsActive, &scannedRoute.CreatedAt, &scannedRoute.UpdatedAt,
			&segmentID, &segmentRouteID, &segmentOrder, &startLat, &startLng, &endLat, &endLng,
			&distance, &baselineDuration, &baselineSpeed, &roadType, &trafficFactor,
			&loungeID, &loungeName, &loungeAddress, &loungeDistrict, &loungeLatitude, &loungeLongitude,
			&stopBeforeID, &stopAfterID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan active route details: %w", err)
		}

		route, ok := routeByID[scannedRoute.ID]
		if !ok {
			route = &scannedRoute
			if totalDistance.Valid {
				route.TotalDistanceKM = totalDistance.String
			}
			if estimatedDuration.Valid {
				route.EstimatedDurationMinutes = int(estimatedDuration.Int32)
			}
			if encodedPolyline.Valid {
				route.EncodedPolyline = &encodedPolyline.String
			}
			route.Segments = make([]models.RouteSegment, 0)
			route.Lounges = make([]models.RouteLounge, 0)
			routeByID[route.ID] = route
			segmentSeen[route.ID] = make(map[int]bool)
			loungeSeen[route.ID] = make(map[string]bool)
			orderedRoutes = append(orderedRoutes, route)
		}

		if segmentID.Valid && !segmentSeen[route.ID][int(segmentOrder.Int32)] {
			segment.ID = segmentID.String
			segment.MasterRouteID = segmentRouteID.String
			segment.SegmentOrder = int(segmentOrder.Int32)
			segment.StartLatitude = startLat.Float64
			segment.StartLongitude = startLng.Float64
			segment.EndLatitude = endLat.Float64
			segment.EndLongitude = endLng.Float64
			segment.DistanceKM = distance.Float64
			segment.BaselineDurationMinutes = int(baselineDuration.Int32)
			segment.BaselineSpeedKMH = baselineSpeed.Float64
			segment.RoadType = roadType.String
			segment.TrafficSensitivityFactor = trafficFactor.Float64
			route.Segments = append(route.Segments, segment)
			segmentSeen[route.ID][int(segmentOrder.Int32)] = true
		}

		if loungeID.Valid && !loungeSeen[route.ID][loungeID.String] {
			lounge := models.RouteLounge{ID: loungeID.String, LoungeName: loungeName.String}
			if loungeAddress.Valid { lounge.Address = &loungeAddress.String }
			if loungeDistrict.Valid { lounge.District = &loungeDistrict.String }
			if loungeLatitude.Valid { lounge.Latitude = &loungeLatitude.Float64 }
			if loungeLongitude.Valid { lounge.Longitude = &loungeLongitude.Float64 }
			if stopBeforeID.Valid { lounge.StopBeforeID = &stopBeforeID.String }
			if stopAfterID.Valid { lounge.StopAfterID = &stopAfterID.String }
			route.Lounges = append(route.Lounges, lounge)
			loungeSeen[route.ID][loungeID.String] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate active route details: %w", err)
	}
	return orderedRoutes, nil
}

func (r *RouteRepository) GetByID(ctx context.Context, id string) (*models.MasterRoute, error) {
	query := `
		SELECT id, route_number, route_name, origin_city, destination_city,
		       total_distance_km, estimated_duration_minutes, encoded_polyline,
		       is_active, created_at, updated_at
		FROM master_routes
		WHERE id = $1
	`

	route := &models.MasterRoute{}
	var totalDistance sql.NullString
	var estimatedDuration sql.NullInt32
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&route.ID,
		&route.RouteNumber,
		&route.RouteName,
		&route.OriginCity,
		&route.DestinationCity,
		&totalDistance,
		&estimatedDuration,
		&route.EncodedPolyline,
		&route.IsActive,
		&route.CreatedAt,
		&route.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("route not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get route: %w", err)
	}

	if totalDistance.Valid {
		route.TotalDistanceKM = totalDistance.String
	} else {
		route.TotalDistanceKM = ""
	}

	if estimatedDuration.Valid {
		route.EstimatedDurationMinutes = int(estimatedDuration.Int32)
	} else {
		route.EstimatedDurationMinutes = 0
	}

	segments, err := r.getSegments(ctx, id)
	if err != nil {
		return nil, err
	}
	route.Segments = segments

	return route, nil
}

func (r *RouteRepository) getSegments(ctx context.Context, routeID string) ([]models.RouteSegment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, master_route_id, segment_order,
			start_latitude, start_longitude, end_latitude, end_longitude,
			distance_km, baseline_duration_minutes, baseline_speed_kmh,
			road_type, traffic_sensitivity_factor
		FROM route_segments
		WHERE master_route_id = $1
		ORDER BY segment_order`, routeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get route segments: %w", err)
	}
	defer rows.Close()

	segments := make([]models.RouteSegment, 0)
	for rows.Next() {
		var segment models.RouteSegment
		if err := rows.Scan(
			&segment.ID, &segment.MasterRouteID, &segment.SegmentOrder,
			&segment.StartLatitude, &segment.StartLongitude,
			&segment.EndLatitude, &segment.EndLongitude,
			&segment.DistanceKM, &segment.BaselineDurationMinutes,
			&segment.BaselineSpeedKMH, &segment.RoadType,
			&segment.TrafficSensitivityFactor,
		); err != nil {
			return nil, fmt.Errorf("failed to scan route segment: %w", err)
		}
		segments = append(segments, segment)
	}
	return segments, rows.Err()
}

func (r *RouteRepository) CreateWithSegments(ctx context.Context, route *models.MasterRoute, segments []models.RouteSegment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return err }
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx, `
		INSERT INTO master_routes (
			route_number, route_name, origin_city, destination_city,
			total_distance_km, estimated_duration_minutes, encoded_polyline, is_active
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, created_at, updated_at`,
		route.RouteNumber, route.RouteName, route.OriginCity, route.DestinationCity,
		route.TotalDistanceKM, route.EstimatedDurationMinutes, route.EncodedPolyline,
		route.IsActive,
	).Scan(&route.ID, &route.CreatedAt, &route.UpdatedAt)
	if err != nil { return fmt.Errorf("failed to create route: %w", err) }
	if err := insertSegments(ctx, tx, route.ID, segments); err != nil { return err }
	route.Segments = segments
	return tx.Commit()
}

func (r *RouteRepository) UpdateWithSegments(ctx context.Context, route *models.MasterRoute, segments []models.RouteSegment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return err }
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx, `
		UPDATE master_routes SET route_number=$1, route_name=$2, origin_city=$3,
			destination_city=$4, total_distance_km=$5, estimated_duration_minutes=$6,
			encoded_polyline=$7, is_active=$8, updated_at=NOW()
		WHERE id=$9 RETURNING updated_at`,
		route.RouteNumber, route.RouteName, route.OriginCity, route.DestinationCity,
		route.TotalDistanceKM, route.EstimatedDurationMinutes, route.EncodedPolyline,
		route.IsActive, route.ID,
	).Scan(&route.UpdatedAt)
	if err != nil { return fmt.Errorf("failed to update route: %w", err) }
	if _, err := tx.ExecContext(ctx, `DELETE FROM route_segments WHERE master_route_id=$1`, route.ID); err != nil {
		return fmt.Errorf("failed to replace route segments: %w", err)
	}
	if err := insertSegments(ctx, tx, route.ID, segments); err != nil { return err }
	route.Segments = segments
	return tx.Commit()
}

func insertSegments(ctx context.Context, tx *sql.Tx, routeID string, segments []models.RouteSegment) error {
	for index, segment := range segments {
		startType := "stop"
		endType := "stop"
		if index == 0 { startType = "origin" }
		if index == len(segments)-1 { endType = "destination" }
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO route_segments (
				master_route_id, start_point_type, end_point_type, segment_order,
				start_latitude, start_longitude, end_latitude, end_longitude,
				distance_km, baseline_duration_minutes, baseline_speed_kmh,
				road_type, traffic_sensitivity_factor
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			routeID, startType, endType, segment.SegmentOrder,
			segment.StartLatitude, segment.StartLongitude,
			segment.EndLatitude, segment.EndLongitude, segment.DistanceKM,
			segment.BaselineDurationMinutes, segment.BaselineSpeedKMH,
			segment.RoadType, segment.TrafficSensitivityFactor,
		); err != nil {
			return fmt.Errorf("failed to insert route segment %d: %w", segment.SegmentOrder, err)
		}
	}
	return nil
}

func (r *RouteRepository) Create(ctx context.Context, route *models.MasterRoute) error {
	query := `
		INSERT INTO master_routes (
			route_number, route_name, origin_city, destination_city,
			total_distance_km, estimated_duration_minutes, encoded_polyline, is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		route.RouteNumber,
		route.RouteName,
		route.OriginCity,
		route.DestinationCity,
		route.TotalDistanceKM,
		route.EstimatedDurationMinutes,
		route.EncodedPolyline,
		route.IsActive,
	).Scan(&route.ID, &route.CreatedAt, &route.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create route: %w", err)
	}

	return nil
}

func (r *RouteRepository) Update(ctx context.Context, route *models.MasterRoute) error {
	query := `
		UPDATE master_routes
		SET route_number = $1,
		    route_name = $2,
		    origin_city = $3,
		    destination_city = $4,
		    total_distance_km = $5,
		    estimated_duration_minutes = $6,
		    encoded_polyline = $7,
		    is_active = $8,
		    updated_at = NOW()
		WHERE id = $9
		RETURNING updated_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		route.RouteNumber,
		route.RouteName,
		route.OriginCity,
		route.DestinationCity,
		route.TotalDistanceKM,
		route.EstimatedDurationMinutes,
		route.EncodedPolyline,
		route.IsActive,
		route.ID,
	).Scan(&route.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update route: %w", err)
	}

	return nil
}

func (r *RouteRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM master_routes WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("route not found")
	}

	return nil
}
