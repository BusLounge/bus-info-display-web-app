-- Check master_route_stops for the active route
SELECT 
    mrs.id,
    mrs.route_id,
    mrs.stop_name,
    mrs.arrival_time_offset_minutes,
    l.lounge_name,
    l.lounge_id
FROM master_route_stops mrs
LEFT JOIN lounges l ON (
    LOWER(mrs.stop_name) LIKE '%' || LOWER(l.lounge_name) || '%' 
    OR LOWER(l.lounge_name) LIKE '%' || LOWER(mrs.stop_name) || '%'
)
WHERE mrs.route_id = 'a4f75c03-3a1e-46d5-90fd-91a1a5e18c30'
ORDER BY mrs.arrival_time_offset_minutes
LIMIT 20;

-- Check if arrival_time_offset_minutes column exists and has data  
SELECT 
    stop_name,
    arrival_time_offset_minutes
FROM master_route_stops
WHERE route_id = 'a4f75c03-3a1e-46d5-90fd-91a1a5e18c30'
AND arrival_time_offset_minutes IS NOT NULL
ORDER BY arrival_time_offset_minutes;
