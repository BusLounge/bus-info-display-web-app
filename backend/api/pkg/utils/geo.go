package utils

import (
	"math"
)

const (
	// EarthRadiusKm is the Earth's radius in kilometers
	EarthRadiusKm = 6371.0

	// DefaultFallbackSpeedKmh is the default speed used when actual speed is unavailable or zero
	DefaultFallbackSpeedKmh = 40.0

	// MinSpeedThresholdKmh is the minimum speed threshold (below this, use fallback)
	MinSpeedThresholdKmh = 5.0
)

// HaversineDistance calculates the distance between two geographic coordinates using the Haversine formula
// Returns distance in kilometers
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// Convert degrees to radians
	lat1Rad := degreesToRadians(lat1)
	lon1Rad := degreesToRadians(lon1)
	lat2Rad := degreesToRadians(lat2)
	lon2Rad := degreesToRadians(lon2)

	// Differences
	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	// Haversine formula
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Distance in kilometers
	distance := EarthRadiusKm * c

	return distance
}

// CalculateETAMinutes calculates ETA in minutes based on distance and speed
// distance: in kilometers
// speedKmh: speed in km/h
// Returns ETA in minutes
func CalculateETAMinutes(distance float64, speedKmh float64) float64 {
	// If speed is too low or zero, use fallback speed
	if speedKmh < MinSpeedThresholdKmh {
		speedKmh = DefaultFallbackSpeedKmh
	}

	// ETA (minutes) = distance (km) / speed (km/h) * 60
	etaMinutes := (distance / speedKmh) * 60.0

	return etaMinutes
}

// CalculateAdjustedETA applies smoothing adjustment to ETA
// previousETAMinutes: historical offset from route data
// currentETAMinutes: real-time calculated ETA
// Returns adjusted ETA in minutes
func CalculateAdjustedETA(previousETAMinutes float64, currentETAMinutes float64) float64 {
	// ETA = previous ETA + (current ETA / 2)
	// This helps stabilize ETA using historical route timing
	adjustedETA := previousETAMinutes + (currentETAMinutes / 2.0)

	return adjustedETA
}

// degreesToRadians converts degrees to radians
func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}

// ValidateCoordinates checks if latitude and longitude are valid
func ValidateCoordinates(lat, lon float64) bool {
	return lat >= -90.0 && lat <= 90.0 && lon >= -180.0 && lon <= 180.0
}
