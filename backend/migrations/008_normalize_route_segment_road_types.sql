BEGIN;

ALTER TABLE public.route_segments
  DROP CONSTRAINT IF EXISTS route_segments_road_type_check;

DO $$
DECLARE
  unsupported_count INTEGER;
BEGIN
  SELECT COUNT(*)
  INTO unsupported_count
  FROM public.route_segments
  WHERE road_type IS NOT NULL
    AND UPPER(BTRIM(road_type)) NOT IN (
      'HIGHWAY', 'EXPRESSWAY', 'ARTERIAL', 'COLLECTOR',
      'URBAN', 'RURAL', 'LOCAL', 'SERVICE', 'UNKNOWN',
      'MIXED', 'SUBURBAN'
    );

  IF unsupported_count > 0 THEN
    RAISE NOTICE 'Mapping % unsupported road_type value(s) to UNKNOWN', unsupported_count;
  END IF;
END $$;

UPDATE public.route_segments
SET road_type = CASE
  WHEN road_type IS NULL OR BTRIM(road_type) = '' THEN 'UNKNOWN'
  WHEN UPPER(BTRIM(road_type)) IN (
    'HIGHWAY', 'EXPRESSWAY', 'ARTERIAL', 'COLLECTOR',
    'URBAN', 'RURAL', 'LOCAL', 'SERVICE', 'UNKNOWN'
  ) THEN UPPER(BTRIM(road_type))
  ELSE 'UNKNOWN'
END;

ALTER TABLE public.route_segments
  ALTER COLUMN road_type SET DEFAULT 'UNKNOWN',
  ALTER COLUMN road_type SET NOT NULL;

ALTER TABLE public.route_segments
  ADD CONSTRAINT route_segments_road_type_check
  CHECK (road_type IN (
    'HIGHWAY', 'EXPRESSWAY', 'ARTERIAL', 'COLLECTOR',
    'URBAN', 'RURAL', 'LOCAL', 'SERVICE', 'UNKNOWN'
  ));

COMMIT;