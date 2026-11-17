package services

import (
	"bus-schedule-lounge/internal/models"
)

type LoungeService struct {
	repo LoungeRepositoryInterface
}

type LoungeRepositoryInterface interface {
	GetAllLounges() ([]models.Lounge, error)
	GetLoungeByID(id string) (*models.Lounge, error)
	ValidateLoungeRouteSegments(loungeID string) (*models.LoungeRouteValidationResponse, error)
}

func NewLoungeService(repo LoungeRepositoryInterface) *LoungeService {
	return &LoungeService{repo: repo}
}

func (s *LoungeService) GetAllLounges() ([]models.Lounge, error) {
	return s.repo.GetAllLounges()
}

func (s *LoungeService) GetLoungeByID(id string) (*models.Lounge, error) {
	return s.repo.GetLoungeByID(id)
}

func (s *LoungeService) ValidateLoungeRouteSegments(loungeID string) (*models.LoungeRouteValidationResponse, error) {
	return s.repo.ValidateLoungeRouteSegments(loungeID)
}
