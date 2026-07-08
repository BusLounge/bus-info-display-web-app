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

	return route, nil
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
