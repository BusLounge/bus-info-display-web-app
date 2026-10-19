package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// TripSchedule contains the fields needed to decide whether a base schedule
// runs and to create its daily trip.
type TripSchedule struct {
	ID                       string
	BusOwnerID               string
	ScheduleName             sql.NullString
	RecurrenceType           string
	DepartureTime            string
	BaseFare                 string
	ValidFrom                sql.NullString
	ValidUntil               sql.NullString
	RecurrenceInterval       sql.NullInt64
	BusOwnerRouteID          sql.NullString
	RecurrenceDays           sql.NullString
	SpecificDates            sql.NullString
	EstimatedDurationMinutes sql.NullInt64
}

type TripScheduleRepository struct {
	db *sql.DB
}

func NewTripScheduleRepository(db *sql.DB) *TripScheduleRepository {
	return &TripScheduleRepository{db: db}
}

func (r *TripScheduleRepository) ListActiveTripSchedules(ctx context.Context) ([]TripSchedule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			ts.id::text,
			ts.bus_owner_id::text,
			ts.schedule_name,
			ts.recurrence_type,
			ts.departure_time::text,
			ts.base_fare::text,
			ts.valid_from::text,
			ts.valid_until::text,
			ts.recurrence_interval,
			ts.bus_owner_route_id::text,
			ts.recurrence_days,
			ts.specific_dates,
			ts.estimated_duration_minutes
		FROM public.trip_schedules ts
		WHERE ts.is_active IS TRUE
		ORDER BY ts.id
	`)
	if err != nil {
		return nil, fmt.Errorf("query active trip schedules: %w", err)
	}
	defer rows.Close()

	schedules := make([]TripSchedule, 0)
	for rows.Next() {
		var schedule TripSchedule
		if err := rows.Scan(
			&schedule.ID,
			&schedule.BusOwnerID,
			&schedule.ScheduleName,
			&schedule.RecurrenceType,
			&schedule.DepartureTime,
			&schedule.BaseFare,
			&schedule.ValidFrom,
			&schedule.ValidUntil,
			&schedule.RecurrenceInterval,
			&schedule.BusOwnerRouteID,
			&schedule.RecurrenceDays,
			&schedule.SpecificDates,
			&schedule.EstimatedDurationMinutes,
		); err != nil {
			return nil, fmt.Errorf("scan active trip schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active trip schedules: %w", err)
	}
	return schedules, nil
}

// GenerateTrip inserts a scheduled trip. Bus and staff assignments are not
// stored on trip_schedules and are left for dispatch to set later.
func (r *TripScheduleRepository) GenerateTrip(
	ctx context.Context,
	schedule TripSchedule,
	departure time.Time,
	durationMinutes int64,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin trip generation transaction: %w", err)
	}
	defer tx.Rollback()

	lockKey := schedule.ID + "/" + departure.Format("2006-01-02 15:04:05.999999999")
	if _, err := tx.ExecContext(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended($1, 0))
	`, lockKey); err != nil {
		return fmt.Errorf("lock trip generation for schedule %s: %w", schedule.ID, err)
	}

	var scheduledTripID string
	err = tx.QueryRowContext(ctx, `
		SELECT id::text
		FROM public.scheduled_trips
		WHERE trip_schedule_id = $1::uuid
			AND departure_datetime = $2
	`, schedule.ID, departure).Scan(&scheduledTripID)
	if err == sql.ErrNoRows {
		err = tx.QueryRowContext(ctx, `
			INSERT INTO public.scheduled_trips (
				trip_schedule_id,
				base_fare,
				bus_owner_route_id,
				departure_datetime,
				estimated_duration_minutes
			)
			VALUES ($1::uuid, $2::numeric, NULLIF($3, '')::uuid, $4, $5)
			RETURNING id::text
		`,
			schedule.ID,
			schedule.BaseFare,
			nullStringValue(schedule.BusOwnerRouteID),
			departure,
			durationMinutes,
		).Scan(&scheduledTripID)
	}
	if err != nil {
		return fmt.Errorf("insert or find scheduled trip for schedule %s at %s: %w",
			schedule.ID, departure.Format(time.RFC3339), err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit generated trip for schedule %s: %w", schedule.ID, err)
	}
	return nil
}

func nullStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
