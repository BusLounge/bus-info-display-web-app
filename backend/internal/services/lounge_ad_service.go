package services

import (
	"bus-schedule-lounge/internal/database"
	"bus-schedule-lounge/internal/models"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type LoungeAdService struct {
	repo *database.LoungeAdRepository
}

func NewLoungeAdService(repo *database.LoungeAdRepository) *LoungeAdService {
	return &LoungeAdService{repo: repo}
}

func (s *LoungeAdService) EnsureSchema() error {
	return s.repo.EnsureSchema()
}

func (s *LoungeAdService) GetAll() ([]models.LoungeAd, error) {
	return s.repo.GetAll()
}

func (s *LoungeAdService) GetForLounge(loungeID string) ([]models.LoungeAd, error) {
	return s.repo.GetForLounge(loungeID)
}

func (s *LoungeAdService) Create(req *models.LoungeAdRequest) (*models.LoungeAd, error) {
	item, err := s.requestToModel("", req)
	if err != nil {
		return nil, err
	}
	if err := s.validateFitsLoungeWindow(item); err != nil {
		return nil, err
	}
	item.ID = uuid.NewString()
	if err := s.repo.Create(item); err != nil {
		return nil, fmt.Errorf("create lounge ad: %w", err)
	}
	return item, nil
}

func (s *LoungeAdService) Update(id string, req *models.LoungeAdRequest) (*models.LoungeAd, error) {
	item, err := s.requestToModel(id, req)
	if err != nil {
		return nil, err
	}
	if err := s.validateFitsLoungeWindow(item); err != nil {
		return nil, err
	}
	if err := s.repo.Update(item); err != nil {
		return nil, fmt.Errorf("update lounge ad: %w", err)
	}
	return item, nil
}

func (s *LoungeAdService) Delete(id string) error {
	return s.repo.Delete(id)
}

func (s *LoungeAdService) GetSlotSummary(loungeID string) (*models.LoungeAdSlotSummary, error) {
	booked, byScheduleType, groups, err := s.repo.GetBookedSecondsBreakdownForLounge(loungeID)
	if err != nil {
		return nil, err
	}
	summary := &models.LoungeAdSlotSummary{
		ScheduleWindowSeconds: 6,
		AdWindowSeconds:       24,
		BookedSeconds:         booked,
		BookedByScheduleType:  byScheduleType,
		EffectiveLoungeGroups: groups,
	}
	summary.RemainingSeconds = summary.AdWindowSeconds - summary.BookedSeconds
	if summary.RemainingSeconds < 0 {
		summary.RemainingSeconds = 0
	}
	summary.TimeSlots, summary.AvailableSlots = buildHybridTimeSlots(summary.ScheduleWindowSeconds, summary.AdWindowSeconds, summary.BookedSeconds)
	return summary, nil
}

func (s *LoungeAdService) validateFitsLoungeWindow(item *models.LoungeAd) error {
	if !item.IsActive {
		return nil
	}
	if item.IsDefaultForAll || item.LoungeID == nil || strings.TrimSpace(*item.LoungeID) == "" {
		return nil
	}

	summary, err := s.GetSlotSummary(strings.TrimSpace(*item.LoungeID))
	if err != nil {
		return fmt.Errorf("resolve lounge slot summary: %w", err)
	}

	if item.DurationSeconds > summary.RemainingSeconds {
		return fmt.Errorf("durationSeconds exceeds available ad window (%ds remaining after company ads)", summary.RemainingSeconds)
	}

	return nil
}

func buildHybridTimeSlots(scheduleSeconds, adWindowSeconds, bookedSeconds int) ([]models.LoungeAdTimeSlot, []models.LoungeAdTimeSlot) {
	if scheduleSeconds < 0 {
		scheduleSeconds = 0
	}
	if adWindowSeconds < 0 {
		adWindowSeconds = 0
	}
	if bookedSeconds < 0 {
		bookedSeconds = 0
	}
	if bookedSeconds > adWindowSeconds {
		bookedSeconds = adWindowSeconds
	}

	cycleStart := 0
	scheduleEnd := scheduleSeconds
	adStart := scheduleEnd
	companyEnd := adStart + bookedSeconds
	adEnd := adStart + adWindowSeconds

	slots := []models.LoungeAdTimeSlot{
		{
			Type:            "schedule",
			Label:           "Bus schedules",
			StartSecond:     cycleStart,
			EndSecond:       scheduleEnd,
			DurationSeconds: scheduleSeconds,
			Interactive:     false,
		},
	}

	if bookedSeconds > 0 {
		slots = append(slots, models.LoungeAdTimeSlot{
			Type:            "company",
			Label:           "Company ads",
			StartSecond:     adStart,
			EndSecond:       companyEnd,
			DurationSeconds: bookedSeconds,
			Interactive:     false,
		})
	}

	available := make([]models.LoungeAdTimeSlot, 0, 1)
	remaining := adEnd - companyEnd
	if remaining > 0 {
		slot := models.LoungeAdTimeSlot{
			Type:            "available",
			Label:           "Available for lounge-specific ads",
			StartSecond:     companyEnd,
			EndSecond:       adEnd,
			DurationSeconds: remaining,
			Interactive:     true,
		}
		slots = append(slots, slot)
		available = append(available, slot)
	}

	return slots, available
}

func (s *LoungeAdService) requestToModel(id string, req *models.LoungeAdRequest) (*models.LoungeAd, error) {
	name := strings.TrimSpace(req.AdvertisementName)
	if name == "" {
		return nil, fmt.Errorf("advertisementName is required")
	}
	mediaURL := strings.TrimSpace(req.MediaURL)
	if mediaURL == "" {
		return nil, fmt.Errorf("mediaUrl is required")
	}
	mediaType := strings.TrimSpace(strings.ToLower(req.MediaType))
	if mediaType == "" {
		mediaType = "image"
	}
	if mediaType != "image" && mediaType != "video" {
		return nil, fmt.Errorf("mediaType must be image or video")
	}

	duration := req.DurationSeconds
	if duration <= 0 {
		return nil, fmt.Errorf("durationSeconds must be greater than zero")
	}
	if duration > 24 {
		return nil, fmt.Errorf("durationSeconds cannot exceed 24 seconds ad window")
	}

	priority := strings.TrimSpace(strings.ToLower(req.Priority))
	if priority == "" {
		priority = "normal"
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	isDefaultForAll := false
	if req.IsDefaultForAll != nil {
		isDefaultForAll = *req.IsDefaultForAll
	}

	var loungeID *string
	if req.LoungeID != nil {
		trimmed := strings.TrimSpace(*req.LoungeID)
		if trimmed != "" {
			loungeID = &trimmed
		}
	}
	if isDefaultForAll {
		loungeID = nil
	}

	return &models.LoungeAd{
		ID:                id,
		LoungeID:          loungeID,
		AdvertisementName: name,
		MediaURL:          mediaURL,
		MediaType:         mediaType,
		DurationSeconds:   duration,
		Priority:          priority,
		IsActive:          isActive,
		IsDefaultForAll:   isDefaultForAll,
	}, nil
}
