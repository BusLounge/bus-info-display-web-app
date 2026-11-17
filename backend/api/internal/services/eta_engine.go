package services

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client for the optional Python ETA engine (backend/eta_engine).
//
// The engine is off unless ETA_ENGINE_ENABLED=true: it cannot make predictions
// until its trained model files exist. When it is off or fails, the API uses its
// own estimates.

type ETARequest struct {
	MasterRouteID     string `json:"master_route_id"`
	LoungeID          string `json:"lounge_id"`
	DepartureDatetime string `json:"departure_datetime"`
	DriverID          string `json:"driver_id"`
	BusID             string `json:"bus_id"`
}

type BatchETARequest struct {
	Requests []ETARequest `json:"requests"`
}

type ETAResponseData struct {
	MasterRouteID string `json:"master_route_id"`
	LoungeID      string `json:"lounge_id"`
	ETA           string `json:"eta"`
	ETD           string `json:"etd"`
}

type BatchETAResponseItem struct {
	Status string          `json:"status"`
	Data   ETAResponseData `json:"data"`
}

var etaEngineClient = &http.Client{Timeout: 3 * time.Second}

// etaEngineEnabled reports whether ETA_ENGINE_ENABLED is set to true.
func etaEngineEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ETA_ENGINE_ENABLED"))) {
	case "true", "1", "yes":
		return true
	}
	return false
}

// etaEngineURL returns the base URL of the Python ETA engine. Set
// ETA_ENGINE_URL when the engine does not run on the same host as the API.
func etaEngineURL() string {
	if url := strings.TrimRight(strings.TrimSpace(os.Getenv("ETA_ENGINE_URL")), "/"); url != "" {
		return url
	}
	return "http://localhost:8000"
}

// predictWithETAEngine asks the engine for arrival times. It returns one entry
// per request, nil where the engine gave no usable prediction, or nil when the
// engine is off or unreachable.
func predictWithETAEngine(requests []ETARequest) []*time.Time {
	if !etaEngineEnabled() || len(requests) == 0 {
		return nil
	}
	body, err := json.Marshal(BatchETARequest{Requests: requests})
	if err != nil {
		return nil
	}
	resp, err := etaEngineClient.Post(etaEngineURL()+"/api/v1/predict_batch", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var items []BatchETAResponseItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil
	}

	predictions := make([]*time.Time, len(requests))
	for i, item := range items {
		if i >= len(predictions) || item.Status != "success" {
			continue
		}
		if eta, err := time.Parse(time.RFC3339, item.Data.ETA); err == nil {
			predictions[i] = &eta
		}
	}
	return predictions
}
