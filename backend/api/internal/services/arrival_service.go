package services

import (
	"context"
	"math"
	"strings"
	"time"

	"bus-schedule-lounge/internal/database"
	"bus-schedule-lounge/internal/models"
	"bus-schedule-lounge/pkg/utils"
)

type ArrivalService struct {
	repo        *database.ArrivalRepository
	board       *LoungeBoardService
	etaProfiles map[string]float64
	location    *time.Location
}

func NewArrivalService(
	repo *database.ArrivalRepository,
	location *time.Location,
	board *LoungeBoardService,
	etaProfiles ...map[string]float64,
) *ArrivalService {
	service := &ArrivalService{repo: repo, board: board, location: location}
	if len(etaProfiles) > 0 {
		service.etaProfiles = etaProfiles[0]
	}
	return service
}

// GetAllLoungeArrivals gets arrivals for all lounges with calculated remarks
// (admin Arrival Management page).
func (s *ArrivalService) GetAllLoungeArrivals() ([]models.LoungeArrivalResponse, error) {
	start, end, err := currentServiceDayBounds(time.Now(), s.location)
	if err != nil {
		return nil, err
	}
	arrivals, err := s.repo.GetAllLoungeArrivals(start, end)
	if err != nil {
		return nil, err
	}

	// Process arrivals and calculate remarks
	s.processArrivals(arrivals)

	// Group by lounge
	return s.groupByLounge(arrivals, start.Format("2006-01-02"), s.location.String()), nil
}

// GetArrivalsByLoungeID returns the arrivals board for one lounge (used by the kiosks):
// buses due from 15 minutes ago to 6 hours ahead, earliest first, at most 10.
func (s *ArrivalService) GetArrivalsByLoungeID(loungeID string) (*models.LoungeArrivalResponse, error) {
	board, err := s.board.Board(context.Background(), loungeID)
	if err != nil {
		return nil, err
	}

	response := &models.LoungeArrivalResponse{
		LoungeID:        loungeID,
		LoungeName:      board.LoungeName,
		ServiceDate:     board.ServiceDate,
		ServiceTimeZone: board.ServiceTimeZone,
		Arrivals:        []models.ArrivalInfo{},
	}
	for _, entry := range board.Arrivals() {
		response.Arrivals = append(response.Arrivals, arrivalInfoFromETA(entry, board.Now, s.location))
	}
	return response, nil
}

func arrivalInfoFromETA(entry LoungeTripETA, now time.Time, location *time.Location) models.ArrivalInfo {
	trip := entry.Trip
	arrival := entry.ArrivalETA
	departure := trip.DepartureTime
	offset := int(math.Round(entry.MinutesFromDeparture))
	return models.ArrivalInfo{
		LoungeID:             trip.LoungeID,
		LoungeName:           trip.LoungeName,
		LoungeLat:            trip.LoungeLat,
		LoungeLng:            trip.LoungeLng,
		ActiveTripID:         trip.ActiveTripID,
		MasterRouteID:        trip.MasterRouteID,
		BusID:                trip.BusID,
		DriverID:             trip.DriverID,
		BusNumber:            busNumberOrTBA(trip.BusNumber),
		BusCompanyName:       trip.BusCompanyName,
		RouteNumber:          trip.RouteNumber,
		Origin:               trip.Origin,
		Destination:          trip.Destination,
		CurrentLat:           trip.CurrentLat,
		CurrentLng:           trip.CurrentLng,
		CurrentSpeedKmh:      trip.CurrentSpeedKmh,
		LastLocationUpdate:   trip.LastLocationUpdate,
		OffsetMinutes:        &offset,
		ETA:                  &arrival,
		DepartureTime:        &departure,
		Status:               tripStatusOrNotStarted(trip.ActiveTripStatus),
		Remarks:              arrivalRemark(arrival, now, location),
		TimeDisplay:          clockTime(arrival, location),
		DistanceKm:           entry.DistanceKm,
		CalculatedETAMinutes: entry.MinutesFromDeparture,
		ETASource:            entry.Source,
	}
}

func busNumberOrTBA(busNumber string) string {
	if strings.TrimSpace(busNumber) == "" {
		return "Bus TBA"
	}
	return busNumber
}

func tripStatusOrNotStarted(status string) string {
	if strings.TrimSpace(status) == "" {
		return "not_started"
	}
	return status
}

// processArrivals calculates ETA, remarks and time display for each arrival
// (admin Arrival Management page).
func (s *ArrivalService) processArrivals(arrivals []models.ArrivalInfo) {
	now := time.Now()

	// Optional Python ETA engine (off unless ETA_ENGINE_ENABLED=true).
	var requests []ETARequest
	var requestIndexes []int
	for i, arrival := range arrivals {
		if arrival.DepartureTime != nil && arrival.MasterRouteID != "" {
			requests = append(requests, ETARequest{
				MasterRouteID:     arrival.MasterRouteID,
				LoungeID:          arrival.LoungeID,
				DepartureDatetime: arrival.DepartureTime.Format(time.RFC3339),
				DriverID:          arrival.DriverID,
				BusID:             arrival.BusID,
			})
			requestIndexes = append(requestIndexes, i)
		}
	}
	mlPredictions := make(map[int]*time.Time)
	for i, prediction := range predictWithETAEngine(requests) {
		if prediction != nil {
			mlPredictions[requestIndexes[i]] = prediction
		}
	}

	for i := range arrivals {
		arrival := &arrivals[i]

		hasGPSData := arrival.CurrentLat != nil && arrival.CurrentLng != nil &&
			arrival.LoungeLat != nil && arrival.LoungeLng != nil &&
			utils.ValidateCoordinates(*arrival.CurrentLat, *arrival.CurrentLng) &&
			utils.ValidateCoordinates(*arrival.LoungeLat, *arrival.LoungeLng)

		if mlETA, ok := mlPredictions[i]; ok {
			arrival.ETA = mlETA
			arrival.ETASource = ETASourceETAEngine
			if hasGPSData {
				arrival.DistanceKm = utils.HaversineDistance(*arrival.CurrentLat, *arrival.CurrentLng, *arrival.LoungeLat, *arrival.LoungeLng)
			}
		} else if hasGPSData {
			distance, currentETAMinutes, foundRouteETA := calculateSegmentBasedETA(
				*arrival.CurrentLat, *arrival.CurrentLng,
				*arrival.LoungeLat, *arrival.LoungeLng,
				arrival.RouteSegments, s.etaProfiles,
			)
			if !foundRouteETA {
				distance = utils.HaversineDistance(
					*arrival.CurrentLat, *arrival.CurrentLng,
					*arrival.LoungeLat, *arrival.LoungeLng,
				)
				speed := utils.DefaultFallbackSpeedKmh
				if arrival.CurrentSpeedKmh != nil && *arrival.CurrentSpeedKmh >= utils.MinSpeedThresholdKmh {
					speed = *arrival.CurrentSpeedKmh
				}
				currentETAMinutes = utils.CalculateETAMinutes(distance, speed)
			}
			arrival.DistanceKm = distance
			arrival.CalculatedETAMinutes = currentETAMinutes

			// Live ETA counts from now. (The schedule offset is no longer added on top:
			// it measures time from the start of the route, not from the bus's position.)
			newETA := now.Add(minutesToDuration(currentETAMinutes))
			arrival.ETA = &newETA
			arrival.ETASource = ETASourceLiveGPS
		} else if routeDuration, ok := calculateRoadTypeRouteDuration(
			arrival.RouteSegments,
			s.etaProfiles,
		); ok && arrival.DepartureTime != nil {
			routeETA := arrival.DepartureTime.Add(routeDuration)
			arrival.CalculatedETAMinutes = routeDuration.Minutes()
			arrival.ETA = &routeETA
			arrival.ETASource = ETASourceRoadType
		} else if arrival.ETA == nil && arrival.DepartureTime != nil && arrival.OffsetMinutes != nil {
			loungeSpecificETA := arrival.DepartureTime.Add(time.Duration(*arrival.OffsetMinutes) * time.Minute)
			arrival.ETA = &loungeSpecificETA
			arrival.ETASource = ETASourceStopOffsets
		} else if arrival.ETA != nil {
			arrival.ETASource = ETASourceStopOffsets
		}

		if arrival.ETA == nil {
			arrival.Remarks = "In Transit"
			arrival.TimeDisplay = "-"
			continue
		}

		arrival.TimeDisplay = clockTime(*arrival.ETA, s.location)
		arrival.Remarks = arrivalRemark(*arrival.ETA, now, s.location)
	}
}

func calculateSegmentBasedETA(
	currentLat, currentLng, loungeLat, loungeLng float64,
	segments []models.RouteSegment,
	etaProfiles map[string]float64,
) (distanceKm, etaMinutes float64, ok bool) {
	if len(segments) == 0 {
		return 0, 0, false
	}

	referenceLatitude := (currentLat + loungeLat) / 2
	currentPosition, currentDistance := nearestRoutePosition(currentLat, currentLng, segments, referenceLatitude)
	loungePosition, loungeDistance := nearestRoutePosition(loungeLat, loungeLng, segments, referenceLatitude)
	if currentDistance < 0 || loungeDistance < 0 || loungePosition <= currentPosition {
		return 0, 0, false
	}

	cumulativeDistance := 0.0
	for _, segment := range segments {
		segmentEnd := cumulativeDistance + segment.DistanceKM
		overlapStart := math.Max(currentPosition, cumulativeDistance)
		overlapEnd := math.Min(loungePosition, segmentEnd)
		if overlapEnd > overlapStart {
			segmentDistance := overlapEnd - overlapStart
			distanceKm += segmentDistance
			etaMinutes += segmentDistance / segmentBaselineSpeed(segment, etaProfiles) * 60
		}
		cumulativeDistance = segmentEnd
	}

	return distanceKm, etaMinutes, distanceKm > 0
}

func calculateRoadTypeRouteDuration(
	segments []models.RouteSegment,
	etaProfiles map[string]float64,
) (time.Duration, bool) {
	if len(segments) == 0 {
		return 0, false
	}

	hasKnownRoadType := false
	totalMinutes := 0.0
	for _, segment := range segments {
		if segment.DistanceKM <= 0 ||
			math.IsNaN(segment.DistanceKM) ||
			math.IsInf(segment.DistanceKM, 0) {
			return 0, false
		}

		roadType := strings.ToUpper(strings.TrimSpace(segment.RoadType))
		if multiplier := etaProfiles[roadType]; roadType != "" && roadType != "UNKNOWN" &&
			multiplier > 0 && !math.IsNaN(multiplier) && !math.IsInf(multiplier, 0) {
			hasKnownRoadType = true
		}

		speed := segmentBaselineSpeed(segment, etaProfiles)
		if speed <= 0 || math.IsNaN(speed) || math.IsInf(speed, 0) {
			return 0, false
		}
		totalMinutes += segment.DistanceKM / speed * 60
	}

	if !hasKnownRoadType || totalMinutes <= 0 ||
		math.IsNaN(totalMinutes) || math.IsInf(totalMinutes, 0) {
		return 0, false
	}

	return time.Duration(totalMinutes * float64(time.Minute)), true
}

func nearestRoutePosition(latitude, longitude float64, segments []models.RouteSegment, referenceLatitude float64) (positionKm, distanceKm float64) {
	bestDistance := math.Inf(1)
	cumulativeDistance := 0.0
	positionKm = -1
	distanceKm = -1
	cosLatitude := math.Cos(referenceLatitude * math.Pi / 180)

	pointX := longitude * 111.32 * cosLatitude
	pointY := latitude * 110.574
	for _, segment := range segments {
		if segment.DistanceKM <= 0 || math.IsNaN(segment.DistanceKM) || math.IsInf(segment.DistanceKM, 0) {
			continue
		}

		startX := segment.StartLongitude * 111.32 * cosLatitude
		startY := segment.StartLatitude * 110.574
		endX := segment.EndLongitude * 111.32 * cosLatitude
		endY := segment.EndLatitude * 110.574
		deltaX := endX - startX
		deltaY := endY - startY
		lengthSquared := deltaX*deltaX + deltaY*deltaY
		fraction := 0.0
		if lengthSquared > 0 {
			fraction = ((pointX-startX)*deltaX + (pointY-startY)*deltaY) / lengthSquared
			fraction = math.Max(0, math.Min(fraction, 1))
		}

		projectedX := startX + fraction*deltaX
		projectedY := startY + fraction*deltaY
		distance := math.Hypot(pointX-projectedX, pointY-projectedY)
		if distance < bestDistance {
			bestDistance = distance
			positionKm = cumulativeDistance + fraction*segment.DistanceKM
			distanceKm = distance
		}
		cumulativeDistance += segment.DistanceKM
	}

	return positionKm, distanceKm
}

func segmentBaselineSpeed(segment models.RouteSegment, etaProfiles map[string]float64) float64 {
	if multiplier := etaProfiles[strings.ToUpper(strings.TrimSpace(segment.RoadType))]; multiplier > 0 && !math.IsNaN(multiplier) && !math.IsInf(multiplier, 0) {
		return utils.DefaultFallbackSpeedKmh / multiplier
	}
	if segment.BaselineSpeedKMH > 0 && !math.IsNaN(segment.BaselineSpeedKMH) && !math.IsInf(segment.BaselineSpeedKMH, 0) {
		return segment.BaselineSpeedKMH
	}
	return utils.DefaultFallbackSpeedKmh
}

// groupByLounge groups arrivals by lounge
func (s *ArrivalService) groupByLounge(
	arrivals []models.ArrivalInfo,
	serviceDate, serviceTimeZone string,
) []models.LoungeArrivalResponse {
	loungeMap := make(map[string]*models.LoungeArrivalResponse)
	var loungeOrder []string

	for _, arrival := range arrivals {
		if _, exists := loungeMap[arrival.LoungeID]; !exists {
			loungeMap[arrival.LoungeID] = &models.LoungeArrivalResponse{
				LoungeID:        arrival.LoungeID,
				LoungeName:      arrival.LoungeName,
				ServiceDate:     serviceDate,
				ServiceTimeZone: serviceTimeZone,
				Arrivals:        []models.ArrivalInfo{},
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
