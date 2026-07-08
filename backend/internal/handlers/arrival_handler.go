package handlers

import (
	"bus-schedule-lounge/internal/services"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type ArrivalHandler struct {
	service *services.ArrivalService
	cache   *responseCache
}

func NewArrivalHandler(service *services.ArrivalService) *ArrivalHandler {
	return &ArrivalHandler{
		service: service,
		cache:   newResponseCache(5 * time.Second),
	}
}

// GetAllLoungeArrivals handles GET /api/arrivals
func (h *ArrivalHandler) GetAllLoungeArrivals(w http.ResponseWriter, r *http.Request) {
	payload, err := h.cache.getOrSet("arrivals:all", func() ([]byte, error) {
		arrivals, err := h.service.GetAllLoungeArrivals()
		if err != nil {
			return nil, err
		}
		return json.Marshal(arrivals)
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}

// GetArrivalsByLoungeID handles GET /api/arrivals/lounge/{loungeId}
func (h *ArrivalHandler) GetArrivalsByLoungeID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	loungeID := vars["loungeId"]

	if loungeID == "" {
		http.Error(w, "Lounge ID is required", http.StatusBadRequest)
		return
	}

	payload, err := h.cache.getOrSet("arrivals:lounge:"+loungeID, func() ([]byte, error) {
		arrivals, err := h.service.GetArrivalsByLoungeID(loungeID)
		if err != nil {
			return nil, err
		}
		return json.Marshal(arrivals)
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}
