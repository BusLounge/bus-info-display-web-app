package models

import "time"

// ArrivalInfo represents bus arrival information for a lounge
type ArrivalInfo struct {
	LoungeID             string     `json:"loungeId" db:"lounge_id"`
	LoungeName           string     `json:"loungeName" db:"lounge_name"`
	LoungeLat            *float64   `json:"loungeLatitude" db:"lounge_latitude"`
	LoungeLng            *float64   `json:"loungeLongitude" db:"lounge_longitude"`
	ActiveTripID         string     `json:"activeTripId" db:"active_trip_id"`
	MasterRouteID        string     `json:"masterRouteId" db:"master_route_id"`
	BusID                string     `json:"busId" db:"bus_id"`
	DriverID             string     `json:"driverId" db:"driver_id"`
	BusNumber            string     `json:"busNo" db:"bus_number"`
	RouteNumber          string     `json:"routeNumber" db:"route_number"`
	Origin               string     `json:"origin" db:"origin_city"`
	Destination          string     `json:"destination" db:"destination_city"`
	CurrentLat           *float64   `json:"currentLatitude" db:"current_latitude"`
	CurrentLng           *float64   `json:"currentLongitude" db:"current_longitude"`
	CurrentSpeedKmh      *float64   `json:"currentSpeedKmh" db:"current_speed_kmh"`
	LastLocationUpdate   *time.Time `json:"lastLocationUpdate" db:"last_location_update"`
	OffsetMinutes        *int       `json:"offsetMinutes" db:"arrival_time_offset_minutes"`
	ETA                  *time.Time `json:"eta" db:"estimated_arrival_time"`
	ActualArrival        *time.Time `json:"actualArrival" db:"actual_arrival_time"`
	DepartureTime        *time.Time `json:"departureTime" db:"departure_datetime"`
	Status               string     `json:"status" db:"status"`
	Remarks              string     `json:"remarks"`
	TimeDisplay          string     `json:"time"`
	DistanceKm           float64    `json:"distanceKm"`
	CalculatedETAMinutes float64    `json:"calculatedETAMinutes"`
}

// LoungeArrivalResponse represents arrivals grouped by lounge
type LoungeArrivalResponse struct {
	LoungeID   string        `json:"loungeId"`
	LoungeName string        `json:"loungeName"`
	Arrivals   []ArrivalInfo `json:"arrivals"`
}
