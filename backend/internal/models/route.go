package models

import "time"

type MasterRoute struct {
	ID                       string    `json:"id" db:"id"`
	RouteNumber              string    `json:"route_number" db:"route_number"`
	RouteName                string    `json:"route_name" db:"route_name"`
	OriginCity               string    `json:"origin_city" db:"origin_city"`
	DestinationCity          string    `json:"destination_city" db:"destination_city"`
	TotalDistanceKM          string    `json:"total_distance_km" db:"total_distance_km"`
	EstimatedDurationMinutes int       `json:"estimated_duration_minutes" db:"estimated_duration_minutes"`
	EncodedPolyline          *string   `json:"encoded_polyline" db:"encoded_polyline"`
	IsActive                 bool      `json:"is_active" db:"is_active"`
	CreatedAt                time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time `json:"updated_at" db:"updated_at"`
}
