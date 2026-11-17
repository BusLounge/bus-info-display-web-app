-- Sample GPS Update for Active Trips
-- This will add mock GPS coordinates to make ETA calculations work

-- Update the active trip with sample GPS coordinates
-- This simulates a bus traveling from Colombo towards Kandy

UPDATE active_trips 
SET 
    current_latitude = 7.1500,  -- Somewhere between Colombo and Kandy
    current_longitude = 80.2000,
    current_speed_kmh = 55.0,   -- Traveling at 55 km/h
    last_location_update = NOW()
WHERE id = 'fe6dcdb0-0a14-4369-a511-90b0d5188a2b';

-- Verify the update
SELECT 
    id,
    current_latitude,
    current_longitude,
    current_speed_kmh,
    last_location_update,
    status
FROM active_trips
WHERE id = 'fe6dcdb0-0a14-4369-a511-90b0d5188a2b';

-- Alternative: If you want to add GPS to ALL active trips for today
-- UPDATE active_trips at
-- SET 
--     current_latitude = 7.1500 + (RANDOM() * 0.5 - 0.25),  -- Random nearby location
--     current_longitude = 80.2000 + (RANDOM() * 0.5 - 0.25),
--     current_speed_kmh = 40.0 + (RANDOM() * 30),  -- Random speed 40-70 km/h
--     last_location_update = NOW()
-- WHERE EXISTS (
--     SELECT 1 FROM scheduled_trips st 
--     WHERE st.id = at.scheduled_trip_id 
--     AND DATE(st.departure_datetime) = CURRENT_DATE
-- );
