package services

// Lounge ETA: when does each bus reach (and leave) a given lounge?
//
// One calculation feeds both the arrivals and the departures board, so the two
// can never disagree. For every trip that passes the lounge:
//
//  1. Live GPS (the conductor app updates the bus position): if the position is
//     recent and on the route, ETA = remaining road distance at road-type speeds.
//     If the bus is already past the lounge, the trip is dropped from the board.
//  2. Otherwise a schedule estimate, minutes from the start of the route, using
//     the first of these that gives a plausible average speed (15-60 km/h):
//     stop offsets, the trip's duration, the route's duration (both spread by
//     how far along the route the lounge is), then road-type speeds.
//  3. Optionally the Python ETA engine (ETA_ENGINE_ENABLED=true) replaces the
//     schedule estimate when it returns a plausible time.
//
// Departure = arrival + the lounge stopover (lounge_stopover_minutes, default 3).
// A board shows buses from 15 minutes before now to 6 hours ahead, earliest first,
// at most 10.

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"bus-schedule-lounge/internal/database"
	"bus-schedule-lounge/internal/models"
	"bus-schedule-lounge/pkg/utils"
)

const (
	boardLookAhead   = 6 * time.Hour
	boardGracePeriod = 15 * time.Minute
	boardMaxRows     = 10
	// Trips that left the start of their route up to this long ago can still be on their way.
	tripLookBack = 12 * time.Hour

	liveGPSMaxAge      = 5 * time.Minute
	liveMaxOffRouteKm  = 3.0
	livePassedMarginKm = 0.3

	minPlausibleSpeedKmh       = 15.0
	maxPlausibleSpeedKmh       = 60.0
	minDistanceForSpeedCheckKm = 2.0

	defaultStopoverMinutes = 3

	// The arrivals and departures requests for a lounge share one calculation.
	boardCacheTTL = 10 * time.Second
	// Road segments and stops rarely change.
	routeCacheTTL = 10 * time.Minute
)

// Where an ETA came from (returned to clients as etaSource).
const (
	ETASourceLiveGPS       = "live-gps"
	ETASourceETAEngine     = "eta-engine"
	ETASourceStopOffsets   = "stop-offsets"
	ETASourceTripDuration  = "trip-duration"
	ETASourceRouteDuration = "route-duration"
	ETASourceRoadType      = "road-type"
)

// LoungeTripETA is one bus's estimated arrival at and departure from a lounge.
type LoungeTripETA struct {
	Trip         database.LoungeTripRow
	ArrivalETA   time.Time
	DepartureETA time.Time
	Source       string
	// Schedule estimate: minutes from the start of the route to the lounge.
	MinutesFromDeparture float64
	// Live GPS: road km still to go; otherwise km from the start of the route (0 if unknown).
	DistanceKm float64

	loungeKm      float64
	loungeKmKnown bool
}

// ---------------------------------------------------------------- route model

type routeModel struct {
	segments  []models.RouteSegment // usable segments, in order
	totalKm   float64
	stops     map[string]database.RouteStop
	stopRank  map[string]int
	stopCount int
}

func newRouteModel(geometry database.RouteGeometry) *routeModel {
	model := &routeModel{
		stops:    make(map[string]database.RouteStop),
		stopRank: make(map[string]int),
	}
	for _, segment := range geometry.Segments {
		if !finitePositive(segment.DistanceKM) ||
			!utils.ValidateCoordinates(segment.StartLatitude, segment.StartLongitude) ||
			!utils.ValidateCoordinates(segment.EndLatitude, segment.EndLongitude) {
			continue
		}
		model.segments = append(model.segments, segment)
		model.totalKm += segment.DistanceKM
	}
	for i, stop := range geometry.Stops {
		model.stops[stop.ID] = stop
		model.stopRank[stop.ID] = i
	}
	model.stopCount = len(geometry.Stops)
	return model
}

// position projects a point onto the route: km from the start, and how far the point is from the route.
func (m *routeModel) position(lat, lng float64) (km, offRouteKm float64, ok bool) {
	if len(m.segments) == 0 || !utils.ValidateCoordinates(lat, lng) {
		return 0, 0, false
	}
	km, offRouteKm = nearestRoutePosition(lat, lng, m.segments, lat)
	if km < 0 || offRouteKm < 0 {
		return 0, 0, false
	}
	return km, offRouteKm, true
}

func (m *routeModel) stopPosition(stopID string) (float64, bool) {
	stop, exists := m.stops[stopID]
	if !exists || stop.Lat == nil || stop.Lng == nil {
		return 0, false
	}
	km, _, ok := m.position(*stop.Lat, *stop.Lng)
	return km, ok
}

// loungePosition is how far along the route the lounge is. It is kept between the
// stops either side of the lounge, so a lounge with inaccurate coordinates still
// lands in the right part of the route.
func (m *routeModel) loungePosition(trip database.LoungeTripRow) (float64, bool) {
	if trip.LoungeLat == nil || trip.LoungeLng == nil {
		return 0, false
	}
	km, _, ok := m.position(*trip.LoungeLat, *trip.LoungeLng)
	if !ok {
		return 0, false
	}
	before, okBefore := m.stopPosition(trip.StopBeforeID)
	after, okAfter := m.stopPosition(trip.StopAfterID)
	if okBefore && okAfter && after > before {
		km = math.Max(before, math.Min(km, after))
	}
	return km, true
}

// routeFraction is the share of the route covered on reaching the lounge (0-1).
// Without road geometry it falls back to the lounge's place in the stop list.
func (m *routeModel) routeFraction(trip database.LoungeTripRow, loungeKm float64, known bool) (float64, bool) {
	if known && m.totalKm > 0 {
		return clamp01(loungeKm / m.totalKm), true
	}
	rankBefore, okBefore := m.stopRank[trip.StopBeforeID]
	rankAfter, okAfter := m.stopRank[trip.StopAfterID]
	if okBefore && okAfter && m.stopCount > 1 {
		return clamp01((float64(rankBefore+rankAfter) / 2) / float64(m.stopCount-1)), true
	}
	return 0, false
}

// stopOffsetMinutes estimates minutes to the lounge from the stops' time offsets,
// interpolated by the lounge's position between the two stops.
func (m *routeModel) stopOffsetMinutes(trip database.LoungeTripRow, loungeKm float64, known bool) (float64, bool) {
	before := m.stops[trip.StopBeforeID].OffsetMinutes
	after := m.stops[trip.StopAfterID].OffsetMinutes
	switch {
	case before != nil && after != nil:
		if known {
			kmBefore, okBefore := m.stopPosition(trip.StopBeforeID)
			kmAfter, okAfter := m.stopPosition(trip.StopAfterID)
			if okBefore && okAfter && kmAfter > kmBefore {
				share := clamp01((loungeKm - kmBefore) / (kmAfter - kmBefore))
				return *before + (*after-*before)*share, true
			}
		}
		return (*before + *after) / 2, true
	case before != nil:
		return *before, true
	case after != nil:
		return *after, true
	}
	return 0, false
}

// travelMinutes is the time to drive from fromKm to toKm along the route at road-type speeds.
func travelMinutes(segments []models.RouteSegment, fromKm, toKm float64, etaProfiles map[string]float64) float64 {
	minutes := 0.0
	cumulative := 0.0
	for _, segment := range segments {
		end := cumulative + segment.DistanceKM
		overlapStart := math.Max(fromKm, cumulative)
		overlapEnd := math.Min(toKm, end)
		if overlapEnd > overlapStart {
			minutes += (overlapEnd - overlapStart) / segmentBaselineSpeed(segment, etaProfiles) * 60
		}
		cumulative = end
	}
	return minutes
}

// plausibleTravel checks that covering km in the given minutes means an average
// bus speed of 15-60 km/h. Short or unknown distances cannot be checked.
func plausibleTravel(minutes, km float64, kmKnown bool) bool {
	if math.IsNaN(minutes) || math.IsInf(minutes, 0) || minutes < 0 {
		return false
	}
	if !kmKnown || km < minDistanceForSpeedCheckKm {
		return true
	}
	if minutes == 0 {
		return false
	}
	speed := km / (minutes / 60)
	return speed >= minPlausibleSpeedKmh && speed <= maxPlausibleSpeedKmh
}

// ---------------------------------------------------------------- estimates

type scheduleEstimate struct {
	minutes  float64
	source   string
	loungeKm float64
	kmKnown  bool
}

// estimateFromSchedule returns minutes from the start of the route to the lounge.
func estimateFromSchedule(trip database.LoungeTripRow, m *routeModel, etaProfiles map[string]float64) (scheduleEstimate, bool) {
	loungeKm, kmKnown := m.loungePosition(trip)
	type candidate struct {
		minutes float64
		source  string
	}
	var candidates []candidate

	if minutes, ok := m.stopOffsetMinutes(trip, loungeKm, kmKnown); ok {
		candidates = append(candidates, candidate{minutes, ETASourceStopOffsets})
	}
	if share, ok := m.routeFraction(trip, loungeKm, kmKnown); ok {
		if trip.TripDurationMinutes != nil && finitePositive(*trip.TripDurationMinutes) {
			candidates = append(candidates, candidate{*trip.TripDurationMinutes * share, ETASourceTripDuration})
		}
		if trip.RouteDurationMinutes != nil && finitePositive(*trip.RouteDurationMinutes) {
			candidates = append(candidates, candidate{*trip.RouteDurationMinutes * share, ETASourceRouteDuration})
		}
	}
	roadTypeAvailable := kmKnown && len(m.segments) > 0
	if roadTypeAvailable {
		candidates = append(candidates, candidate{travelMinutes(m.segments, 0, loungeKm, etaProfiles), ETASourceRoadType})
	}

	pick := func(c candidate) (scheduleEstimate, bool) {
		return scheduleEstimate{minutes: c.minutes, source: c.source, loungeKm: loungeKm, kmKnown: kmKnown}, true
	}
	for _, c := range candidates {
		if plausibleTravel(c.minutes, loungeKm, kmKnown) {
			return pick(c)
		}
	}
	// Nothing passed the speed check: road-type speeds are the safest guess, then anything at all.
	if roadTypeAvailable {
		return pick(candidates[len(candidates)-1])
	}
	if len(candidates) > 0 && candidates[0].minutes >= 0 {
		return pick(candidates[0])
	}
	return scheduleEstimate{loungeKm: loungeKm, kmKnown: kmKnown}, false
}

type liveEstimate struct {
	arrival     time.Time
	remainingKm float64
	passed      bool
}

// estimateFromLiveGPS uses the bus position the conductor app reports. It only
// applies to trips in progress with a recent position close to the route.
func estimateFromLiveGPS(
	trip database.LoungeTripRow,
	m *routeModel,
	loungeKm float64,
	kmKnown bool,
	now time.Time,
	etaProfiles map[string]float64,
) (liveEstimate, bool) {
	if trip.ActiveTripStatus != "in_transit" && trip.ActiveTripStatus != "at_stop" {
		return liveEstimate{}, false
	}
	if trip.CurrentLat == nil || trip.CurrentLng == nil || trip.LastLocationUpdate == nil || !kmKnown {
		return liveEstimate{}, false
	}
	age := now.Sub(*trip.LastLocationUpdate)
	if age > liveGPSMaxAge || age < -2*time.Minute {
		return liveEstimate{}, false
	}
	busKm, offRouteKm, ok := m.position(*trip.CurrentLat, *trip.CurrentLng)
	if !ok || offRouteKm > liveMaxOffRouteKm {
		return liveEstimate{}, false
	}
	if busKm > loungeKm+livePassedMarginKm {
		return liveEstimate{passed: true}, true
	}

	reportedAt := *trip.LastLocationUpdate
	if reportedAt.After(now) {
		reportedAt = now
	}
	minutes := travelMinutes(m.segments, busKm, loungeKm, etaProfiles)
	return liveEstimate{
		arrival:     reportedAt.Add(minutesToDuration(minutes)),
		remainingKm: math.Max(0, loungeKm-busKm),
	}, true
}

// computeLoungeETAs estimates arrival and departure for every trip. Trips whose
// bus has already passed the lounge, or that cannot be placed at all, are left out.
func computeLoungeETAs(
	trips []database.LoungeTripRow,
	routes map[string]*routeModel,
	now time.Time,
	etaProfiles map[string]float64,
	stopover time.Duration,
) []LoungeTripETA {
	empty := newRouteModel(database.RouteGeometry{})
	var entries []LoungeTripETA

	for _, trip := range trips {
		model := routes[trip.MasterRouteID]
		if model == nil {
			model = empty
		}

		entry := LoungeTripETA{Trip: trip}
		estimate, haveSchedule := estimateFromSchedule(trip, model, etaProfiles)
		entry.loungeKm, entry.loungeKmKnown = estimate.loungeKm, estimate.kmKnown
		if haveSchedule {
			entry.ArrivalETA = trip.DepartureTime.Add(minutesToDuration(estimate.minutes))
			entry.MinutesFromDeparture = estimate.minutes
			entry.Source = estimate.source
			if estimate.kmKnown {
				entry.DistanceKm = estimate.loungeKm
			}
		}

		live, haveLive := estimateFromLiveGPS(trip, model, estimate.loungeKm, estimate.kmKnown, now, etaProfiles)
		if haveLive {
			if live.passed {
				continue // the bus has already gone past this lounge
			}
			entry.ArrivalETA = live.arrival
			entry.DistanceKm = live.remainingKm
			entry.Source = ETASourceLiveGPS
		}

		if !haveSchedule && !haveLive {
			continue
		}
		entry.DepartureETA = entry.ArrivalETA.Add(stopover)
		entries = append(entries, entry)
	}
	return entries
}

// applyETAEngine replaces schedule estimates with the Python ETA engine's
// predictions when it is enabled and its answer is plausible.
func applyETAEngine(entries []LoungeTripETA, stopover time.Duration) {
	if !etaEngineEnabled() {
		return
	}
	var requests []ETARequest
	var indexes []int
	for i, entry := range entries {
		if entry.Source == ETASourceLiveGPS {
			continue
		}
		requests = append(requests, ETARequest{
			MasterRouteID:     entry.Trip.MasterRouteID,
			LoungeID:          entry.Trip.LoungeID,
			DepartureDatetime: entry.Trip.DepartureTime.Format(time.RFC3339),
			DriverID:          entry.Trip.DriverID,
			BusID:             entry.Trip.BusID,
		})
		indexes = append(indexes, i)
	}
	predictions := predictWithETAEngine(requests)
	for i, prediction := range predictions {
		if prediction == nil {
			continue
		}
		entry := &entries[indexes[i]]
		minutes := prediction.Sub(entry.Trip.DepartureTime).Minutes()
		if !plausibleTravel(minutes, entry.loungeKm, entry.loungeKmKnown) {
			continue
		}
		entry.ArrivalETA = *prediction
		entry.DepartureETA = prediction.Add(stopover)
		entry.MinutesFromDeparture = minutes
		entry.Source = ETASourceETAEngine
	}
}

// selectForBoard keeps buses due between 15 minutes ago and 6 hours ahead,
// earliest first, at most 10.
func selectForBoard(entries []LoungeTripETA, now time.Time, timeOf func(LoungeTripETA) time.Time) []LoungeTripETA {
	from := now.Add(-boardGracePeriod)
	to := now.Add(boardLookAhead)
	var selected []LoungeTripETA
	for _, entry := range entries {
		at := timeOf(entry)
		if !at.Before(from) && !at.After(to) {
			selected = append(selected, entry)
		}
	}
	sort.SliceStable(selected, func(i, j int) bool {
		return timeOf(selected[i]).Before(timeOf(selected[j]))
	})
	if len(selected) > boardMaxRows {
		selected = selected[:boardMaxRows]
	}
	return selected
}

// ---------------------------------------------------------------- remarks

func arrivalRemark(eta, now time.Time, location *time.Location) string {
	minutes := int(eta.Sub(now).Minutes())
	switch {
	case minutes < -5:
		return fmt.Sprintf("Delayed %d min", -minutes)
	case minutes <= 0:
		return "Arriving Now"
	case minutes <= 5:
		return fmt.Sprintf("Arriving in %d min", minutes)
	case minutes <= 15:
		return fmt.Sprintf("Expected in %d min", minutes)
	default:
		return "Expected at " + clockTime(eta, location)
	}
}

func departureRemark(etd, now time.Time, location *time.Location) string {
	minutes := int(etd.Sub(now).Minutes())
	switch {
	case minutes < -10:
		return fmt.Sprintf("Delayed %d min", -minutes)
	case minutes <= 0:
		return "Departing Now"
	case minutes <= 5:
		return "Boarding Now"
	case minutes <= 15:
		return fmt.Sprintf("Final Call - %d min", minutes)
	case minutes <= 30:
		return fmt.Sprintf("Boarding Soon - %d min", minutes)
	default:
		return "Departs at " + clockTime(etd, location)
	}
}

// clockTime formats a time as HH:MM in the service timezone.
func clockTime(t time.Time, location *time.Location) string {
	if location != nil {
		t = t.In(location)
	}
	return t.Format("15:04")
}

// ---------------------------------------------------------------- service

// LoungeBoardService builds the arrivals/departures board for one lounge.
type LoungeBoardService struct {
	repo        *database.LoungeETARepository
	location    *time.Location
	etaProfiles map[string]float64
	stopover    time.Duration
	now         func() time.Time

	mu     sync.Mutex
	boards map[string]cachedBoard
	routes map[string]cachedRoute
}

type cachedBoard struct {
	board   *LoungeBoard
	expires time.Time
}

type cachedRoute struct {
	model   *routeModel
	expires time.Time
}

func NewLoungeBoardService(
	repo *database.LoungeETARepository,
	location *time.Location,
	etaProfiles map[string]float64,
	stopoverMinutes int,
) *LoungeBoardService {
	if stopoverMinutes <= 0 {
		stopoverMinutes = defaultStopoverMinutes
	}
	return &LoungeBoardService{
		repo:        repo,
		location:    location,
		etaProfiles: etaProfiles,
		stopover:    time.Duration(stopoverMinutes) * time.Minute,
		now:         time.Now,
		boards:      make(map[string]cachedBoard),
		routes:      make(map[string]cachedRoute),
	}
}

// LoungeBoard is every bus currently relevant to a lounge, with its ETAs.
type LoungeBoard struct {
	LoungeID        string
	LoungeName      string
	ServiceDate     string
	ServiceTimeZone string
	Now             time.Time
	StopoverMinutes int
	Entries         []LoungeTripETA
}

func (b *LoungeBoard) Arrivals() []LoungeTripETA {
	return selectForBoard(b.Entries, b.Now, func(e LoungeTripETA) time.Time { return e.ArrivalETA })
}

func (b *LoungeBoard) Departures() []LoungeTripETA {
	return selectForBoard(b.Entries, b.Now, func(e LoungeTripETA) time.Time { return e.DepartureETA })
}

// Board loads and estimates every trip relevant to the lounge right now.
func (s *LoungeBoardService) Board(ctx context.Context, loungeID string) (*LoungeBoard, error) {
	if s.location == nil {
		return nil, fmt.Errorf("service timezone is not configured")
	}
	now := s.now()
	if board := s.cachedBoard(loungeID, now); board != nil {
		return board, nil
	}

	trips, err := s.repo.GetLoungeTrips(ctx, loungeID, now.Add(-tripLookBack), now.Add(boardLookAhead))
	if err != nil {
		return nil, err
	}

	routes := make(map[string]*routeModel)
	for _, trip := range trips {
		if _, loaded := routes[trip.MasterRouteID]; loaded {
			continue
		}
		model, err := s.routeModel(ctx, trip.MasterRouteID, now)
		if err != nil {
			return nil, err
		}
		routes[trip.MasterRouteID] = model
	}

	entries := computeLoungeETAs(trips, routes, now, s.etaProfiles, s.stopover)
	applyETAEngine(entries, s.stopover)

	board := &LoungeBoard{
		LoungeID:        loungeID,
		ServiceDate:     now.In(s.location).Format("2006-01-02"),
		ServiceTimeZone: s.location.String(),
		Now:             now,
		StopoverMinutes: int(s.stopover / time.Minute),
		Entries:         entries,
	}
	if len(trips) > 0 {
		board.LoungeName = trips[0].LoungeName
	}

	s.mu.Lock()
	s.boards[loungeID] = cachedBoard{board: board, expires: now.Add(boardCacheTTL)}
	s.mu.Unlock()
	return board, nil
}

func (s *LoungeBoardService) cachedBoard(loungeID string, now time.Time) *LoungeBoard {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, cached := range s.boards {
		if now.After(cached.expires) {
			delete(s.boards, id)
		}
	}
	if cached, ok := s.boards[loungeID]; ok {
		return cached.board
	}
	return nil
}

func (s *LoungeBoardService) routeModel(ctx context.Context, masterRouteID string, now time.Time) (*routeModel, error) {
	s.mu.Lock()
	cached, ok := s.routes[masterRouteID]
	s.mu.Unlock()
	if ok && now.Before(cached.expires) {
		return cached.model, nil
	}

	geometry, err := s.repo.GetRouteGeometry(ctx, masterRouteID)
	if err != nil {
		return nil, err
	}
	model := newRouteModel(geometry)

	s.mu.Lock()
	s.routes[masterRouteID] = cachedRoute{model: model, expires: now.Add(routeCacheTTL)}
	s.mu.Unlock()
	return model, nil
}

// ---------------------------------------------------------------- helpers

func minutesToDuration(minutes float64) time.Duration {
	return time.Duration(math.Round(minutes * float64(time.Minute)))
}

func finitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func clamp01(value float64) float64 {
	return math.Max(0, math.Min(1, value))
}
