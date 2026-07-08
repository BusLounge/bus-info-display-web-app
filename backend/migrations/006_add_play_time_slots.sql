-- Add multi-slot support for advertisement play windows.
ALTER TABLE advertisements
  ADD COLUMN IF NOT EXISTS play_time_slots TEXT[] NOT NULL DEFAULT '{}';

-- Optional backfill from legacy single-string column if it exists in some environments.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'advertisements'
      AND column_name = 'play_time_slot'
  ) THEN
    EXECUTE $$
      UPDATE advertisements
      SET play_time_slots = CASE
        WHEN play_time_slot IS NULL OR btrim(play_time_slot) = '' THEN '{}'
        ELSE string_to_array(play_time_slot, ',')
      END
      WHERE play_time_slots = '{}'::text[];
    $$;
  END IF;
END $$;
