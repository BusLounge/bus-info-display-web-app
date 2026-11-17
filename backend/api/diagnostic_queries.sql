-- Diagnostic queries for Arrival Management debugging

-- 1. Check if we have any active trips today
SELECT 
    COUNT(*) as total_active_trips,
    COUNT(CASE WHEN current_latitude IS NOT NULL THEN 1 END) as with_gps,
    COUNT(CASE WHEN current_latitude IS NULL THEN 1 END) as without_gps
FROM active_trips at
JOIN scheduled_trips st ON at.scheduled_trip_id = st.id
WHERE DATE(st.departure_datetime) = CURRENT_DATE;

-- 2. Check active trip statuses
SELECT status, COUNT(*) as count
FROM active_trips at
JOIN scheduled_trips st ON at.scheduled_trip_id = st.id
WHERE DATE(st.departure_datetime) = CURRENT_DATE
GROUP BY status;

-- 3. Check if lounges are operational
SELECT 
    COUNT(*) as total_lounges,
    COUNT(CASE WHEN is_operational = true THEN 1 END) as operational,
    COUNT(CASE WHEN latitude IS NOT NULL AND longitude IS NOT NULL THEN 1 END) as with_coordinates
FROM lounges;

-- 4. Check scheduled trips for today
SELECT COUNT(*) as scheduled_trips_today
FROM scheduled_trips
WHERE DATE(departure_datetime) = CURRENT_DATE;

-- 5. Full diagnostic - what's missing?
SELECT 
    'Active Trips Today' as check_item,
    COUNT(*) as count
FROM active_trips at
JOIN scheduled_trips st ON at.scheduled_trip_id = st.id
WHERE DATE(st.departure_datetime) = CURRENT_DATE

UNION ALL

SELECT 
    'Active Trips with GPS',
    COUNT(*)
FROM active_trips
WHERE current_latitude IS NOT NULL AND current_longitude IS NOT NULL

UNION ALL

SELECT 
    'Operational Lounges',
    COUNT(*)
FROM lounges
WHERE is_operational = true

UNION ALL

SELECT 
    'Lounges with Coordinates',
    COUNT(*)
FROM lounges
WHERE latitude IS NOT NULL AND longitude IS NOT NULL

UNION ALL

SELECT 
    'Lounge Routes',
    COUNT(*)
FROM lounge_routes;

-- 6. Sample data check - see what we have
SELECT 
    at.id as trip_id,
    at.status,
    at.current_latitude,
    at.current_longitude,
    st.departure_datetime,
    b.bus_number
FROM active_trips at
JOIN scheduled_trips st ON at.scheduled_trip_id = st.id
JOIN buses b ON at.bus_id = b.id
ORDER BY st.departure_datetime DESC
LIMIT 5;
