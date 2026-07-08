package services

import (
	"bus-schedule-lounge/internal/database"
	"bus-schedule-lounge/internal/models"
	"fmt"
	"time"
)

type DepartureService struct {
	repo *database.DepartureRepository
}

func NewDepartureService(repo *database.DepartureRepository) *DepartureService {
	return &DepartureService{repo: repo}
}

// GetAllLoungeDepartures gets departures for all lounges with calculated remarks
func (s *DepartureService) GetAllLoungeDepartures() ([]models.LoungeDepartureResponse, error) {
	departures, err := s.repo.GetAllLoungeDepartures()
	if err != nil {
		return nil, err
	}

	// Process departures and calculate remarks
	s.processDepartures(departures)

	// Group by lounge
	return s.groupByLounge(departures), nil
}

// GetDeparturesByLoungeID gets departures for a specific lounge with calculated remarks
func (s *DepartureService) GetDeparturesByLoungeID(loungeID string) (*models.LoungeDepartureResponse, error) {
	departures, err := s.repo.GetDeparturesByLoungeID(loungeID)
	if err != nil {
		return nil, err
	}

	if len(departures) == 0 {
		return &models.LoungeDepartureResponse{
			LoungeID:   loungeID,
			LoungeName: "",
			Departures: []models.DepartureInfo{},
		}, nil
	}

	// Process departures and calculate remarks
	s.processDepartures(departures)

	return &models.LoungeDepartureResponse{
		LoungeID:   departures[0].LoungeID,
		LoungeName: departures[0].LoungeName,
		Departures: departures,
	}, nil
}

// processDepartures calculates remarks and time display for each departure
func (s *DepartureService) processDepartures(departures []models.DepartureInfo) {
	now := time.Now()

	for i := range departures {
		departure := &departures[i]

		// If already departed
		if departure.ActualDeparture != nil {
			departure.Remarks = "Departed"
			departure.TimeDisplay = departure.ActualDeparture.Format("15:04")
			continue
		}

		// Use calculated departure ETA if available
		if departure.DepartureETA == nil {
			departure.Remarks = "Scheduled"
			departure.TimeDisplay = "-"
			continue
		}

		// Calculate arrival delay and adjust departure ETA accordingly
		// If arrival is delayed by X minutes, departure should also be delayed by X minutes
		var delayMinutes int
		if departure.ArrivalETA != nil {
			arrivalDelay := now.Sub(*departure.ArrivalETA)
			if arrivalDelay > 0 {
				delayMinutes = int(arrivalDelay.Minutes())
				// Adjust departure ETA by adding the arrival delay
				adjustedDeparture := departure.DepartureETA.Add(time.Duration(delayMinutes) * time.Minute)
				departure.DepartureETA = &adjustedDeparture
			}
		}

		// Calculate time difference to departure
		diff := departure.DepartureETA.Sub(now)
		minutesDiff := int(diff.Minutes())

		// Set time display
		departure.TimeDisplay = departure.DepartureETA.Format("15:04")

		// Calculate remarks based on delay and time to departure
		// Priority: Show delay information if there's a significant delay
		if delayMinutes > 10 {
			// Significant delay from arrival - show delay with departure time
			departure.Remarks = fmt.Sprintf("Delayed %d min", delayMinutes)
		} else if minutesDiff < -10 {
			// Departure is delayed (more than 10 minutes past departure time)
			pastMinutes := -minutesDiff
			departure.Remarks = fmt.Sprintf("Delayed %d min", pastMinutes)
		} else if minutesDiff <= 0 {
			// Departure time is now or just passed (within 10 minutes)
			departure.Remarks = "Departing Now"
		} else if minutesDiff <= 5 {
			// Within 5 minutes
			departure.Remarks = "Boarding Now"
		} else if minutesDiff <= 15 {
			// Within 15 minutes - final call
			departure.Remarks = fmt.Sprintf("Final Call - %d min", minutesDiff)
		} else if minutesDiff <= 30 {
			// Within 30 minutes - boarding soon
			departure.Remarks = fmt.Sprintf("Boarding Soon - %d min", minutesDiff)
		} else {
			// More than 30 minutes away
			departure.Remarks = fmt.Sprintf("Departs at %s", departure.DepartureETA.Format("15:04"))
		}
	}
}

// groupByLounge groups departures by lounge
func (s *DepartureService) groupByLounge(departures []models.DepartureInfo) []models.LoungeDepartureResponse {
	loungeMap := make(map[string]*models.LoungeDepartureResponse)
	var loungeOrder []string

	for _, departure := range departures {
		if _, exists := loungeMap[departure.LoungeID]; !exists {
			loungeMap[departure.LoungeID] = &models.LoungeDepartureResponse{
				LoungeID:   departure.LoungeID,
				LoungeName: departure.LoungeName,
				Departures: []models.DepartureInfo{},
			}
			loungeOrder = append(loungeOrder, departure.LoungeID)
		}
		loungeMap[departure.LoungeID].Departures = append(loungeMap[departure.LoungeID].Departures, departure)
	}

	// Convert map to slice maintaining order
	result := make([]models.LoungeDepartureResponse, 0, len(loungeMap))
	for _, loungeID := range loungeOrder {
		result = append(result, *loungeMap[loungeID])
	}

	return result
}
