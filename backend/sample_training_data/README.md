# ETA Training CSV Sample Pack

These CSV files contain deterministic, synthetic-but-realistic training data for the ETA/ETD model tables in `backend/migrations/004_create_eta_v3_tables.sql`.

Every CSV has 100 rows and uses the current project table format: UUIDs, ISO-8601 UTC timestamps, PostgreSQL array text for `primary_routes`, and generated columns omitted from import CSVs.

Files: `route_segments.csv`, `trip_contexts.csv`, `segment_performance_facts.csv`, `segment_aggregate_stats.csv`, `driver_performance_profiles.csv`, `bus_performance_profiles.csv`, `lounge_stop_metrics.csv`, `eta_predictions.csv`, and `segment_historical_performance.csv`.

Important: the UUIDs are internally consistent across the CSVs, but they are sample IDs. Replace them with real IDs from your database or seed matching reference rows before importing with foreign keys enabled.

For ETA training, join `segment_performance_facts` to `trip_contexts`. For ETD, train/predict lounge dwell time from `lounge_stop_metrics` and add it to the predicted arrival time.
