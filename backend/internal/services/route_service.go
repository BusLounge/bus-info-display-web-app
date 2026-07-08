package services

import (
	"context"
	"fmt"

	"bus-schedule-lounge/internal/database"
	"bus-schedule-lounge/internal/models"
)

type RouteService struct {
	routeRepo *database.RouteRepository
}

func NewRouteService(routeRepo *database.RouteRepository) *RouteService {
	return &RouteService{
		routeRepo: routeRepo,
	}
}

type CreateRouteRequest struct {
	RouteNumber              string  `json:"route_number"`
	RouteName                string  `json:"route_name"`
	OriginCity               string  `json:"origin_city"`
	DestinationCity          string  `json:"destination_city"`
	TotalDistanceKM          string  `json:"total_distance_km"`
	EstimatedDurationMinutes int     `json:"estimated_duration_minutes"`
	EncodedPolyline          *string `json:"encoded_polyline"`
	IsActive                 bool    `json:"is_active"`
}

type UpdateRouteRequest struct {
	RouteNumber              *string `json:"route_number,omitempty"`
	RouteName                *string `json:"route_name,omitempty"`
	OriginCity               *string `json:"origin_city,omitempty"`
	DestinationCity          *string `json:"destination_city,omitempty"`
	TotalDistanceKM          *string `json:"total_distance_km,omitempty"`
	EstimatedDurationMinutes *int    `json:"estimated_duration_minutes,omitempty"`
	EncodedPolyline          *string `json:"encoded_polyline,omitempty"`
	IsActive                 *bool   `json:"is_active,omitempty"`
}

func (s *RouteService) GetAllRoutes(ctx context.Context) ([]*models.MasterRoute, error) {
	return s.routeRepo.GetAll(ctx)
}

func (s *RouteService) GetRouteByID(ctx context.Context, id string) (*models.MasterRoute, error) {
	route, err := s.routeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("route not found: %w", err)
	}
	return route, nil
}

func (s *RouteService) CreateRoute(ctx context.Context, req *CreateRouteRequest) (*models.MasterRoute, error) {
	route := &models.MasterRoute{
		RouteNumber:              req.RouteNumber,
		RouteName:                req.RouteName,
		OriginCity:               req.OriginCity,
		DestinationCity:          req.DestinationCity,
		TotalDistanceKM:          req.TotalDistanceKM,
		EstimatedDurationMinutes: req.EstimatedDurationMinutes,
		EncodedPolyline:          req.EncodedPolyline,
		IsActive:                 req.IsActive,
	}

	err := s.routeRepo.Create(ctx, route)
	if err != nil {
		return nil, fmt.Errorf("failed to create route: %w", err)
	}

	return route, nil
}

func (s *RouteService) UpdateRoute(ctx context.Context, id string, req *UpdateRouteRequest) (*models.MasterRoute, error) {
	route, err := s.routeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("route not found: %w", err)
	}

	// Update fields if provided
	if req.RouteNumber != nil {
		route.RouteNumber = *req.RouteNumber
	}
	if req.RouteName != nil {
		route.RouteName = *req.RouteName
	}
	if req.OriginCity != nil {
		route.OriginCity = *req.OriginCity
	}
	if req.DestinationCity != nil {
		route.DestinationCity = *req.DestinationCity
	}
	if req.TotalDistanceKM != nil {
		route.TotalDistanceKM = *req.TotalDistanceKM
	}
	if req.EstimatedDurationMinutes != nil {
		route.EstimatedDurationMinutes = *req.EstimatedDurationMinutes
	}
	if req.EncodedPolyline != nil {
		route.EncodedPolyline = req.EncodedPolyline
	}
	if req.IsActive != nil {
		route.IsActive = *req.IsActive
	}

	err = s.routeRepo.Update(ctx, route)
	if err != nil {
		return nil, fmt.Errorf("failed to update route: %w", err)
	}

	return route, nil
}

func (s *RouteService) DeleteRoute(ctx context.Context, id string) error {
	err := s.routeRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}
	return nil
}
