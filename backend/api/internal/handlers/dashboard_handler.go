package handlers

import (
	"bus-schedule-lounge/internal/services"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type DashboardHandler struct {
	service *services.DashboardService
	cache   *responseCache
}

func NewDashboardHandler(service *services.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		service: service,
		cache:   newResponseCache(15 * time.Second),
	}
}

// GetDashboardData handles GET /api/dashboard
func (h *DashboardHandler) GetDashboardData(w http.ResponseWriter, r *http.Request) {
	log.Println("=== GetDashboardData called ===")

	payload, err := h.cache.getOrSet("dashboard:all", func() ([]byte, error) {
		data, err := h.service.GetDashboardData()
		if err != nil {
			return nil, err
		}
		return json.Marshal(data)
	})
	if err != nil {
		log.Printf("Error fetching dashboard data: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully fetched dashboard data")
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}
