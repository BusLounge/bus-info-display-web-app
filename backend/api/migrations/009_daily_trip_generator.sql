BEGIN;

ALTER TABLE public.trip_schedules
  ADD COLUMN IF NOT EXISTS bus_id uuid,
  ADD COLUMN IF NOT EXISTS driver_id uuid,
  ADD COLUMN IF NOT EXISTS conductor_id uuid;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'trip_schedules_bus_id_fkey'
      AND conrelid = 'public.trip_schedules'::regclass
  ) THEN
    ALTER TABLE public.trip_schedules
      ADD CONSTRAINT trip_schedules_bus_id_fkey
      FOREIGN KEY (bus_id) REFERENCES public.buses(id);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'trip_schedules_driver_id_fkey'
      AND conrelid = 'public.trip_schedules'::regclass
  ) THEN
    ALTER TABLE public.trip_schedules
      ADD CONSTRAINT trip_schedules_driver_id_fkey
      FOREIGN KEY (driver_id) REFERENCES public.bus_staff(id);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'trip_schedules_conductor_id_fkey'
      AND conrelid = 'public.trip_schedules'::regclass
  ) THEN
    ALTER TABLE public.trip_schedules
      ADD CONSTRAINT trip_schedules_conductor_id_fkey
      FOREIGN KEY (conductor_id) REFERENCES public.bus_staff(id);
  END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_scheduled_trips_schedule_departure
  ON public.scheduled_trips (trip_schedule_id, departure_datetime)
  WHERE trip_schedule_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_active_trips_scheduled_trip
  ON public.active_trips (scheduled_trip_id);

COMMENT ON COLUMN public.trip_schedules.recurrence_days IS
  'For weekly recurrence: comma-separated weekday names, e.g. Mon,Wed,Fri.';
COMMENT ON COLUMN public.trip_schedules.specific_dates IS
  'Optional comma-separated YYYY-MM-DD allow-list; when set it overrides recurrence_type for those dates.';
COMMENT ON COLUMN public.trip_schedules.valid_from IS
  'Inclusive schedule start date and anchor date for interval recurrence.';
COMMENT ON COLUMN public.trip_schedules.valid_until IS
  'Inclusive schedule end date.';

COMMIT;
