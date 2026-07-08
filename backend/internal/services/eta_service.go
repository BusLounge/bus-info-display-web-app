package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// MODELS
// ============================================================================

type LoungeETA struct {
	LoungeID              uuid.UUID              `json:"lounge_id"`
	LoungeName            string                 `json:"lounge_name"`
	SegmentOrder          int                    `json:"segment_order"`
	DistanceRemainingKm   float64                `json:"distance_remaining_km"`
	EstimatedArrivalTime  time.Time              `json:"estimated_arrival_time"`
	EstimatedDepartureTime time.Time             `json:"estimated_departure_time"`
	ConfidenceScore       float64                `json:"confidence_score"`
	CalculationMethod     string                 `json:"calculation_method"`
	ComponentDetails      map[string]interface{} `json:"component_details"`
}

type ETAPrediction struct {
	ID                   uuid.UUID              `json:"id"`
	ActiveTripID         uuid.UUID              `json:"active_trip_id"`
	LoungeID             *uuid.UUID             `json:"lounge_id,omitempty"`
	StopID               *uuid.UUID             `json:"stop_id,omitempty"`
	PredictionType       string                 `json:"prediction_type"`
	PredictedAt          time.Time              `json:"predicted_at"`
	PredictedArrivalTime time.Time              `json:"predicted_arrival_time"`
	CurrentLocationLat   *float64               `json:"current_location_lat,omitempty"`
	CurrentLocationLng   *float64               `json:"current_location_lng,omitempty"`
	DistanceRemainingKm  *float64               `json:"distance_remaining_km,omitempty"`
	ActualArrivalTime    *time.Time             `json:"actual_arrival_time,omitempty"`
	PredictionErrorMin   *float64               `json:"prediction_error_minutes,omitempty"`
	CalculationMethod    string                 `json:"calculation_method"`
	ConfidenceScore      float64                `json:"confidence_score"`
	ContextData          map[string]interface{} `json:"context_data,omitempty"`
}

type AccuracyReport struct {
	TotalPredictions        int     `json:"total_predictions"`
	PredictionsWithActuals  int     `json:"predictions_with_actuals"`
	AvgErrorMinutes         float64 `json:"avg_error_minutes"`
	MedianErrorMinutes      float64 `json:"median_error_minutes"`
	Within3MinutesPct       float64 `json:"within_3_minutes_pct"`
	Within5MinutesPct       float64 `json:"within_5_minutes_pct"`
	CalculationMethod       string  `json:"calculation_method"`
	ConfidenceRange         string  `json:"confidence_range"`
}

type SegmentPerformance struct {
	RouteSegmentID      uuid.UUID `json:"route_segment_id"`
	TripContextID       uuid.UUID `json:"trip_context_id"`
	SegmentStartTime    time.Time `json:"segment_start_time"`
	SegmentEndTime      time.Time `json:"segment_end_time"`
	ActualDurationMin   float64   `json:"actual_duration_minutes"`
	AverageSpeedKmh     *float64  `json:"average_speed_kmh,omitempty"`
	DurationVarianceMin *float64  `json:"duration_variance_minutes,omitempty"`
	TrafficLevel        *string   `json:"traffic_level,omitempty"`
	ActualDistanceKm    *float64  `json:"actual_distance_km,omitempty"`
	GPSAccuracyMeters   *int      `json:"gps_accuracy_meters,omitempty"`
	DataQualityScore    float64   `json:"data_quality_score"`
}

type TripContext struct {
	ID                   uuid.UUID  `json:"id"`
	ActiveTripID         uuid.UUID  `json:"active_trip_id"`
	ScheduledTripID      *uuid.UUID `json:"scheduled_trip_id,omitempty"`
	TripDate             string     `json:"trip_date"`
	DriverID             uuid.UUID  `json:"driver_id"`
	DriverExperienceYears *int      `json:"driver_experience_years,omitempty"`
	DriverRating         *float64   `json:"driver_rating,omitempty"`
	BusID                uuid.UUID  `json:"bus_id"`
	BusType              *string    `json:"bus_type,omitempty"`
	HasAC                bool       `json:"has_ac"`
	BusAgeYears          *float64   `json:"bus_age_years,omitempty"`
	DepartureTime        string     `json:"departure_time"`
	TimeOfDayCategory    string     `json:"time_of_day_category"`
	DayOfWeek            string     `json:"day_of_week"`
	IsWeekend            bool       `json:"is_weekend"`
	IsHoliday            bool       `json:"is_holiday"`
	WeatherCondition     string     `json:"weather_condition"`
	TemperatureCelsius   *int       `json:"temperature_celsius,omitempty"`
	TotalPassengers      int        `json:"total_passengers"`
	RouteID              uuid.UUID  `json:"route_id"`
}

// ============================================================================
// SERVICE
// ============================================================================

type ETAService struct {
	db *sql.DB
}

func NewETAService(db *sql.DB) *ETAService {
	return &ETAService{db: db}
}

// ============================================================================
// CALCULATE ETA
// ============================================================================

// CalculateTripLoungeETAs calculates estimated arrival times for all lounges on a trip
func (s *ETAService) CalculateTripLoungeETAs(ctx context.Context, tripID uuid.UUID, currentLat, currentLng, currentSpeedKmh *float64) ([]LoungeETA, error) {
	query := `
		SELECT 
			lounge_id,
			lounge_name,
			segment_order,
			distance_remaining_km,
			estimated_arrival_time,
			estimated_departure_time,
			confidence_score,
			calculation_method,
			component_details
		FROM calculate_trip_lounge_etas($1, $2, $3, $4)
	`

	rows, err := s.db.QueryContext(ctx, query, tripID, currentLat, currentLng, currentSpeedKmh)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate ETAs: %w", err)
	}
	defer rows.Close()

	var etas []LoungeETA
	for rows.Next() {
		var eta LoungeETA
		var componentDetailsJSON []byte

		err := rows.Scan(
			&eta.LoungeID,
			&eta.LoungeName,
			&eta.SegmentOrder,
			&eta.DistanceRemainingKm,
			&eta.EstimatedArrivalTime,
			&eta.EstimatedDepartureTime,
			&eta.ConfidenceScore,
			&eta.CalculationMethod,
			&componentDetailsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ETA row: %w", err)
		}

		// Parse component details JSON
		if len(componentDetailsJSON) > 0 {
			if err := json.Unmarshal(componentDetailsJSON, &eta.ComponentDetails); err != nil {
				// Log warning but don't fail
				eta.ComponentDetails = make(map[string]interface{})
			}
		}

		etas = append(etas, eta)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ETA rows: %w", err)
	}

	return etas, nil
}

// ============================================================================
// SAVE PREDICTION
// ============================================================================

// SaveETAPrediction saves an ETA prediction for auditing
func (s *ETAService) SaveETAPrediction(ctx context.Context, pred ETAPrediction) (uuid.UUID, error) {
	contextDataJSON, err := json.Marshal(pred.ContextData)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to marshal context data: %w", err)
	}

	query := `
		SELECT save_eta_prediction($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	var predictionID uuid.UUID
	err = s.db.QueryRowContext(
		ctx,
		query,
		pred.ActiveTripID,
		pred.LoungeID,
		pred.PredictedArrivalTime,
		pred.CurrentLocationLat,
		pred.CurrentLocationLng,
		pred.DistanceRemainingKm,
		pred.ConfidenceScore,
		pred.CalculationMethod,
		contextDataJSON,
	).Scan(&predictionID)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to save prediction: %w", err)
	}

	return predictionID, nil
}

// ============================================================================
// RECORD ACTUAL ARRIVAL
// ============================================================================

// RecordActualArrival updates the actual arrival time for accuracy tracking
func (s *ETAService) RecordActualArrival(ctx context.Context, tripID, loungeID uuid.UUID, actualArrivalTime time.Time) error {
	query := `SELECT record_actual_arrival($1, $2, $3)`

	_, err := s.db.ExecContext(ctx, query, tripID, loungeID, actualArrivalTime)
	if err != nil {
		return fmt.Errorf("failed to record actual arrival: %w", err)
	}

	return nil
}

// ============================================================================
// ACCURACY REPORTING
// ============================================================================

// GetAccuracyReport generates an accuracy report for the specified time period
func (s *ETAService) GetAccuracyReport(ctx context.Context, daysBack int) ([]AccuracyReport, error) {
	query := `
		SELECT 
			total_predictions,
			predictions_with_actuals,
			avg_error_minutes,
			median_error_minutes,
			within_3_minutes_pct,
			within_5_minutes_pct,
			calculation_method,
			confidence_range
		FROM get_eta_accuracy_report($1)
	`

	rows, err := s.db.QueryContext(ctx, query, daysBack)
	if err != nil {
		return nil, fmt.Errorf("failed to get accuracy report: %w", err)
	}
	defer rows.Close()

	var reports []AccuracyReport
	for rows.Next() {
		var report AccuracyReport
		err := rows.Scan(
			&report.TotalPredictions,
			&report.PredictionsWithActuals,
			&report.AvgErrorMinutes,
			&report.MedianErrorMinutes,
			&report.Within3MinutesPct,
			&report.Within5MinutesPct,
			&report.CalculationMethod,
			&report.ConfidenceRange,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan accuracy report row: %w", err)
		}

		reports = append(reports, report)
	}

	return reports, nil
}

// ============================================================================
// DATA COLLECTION
// ============================================================================

// CreateTripContext creates a trip context record
func (s *ETAService) CreateTripContext(ctx context.Context, tc TripContext) (uuid.UUID, error) {
	query := `
		INSERT INTO trip_contexts (
			active_trip_id,
			scheduled_trip_id,
			trip_date,
			driver_id,
			driver_experience_years,
			driver_rating,
			bus_id,
			bus_type,
			has_ac,
			bus_age_years,
			departure_time,
			time_of_day_category,
			day_of_week,
			is_weekend,
			is_holiday,
			weather_condition,
			temperature_celsius,
			total_passengers,
			route_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id
	`

	var id uuid.UUID
	err := s.db.QueryRowContext(
		ctx,
		query,
		tc.ActiveTripID,
		tc.ScheduledTripID,
		tc.TripDate,
		tc.DriverID,
		tc.DriverExperienceYears,
		tc.DriverRating,
		tc.BusID,
		tc.BusType,
		tc.HasAC,
		tc.BusAgeYears,
		tc.DepartureTime,
		tc.TimeOfDayCategory,
		tc.DayOfWeek,
		tc.IsWeekend,
		tc.IsHoliday,
		tc.WeatherCondition,
		tc.TemperatureCelsius,
		tc.TotalPassengers,
		tc.RouteID,
	).Scan(&id)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create trip context: %w", err)
	}

	return id, nil
}

// RecordSegmentPerformance records segment performance data
func (s *ETAService) RecordSegmentPerformance(ctx context.Context, sp SegmentPerformance) error {
	query := `
		INSERT INTO segment_performance_facts (
			route_segment_id,
			trip_context_id,
			segment_start_time,
			segment_end_time,
			actual_duration_minutes,
			average_speed_kmh,
			duration_variance_minutes,
			traffic_level,
			actual_distance_km,
			gps_accuracy_meters,
			data_quality_score
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := s.db.ExecContext(
		ctx,
		query,
		sp.RouteSegmentID,
		sp.TripContextID,
		sp.SegmentStartTime,
		sp.SegmentEndTime,
		sp.ActualDurationMin,
		sp.AverageSpeedKmh,
		sp.DurationVarianceMin,
		sp.TrafficLevel,
		sp.ActualDistanceKm,
		sp.GPSAccuracyMeters,
		sp.DataQualityScore,
	)

	if err != nil {
		return fmt.Errorf("failed to record segment performance: %w", err)
	}

	return nil
}

// ============================================================================
// REFRESH FUNCTIONS
// ============================================================================

// RefreshAggregateStats refreshes pre-computed aggregate statistics
func (s *ETAService) RefreshAggregateStats(ctx context.Context) error {
	queries := []string{
		"REFRESH MATERIALIZED VIEW segment_performance_with_context",
		"SELECT refresh_segment_aggregate_stats()",
		"SELECT refresh_driver_performance_profiles()",
		"SELECT refresh_bus_performance_profiles()",
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, query := range queries {
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query %s: %w", query, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// GetTimeOfDayCategory determines the time category for a given time
func (s *ETAService) GetTimeOfDayCategory(ctx context.Context, t time.Time) (string, error) {
	query := `SELECT get_time_of_day_category($1::TIME)`

	var category string
	err := s.db.QueryRowContext(ctx, query, t.Format("15:04:05")).Scan(&category)
	if err != nil {
		return "", fmt.Errorf("failed to get time of day category: %w", err)
	}

	return category, nil
}

// GetDayOfWeek returns the day of week as a lowercase string
func GetDayOfWeek(t time.Time) string {
	days := []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}
	return days[t.Weekday()]
}

// ============================================================================
// BATCH OPERATIONS
// ============================================================================

// SaveBatchPredictions saves multiple predictions efficiently
func (s *ETAService) SaveBatchPredictions(ctx context.Context, predictions []ETAPrediction) error {
	if len(predictions) == 0 {
		return nil
	}

	txn, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer txn.Rollback()

	stmt, err := txn.PrepareContext(ctx, `
		SELECT save_eta_prediction($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, pred := range predictions {
		contextDataJSON, err := json.Marshal(pred.ContextData)
		if err != nil {
			return fmt.Errorf("failed to marshal context data: %w", err)
		}

		_, err = stmt.ExecContext(
			ctx,
			pred.ActiveTripID,
			pred.LoungeID,
			pred.PredictedArrivalTime,
			pred.CurrentLocationLat,
			pred.CurrentLocationLng,
			pred.DistanceRemainingKm,
			pred.ConfidenceScore,
			pred.CalculationMethod,
			contextDataJSON,
		)
		if err != nil {
			return fmt.Errorf("failed to save prediction: %w", err)
		}
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
