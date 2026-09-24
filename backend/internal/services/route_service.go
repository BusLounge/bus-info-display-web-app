package services

import (
	"context"
	"fmt"
	"math"

	"bus-schedule-lounge/internal/database"
	"bus-schedule-lounge/internal/models"
)

type RouteService struct {
	routeRepo *database.RouteRepository
	etaProfiles map[string]float64
}

func NewRouteService(routeRepo *database.RouteRepository, etaProfiles map[string]float64) *RouteService {
	return &RouteService{
		routeRepo: routeRepo,
		etaProfiles: etaProfiles,
	}
}

type RouteSegmentRequest struct {
	SegmentOrder   int     `json:"segmentOrder"`
	StartLatitude  float64 `json:"startLatitude"`
	StartLongitude float64 `json:"startLongitude"`
	EndLatitude    float64 `json:"endLatitude"`
	EndLongitude   float64 `json:"endLongitude"`
	DistanceKM     float64 `json:"distanceKm"`
	RoadType       string  `json:"roadType"`
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
	Segments                 []RouteSegmentRequest `json:"segments"`
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
	Segments                *[]RouteSegmentRequest `json:"segments,omitempty"`
}

var validRoadTypes = map[string]bool{
	"HIGHWAY": true, "EXPRESSWAY": true, "ARTERIAL": true, "COLLECTOR": true,
	"URBAN": true, "RURAL": true, "LOCAL": true, "SERVICE": true,
}

func (s *RouteService) buildSegments(routeID string, requests []RouteSegmentRequest) ([]models.RouteSegment, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("at least one route segment is required")
	}

	segments := make([]models.RouteSegment, 0, len(requests))
	for index, request := range requests {
		if request.SegmentOrder != index+1 {
			return nil, fmt.Errorf("segment orders must start at 1 and be contiguous")
		}
		if !validRoadTypes[request.RoadType] {
			return nil, fmt.Errorf("invalid road type %q", request.RoadType)
		}
		if request.DistanceKM <= 0 || math.IsNaN(request.DistanceKM) || math.IsInf(request.DistanceKM, 0) {
			return nil, fmt.Errorf("segment %d distance must be greater than zero", request.SegmentOrder)
		}
		coordinates := []float64{
			request.StartLatitude, request.StartLongitude,
			request.EndLatitude, request.EndLongitude,
		}
		for _, coordinate := range coordinates {
			if math.IsNaN(coordinate) || math.IsInf(coordinate, 0) {
				return nil, fmt.Errorf("segment %d contains invalid coordinates", request.SegmentOrder)
			}
		}

		multiplier := s.etaProfiles[request.RoadType]
		if multiplier <= 0 {
			return nil, fmt.Errorf("missing ETA profile for road type %q", request.RoadType)
		}
		baselineSpeed := 40.0 / multiplier
		baselineDuration := int(math.Ceil(request.DistanceKM / baselineSpeed * 60))
		if baselineDuration < 1 {
			baselineDuration = 1
		}

		segments = append(segments, models.RouteSegment{
			MasterRouteID: routeID,
			SegmentOrder: request.SegmentOrder,
			StartLatitude: request.StartLatitude,
			StartLongitude: request.StartLongitude,
			EndLatitude: request.EndLatitude,
			EndLongitude: request.EndLongitude,
			DistanceKM: request.DistanceKM,
			BaselineDurationMinutes: baselineDuration,
			BaselineSpeedKMH: baselineSpeed,
			RoadType: request.RoadType,
			TrafficSensitivityFactor: multiplier,
		})
	}
	return segments, nil
}

func (s *RouteService) GetAllRoutes(ctx context.Context) ([]*models.MasterRoute, error) {
	return s.routeRepo.GetAll(ctx)
}

func (s *RouteService) GetActiveRoutesWithDetails(ctx context.Context) ([]*models.ActiveRouteDetails, error) {
	return s.routeRepo.GetActiveWithDetails(ctx)
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
	segments, err := s.buildSegments(route.ID, req.Segments)
	if err != nil {
		return nil, err
	}

	err = s.routeRepo.CreateWithSegments(ctx, route, segments)
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

	if req.Segments == nil {
		return nil, fmt.Errorf("segments are required when updating a route")
	}
	segments, err := s.buildSegments(id, *req.Segments)
	if err != nil {
		return nil, err
	}
	err = s.routeRepo.UpdateWithSegments(ctx, route, segments)
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
