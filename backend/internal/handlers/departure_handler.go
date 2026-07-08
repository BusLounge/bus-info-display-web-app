package handlers

import (
	"bus-schedule-lounge/internal/services"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type DepartureHandler struct {
	service *services.DepartureService
	cache   *responseCache
}

func NewDepartureHandler(service *services.DepartureService) *DepartureHandler {
	return &DepartureHandler{
		service: service,
		cache:   newResponseCache(5 * time.Second),
	}
}

// GetAllLoungeDepartures handles GET /api/departures
func (h *DepartureHandler) GetAllLoungeDepartures(w http.ResponseWriter, r *http.Request) {
	payload, err := h.cache.getOrSet("departures:all", func() ([]byte, error) {
		departures, err := h.service.GetAllLoungeDepartures()
		if err != nil {
			return nil, err
		}
		return json.Marshal(departures)
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}

// GetDeparturesByLoungeID handles GET /api/departures/lounge/{loungeId}
func (h *DepartureHandler) GetDeparturesByLoungeID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	loungeID := vars["loungeId"]

	if loungeID == "" {
		http.Error(w, "Lounge ID is required", http.StatusBadRequest)
		return
	}

	payload, err := h.cache.getOrSet("departures:lounge:"+loungeID, func() ([]byte, error) {
		departures, err := h.service.GetDeparturesByLoungeID(loungeID)
		if err != nil {
			return nil, err
		}
		return json.Marshal(departures)
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}
