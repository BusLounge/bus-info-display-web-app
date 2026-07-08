package services

import (
	"bytes"
	"encoding/json"
	"net/http"
	"bus-schedule-lounge/internal/database"
	"bus-schedule-lounge/internal/models"
	"bus-schedule-lounge/pkg/utils"
	"fmt"
	"time"
)

type ETARequest struct {
	MasterRouteID     string `json:"master_route_id"`
	LoungeID          string `json:"lounge_id"`
	DepartureDatetime string `json:"departure_datetime"`
	DriverID          string `json:"driver_id"`
	BusID             string `json:"bus_id"`
}

type BatchETARequest struct {
	Requests []ETARequest `json:"requests"`
}

type ETAResponseData struct {
	MasterRouteID string `json:"master_route_id"`
	LoungeID      string `json:"lounge_id"`
	ETA           string `json:"eta"`
	ETD           string `json:"etd"`
}

type BatchETAResponseItem struct {
	Status string          `json:"status"`
	Data   ETAResponseData `json:"data"`
}

type ArrivalService struct {
	repo *database.ArrivalRepository
}

func NewArrivalService(repo *database.ArrivalRepository) *ArrivalService {
	return &ArrivalService{repo: repo}
}

// GetAllLoungeArrivals gets arrivals for all lounges with calculated remarks
func (s *ArrivalService) GetAllLoungeArrivals() ([]models.LoungeArrivalResponse, error) {
	arrivals, err := s.repo.GetAllLoungeArrivals()
	if err != nil {
		return nil, err
	}

	// Process arrivals and calculate remarks
	s.processArrivals(arrivals)

	// Group by lounge
	return s.groupByLounge(arrivals), nil
}

// GetArrivalsByLoungeID gets arrivals for a specific lounge with calculated remarks
func (s *ArrivalService) GetArrivalsByLoungeID(loungeID string) (*models.LoungeArrivalResponse, error) {
	arrivals, err := s.repo.GetArrivalsByLoungeID(loungeID)
	if err != nil {
		return nil, err
	}

	if len(arrivals) == 0 {
		return &models.LoungeArrivalResponse{
			LoungeID:   loungeID,
			LoungeName: "",
			Arrivals:   []models.ArrivalInfo{},
		}, nil
	}

	// Process arrivals and calculate remarks
	s.processArrivals(arrivals)

	return &models.LoungeArrivalResponse{
		LoungeID:   arrivals[0].LoungeID,
		LoungeName: arrivals[0].LoungeName,
		Arrivals:   arrivals,
	}, nil
}

// processArrivals calculates dynamic ETA, remarks and time display for each arrival
func (s *ArrivalService) processArrivals(arrivals []models.ArrivalInfo) {
	now := time.Now()

	// Prepare ETA engine batch request
	var batchReq BatchETARequest
	var validIndices []int

	for i, arrival := range arrivals {
		if arrival.ActualArrival == nil && arrival.DepartureTime != nil && arrival.MasterRouteID != "" {
			req := ETARequest{
				MasterRouteID:     arrival.MasterRouteID,
				LoungeID:          arrival.LoungeID,
				DepartureDatetime: arrival.DepartureTime.Format(time.RFC3339),
				DriverID:          arrival.DriverID,
				BusID:             arrival.BusID,
			}
			batchReq.Requests = append(batchReq.Requests, req)
			validIndices = append(validIndices, i)
		}
	}

	// Call ML engine
	mlPredictions := make(map[int]*time.Time)
	
	if len(batchReq.Requests) > 0 {
		if jsonData, err := json.Marshal(batchReq); err == nil {
			resp, err := http.Post("http://localhost:8000/api/v1/predict_batch", "application/json", bytes.NewBuffer(jsonData))
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					var batchResp []BatchETAResponseItem
					if err := json.NewDecoder(resp.Body).Decode(&batchResp); err == nil {
						for i, resItem := range batchResp {
							if resItem.Status == "success" {
								if parsedETA, err := time.Parse(time.RFC3339, resItem.Data.ETA); err == nil {
									mlPredictions[validIndices[i]] = &parsedETA
								}
							}
						}
					}
				}
			}
		}
	}

	for i := range arrivals {
		arrival := &arrivals[i]

		// If already arrived
		if arrival.ActualArrival != nil {
			arrival.Remarks = "Arrived"
			arrival.TimeDisplay = arrival.ActualArrival.Format("15:04")
			continue
		}

		// Calculate dynamic ETA if we have valid GPS data for BOTH bus and lounge
		hasGPSData := arrival.CurrentLat != nil && arrival.CurrentLng != nil &&
			arrival.LoungeLat != nil && arrival.LoungeLng != nil &&
			utils.ValidateCoordinates(*arrival.CurrentLat, *arrival.CurrentLng) &&
			utils.ValidateCoordinates(*arrival.LoungeLat, *arrival.LoungeLng)

		if mlETA, ok := mlPredictions[i]; ok {
			// ETA Engine priority
			arrival.ETA = mlETA
			// Set distance for UI if possible (from ML or just fallback distance)
			if hasGPSData {
				arrival.DistanceKm = utils.HaversineDistance(*arrival.CurrentLat, *arrival.CurrentLng, *arrival.LoungeLat, *arrival.LoungeLng)
			}
		} else if hasGPSData {
			// Fallback: Calculate distance between bus and THIS specific lounge
			distance := utils.HaversineDistance(
				*arrival.CurrentLat, *arrival.CurrentLng,
				*arrival.LoungeLat, *arrival.LoungeLng,
			)
			arrival.DistanceKm = distance

			// Step 2: Get speed (from database or use fallback)
			speed := utils.DefaultFallbackSpeedKmh
			if arrival.CurrentSpeedKmh != nil && *arrival.CurrentSpeedKmh >= utils.MinSpeedThresholdKmh {
				speed = *arrival.CurrentSpeedKmh
			}

			// Step 3: Calculate current ETA in minutes for THIS lounge
			currentETAMinutes := utils.CalculateETAMinutes(distance, speed)
			arrival.CalculatedETAMinutes = currentETAMinutes

			// Step 4: Apply offset adjustment if available for THIS lounge
			finalETAMinutes := currentETAMinutes
			if arrival.OffsetMinutes != nil && *arrival.OffsetMinutes > 0 {
				// Use historical offset to smooth ETA
				previousETAMinutes := float64(*arrival.OffsetMinutes)
				finalETAMinutes = utils.CalculateAdjustedETA(previousETAMinutes, currentETAMinutes)
			}

			// Calculate new ETA timestamp specific to THIS lounge
			newETA := now.Add(time.Duration(finalETAMinutes) * time.Minute)
			arrival.ETA = &newETA
		} else if arrival.ETA == nil && arrival.DepartureTime != nil && arrival.OffsetMinutes != nil {
			// Fallback: Use route offset from departure time for THIS specific lounge
			// This ensures each lounge gets its own ETA based on its offset
			loungeSpecificETA := arrival.DepartureTime.Add(time.Duration(*arrival.OffsetMinutes) * time.Minute)
			arrival.ETA = &loungeSpecificETA
		}

		// If still no ETA available, show status
		if arrival.ETA == nil {
			arrival.Remarks = "In Transit"
			arrival.TimeDisplay = "-"
			continue
		}

		// Calculate time difference for THIS lounge's ETA
		diff := arrival.ETA.Sub(now)
		minutesDiff := int(diff.Minutes())

		// Set time display
		arrival.TimeDisplay = arrival.ETA.Format("15:04")

		// Calculate remarks based on time difference
		if minutesDiff < -5 {
			// Bus is delayed (more than 5 minutes past ETA)
			delayMinutes := -minutesDiff
			arrival.Remarks = fmt.Sprintf("Delayed %d min", delayMinutes)
		} else if minutesDiff <= 0 {
			// ETA is now or just passed (within 5 minutes)
			arrival.Remarks = "Arriving Now"
		} else if minutesDiff <= 5 {
			// Within 5 minutes
			arrival.Remarks = fmt.Sprintf("Arriving in %d min", minutesDiff)
		} else if minutesDiff <= 15 {
			// Within 15 minutes
			arrival.Remarks = fmt.Sprintf("Expected in %d min", minutesDiff)
		} else {
			// More than 15 minutes away
			arrival.Remarks = fmt.Sprintf("Expected at %s", arrival.ETA.Format("15:04"))
		}
	}
}

// groupByLounge groups arrivals by lounge
func (s *ArrivalService) groupByLounge(arrivals []models.ArrivalInfo) []models.LoungeArrivalResponse {
	loungeMap := make(map[string]*models.LoungeArrivalResponse)
	var loungeOrder []string

	for _, arrival := range arrivals {
		if _, exists := loungeMap[arrival.LoungeID]; !exists {
			loungeMap[arrival.LoungeID] = &models.LoungeArrivalResponse{
				LoungeID:   arrival.LoungeID,
				LoungeName: arrival.LoungeName,
				Arrivals:   []models.ArrivalInfo{},
			}
			loungeOrder = append(loungeOrder, arrival.LoungeID)
		}
		loungeMap[arrival.LoungeID].Arrivals = append(loungeMap[arrival.LoungeID].Arrivals, arrival)
	}

	// Convert map to slice maintaining order
	result := make([]models.LoungeArrivalResponse, 0, len(loungeMap))
	for _, loungeID := range loungeOrder {
		result = append(result, *loungeMap[loungeID])
	}

	return result
}
