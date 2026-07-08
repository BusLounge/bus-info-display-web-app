-- ============================================================================
-- ETA ENGINE SCHEMA v3.0 (Supabase/PostgreSQL)
-- Creates: route_segments, trip_contexts, segment_performance_facts,
--          segment_aggregate_stats, driver_performance_profiles,
--          bus_performance_profiles, lounge_stop_metrics,
--          eta_predictions, segment_historical_performance
-- ============================================================================

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- Shared trigger function for updated_at maintenance
-- ============================================================================
CREATE OR REPLACE FUNCTION public.set_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$;

-- ============================================================================
-- 1) route_segments
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.route_segments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  master_route_id UUID NOT NULL REFERENCES public.master_routes(id) ON DELETE CASCADE,

  start_point_type VARCHAR(20) NOT NULL CHECK (start_point_type IN ('stop', 'lounge', 'origin')),
  start_point_id UUID NOT NULL,
  end_point_type VARCHAR(20) NOT NULL CHECK (end_point_type IN ('stop', 'lounge', 'destination')),
  end_point_id UUID NOT NULL,
  segment_order INTEGER NOT NULL CHECK (segment_order > 0),

  start_latitude NUMERIC(10, 7) NOT NULL,
  start_longitude NUMERIC(10, 7) NOT NULL,
  end_latitude NUMERIC(10, 7) NOT NULL,
  end_longitude NUMERIC(10, 7) NOT NULL,

  distance_km NUMERIC(8, 3) NOT NULL CHECK (distance_km > 0),
  baseline_duration_minutes INTEGER NOT NULL CHECK (baseline_duration_minutes > 0),
  baseline_speed_kmh NUMERIC(5, 2) DEFAULT 40.0 CHECK (baseline_speed_kmh > 0),

  road_type VARCHAR(20) CHECK (road_type IN ('highway', 'urban', 'rural', 'mixed')),
  traffic_sensitivity_factor NUMERIC(3, 2) DEFAULT 1.0 CHECK (traffic_sensitivity_factor > 0),
  encoded_polyline_segment TEXT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT uq_route_segments_route_order UNIQUE (master_route_id, segment_order)
);

CREATE INDEX IF NOT EXISTS idx_route_segments_route ON public.route_segments(master_route_id);
CREATE INDEX IF NOT EXISTS idx_route_segments_route_order ON public.route_segments(master_route_id, segment_order);
CREATE INDEX IF NOT EXISTS idx_route_segments_start_point ON public.route_segments(start_point_type, start_point_id);
CREATE INDEX IF NOT EXISTS idx_route_segments_end_point ON public.route_segments(end_point_type, end_point_id);

DROP TRIGGER IF EXISTS trg_route_segments_updated_at ON public.route_segments;
CREATE TRIGGER trg_route_segments_updated_at
BEFORE UPDATE ON public.route_segments
FOR EACH ROW
EXECUTE FUNCTION public.set_updated_at();

-- ============================================================================
-- 2) trip_contexts
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.trip_contexts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  active_trip_id UUID NOT NULL UNIQUE REFERENCES public.active_trips(id) ON DELETE CASCADE,
  scheduled_trip_id UUID REFERENCES public.scheduled_trips(id) ON DELETE SET NULL,
  trip_date DATE NOT NULL,

  driver_id UUID NOT NULL REFERENCES public.bus_staff(id) ON DELETE RESTRICT,
  driver_experience_years INTEGER CHECK (driver_experience_years IS NULL OR driver_experience_years >= 0),
  driver_rating NUMERIC(3, 2) CHECK (driver_rating IS NULL OR (driver_rating >= 0 AND driver_rating <= 5)),

  bus_id UUID NOT NULL REFERENCES public.buses(id) ON DELETE RESTRICT,
  bus_type VARCHAR(50),
  has_ac BOOLEAN NOT NULL DEFAULT FALSE,
  bus_age_years NUMERIC(4, 2) CHECK (bus_age_years IS NULL OR bus_age_years >= 0),

  departure_time TIME NOT NULL,
  time_of_day_category VARCHAR(20) NOT NULL CHECK (time_of_day_category IN ('early_morning', 'morning_peak', 'midday', 'evening_peak', 'night')),
  day_of_week VARCHAR(10) NOT NULL CHECK (day_of_week IN ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday')),
  is_weekend BOOLEAN NOT NULL DEFAULT FALSE,
  is_holiday BOOLEAN NOT NULL DEFAULT FALSE,

  weather_condition VARCHAR(20) NOT NULL DEFAULT 'clear' CHECK (weather_condition IN ('clear', 'rain', 'heavy_rain', 'fog')),
  temperature_celsius INTEGER,

  total_passengers INTEGER NOT NULL DEFAULT 0 CHECK (total_passengers >= 0),
  route_id UUID NOT NULL REFERENCES public.master_routes(id) ON DELETE RESTRICT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trip_contexts_trip ON public.trip_contexts(active_trip_id);
CREATE INDEX IF NOT EXISTS idx_trip_contexts_route ON public.trip_contexts(route_id);
CREATE INDEX IF NOT EXISTS idx_trip_contexts_driver ON public.trip_contexts(driver_id);
CREATE INDEX IF NOT EXISTS idx_trip_contexts_bus ON public.trip_contexts(bus_id);
CREATE INDEX IF NOT EXISTS idx_trip_contexts_lookup ON public.trip_contexts(route_id, time_of_day_category, day_of_week, trip_date DESC);

-- ============================================================================
-- 3) segment_performance_facts
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.segment_performance_facts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  route_segment_id UUID NOT NULL REFERENCES public.route_segments(id) ON DELETE CASCADE,
  trip_context_id UUID NOT NULL REFERENCES public.trip_contexts(id) ON DELETE CASCADE,

  segment_start_time TIMESTAMPTZ NOT NULL,
  segment_end_time TIMESTAMPTZ NOT NULL,
  actual_duration_minutes NUMERIC(8, 2) NOT NULL CHECK (actual_duration_minutes >= 0),

  average_speed_kmh NUMERIC(6, 2) CHECK (average_speed_kmh IS NULL OR average_speed_kmh >= 0),
  duration_variance_minutes NUMERIC(8, 2),
  traffic_level VARCHAR(20) CHECK (traffic_level IS NULL OR traffic_level IN ('light', 'moderate', 'heavy', 'congested')),
  actual_distance_km NUMERIC(8, 3) CHECK (actual_distance_km IS NULL OR actual_distance_km >= 0),
  gps_accuracy_meters INTEGER CHECK (gps_accuracy_meters IS NULL OR gps_accuracy_meters >= 0),

  data_quality_score NUMERIC(3, 2) NOT NULL CHECK (data_quality_score >= 0 AND data_quality_score <= 1),
  recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT chk_segment_perf_time_order CHECK (segment_end_time > segment_start_time)
);

CREATE INDEX IF NOT EXISTS idx_segment_facts_segment ON public.segment_performance_facts(route_segment_id);
CREATE INDEX IF NOT EXISTS idx_segment_facts_trip_context ON public.segment_performance_facts(trip_context_id);
CREATE INDEX IF NOT EXISTS idx_segment_facts_time ON public.segment_performance_facts(segment_start_time DESC);
CREATE INDEX IF NOT EXISTS idx_segment_facts_lookup ON public.segment_performance_facts(route_segment_id, trip_context_id, recorded_at DESC);

-- ============================================================================
-- 4) segment_aggregate_stats
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.segment_aggregate_stats (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  route_segment_id UUID NOT NULL REFERENCES public.route_segments(id) ON DELETE CASCADE,

  time_of_day_category VARCHAR(20) NOT NULL CHECK (time_of_day_category IN ('early_morning', 'morning_peak', 'midday', 'evening_peak', 'night')),
  day_of_week VARCHAR(10) NOT NULL CHECK (day_of_week IN ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday')),
  weather_condition VARCHAR(20) NOT NULL CHECK (weather_condition IN ('clear', 'rain', 'heavy_rain', 'fog')),

  avg_duration_minutes NUMERIC(8, 2) NOT NULL CHECK (avg_duration_minutes >= 0),
  median_duration_minutes NUMERIC(8, 2),
  stddev_duration_minutes NUMERIC(8, 2),
  min_duration_minutes NUMERIC(8, 2),
  max_duration_minutes NUMERIC(8, 2),
  p95_duration_minutes NUMERIC(8, 2),
  avg_speed_kmh NUMERIC(6, 2),

  sample_count INTEGER NOT NULL CHECK (sample_count >= 0),
  last_trip_date DATE,
  avg_data_quality NUMERIC(3, 2) CHECK (avg_data_quality IS NULL OR (avg_data_quality >= 0 AND avg_data_quality <= 1)),

  last_calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  calculation_window_days INTEGER NOT NULL DEFAULT 90 CHECK (calculation_window_days > 0),

  CONSTRAINT uq_segment_aggregate_stats UNIQUE (route_segment_id, time_of_day_category, day_of_week, weather_condition)
);

CREATE INDEX IF NOT EXISTS idx_segment_agg_segment ON public.segment_aggregate_stats(route_segment_id);
CREATE INDEX IF NOT EXISTS idx_segment_agg_lookup ON public.segment_aggregate_stats(route_segment_id, time_of_day_category, day_of_week, weather_condition);

-- ============================================================================
-- 5) driver_performance_profiles
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.driver_performance_profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  driver_id UUID NOT NULL UNIQUE REFERENCES public.bus_staff(id) ON DELETE CASCADE,

  avg_speed_factor NUMERIC(4, 3) NOT NULL DEFAULT 1.000 CHECK (avg_speed_factor > 0),
  punctuality_score NUMERIC(3, 2) NOT NULL DEFAULT 0.50 CHECK (punctuality_score >= 0 AND punctuality_score <= 1),
  consistency_score NUMERIC(3, 2) NOT NULL DEFAULT 0.50 CHECK (consistency_score >= 0 AND consistency_score <= 1),

  total_trips_analyzed INTEGER NOT NULL DEFAULT 0 CHECK (total_trips_analyzed >= 0),
  total_segments_analyzed INTEGER NOT NULL DEFAULT 0 CHECK (total_segments_analyzed >= 0),
  primary_routes UUID[],

  last_calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  calculation_window_days INTEGER NOT NULL DEFAULT 90 CHECK (calculation_window_days > 0)
);

CREATE INDEX IF NOT EXISTS idx_driver_profiles_driver ON public.driver_performance_profiles(driver_id);

-- ============================================================================
-- 6) bus_performance_profiles
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.bus_performance_profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  bus_id UUID NOT NULL UNIQUE REFERENCES public.buses(id) ON DELETE CASCADE,

  avg_speed_factor NUMERIC(4, 3) NOT NULL DEFAULT 1.000 CHECK (avg_speed_factor > 0),
  reliability_score NUMERIC(3, 2) NOT NULL DEFAULT 0.50 CHECK (reliability_score >= 0 AND reliability_score <= 1),
  fuel_efficiency_kmpl NUMERIC(6, 2) CHECK (fuel_efficiency_kmpl IS NULL OR fuel_efficiency_kmpl >= 0),
  days_since_last_service INTEGER CHECK (days_since_last_service IS NULL OR days_since_last_service >= 0),
  breakdown_count_90d INTEGER NOT NULL DEFAULT 0 CHECK (breakdown_count_90d >= 0),

  total_trips_analyzed INTEGER NOT NULL DEFAULT 0 CHECK (total_trips_analyzed >= 0),
  total_km_traveled NUMERIC(10, 2) NOT NULL DEFAULT 0 CHECK (total_km_traveled >= 0),

  last_calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  calculation_window_days INTEGER NOT NULL DEFAULT 90 CHECK (calculation_window_days > 0)
);

CREATE INDEX IF NOT EXISTS idx_bus_profiles_bus ON public.bus_performance_profiles(bus_id);

-- ============================================================================
-- 7) lounge_stop_metrics
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.lounge_stop_metrics (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lounge_id UUID NOT NULL REFERENCES public.lounges(id) ON DELETE CASCADE,
  master_route_id UUID NOT NULL REFERENCES public.master_routes(id) ON DELETE CASCADE,

  average_dwell_time_minutes NUMERIC(6, 2) NOT NULL CHECK (average_dwell_time_minutes >= 0),
  min_dwell_time_minutes NUMERIC(6, 2) NOT NULL CHECK (min_dwell_time_minutes >= 0),
  max_dwell_time_minutes NUMERIC(6, 2) NOT NULL CHECK (max_dwell_time_minutes >= min_dwell_time_minutes),
  peak_hour_dwell_time_minutes NUMERIC(6, 2) NOT NULL CHECK (peak_hour_dwell_time_minutes >= 0),
  off_peak_dwell_time_minutes NUMERIC(6, 2) NOT NULL CHECK (off_peak_dwell_time_minutes >= 0),

  average_bookings_per_trip INTEGER NOT NULL CHECK (average_bookings_per_trip >= 0),
  total_stops_recorded INTEGER NOT NULL CHECK (total_stops_recorded >= 0),
  last_calculated_at TIMESTAMPTZ,
  data_points_count INTEGER NOT NULL CHECK (data_points_count >= 0),

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT uq_lounge_stop_metrics UNIQUE (lounge_id, master_route_id)
);

CREATE INDEX IF NOT EXISTS idx_lounge_metrics_lounge ON public.lounge_stop_metrics(lounge_id);
CREATE INDEX IF NOT EXISTS idx_lounge_metrics_route ON public.lounge_stop_metrics(master_route_id);

DROP TRIGGER IF EXISTS trg_lounge_stop_metrics_updated_at ON public.lounge_stop_metrics;
CREATE TRIGGER trg_lounge_stop_metrics_updated_at
BEFORE UPDATE ON public.lounge_stop_metrics
FOR EACH ROW
EXECUTE FUNCTION public.set_updated_at();

-- ============================================================================
-- 8) eta_predictions
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.eta_predictions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  active_trip_id UUID NOT NULL REFERENCES public.active_trips(id) ON DELETE CASCADE,

  lounge_id UUID REFERENCES public.lounges(id) ON DELETE SET NULL,
  stop_id UUID REFERENCES public.master_route_stops(id) ON DELETE SET NULL,
  prediction_type VARCHAR(20) NOT NULL CHECK (prediction_type IN ('lounge', 'stop', 'destination')),

  predicted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  predicted_arrival_time TIMESTAMPTZ NOT NULL,
  current_location_lat NUMERIC(10, 7),
  current_location_lng NUMERIC(10, 7),
  distance_remaining_km NUMERIC(8, 3) CHECK (distance_remaining_km IS NULL OR distance_remaining_km >= 0),

  actual_arrival_time TIMESTAMPTZ,
  prediction_error_minutes NUMERIC(8, 2) GENERATED ALWAYS AS (
    CASE
      WHEN actual_arrival_time IS NULL THEN NULL
      ELSE EXTRACT(EPOCH FROM (actual_arrival_time - predicted_arrival_time)) / 60.0
    END
  ) STORED,

  calculation_method VARCHAR(20) NOT NULL CHECK (calculation_method IN ('baseline', 'historical', 'ml', 'hybrid', 'realtime')),
  confidence_score NUMERIC(5, 2) NOT NULL CHECK (confidence_score >= 0 AND confidence_score <= 100),
  context_data JSONB,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT chk_eta_prediction_target_match CHECK (
    (prediction_type = 'lounge' AND lounge_id IS NOT NULL AND stop_id IS NULL)
    OR
    (prediction_type = 'stop' AND stop_id IS NOT NULL AND lounge_id IS NULL)
    OR
    (prediction_type = 'destination' AND lounge_id IS NULL AND stop_id IS NULL)
  )
);

CREATE INDEX IF NOT EXISTS idx_eta_predictions_trip ON public.eta_predictions(active_trip_id);
CREATE INDEX IF NOT EXISTS idx_eta_predictions_lounge ON public.eta_predictions(lounge_id);
CREATE INDEX IF NOT EXISTS idx_eta_predictions_stop ON public.eta_predictions(stop_id);
CREATE INDEX IF NOT EXISTS idx_eta_predictions_predicted_at ON public.eta_predictions(predicted_at DESC);
CREATE INDEX IF NOT EXISTS idx_eta_predictions_context_gin ON public.eta_predictions USING GIN (context_data);

-- ============================================================================
-- 9) segment_historical_performance (legacy)
-- ============================================================================
CREATE TABLE IF NOT EXISTS public.segment_historical_performance (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  route_segment_id UUID NOT NULL REFERENCES public.route_segments(id) ON DELETE CASCADE,
  active_trip_id UUID REFERENCES public.active_trips(id) ON DELETE SET NULL,
  scheduled_trip_id UUID REFERENCES public.scheduled_trips(id) ON DELETE SET NULL,
  bus_id UUID REFERENCES public.buses(id) ON DELETE SET NULL,
  driver_id UUID REFERENCES public.bus_staff(id) ON DELETE SET NULL,

  trip_date DATE NOT NULL,
  segment_start_time TIMESTAMPTZ NOT NULL,
  segment_end_time TIMESTAMPTZ NOT NULL,
  actual_duration_minutes NUMERIC(8, 2) NOT NULL CHECK (actual_duration_minutes >= 0),
  estimated_duration_minutes NUMERIC(8, 2),
  duration_variance_minutes NUMERIC(8, 2) GENERATED ALWAYS AS (
    actual_duration_minutes - estimated_duration_minutes
  ) STORED,

  average_speed_kmh NUMERIC(5, 2),
  max_speed_kmh NUMERIC(5, 2),
  min_speed_kmh NUMERIC(5, 2),
  stop_count INTEGER CHECK (stop_count IS NULL OR stop_count >= 0),
  dwell_time_seconds INTEGER CHECK (dwell_time_seconds IS NULL OR dwell_time_seconds >= 0),

  time_of_day_category VARCHAR(20) CHECK (time_of_day_category IS NULL OR time_of_day_category IN ('early_morning', 'morning_peak', 'midday', 'evening_peak', 'night')),
  day_of_week VARCHAR(10) CHECK (day_of_week IS NULL OR day_of_week IN ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday')),
  is_holiday BOOLEAN,
  weather_condition VARCHAR(20) CHECK (weather_condition IS NULL OR weather_condition IN ('clear', 'rain', 'heavy_rain', 'fog')),
  bus_type VARCHAR(50),
  bus_occupancy_percentage INTEGER CHECK (bus_occupancy_percentage IS NULL OR (bus_occupancy_percentage >= 0 AND bus_occupancy_percentage <= 100)),
  driver_experience_years INTEGER CHECK (driver_experience_years IS NULL OR driver_experience_years >= 0),

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT chk_segment_hist_time_order CHECK (segment_end_time > segment_start_time)
);

CREATE INDEX IF NOT EXISTS idx_segment_hist_segment ON public.segment_historical_performance(route_segment_id);
CREATE INDEX IF NOT EXISTS idx_segment_hist_trip_date ON public.segment_historical_performance(trip_date DESC);
CREATE INDEX IF NOT EXISTS idx_segment_hist_trip ON public.segment_historical_performance(active_trip_id);
CREATE INDEX IF NOT EXISTS idx_segment_hist_context ON public.segment_historical_performance(route_segment_id, time_of_day_category, day_of_week, trip_date DESC);

COMMIT;
