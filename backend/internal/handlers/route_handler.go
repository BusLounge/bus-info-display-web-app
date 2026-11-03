package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"bus-schedule-lounge/internal/database"
	"bus-schedule-lounge/internal/services"
	wsupdates "bus-schedule-lounge/internal/websocket"

	"github.com/gorilla/mux"
)

type RouteHandler struct {
	routeService *services.RouteService
	routeRepo    *database.RouteRepository
	updatesHub   *wsupdates.Hub
}

func NewRouteHandler(
	routeService *services.RouteService,
	routeRepo *database.RouteRepository,
	updatesHub *wsupdates.Hub,
) *RouteHandler {
	return &RouteHandler{
		routeService: routeService,
		routeRepo:    routeRepo,
		updatesHub:   updatesHub,
	}
}

// GetAllRoutes handles GET /api/routes
func (h *RouteHandler) GetAllRoutes(w http.ResponseWriter, r *http.Request) {
	routes, err := h.routeService.GetAllRoutes(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(routes)
}

// GetActiveRoutesWithDetails handles GET /api/routes/active-with-details
func (h *RouteHandler) GetActiveRoutesWithDetails(w http.ResponseWriter, r *http.Request) {
	routes, err := h.routeService.GetActiveRoutesWithDetails(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(routes)
}

// GetRouteByID handles GET /api/routes/{id}
func (h *RouteHandler) GetRouteByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "route ID is required", http.StatusBadRequest)
		return
	}

	route, err := h.routeService.GetRouteByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(route)
}

// CreateRoute handles POST /api/routes
func (h *RouteHandler) CreateRoute(w http.ResponseWriter, r *http.Request) {
	var req services.CreateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	route, err := h.routeService.CreateRoute(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := h.updatesHub.BroadcastAll(wsupdates.ResourceSchedule); err != nil {
		log.Printf("failed to broadcast schedule update after route creation: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(route)
}

// UpdateRoute handles PUT /api/routes/{id}
func (h *RouteHandler) UpdateRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "route ID is required", http.StatusBadRequest)
		return
	}

	var req services.UpdateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	route, err := h.routeService.UpdateRoute(r.Context(), id, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	loungeIDs, err := h.routeRepo.GetLoungeIDsByMasterRouteID(r.Context(), id)
	if err != nil {
		log.Printf("failed to find lounges for updated route %q: %v", id, err)
	} else {
		for _, loungeID := range loungeIDs {
			if _, err := h.updatesHub.Broadcast(wsupdates.Notification{
				LoungeID: loungeID,
				Resource: "schedule",
			}); err != nil {
				log.Printf("failed to broadcast schedule update for lounge %q: %v", loungeID, err)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(route)
}

// DeleteRoute handles DELETE /api/routes/{id}
func (h *RouteHandler) DeleteRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "route ID is required", http.StatusBadRequest)
		return
	}

	err := h.routeService.DeleteRoute(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := h.updatesHub.BroadcastAll(wsupdates.ResourceSchedule); err != nil {
		log.Printf("failed to broadcast schedule update after route deletion: %v", err)
	}

	w.WriteHeader(http.StatusNoContent)
}
