package models

import "time"

// DepartureInfo represents bus departure information for a lounge
type DepartureInfo struct {
	LoungeID               string     `json:"loungeId" db:"lounge_id"`
	LoungeName             string     `json:"loungeName" db:"lounge_name"`
	LoungeLat              *float64   `json:"loungeLatitude" db:"lounge_latitude"`
	LoungeLng              *float64   `json:"loungeLongitude" db:"lounge_longitude"`
	ActiveTripID           string     `json:"activeTripId" db:"active_trip_id"`
	BusNumber              string     `json:"busNo" db:"bus_number"`
	RouteNumber            string     `json:"routeNumber" db:"route_number"`
	Origin                 string     `json:"origin" db:"origin_city"`
	Destination            string     `json:"destination" db:"destination_city"`
	CurrentLat             *float64   `json:"currentLatitude" db:"current_latitude"`
	CurrentLng             *float64   `json:"currentLongitude" db:"current_longitude"`
	ScheduledDepartureTime *time.Time `json:"scheduledDeparture" db:"departure_datetime"`
	ActualDeparture        *time.Time `json:"actualDeparture" db:"actual_departure_time"`
	ArrivalETA             *time.Time `json:"arrivalEta" db:"arrival_eta"`
	DepartureETA           *time.Time `json:"departureEta" db:"departure_eta"`
	StopoverMinutes        *int       `json:"stopoverMinutes" db:"stopover_minutes"`
	OffsetMinutes          *int       `json:"offsetMinutes" db:"offset_minutes_calculated"`
	Status                 string     `json:"status" db:"status"`
	Remarks                string     `json:"remarks"`
	TimeDisplay            string     `json:"time"`
	DepartureTime          *time.Time `json:"departureTime" db:"trip_departure_datetime"`
}

// LoungeDepartureResponse represents departures grouped by lounge
type LoungeDepartureResponse struct {
	LoungeID   string          `json:"loungeId"`
	LoungeName string          `json:"loungeName"`
	Departures []DepartureInfo `json:"departures"`
}
