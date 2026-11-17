package services

import (
	"bus-schedule-lounge/internal/models"
)

type DashboardService struct {
	repo DashboardRepositoryInterface
}

type DashboardRepositoryInterface interface {
	GetDashboardStats() (*models.DashboardStats, error)
	GetRoutesPerLounges() ([]models.RouteStats, error)
	GetAdvertisementCategories() ([]models.AdvertisementCategoryStats, error)
	GetAdvertisementStatus() ([]models.AdvertisementStatusStats, error)
	GetMediaCategories() ([]models.MediaCategoryStats, error)
}

func NewDashboardService(repo DashboardRepositoryInterface) *DashboardService {
	return &DashboardService{repo: repo}
}

// GetDashboardData retrieves all dashboard data
func (s *DashboardService) GetDashboardData() (*models.DashboardResponse, error) {
	response := &models.DashboardResponse{}

	// Get stats
	stats, err := s.repo.GetDashboardStats()
	if err != nil {
		return nil, err
	}
	response.Stats = *stats

	// Get routes per lounges
	routesPerLounges, err := s.repo.GetRoutesPerLounges()
	if err != nil {
		return nil, err
	}
	response.RoutesPerLounges = routesPerLounges

	// Get advertisement categories
	adCategories, err := s.repo.GetAdvertisementCategories()
	if err != nil {
		return nil, err
	}
	response.AdvertisementCategories = adCategories

	// Get advertisement status
	adStatus, err := s.repo.GetAdvertisementStatus()
	if err != nil {
		return nil, err
	}
	response.AdvertisementStatus = adStatus

	// Get media categories
	mediaCategories, err := s.repo.GetMediaCategories()
	if err != nil {
		return nil, err
	}
	response.MediaCategories = mediaCategories

	return response, nil
}
