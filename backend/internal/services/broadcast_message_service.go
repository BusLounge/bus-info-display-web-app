package services

import (
	"bus-schedule-lounge/internal/database"
	"bus-schedule-lounge/internal/models"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type BroadcastMessageService struct {
	repo *database.BroadcastMessageRepository
}

func NewBroadcastMessageService(repo *database.BroadcastMessageRepository) *BroadcastMessageService {
	return &BroadcastMessageService{repo: repo}
}

func (s *BroadcastMessageService) EnsureSchema() error {
	return s.repo.EnsureSchema()
}

func (s *BroadcastMessageService) GetAll() ([]models.BroadcastMessage, error) {
	return s.repo.GetAll()
}

func (s *BroadcastMessageService) GetActiveForTV() ([]models.BroadcastMessage, error) {
	return s.repo.GetActiveForTV(time.Now().UTC())
}

func (s *BroadcastMessageService) Create(req *models.BroadcastMessageRequest) (*models.BroadcastMessage, error) {
	item, err := s.requestToModel("", req)
	if err != nil {
		return nil, err
	}
	item.ID = uuid.NewString()

	if err := s.repo.Create(item); err != nil {
		return nil, fmt.Errorf("create broadcast message: %w", err)
	}
	return item, nil
}

func (s *BroadcastMessageService) Update(id string, req *models.BroadcastMessageRequest) (*models.BroadcastMessage, error) {
	item, err := s.requestToModel(id, req)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Update(item); err != nil {
		return nil, fmt.Errorf("update broadcast message: %w", err)
	}
	return item, nil
}

func (s *BroadcastMessageService) Delete(id string) error {
	return s.repo.Delete(id)
}

func (s *BroadcastMessageService) requestToModel(id string, req *models.BroadcastMessageRequest) (*models.BroadcastMessage, error) {
	message := strings.TrimSpace(req.Message)
	if message == "" {
		return nil, fmt.Errorf("message is required")
	}

	duration := req.DisplayDurationSeconds
	if duration <= 0 {
		return nil, fmt.Errorf("displayDurationSeconds must be greater than zero")
	}

	frequency := req.FrequencySeconds
	if frequency <= 0 {
		return nil, fmt.Errorf("frequencySeconds must be greater than zero")
	}
	if duration > frequency {
		return nil, fmt.Errorf("displayDurationSeconds cannot exceed frequencySeconds")
	}

	startAt := time.Now().UTC()
	if req.StartAt != nil && strings.TrimSpace(*req.StartAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.StartAt))
		if err != nil {
			return nil, fmt.Errorf("invalid startAt, expected RFC3339")
		}
		startAt = parsed.UTC()
	}

	var endAt *time.Time
	if req.EndAt != nil && strings.TrimSpace(*req.EndAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.EndAt))
		if err != nil {
			return nil, fmt.Errorf("invalid endAt, expected RFC3339")
		}
		value := parsed.UTC()
		endAt = &value
	}

	if endAt != nil && endAt.Before(startAt) {
		return nil, fmt.Errorf("endAt must be after startAt")
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	showOnLoungeTV := true
	if req.ShowOnLoungeTV != nil {
		showOnLoungeTV = *req.ShowOnLoungeTV
	}

	priority := strings.TrimSpace(strings.ToLower(req.Priority))
	if priority == "" {
		priority = "normal"
	}

	return &models.BroadcastMessage{
		ID:                     id,
		Message:                message,
		Priority:               priority,
		DisplayDurationSeconds: duration,
		FrequencySeconds:       frequency,
		StartAt:                startAt,
		EndAt:                  endAt,
		IsActive:               isActive,
		ShowOnLoungeTV:         showOnLoungeTV,
	}, nil
}
