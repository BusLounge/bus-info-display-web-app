# Daily trip generator

The server starts a background worker that generates trips for the current
service date on startup and then at each midnight. The default timezone is
`Asia/Colombo`; set `TRIP_GENERATOR_TIMEZONE` to another IANA timezone if the
service operates in a different timezone.

Recurrence fields use these formats:

- `daily`: every date inside the inclusive `valid_from` / `valid_until` range.
- `weekly`: `recurrence_days` is a comma-separated list of weekday names or
  abbreviations, such as `Mon,Wed,Fri`.
- `interval`: `valid_from` is the inclusive anchor date and
  `recurrence_interval` is the number of days between departures.
- A non-empty `specific_dates` is a comma-separated `YYYY-MM-DD` allow-list
  that overrides the recurrence type, while still respecting validity dates.

Each selected schedule must have a positive `estimated_duration_minutes`.
The worker creates the scheduled trip using columns available on
`trip_schedules`. Bus, permit, driver, and conductor assignments are not stored
on that table, so the worker does not create an `active_trips` record;
dispatch can assign the bus and staff later.

The worker takes a transaction-scoped PostgreSQL advisory lock for each
schedule/departure pair before checking for or inserting a scheduled trip. This
makes startup runs, retries, process restarts, and multiple worker replicas
idempotent without requiring schema changes or additional indexes.
