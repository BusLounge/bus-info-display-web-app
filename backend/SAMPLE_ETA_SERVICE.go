// Package eta provides ETA calculation services for bus lounges
// FILE: backend/internal/services/eta_service.go
package services

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"
)

// ===================================================================
// DATA STRUCTURES
// ===================================================================

// ETACalculator handles ETA calculations for trips
type ETACalculator struct {
	db *sql.DB
}

// NewETACalculator creates a new ETA calculator instance
func NewETACalculator(db *sql.DB) *ETACalculator {
	return &ETACalculator{db: db}
}

// Segment represents a route segment
type Segment struct {
	ID                       string
	MasterRouteID            string
	StartPointType           string
	StartPointID             string
	EndPointType             string
	EndPointID               string
	SegmentOrder             int
	StartLatitude            float64
	StartLongitude           float64
	EndLatitude              float64
	EndLongitude             float64
	DistanceKM               float64
	BaselineDurationMinutes  int
	BaselineSpeedKMH         float64
	RoadType                 string
	TrafficSensitivityFactor float64
}

// TripPosition represents current trip position
type TripPosition struct {
	TripID           string
	Latitude         float64
	Longitude        float64
	CurrentSpeedKMH  float64
	CurrentSegmentID *string
	DriverID         string
	BusID            string
}

// ETAResult represents ETA calculation result for a single point
type ETAResult struct {
	PointType           string // "lounge" or "stop"
	PointID             string
	PointName           string
	DistanceRemainingKM float64
	EstimatedArrival    time.Time
	EstimatedDeparture  *time.Time // Only for lounges
	ConfidenceScore     float64
	Delays              DelayBreakdown
}

// DelayBreakdown shows delay components
type DelayBreakdown struct {
	TrafficDelayMinutes  int
	WeatherImpactMinutes int
}

// TripETAResponse is the complete ETA response for a trip
type TripETAResponse struct {
	TripID              string
	RouteID             string
	CurrentPosition     TripPosition
	LastUpdated         time.Time
	Lounges             []ETAResult
	Stops               []ETAResult
	FinalDestinationETA time.Time
}

// ===================================================================
// CORE CALCULATION METHODS
// ===================================================================

// CalculateTripETA calculates ETA for all upcoming lounges and stops
func (ec *ETACalculator) CalculateTripETA(ctx context.Context, tripID string) (*TripETAResponse, error) {
	// 1. Get current trip position
	position, err := ec.getCurrentPosition(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip position: %w", err)
	}

	// 2. Get all remaining segments
	segments, err := ec.getRemainingSegments(ctx, position)
	if err != nil {
		return nil, fmt.Errorf("failed to get remaining segments: %w", err)
	}

	// 3. Calculate cumulative ETAs
	loungeETAs, stopETAs, finalETA, err := ec.calculateCumulativeETAs(ctx, position, segments)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate ETAs: %w", err)
	}

	return &TripETAResponse{
		TripID:              tripID,
		CurrentPosition:     *position,
		LastUpdated:         time.Now(),
		Lounges:             loungeETAs,
		Stops:               stopETAs,
		FinalDestinationETA: finalETA,
	}, nil
}

// CalculateLoungeETA calculates ETA for a specific lounge
func (ec *ETACalculator) CalculateLoungeETA(ctx context.Context, tripID, loungeID string) (*ETAResult, error) {
	// Get trip ETA and filter for specific lounge
	tripETA, err := ec.CalculateTripETA(ctx, tripID)
	if err != nil {
		return nil, err
	}

	for _, lounge := range tripETA.Lounges {
		if lounge.PointID == loungeID {
			return &lounge, nil
		}
	}

	return nil, fmt.Errorf("lounge %s not found in trip route", loungeID)
}

// ===================================================================
// SEGMENT DURATION CALCULATION
// ===================================================================

// calculateSegmentDuration calculates expected duration for a segment
func (ec *ETACalculator) calculateSegmentDuration(
	ctx context.Context,
	segment Segment,
	position TripPosition,
	currentTime time.Time,
) (float64, float64, error) {
	// 1. Get baseline duration
	baselineDuration := float64(segment.BaselineDurationMinutes)

	// 2. Calculate historical factor
	historicalFactor, sampleSize, err := ec.getHistoricalFactor(ctx, segment.ID, currentTime)
	if err != nil {
		// Fall back to baseline if no historical data
		historicalFactor = 1.0
		sampleSize = 0
	}

	// 3. Calculate context factor
	contextFactor := ec.calculateContextFactor(ctx, position, currentTime)

	// 4. Calculate real-time factor (only for current segment)
	realTimeFactor := 1.0
	if position.CurrentSegmentID != nil && *position.CurrentSegmentID == segment.ID {
		realTimeFactor = ec.calculateRealTimeFactor(position, segment)
	}

	// 5. Final duration calculation
	finalDuration := baselineDuration * historicalFactor * contextFactor * realTimeFactor

	// 6. Calculate confidence score
	confidence := ec.calculateConfidence(sampleSize, historicalFactor, position.CurrentSegmentID != nil)

	return finalDuration, confidence, nil
}

// getHistoricalFactor queries historical performance data
func (ec *ETACalculator) getHistoricalFactor(
	ctx context.Context,
	segmentID string,
	currentTime time.Time,
) (float64, int, error) {
	timeCategory := getTimeCategory(currentTime)
	dayOfWeek := currentTime.Weekday().String()

	query := `
		SELECT 
			AVG(actual_duration_minutes) as avg_duration,
			COUNT(*) as sample_size
		FROM segment_historical_performance
		WHERE route_segment_id = $1
			AND time_of_day_category = $2
			AND LOWER(day_of_week) = LOWER($3)
			AND trip_date >= CURRENT_DATE - INTERVAL '90 days'
	`

	var avgDuration sql.NullFloat64
	var sampleSize int

	err := ec.db.QueryRowContext(ctx, query, segmentID, timeCategory, dayOfWeek).
		Scan(&avgDuration, &sampleSize)
	if err != nil {
		return 1.0, 0, err
	}

	if !avgDuration.Valid || sampleSize == 0 {
		return 1.0, 0, nil
	}

	// Get baseline duration for this segment
	var baselineDuration float64
	err = ec.db.QueryRowContext(ctx,
		"SELECT baseline_duration_minutes FROM route_segments WHERE id = $1",
		segmentID).Scan(&baselineDuration)
	if err != nil {
		return 1.0, sampleSize, err
	}

	factor := avgDuration.Float64 / baselineDuration
	return factor, sampleSize, nil
}

// calculateContextFactor calculates multipliers based on context
func (ec *ETACalculator) calculateContextFactor(
	ctx context.Context,
	position TripPosition,
	currentTime time.Time,
) float64 {
	factor := 1.0

	// Driver experience factor
	driverFactor := ec.getDriverFactor(ctx, position.DriverID)
	factor *= driverFactor

	// Bus type factor
	busFactor := ec.getBusFactor(ctx, position.BusID)
	factor *= busFactor

	// Weather factor
	weatherFactor := ec.getWeatherFactor(ctx)
	factor *= weatherFactor

	// Time of day factor
	timeFactor := ec.getTimeFactor(ctx, currentTime)
	factor *= timeFactor

	return factor
}

// calculateRealTimeFactor adjusts based on current speed
func (ec *ETACalculator) calculateRealTimeFactor(
	position TripPosition,
	segment Segment,
) float64 {
	if position.CurrentSpeedKMH == 0 {
		return 1.0 // Avoid division by zero
	}

	// Calculate remaining distance in current segment
	remainingDistance := ec.calculateDistance(
		position.Latitude, position.Longitude,
		segment.EndLatitude, segment.EndLongitude,
	)

	// Time at current speed vs. baseline speed
	timeAtCurrentSpeed := remainingDistance / position.CurrentSpeedKMH * 60 // minutes
	timeAtBaselineSpeed := remainingDistance / segment.BaselineSpeedKMH * 60

	if timeAtBaselineSpeed == 0 {
		return 1.0
	}

	return timeAtCurrentSpeed / timeAtBaselineSpeed
}

// ===================================================================
// CUMULATIVE ETA CALCULATION
// ===================================================================

// calculateCumulativeETAs calculates ETAs for all points in sequence
func (ec *ETACalculator) calculateCumulativeETAs(
	ctx context.Context,
	position *TripPosition,
	segments []Segment,
) ([]ETAResult, []ETAResult, time.Time, error) {
	currentTime := time.Now()
	cumulativeTime := 0.0 // minutes
	loungeResults := []ETAResult{}
	stopResults := []ETAResult{}

	for _, segment := range segments {
		// Calculate segment duration
		duration, confidence, err := ec.calculateSegmentDuration(ctx, segment, *position, currentTime)
		if err != nil {
			return nil, nil, time.Time{}, err
		}

		cumulativeTime += duration

		// Add dwell time if applicable
		dwellTime := 0.0
		if segment.EndPointType == "lounge" {
			dwellTime = ec.getLoungeDwellTime(ctx, segment.EndPointID, currentTime)
		} else if segment.EndPointType == "stop" {
			dwellTime = 3.0 // Standard 3 minutes for major stops
		}

		arrivalTime := currentTime.Add(time.Duration(cumulativeTime) * time.Minute)

		// Create result based on point type
		if segment.EndPointType == "lounge" {
			departureTime := arrivalTime.Add(time.Duration(dwellTime) * time.Minute)
			loungeResults = append(loungeResults, ETAResult{
				PointType:           "lounge",
				PointID:             segment.EndPointID,
				PointName:           ec.getLoungeName(ctx, segment.EndPointID),
				DistanceRemainingKM: ec.calculateTotalRemainingDistance(segments, segment.SegmentOrder),
				EstimatedArrival:    arrivalTime,
				EstimatedDeparture:  &departureTime,
				ConfidenceScore:     confidence,
				Delays:              ec.calculateDelays(duration, float64(segment.BaselineDurationMinutes)),
			})
		} else if segment.EndPointType == "stop" {
			stopResults = append(stopResults, ETAResult{
				PointType:           "stop",
				PointID:             segment.EndPointID,
				PointName:           ec.getStopName(ctx, segment.EndPointID),
				DistanceRemainingKM: ec.calculateTotalRemainingDistance(segments, segment.SegmentOrder),
				EstimatedArrival:    arrivalTime,
				ConfidenceScore:     confidence,
			})
		}

		cumulativeTime += dwellTime
	}

	finalETA := currentTime.Add(time.Duration(cumulativeTime) * time.Minute)
	return loungeResults, stopResults, finalETA, nil
}

// ===================================================================
// HELPER FUNCTIONS
// ===================================================================

// getTimeCategory determines time of day category
func getTimeCategory(t time.Time) string {
	hour := t.Hour()
	if hour >= 5 && hour < 7 {
		return "early_morning"
	} else if hour >= 7 && hour < 10 {
		return "morning_peak"
	} else if hour >= 10 && hour < 16 {
		return "midday"
	} else if hour >= 16 && hour < 20 {
		return "evening_peak"
	}
	return "night"
}

// calculateDistance calculates distance between two lat/lng points (Haversine)
func (ec *ETACalculator) calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLng := (lng2 - lng1) * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// calculateConfidence calculates confidence score (0-100)
func (ec *ETACalculator) calculateConfidence(
	sampleSize int,
	historicalFactor float64,
	isCurrentSegment bool,
) float64 {
	confidence := 50.0 // Base confidence

	// Increase confidence based on sample size
	if sampleSize > 100 {
		confidence += 30
	} else if sampleSize > 50 {
		confidence += 20
	} else if sampleSize > 10 {
		confidence += 10
	}

	// Increase confidence if we have real-time data
	if isCurrentSegment {
		confidence += 15
	}

	// Decrease confidence if historical factor is extreme (unusual conditions)
	if historicalFactor > 1.5 || historicalFactor < 0.7 {
		confidence -= 10
	}

	// Cap between 0 and 100
	if confidence > 100 {
		confidence = 100
	}
	if confidence < 0 {
		confidence = 0
	}

	return confidence
}

// calculateDelays breaks down delay components
func (ec *ETACalculator) calculateDelays(actualDuration, baselineDuration float64) DelayBreakdown {
	totalDelay := actualDuration - baselineDuration
	if totalDelay <= 0 {
		return DelayBreakdown{0, 0}
	}

	// Simplified: assume 70% traffic, 30% weather
	trafficDelay := int(totalDelay * 0.7)
	weatherDelay := int(totalDelay * 0.3)

	return DelayBreakdown{
		TrafficDelayMinutes:  trafficDelay,
		WeatherImpactMinutes: weatherDelay,
	}
}

// ===================================================================
// DATABASE QUERY HELPERS
// ===================================================================

// getCurrentPosition gets current trip position
func (ec *ETACalculator) getCurrentPosition(ctx context.Context, tripID string) (*TripPosition, error) {
	query := `
		SELECT 
			id, current_latitude, current_longitude, 
			COALESCE(current_speed_kmh, 0), current_segment_id,
			driver_id, bus_id
		FROM active_trips
		WHERE id = $1
	`

	var position TripPosition
	var currentSegmentID sql.NullString

	err := ec.db.QueryRowContext(ctx, query, tripID).Scan(
		&position.TripID,
		&position.Latitude,
		&position.Longitude,
		&position.CurrentSpeedKMH,
		&currentSegmentID,
		&position.DriverID,
		&position.BusID,
	)

	if err != nil {
		return nil, err
	}

	if currentSegmentID.Valid {
		position.CurrentSegmentID = &currentSegmentID.String
	}

	return &position, nil
}

// getRemainingSegments gets all segments from current position to destination
func (ec *ETACalculator) getRemainingSegments(ctx context.Context, position *TripPosition) ([]Segment, error) {
	// Implementation would query route_segments table
	// and return only segments after current position
	// This is a placeholder - actual implementation needed
	return []Segment{}, nil
}

// Context factor getters (placeholders - implement with actual queries)
func (ec *ETACalculator) getDriverFactor(ctx context.Context, driverID string) float64 {
	// Query driver experience from bus_staff table
	// Experience > 5 years: 0.95
	// Experience 2-5 years: 1.0
	// Experience < 2 years: 1.1
	return 1.0
}

func (ec *ETACalculator) getBusFactor(ctx context.Context, busID string) float64 {
	// Query bus characteristics
	// AC bus: 1.0, Non-AC: 0.95
	return 1.0
}

func (ec *ETACalculator) getWeatherFactor(ctx context.Context) float64 {
	// Query weather API or manual weather setting
	// Clear: 1.0, Rain: 1.15, Heavy rain: 1.35
	return 1.0
}

func (ec *ETACalculator) getTimeFactor(ctx context.Context, currentTime time.Time) float64 {
	// Query eta_time_categories table
	category := getTimeCategory(currentTime)

	query := `
		SELECT multiplier 
		FROM eta_time_categories 
		WHERE category_name = $1 AND is_active = TRUE
	`

	var multiplier float64
	err := ec.db.QueryRowContext(ctx, query, category).Scan(&multiplier)
	if err != nil {
		return 1.0 // Default
	}

	return multiplier
}

func (ec *ETACalculator) getLoungeDwellTime(ctx context.Context, loungeID string, currentTime time.Time) float64 {
	// Query lounge_stop_metrics table
	// Return peak vs off-peak dwell time
	return 15.0 // Default 15 minutes
}

func (ec *ETACalculator) getLoungeName(ctx context.Context, loungeID string) string {
	var name string
	ec.db.QueryRowContext(ctx, "SELECT lounge_name FROM lounges WHERE id = $1", loungeID).Scan(&name)
	return name
}

func (ec *ETACalculator) getStopName(ctx context.Context, stopID string) string {
	var name string
	ec.db.QueryRowContext(ctx, "SELECT stop_name FROM master_route_stops WHERE id = $1", stopID).Scan(&name)
	return name
}

func (ec *ETACalculator) calculateTotalRemainingDistance(segments []Segment, currentOrder int) float64 {
	total := 0.0
	for _, seg := range segments {
		if seg.SegmentOrder >= currentOrder {
			total += seg.DistanceKM
		}
	}
	return total
}

// ===================================================================
// PREDICTION RECORDING (for ML improvement)
// ===================================================================

// RecordPrediction saves ETA prediction for future accuracy analysis
func (ec *ETACalculator) RecordPrediction(
	ctx context.Context,
	tripID, pointID, pointType string,
	predictedArrival time.Time,
	confidence float64,
	contextData map[string]interface{},
) error {
	query := `
		INSERT INTO eta_predictions (
			active_trip_id,
			lounge_id,
			stop_id,
			prediction_type,
			predicted_arrival_time,
			confidence_score,
			context_data
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	var loungeID, stopID *string
	if pointType == "lounge" {
		loungeID = &pointID
	} else {
		stopID = &pointID
	}

	_, err := ec.db.ExecContext(ctx, query,
		tripID, loungeID, stopID, pointType,
		predictedArrival, confidence, contextData)

	return err
}

// UpdateActualArrival updates prediction with actual arrival time
func (ec *ETACalculator) UpdateActualArrival(
	ctx context.Context,
	predictionID string,
	actualArrival time.Time,
) error {
	query := `
		UPDATE eta_predictions
		SET actual_arrival_time = $1
		WHERE id = $2
	`

	_, err := ec.db.ExecContext(ctx, query, actualArrival, predictionID)
	return err
}
