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
	Segments                 []RouteSegment `json:"segments,omitempty"`
}

type RouteSegment struct {
	ID                       string  `json:"id" db:"id"`
	MasterRouteID            string  `json:"masterRouteId" db:"master_route_id"`
	SegmentOrder             int     `json:"segmentOrder" db:"segment_order"`
	StartLatitude            float64 `json:"startLatitude" db:"start_latitude"`
	StartLongitude           float64 `json:"startLongitude" db:"start_longitude"`
	EndLatitude              float64 `json:"endLatitude" db:"end_latitude"`
	EndLongitude             float64 `json:"endLongitude" db:"end_longitude"`
	DistanceKM               float64 `json:"distanceKm" db:"distance_km"`
	BaselineDurationMinutes  int     `json:"baselineDurationMinutes" db:"baseline_duration_minutes"`
	BaselineSpeedKMH         float64 `json:"baselineSpeedKmh" db:"baseline_speed_kmh"`
	RoadType                 string  `json:"roadType" db:"road_type"`
	TrafficSensitivityFactor float64 `json:"trafficSensitivityFactor" db:"traffic_sensitivity_factor"`
}

type RouteLounge struct {
	ID           string   `json:"id"`
	LoungeName   string   `json:"loungeName"`
	Address      *string  `json:"address,omitempty"`
	District     *string  `json:"district,omitempty"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
	StopBeforeID *string  `json:"stopBeforeId,omitempty"`
	StopAfterID  *string  `json:"stopAfterId,omitempty"`
}

type ActiveRouteDetails struct {
	MasterRoute
	Segments []RouteSegment `json:"segments"`
	Lounges  []RouteLounge  `json:"lounges"`
}
