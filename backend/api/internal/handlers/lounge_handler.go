package handlers

import (
	"bus-schedule-lounge/internal/services"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type LoungeHandler struct {
	service *services.LoungeService
}

func NewLoungeHandler(service *services.LoungeService) *LoungeHandler {
	return &LoungeHandler{service: service}
}

// GetAllLounges handles GET /api/lounges
func (h *LoungeHandler) GetAllLounges(w http.ResponseWriter, r *http.Request) {
	lounges, err := h.service.GetAllLounges()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lounges)
}

// GetLoungeByID handles GET /api/lounges/{id}
func (h *LoungeHandler) GetLoungeByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	lounge, err := h.service.GetLoungeByID(id)
	if err != nil {
		if err.Error() == "lounge not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lounge)
}

// ValidateLoungeRouteSegments handles GET /api/lounges/{id}/route-segment-validation
func (h *LoungeHandler) ValidateLoungeRouteSegments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if id == "" {
		http.Error(w, "Lounge ID is required", http.StatusBadRequest)
		return
	}

	result, err := h.service.ValidateLoungeRouteSegments(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
