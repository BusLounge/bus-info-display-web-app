BEGIN;

ALTER TABLE public.route_segments
  DROP CONSTRAINT IF EXISTS route_segments_road_type_check;

ALTER TABLE public.route_segments
  ADD CONSTRAINT route_segments_road_type_check
  CHECK (road_type IN (
    'HIGHWAY', 'EXPRESSWAY', 'ARTERIAL', 'COLLECTOR',
    'URBAN', 'RURAL', 'LOCAL', 'SERVICE'
  ));

ALTER TABLE public.route_segments
  ALTER COLUMN start_point_id DROP NOT NULL,
  ALTER COLUMN end_point_id DROP NOT NULL;

COMMIT;