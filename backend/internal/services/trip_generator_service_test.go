package services

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"bus-schedule-lounge/internal/database"
	wsupdates "bus-schedule-lounge/internal/websocket"
)

type tripGeneratorRepositoryStub struct {
	schedules          []database.TripSchedule
	generatedDurations []int64
}

func (r *tripGeneratorRepositoryStub) ListActiveTripSchedules(context.Context) ([]database.TripSchedule, error) {
	return r.schedules, nil
}

func (r *tripGeneratorRepositoryStub) GenerateTrip(
	_ context.Context,
	_ database.TripSchedule,
	_ time.Time,
	durationMinutes int64,
) error {
	r.generatedDurations = append(r.generatedDurations, durationMinutes)
	return nil
}

func (r *tripGeneratorRepositoryStub) GetLoungeIDsByBusOwnerRouteID(context.Context, string) ([]string, error) {
	return nil, nil
}

func TestGenerateForDateFallsBackToRouteDuration(t *testing.T) {
	repository := &tripGeneratorRepositoryStub{
		schedules: []database.TripSchedule{{
			ID:                           "schedule-1",
			RecurrenceType:               "daily",
			DepartureTime:                "08:30:00",
			EstimatedDurationMinutes:     sql.NullInt64{Int64: 0, Valid: true},
			RouteEstimatedDurationMinutes: sql.NullInt64{Int64: 95, Valid: true},
		}},
	}
	location := time.FixedZone("service", 5*60*60+30*60)
	service := NewTripGeneratorService(repository, location, wsupdates.NewHub(nil))
	serviceDate := time.Date(2026, 11, 11, 0, 0, 0, 0, location)

	if err := service.GenerateForDate(context.Background(), serviceDate); err != nil {
		t.Fatalf("GenerateForDate() error = %v", err)
	}
	if len(repository.generatedDurations) != 1 || repository.generatedDurations[0] != 95 {
		t.Fatalf("generated durations = %v, want [95]", repository.generatedDurations)
	}
}

func TestGenerateForDateSkipsWhenBothDurationsAreInvalid(t *testing.T) {
	repository := &tripGeneratorRepositoryStub{
		schedules: []database.TripSchedule{{
			ID:                           "schedule-1",
			RecurrenceType:               "daily",
			DepartureTime:                "08:30:00",
			EstimatedDurationMinutes:     sql.NullInt64{Int64: 0, Valid: true},
			RouteEstimatedDurationMinutes: sql.NullInt64{Int64: -10, Valid: true},
		}},
	}
	location := time.FixedZone("service", 5*60*60+30*60)
	service := NewTripGeneratorService(repository, location, wsupdates.NewHub(nil))
	serviceDate := time.Date(2026, 11, 11, 0, 0, 0, 0, location)

	err := service.GenerateForDate(context.Background(), serviceDate)
	if err == nil || !strings.Contains(err.Error(), "no positive estimated_duration_minutes") {
		t.Fatalf("GenerateForDate() error = %v, want missing-duration error", err)
	}
	if len(repository.generatedDurations) != 0 {
		t.Fatalf("generated durations = %v, want no generated trips", repository.generatedDurations)
	}
}

func TestScheduleDurationUsesScheduleBeforeRoute(t *testing.T) {
	tests := []struct {
		name        string
		schedule    sql.NullInt64
		route       sql.NullInt64
		want        int64
		wantValid   bool
	}{
		{
			name:      "prefer positive schedule duration",
			schedule:  sql.NullInt64{Int64: 60, Valid: true},
			route:     sql.NullInt64{Int64: 95, Valid: true},
			want:      60,
			wantValid: true,
		},
		{
			name:      "fallback when schedule duration is null",
			schedule:  sql.NullInt64{},
			route:     sql.NullInt64{Int64: 95, Valid: true},
			want:      95,
			wantValid: true,
		},
		{
			name:      "fallback when schedule duration is negative",
			schedule:  sql.NullInt64{Int64: -5, Valid: true},
			route:     sql.NullInt64{Int64: 95, Valid: true},
			want:      95,
			wantValid: true,
		},
		{
			name:      "invalid when both durations are missing or non-positive",
			schedule:  sql.NullInt64{Int64: 0, Valid: true},
			route:     sql.NullInt64{Int64: -10, Valid: true},
			wantValid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := scheduleDurationMinutes(database.TripSchedule{
				EstimatedDurationMinutes:     test.schedule,
				RouteEstimatedDurationMinutes: test.route,
			})
			if got != test.want || ok != test.wantValid {
				t.Fatalf("scheduleDurationMinutes() = (%d, %v), want (%d, %v)",
					got, ok, test.want, test.wantValid)
			}
		})
	}
}

func TestScheduleRunsOn(t *testing.T) {
	date := func(value string) time.Time {
		parsed, err := time.Parse("2006-01-02", value)
		if err != nil {
			t.Fatal(err)
		}
		return parsed
	}

	tests := []struct {
		name     string
		schedule database.TripSchedule
		date     string
		want     bool
		wantErr  bool
	}{
		{
			name:     "daily within inclusive validity",
			schedule: database.TripSchedule{RecurrenceType: "daily", ValidFrom: sql.NullString{String: "2026-10-01", Valid: true}, ValidUntil: sql.NullString{String: "2026-10-31", Valid: true}},
			date:     "2026-10-15",
			want:     true,
		},
		{
			name:     "inclusive start date in service timezone",
			schedule: database.TripSchedule{RecurrenceType: "daily", ValidFrom: sql.NullString{String: "2026-10-15", Valid: true}},
			date:     "2026-10-15",
			want:     true,
		},
		{
			name:     "outside valid until",
			schedule: database.TripSchedule{RecurrenceType: "daily", ValidUntil: sql.NullString{String: "2026-10-14", Valid: true}},
			date:     "2026-10-15",
			want:     false,
		},
		{
			name:     "weekly matching day",
			schedule: database.TripSchedule{RecurrenceType: "weekly", RecurrenceDays: sql.NullString{String: "Mon, Wed, Fri", Valid: true}},
			date:     "2026-10-16",
			want:     true,
		},
		{
			name:     "weekly non-matching day",
			schedule: database.TripSchedule{RecurrenceType: "weekly", RecurrenceDays: sql.NullString{String: "Mon,Wed,Fri", Valid: true}},
			date:     "2026-10-15",
			want:     false,
		},
		{
			name:     "weekly numeric weekdays",
			schedule: database.TripSchedule{RecurrenceType: "weekly", RecurrenceDays: sql.NullString{String: "1,3,5", Valid: true}},
			date:     "2026-10-16",
			want:     true,
		},
		{
			name:     "weekly numeric Sunday",
			schedule: database.TripSchedule{RecurrenceType: "weekly", RecurrenceDays: sql.NullString{String: "0", Valid: true}},
			date:     "2026-10-18",
			want:     true,
		},
		{
			name:     "weekly mixed numeric and named weekdays",
			schedule: database.TripSchedule{RecurrenceType: "weekly", RecurrenceDays: sql.NullString{String: "0,Mon,Fri", Valid: true}},
			date:     "2026-10-19",
			want:     true,
		},
		{
			name:     "weekly invalid numeric weekday",
			schedule: database.TripSchedule{RecurrenceType: "weekly", RecurrenceDays: sql.NullString{String: "7", Valid: true}},
			date:     "2026-10-16",
			wantErr:  true,
		},
		{
			name: "specific date overrides recurrence",
			schedule: database.TripSchedule{
				RecurrenceType: "weekly",
				RecurrenceDays: sql.NullString{String: "Mon", Valid: true},
				SpecificDates:  sql.NullString{String: "2026-10-15,2026-10-20", Valid: true},
			},
			date: "2026-10-15",
			want: true,
		},
		{
			name: "interval anchored to valid from",
			schedule: database.TripSchedule{
				RecurrenceType:     "interval",
				ValidFrom:          sql.NullString{String: "2026-10-01", Valid: true},
				RecurrenceInterval: sql.NullInt64{Int64: 3, Valid: true},
			},
			date: "2026-10-07",
			want: true,
		},
		{
			name: "interval skips intervening days",
			schedule: database.TripSchedule{
				RecurrenceType:     "interval",
				ValidFrom:          sql.NullString{String: "2026-10-01", Valid: true},
				RecurrenceInterval: sql.NullInt64{Int64: 3, Valid: true},
			},
			date: "2026-10-06",
			want: false,
		},
		{
			name:    "weekly without days is invalid",
			schedule: database.TripSchedule{RecurrenceType: "weekly"},
			date:    "2026-10-15",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := scheduleRunsOn(test.schedule, date(test.date))
			if (err != nil) != test.wantErr {
				t.Fatalf("scheduleRunsOn() error = %v, wantErr %v", err, test.wantErr)
			}
			if err == nil && got != test.want {
				t.Errorf("scheduleRunsOn() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestParseDepartureTime(t *testing.T) {
	location, err := time.LoadLocation("Asia/Colombo")
	if err != nil {
		t.Fatal(err)
	}
	date, err := time.Parse("2006-01-02", "2026-10-15")
	if err != nil {
		t.Fatal(err)
	}

	got, err := parseDepartureTime("08:30:15.250000", date, location)
	if err != nil {
		t.Fatal(err)
	}
	if got.Location() != location || got.Hour() != 8 || got.Minute() != 30 || got.Second() != 15 || got.Nanosecond() != 250000000 {
		t.Errorf("parseDepartureTime() = %s, want 2026-10-15 08:30:15.25 in %s", got, location)
	}
}

func TestNextMidnight(t *testing.T) {
	location, err := time.LoadLocation("Asia/Colombo")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 15, 23, 59, 59, 0, location)
	got := nextMidnight(now)
	want := time.Date(2026, 10, 16, 0, 0, 0, 0, location)
	if !got.Equal(want) {
		t.Errorf("nextMidnight() = %s, want %s", got, want)
	}
}
