package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"bus-schedule-lounge/internal/database"
	wsupdates "bus-schedule-lounge/internal/websocket"
)

const defaultTripGeneratorTimezone = "Asia/Colombo"

type TripGeneratorRepository interface {
	ListActiveTripSchedules(context.Context) ([]database.TripSchedule, error)
	GenerateTrip(context.Context, database.TripSchedule, time.Time, int64) error
	GetLoungeIDsByBusOwnerRouteID(context.Context, string) ([]string, error)
}

type TripGeneratorService struct {
	repository TripGeneratorRepository
	location   *time.Location
	updatesHub *wsupdates.Hub
}

func NewTripGeneratorService(
	repository TripGeneratorRepository,
	location *time.Location,
	updatesHub *wsupdates.Hub,
) *TripGeneratorService {
	if location == nil {
		location = time.Local
	}
	return &TripGeneratorService{
		repository: repository,
		location:   location,
		updatesHub: updatesHub,
	}
}

// GenerateForDate creates all scheduled departures that recur on the given
// service date. Date calculations and departure timestamps use the configured
// service timezone.
func (s *TripGeneratorService) GenerateForDate(ctx context.Context, serviceDate time.Time) error {
	serviceDate = dateOnly(serviceDate.In(s.location))
	schedules, err := s.repository.ListActiveTripSchedules(ctx)
	if err != nil {
		return err
	}

	var generationErrors []error
	for _, schedule := range schedules {
		runs, err := scheduleRunsOn(schedule, serviceDate)
		if err != nil {
			generationErrors = append(generationErrors, fmt.Errorf("schedule %s: %w", schedule.ID, err))
			continue
		}
		if !runs {
			continue
		}
		durationMinutes, ok := scheduleDurationMinutes(schedule)
		if !ok {
			generationErrors = append(generationErrors, fmt.Errorf(
				"schedule %s has no positive estimated_duration_minutes on the schedule or its route", schedule.ID))
			continue
		}

		departure, err := parseDepartureTime(schedule.DepartureTime, serviceDate, s.location)
		if err != nil {
			generationErrors = append(generationErrors, fmt.Errorf("schedule %s: %w", schedule.ID, err))
			continue
		}
		if err := s.repository.GenerateTrip(
			ctx,
			schedule,
			departure,
			durationMinutes,
		); err != nil {
			generationErrors = append(generationErrors, err)
			continue
		}

		if !schedule.BusOwnerRouteID.Valid || schedule.BusOwnerRouteID.String == "" {
			continue
		}

		loungeIDs, err := s.repository.GetLoungeIDsByBusOwnerRouteID(
			ctx,
			schedule.BusOwnerRouteID.String,
		)
		if err != nil {
			generationErrors = append(generationErrors, fmt.Errorf(
				"find lounges for bus-owner route %s: %w",
				schedule.BusOwnerRouteID.String,
				err,
			))
			continue
		}

		for _, loungeID := range loungeIDs {
			if _, err := s.updatesHub.Broadcast(wsupdates.Notification{
				LoungeID: loungeID,
				Resource: wsupdates.ResourceSchedule,
			}); err != nil {
				generationErrors = append(generationErrors, fmt.Errorf(
					"broadcast schedule update for lounge %s: %w",
					loungeID,
					err,
				))
			}
		}
	}
	return errors.Join(generationErrors...)
}

func scheduleDurationMinutes(schedule database.TripSchedule) (int64, bool) {
	if schedule.EstimatedDurationMinutes.Valid && schedule.EstimatedDurationMinutes.Int64 > 0 {
		return schedule.EstimatedDurationMinutes.Int64, true
	}
	if schedule.RouteEstimatedDurationMinutes.Valid && schedule.RouteEstimatedDurationMinutes.Int64 > 0 {
		return schedule.RouteEstimatedDurationMinutes.Int64, true
	}
	return 0, false
}

// Run immediately generates trips for today, then schedules future runs at
// local midnight. Errors are logged, and the worker remains alive for the
// next scheduled run.
func (s *TripGeneratorService) Run(ctx context.Context) {
	s.runForDate(ctx, time.Now().In(s.location))
	for {
		nextRun := nextMidnight(time.Now().In(s.location))
		timer := time.NewTimer(time.Until(nextRun))
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-timer.C:
			s.runForDate(ctx, nextRun)
		}
	}
}

func (s *TripGeneratorService) runForDate(ctx context.Context, date time.Time) {
	if err := s.GenerateForDate(ctx, date); err != nil {
		log.Printf("Daily trip generation failed for %s: %v", date.Format("2006-01-02"), err)
	}
}

func scheduleRunsOn(schedule database.TripSchedule, serviceDate time.Time) (bool, error) {
	date := dateOnly(serviceDate)
	if schedule.ValidFrom.Valid {
		start, err := parseTripScheduleDate(schedule.ValidFrom.String)
		if err != nil {
			return false, fmt.Errorf("invalid valid_from %q: %w", schedule.ValidFrom.String, err)
		}
		if calendarDaysBetween(start, date) < 0 {
			return false, nil
		}
	}
	if schedule.ValidUntil.Valid {
		end, err := parseTripScheduleDate(schedule.ValidUntil.String)
		if err != nil {
			return false, fmt.Errorf("invalid valid_until %q: %w", schedule.ValidUntil.String, err)
		}
		if calendarDaysBetween(end, date) > 0 {
			return false, nil
		}
	}

	if schedule.SpecificDates.Valid && strings.TrimSpace(schedule.SpecificDates.String) != "" {
		dates, err := parseSpecificDates(schedule.SpecificDates.String)
		if err != nil {
			return false, err
		}
		for _, allowed := range dates {
			if calendarDaysBetween(allowed, date) == 0 {
				return true, nil
			}
		}
		return false, nil
	}

	switch strings.ToLower(strings.TrimSpace(schedule.RecurrenceType)) {
	case "daily":
		return true, nil
	case "weekly":
		if !schedule.RecurrenceDays.Valid || strings.TrimSpace(schedule.RecurrenceDays.String) == "" {
			return false, errors.New("weekly recurrence requires recurrence_days")
		}
		return containsWeekday(schedule.RecurrenceDays.String, date.Weekday())
	case "interval":
		if !schedule.ValidFrom.Valid {
			return false, errors.New("interval recurrence requires valid_from as its anchor date")
		}
		if !schedule.RecurrenceInterval.Valid || schedule.RecurrenceInterval.Int64 <= 0 {
			return false, errors.New("interval recurrence requires a positive recurrence_interval")
		}
		anchor, err := parseTripScheduleDate(schedule.ValidFrom.String)
		if err != nil {
			return false, fmt.Errorf("invalid interval anchor valid_from %q: %w", schedule.ValidFrom.String, err)
		}
		days := calendarDaysBetween(anchor, date)
		return days%schedule.RecurrenceInterval.Int64 == 0, nil
	default:
		return false, fmt.Errorf("unsupported recurrence_type %q", schedule.RecurrenceType)
	}
}

func containsWeekday(encoded string, weekday time.Weekday) (bool, error) {
	weekdays := map[string]time.Weekday{
		"sun": time.Sunday, "sunday": time.Sunday,
		"mon": time.Monday, "monday": time.Monday,
		"tue": time.Tuesday, "tuesday": time.Tuesday,
		"wed": time.Wednesday, "wednesday": time.Wednesday,
		"thu": time.Thursday, "thursday": time.Thursday,
		"fri": time.Friday, "friday": time.Friday,
		"sat": time.Saturday, "saturday": time.Saturday,
	}
	found := false
	for _, value := range strings.Split(encoded, ",") {
		normalized := strings.ToLower(strings.TrimSpace(value))
		day, ok := weekdays[normalized]
		if !ok {
			switch normalized {
			case "0":
				day, ok = time.Sunday, true
			case "1":
				day, ok = time.Monday, true
			case "2":
				day, ok = time.Tuesday, true
			case "3":
				day, ok = time.Wednesday, true
			case "4":
				day, ok = time.Thursday, true
			case "5":
				day, ok = time.Friday, true
			case "6":
				day, ok = time.Saturday, true
			}
		}
		if !ok {
			return false, fmt.Errorf("invalid recurrence_days value %q; use comma-separated weekday names or numbers 0-6 (Sunday-Saturday)", value)
		}
		if day == weekday {
			found = true
		}
	}
	return found, nil
}

func parseSpecificDates(encoded string) ([]time.Time, error) {
	values := strings.Split(encoded, ",")
	dates := make([]time.Time, 0, len(values))
	for _, value := range values {
		date, err := parseTripScheduleDate(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("invalid specific_dates value %q; use comma-separated YYYY-MM-DD dates: %w", value, err)
		}
		dates = append(dates, date)
	}
	return dates, nil
}

func parseDepartureTime(encoded string, date time.Time, location *time.Location) (time.Time, error) {
	var parsed time.Time
	var err error
	for _, layout := range []string{"15:04:05.999999999", "15:04:05"} {
		parsed, err = time.Parse(layout, encoded)
		if err == nil {
			return time.Date(date.Year(), date.Month(), date.Day(),
				parsed.Hour(), parsed.Minute(), parsed.Second(), parsed.Nanosecond(), location), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid departure_time %q: %w", encoded, err)
}

func parseTripScheduleDate(encoded string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", encoded)
	if err != nil {
		return time.Time{}, err
	}
	return dateOnly(parsed), nil
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func calendarDaysBetween(start, end time.Time) int64 {
	startUTC := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	endUTC := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	return int64(endUTC.Sub(startUTC).Hours() / 24)
}

func nextMidnight(now time.Time) time.Time {
	tomorrow := now.AddDate(0, 0, 1)
	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, now.Location())
}

func LoadTripGeneratorLocation() (*time.Location, error) {
	timezone := strings.TrimSpace(os.Getenv("TRIP_GENERATOR_TIMEZONE"))
	if timezone == "" {
		timezone = defaultTripGeneratorTimezone
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("load trip generator timezone %q: %w", timezone, err)
	}
	return location, nil
}
