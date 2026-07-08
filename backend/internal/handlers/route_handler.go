package handlers

import (
	"encoding/json"
	"net/http"

	"bus-schedule-lounge/internal/services"
)

type RouteHandler struct {
	routeService *services.RouteService
}

func NewRouteHandler(routeService *services.RouteService) *RouteHandler {
	return &RouteHandler{
		routeService: routeService,
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

// GetRouteByID handles GET /api/routes/{id}
func (h *RouteHandler) GetRouteByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(route)
}

// UpdateRoute handles PUT /api/routes/{id}
func (h *RouteHandler) UpdateRoute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(route)
}

// DeleteRoute handles DELETE /api/routes/{id}
func (h *RouteHandler) DeleteRoute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "route ID is required", http.StatusBadRequest)
		return
	}

	err := h.routeService.DeleteRoute(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
