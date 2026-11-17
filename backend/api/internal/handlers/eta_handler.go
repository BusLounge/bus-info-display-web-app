package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"bus-schedule-lounge/internal/services"
)

type ETAHandler struct {
	etaService *services.ETAService
}

func NewETAHandler(db *sql.DB) *ETAHandler {
	return &ETAHandler{
		etaService: services.NewETAService(db),
	}
}

// ============================================================================
// API ENDPOINTS
// ============================================================================

// GetTripLoungeETAs calculates and returns ETA for all lounges on a trip
// GET /api/v1/eta/trip/{tripId}?lat={lat}&lng={lng}&speed={speed}
func (h *ETAHandler) GetTripLoungeETAs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tripIDStr := vars["tripId"]

	tripID, err := uuid.Parse(tripIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid trip ID")
		return
	}

	// Parse optional query parameters
	var lat, lng, speed *float64

	if latStr := r.URL.Query().Get("lat"); latStr != "" {
		latVal, err := strconv.ParseFloat(latStr, 64)
		if err == nil {
			lat = &latVal
		}
	}

	if lngStr := r.URL.Query().Get("lng"); lngStr != "" {
		lngVal, err := strconv.ParseFloat(lngStr, 64)
		if err == nil {
			lng = &lngVal
		}
	}

	if speedStr := r.URL.Query().Get("speed"); speedStr != "" {
		speedVal, err := strconv.ParseFloat(speedStr, 64)
		if err == nil {
			speed = &speedVal
		}
	}

	etas, err := h.etaService.CalculateTripLoungeETAs(r.Context(), tripID, lat, lng, speed)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to calculate ETAs: "+err.Error())
		return
	}

	// Save predictions for auditing
	go h.savePredictionsAsync(tripID, etas, lat, lng)

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"trip_id":       tripID,
		"calculated_at": time.Now().UTC(),
		"lounges":       etas,
		"total_lounges": len(etas),
	})
}

// GetAccuracyReport generates ETA accuracy report
// GET /api/v1/eta/accuracy?days={days}
func (h *ETAHandler) GetAccuracyReport(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days := 30 // Default to 30 days

	if daysStr != "" {
		if parsedDays, err := strconv.Atoi(daysStr); err == nil && parsedDays > 0 {
			days = parsedDays
		}
	}

	reports, err := h.etaService.GetAccuracyReport(r.Context(), days)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get accuracy report: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"period_days": days,
		"generated_at": time.Now().UTC(),
		"reports":     reports,
	})
}

// RecordActualArrival records when a bus actually arrives at a lounge
// POST /api/v1/eta/arrival
func (h *ETAHandler) RecordActualArrival(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TripID            string    `json:"trip_id"`
		LoungeID          string    `json:"lounge_id"`
		ActualArrivalTime time.Time `json:"actual_arrival_time"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	tripID, err := uuid.Parse(req.TripID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid trip ID")
		return
	}

	loungeID, err := uuid.Parse(req.LoungeID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid lounge ID")
		return
	}

	if err := h.etaService.RecordActualArrival(r.Context(), tripID, loungeID, req.ActualArrivalTime); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to record arrival: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Actual arrival time recorded successfully",
	})
}

// RecordSegmentPerformance records performance data for a completed segment
// POST /api/v1/eta/segment-performance
func (h *ETAHandler) RecordSegmentPerformance(w http.ResponseWriter, r *http.Request) {
	var req services.SegmentPerformance

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.etaService.RecordSegmentPerformance(r.Context(), req); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to record segment performance: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]string{
		"message": "Segment performance recorded successfully",
	})
}

// CreateTripContext creates a trip context for data collection
// POST /api/v1/eta/trip-context
func (h *ETAHandler) CreateTripContext(w http.ResponseWriter, r *http.Request) {
	var req services.TripContext

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	id, err := h.etaService.CreateTripContext(r.Context(), req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create trip context: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"message":         "Trip context created successfully",
		"trip_context_id": id,
	})
}

// RefreshStatistics manually triggers refresh of aggregate statistics
// POST /api/v1/eta/refresh-stats
func (h *ETAHandler) RefreshStatistics(w http.ResponseWriter, r *http.Request) {
	if err := h.etaService.RefreshAggregateStats(r.Context()); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to refresh statistics: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message":      "Statistics refreshed successfully",
		"refreshed_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (h *ETAHandler) savePredictionsAsync(tripID uuid.UUID, etas []services.LoungeETA, lat, lng *float64) {
	// This runs asynchronously to not block the response
	predictions := make([]services.ETAPrediction, 0, len(etas))

	for _, eta := range etas {
		predictions = append(predictions, services.ETAPrediction{
			ActiveTripID:         tripID,
			LoungeID:             &eta.LoungeID,
			PredictionType:       "lounge",
			PredictedArrivalTime: eta.EstimatedArrivalTime,
			CurrentLocationLat:   lat,
			CurrentLocationLng:   lng,
			DistanceRemainingKm:  &eta.DistanceRemainingKm,
			CalculationMethod:    eta.CalculationMethod,
			ConfidenceScore:      eta.ConfidenceScore,
			ContextData:          eta.ComponentDetails,
		})
	}

	// Use batch save for efficiency
	if err := h.etaService.SaveBatchPredictions(context.Background(), predictions); err != nil {
		// Log error but don't fail the request
		// TODO: Add proper logging
		println("Failed to save predictions:", err.Error())
	}
}

// ============================================================================
// ROUTE REGISTRATION
// ============================================================================

func (h *ETAHandler) RegisterRoutes(r *mux.Router) {
	// ETA calculation
	r.HandleFunc("/api/v1/eta/trip/{tripId}", h.GetTripLoungeETAs).Methods("GET")
	
	// Accuracy and reporting
	r.HandleFunc("/api/v1/eta/accuracy", h.GetAccuracyReport).Methods("GET")
	
	// Data collection
	r.HandleFunc("/api/v1/eta/arrival", h.RecordActualArrival).Methods("POST")
	r.HandleFunc("/api/v1/eta/segment-performance", h.RecordSegmentPerformance).Methods("POST")
	r.HandleFunc("/api/v1/eta/trip-context", h.CreateTripContext).Methods("POST")
	
	// Maintenance
	r.HandleFunc("/api/v1/eta/refresh-stats", h.RefreshStatistics).Methods("POST")
}
