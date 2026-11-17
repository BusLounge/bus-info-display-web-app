const fs = require('fs');
const path = require('path');

const base = __dirname;
const uuid = (n) => `00000000-0000-0000-0000-${String(n).padStart(12, '0')}`;
const iso = (value) => new Date(value).toISOString().replace('.000Z', 'Z');
const addMinutes = (date, minutes) => new Date(date.getTime() + minutes * 60000);
const days = ['sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday'];
const bucketFor = (time) => {
  const hour = Number(time.slice(0, 2));
  if (hour < 6) return 'early_morning';
  if (hour < 10) return 'morning_peak';
  if (hour < 16) return 'midday';
  if (hour < 20) return 'evening_peak';
  return 'night';
};
const weatherFor = (i) => (i % 29 === 0 ? 'heavy_rain' : i % 13 === 0 ? 'fog' : i % 5 === 0 ? 'rain' : 'clear');
const roadTypeFor = (i) => ['urban', 'mixed', 'rural', 'highway', 'mixed'][i % 5];
const trafficFor = (bucket, weather, i) => {
  if (weather === 'heavy_rain') return 'congested';
  if (bucket.includes('peak')) return i % 3 === 0 ? 'heavy' : 'moderate';
  if (weather === 'rain' || weather === 'fog' || i % 7 === 0) return 'moderate';
  return 'light';
};
const durationFactor = (traffic, weather, bucket, driverFactor, busFactor) => {
  const trafficFactor = { light: 0.94, moderate: 1.08, heavy: 1.25, congested: 1.48 }[traffic] || 1;
  const weatherFactor = { clear: 1, rain: 1.1, heavy_rain: 1.24, fog: 1.14 }[weather] || 1;
  const peakFactor = bucket.includes('peak') ? 1.08 : 1;
  return trafficFactor * weatherFactor * peakFactor * driverFactor * busFactor;
};
const csv = (rows) => {
  const headers = Object.keys(rows[0]);
  const escape = (value) => {
    if (value === null || value === undefined) return '';
    const text = String(value);
    return /[",\n\r]/.test(text) ? `"${text.replace(/"/g, '""')}"` : text;
  };
  return `${headers.join(',')}\n${rows.map((row) => headers.map((h) => escape(row[h])).join(',')).join('\n')}\n`;
};
const write = (name, rows) => fs.writeFileSync(path.join(base, name), csv(rows), 'utf8');

const routes = [
  { id: uuid(1001), name: 'Colombo - Kandy', lat1: 6.9271, lng1: 79.8612, lat2: 7.2906, lng2: 80.6337, dist: 116 },
  { id: uuid(1002), name: 'Kandy - Colombo', lat1: 7.2906, lng1: 80.6337, lat2: 6.9271, lng2: 79.8612, dist: 116 },
  { id: uuid(1003), name: 'Colombo - Galle', lat1: 6.9271, lng1: 79.8612, lat2: 6.0535, lng2: 80.221, dist: 126 },
  { id: uuid(1004), name: 'Colombo - Jaffna', lat1: 6.9271, lng1: 79.8612, lat2: 9.6615, lng2: 80.0255, dist: 395 },
  { id: uuid(1005), name: 'Matara - Kandy', lat1: 5.9549, lng1: 80.555, lat2: 7.2906, lng2: 80.6337, dist: 245 },
];

const routeSegments = [];
for (let routeIndex = 0; routeIndex < routes.length; routeIndex += 1) {
  const route = routes[routeIndex];
  const routeNo = routeIndex + 1;
  const loungeIndexes = [5, 10, 15];

  for (let segmentOrder = 1; segmentOrder <= 20; segmentOrder += 1) {
    const startIndex = segmentOrder - 1;
    const endIndex = segmentOrder;
    const pointType = (index, isEnd) => {
      if (index === 0 && !isEnd) return 'origin';
      if (index === 20 && isEnd) return 'destination';
      return loungeIndexes.includes(index) ? 'lounge' : 'stop';
    };
    const pointId = (type, index) => {
      if (type === 'origin') return uuid(110000 + routeNo);
      if (type === 'destination') return uuid(111000 + routeNo);
      if (type === 'lounge') return uuid(410000 + routeNo * 100 + index);
      return uuid(210000 + routeNo * 100 + index);
    };
    const startType = pointType(startIndex, false);
    const endType = pointType(endIndex, true);
    const curve1 = Math.sin(segmentOrder * 0.75) * 0.015;
    const curve2 = Math.sin((segmentOrder + 1) * 0.75) * 0.015;
    const roadType = roadTypeFor(segmentOrder);
    const baselineSpeed = roadType === 'urban'
      ? 28 + (segmentOrder % 4) * 2
      : roadType === 'mixed'
        ? 38 + (segmentOrder % 5) * 2
        : roadType === 'rural'
          ? 45 + (segmentOrder % 4) * 3
          : 58 + (segmentOrder % 4) * 4;
    const distance = (route.dist / 20) * (0.88 + (segmentOrder % 5) * 0.06);
    const baselineMinutes = Math.max(3, Math.round((distance / baselineSpeed) * 60));
    const ratio1 = startIndex / 20;
    const ratio2 = endIndex / 20;

    routeSegments.push({
      id: uuid(310000 + routeNo * 100 + segmentOrder),
      master_route_id: route.id,
      start_point_type: startType,
      start_point_id: pointId(startType, startIndex),
      end_point_type: endType,
      end_point_id: pointId(endType, endIndex),
      segment_order: segmentOrder,
      start_latitude: (route.lat1 + (route.lat2 - route.lat1) * ratio1 + curve1).toFixed(7),
      start_longitude: (route.lng1 + (route.lng2 - route.lng1) * ratio1 - curve1 / 2).toFixed(7),
      end_latitude: (route.lat1 + (route.lat2 - route.lat1) * ratio2 + curve2).toFixed(7),
      end_longitude: (route.lng1 + (route.lng2 - route.lng1) * ratio2 - curve2 / 2).toFixed(7),
      distance_km: distance.toFixed(3),
      baseline_duration_minutes: baselineMinutes,
      baseline_speed_kmh: baselineSpeed.toFixed(2),
      road_type: roadType,
      traffic_sensitivity_factor: (roadType === 'urban' ? 1.45 : roadType === 'mixed' ? 1.15 : roadType === 'rural' ? 0.95 : 0.75).toFixed(2),
      elevation_change_meters: Math.round(Math.sin((segmentOrder + routeNo) / 3) * 42 + routeNo * 8),
      encoded_polyline_segment: `poly_r${String(routeNo).padStart(2, '0')}_s${String(segmentOrder).padStart(2, '0')}`,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-05-03T00:00:00Z',
    });
  }
}

const drivers = Array.from({ length: 100 }, (_, index) => {
  const i = index + 1;
  const experience = 1 + ((i * 3) % 24);
  const speedFactor = Number((1.12 - Math.min(experience, 18) * 0.009 + ((i % 7) - 3) * 0.006).toFixed(3));
  return {
    id: uuid(810000 + i),
    experience,
    rating: Number(Math.min(4.95, 3.65 + experience * 0.045 + (i % 5) * 0.04).toFixed(2)),
    speedFactor,
    punctuality: Number(Math.max(0.45, Math.min(0.96, 0.55 + experience * 0.015 - Math.abs(speedFactor - 1))).toFixed(2)),
    consistency: Number(Math.max(0.48, Math.min(0.97, 0.62 + experience * 0.012 - (i % 9) * 0.01)).toFixed(2)),
  };
});
const busTypes = ['Normal', 'Semi Luxury', 'AC Luxury', 'Express', 'Intercity'];
const buses = Array.from({ length: 100 }, (_, index) => {
  const i = index + 1;
  const age = Number((1 + ((i * 7) % 150) / 10).toFixed(1));
  const type = busTypes[i % busTypes.length];
  const speedFactor = Number((0.94 + age * 0.008 + (i % 6) * 0.004).toFixed(3));
  return {
    id: uuid(910000 + i),
    type,
    hasAc: type === 'AC Luxury' || type === 'Express',
    age,
    speedFactor,
    reliability: Number(Math.max(0.6, 0.98 - age * 0.018 - (i % 8) * 0.01).toFixed(2)),
  };
});

const slots = ['05:15:00', '06:30:00', '08:03:00', '10:15:00', '12:40:00', '14:00:00', '16:20:00', '17:45:00', '20:10:00', '22:30:00'];
const tripContexts = Array.from({ length: 100 }, (_, index) => {
  const i = index + 1;
  const tripDate = new Date(Date.UTC(2026, 0, 19 + i));
  const departureTime = slots[(i - 1) % slots.length];
  const driver = drivers[(i - 1) % drivers.length];
  const bus = buses[(i * 3 - 1) % buses.length];
  const route = routes[(i - 1) % routes.length];
  const dayOfWeek = days[tripDate.getUTCDay()];
  const weather = weatherFor(i);
  return {
    id: uuid(510000 + i),
    active_trip_id: uuid(610000 + i),
    scheduled_trip_id: uuid(710000 + i),
    trip_date: tripDate.toISOString().slice(0, 10),
    driver_id: driver.id,
    driver_experience_years: driver.experience,
    driver_rating: driver.rating.toFixed(2),
    bus_id: bus.id,
    bus_type: bus.type,
    has_ac: String(bus.hasAc),
    bus_age_years: bus.age.toFixed(2),
    departure_time: departureTime,
    time_of_day_category: bucketFor(departureTime),
    day_of_week: dayOfWeek,
    is_weekend: String(dayOfWeek === 'saturday' || dayOfWeek === 'sunday'),
    is_holiday: String(i % 17 === 0 || tripDate.toISOString().slice(5, 10) === '05-01'),
    weather_condition: weather,
    temperature_celsius: weather === 'fog' ? 23 + (i % 4) : weather === 'heavy_rain' ? 25 + (i % 3) : 28 + (i % 7),
    total_passengers: 18 + ((i * 7) % 35),
    route_id: route.id,
    created_at: iso(`${tripDate.toISOString().slice(0, 10)}T${departureTime}Z`),
  };
});

const facts = [];
const historical = [];
const predictions = [];
for (let i = 1; i <= 100; i += 1) {
  const context = tripContexts[i - 1];
  const routeNo = routes.findIndex((route) => route.id === context.route_id) + 1;
  const segmentOrder = ((i - 1) % 20) + 1;
  const segment = routeSegments.find((item) => item.master_route_id === context.route_id && Number(item.segment_order) === segmentOrder);
  const tripStart = new Date(`${context.trip_date}T${context.departure_time}Z`);
  const segmentStart = addMinutes(tripStart, (segmentOrder - 1) * 14 + (i % 5) * 2);
  const driver = drivers.find((item) => item.id === context.driver_id);
  const bus = buses.find((item) => item.id === context.bus_id);
  const trafficLevel = trafficFor(context.time_of_day_category, context.weather_condition, i);
  const actualDuration = Math.max(
    2.5,
    Number((Number(segment.baseline_duration_minutes) * durationFactor(trafficLevel, context.weather_condition, context.time_of_day_category, driver.speedFactor, bus.speedFactor) + ((i % 9) - 4) * 0.55).toFixed(2)),
  );
  const segmentEnd = addMinutes(segmentStart, actualDuration);
  const speed = Number((Number(segment.distance_km) / (actualDuration / 60)).toFixed(2));
  const quality = Number(Math.max(0.72, Math.min(0.99, bus.reliability - (context.weather_condition === 'heavy_rain' ? 0.06 : 0) + (i % 5) * 0.005)).toFixed(2));

  facts.push({
    id: uuid(1010000 + i),
    route_segment_id: segment.id,
    trip_context_id: context.id,
    segment_start_time: iso(segmentStart),
    segment_end_time: iso(segmentEnd),
    actual_duration_minutes: actualDuration.toFixed(2),
    average_speed_kmh: speed.toFixed(2),
    duration_variance_minutes: (actualDuration - Number(segment.baseline_duration_minutes)).toFixed(2),
    traffic_level: trafficLevel,
    actual_distance_km: (Number(segment.distance_km) + ((i % 5) - 2) * 0.04).toFixed(3),
    gps_accuracy_meters: 6 + (i % 16),
    data_quality_score: quality.toFixed(2),
    recorded_at: iso(addMinutes(segmentEnd, 5 / 60)),
  });

  historical.push({
    id: uuid(1610000 + i),
    route_segment_id: segment.id,
    active_trip_id: context.active_trip_id,
    scheduled_trip_id: context.scheduled_trip_id,
    bus_id: context.bus_id,
    driver_id: context.driver_id,
    trip_date: context.trip_date,
    segment_start_time: iso(segmentStart),
    segment_end_time: iso(segmentEnd),
    actual_duration_minutes: actualDuration.toFixed(2),
    estimated_duration_minutes: Number(segment.baseline_duration_minutes).toFixed(2),
    average_speed_kmh: speed.toFixed(2),
    max_speed_kmh: (speed + 14 + (i % 9)).toFixed(2),
    min_speed_kmh: Math.max(3, speed - 16 - (i % 6)).toFixed(2),
    stop_count: i % 5,
    dwell_time_seconds: segment.end_point_type === 'lounge' || segment.start_point_type === 'lounge' ? 300 + ((i * 37) % 1200) : 20 + ((i * 11) % 160),
    time_of_day_category: context.time_of_day_category,
    day_of_week: context.day_of_week,
    is_holiday: context.is_holiday,
    weather_condition: context.weather_condition,
    bus_type: context.bus_type,
    bus_occupancy_percentage: Math.min(100, Math.round((Number(context.total_passengers) / 52) * 100)),
    driver_experience_years: context.driver_experience_years,
    created_at: iso(addMinutes(segmentEnd, 5 / 60)),
  });

  const targetMode = i % 3;
  const predictionType = targetMode === 0 ? 'destination' : targetMode === 1 ? 'lounge' : 'stop';
  const predictedAt = addMinutes(segmentStart, -Math.min(25, ((segmentOrder - 1) * 14 + (i % 5) * 2) / 2));
  const predictionError = (i % 11) - 5;
  const method = ['baseline', 'historical', 'ml', 'hybrid', 'realtime'][i % 5];
  predictions.push({
    id: uuid(1510000 + i),
    active_trip_id: context.active_trip_id,
    lounge_id: predictionType === 'lounge' ? uuid(410000 + routeNo * 100 + [5, 10, 15][i % 3]) : '',
    stop_id: predictionType === 'stop' ? uuid(210000 + routeNo * 100 + (((segmentOrder + 3) % 19) + 1)) : '',
    prediction_type: predictionType,
    predicted_at: iso(predictedAt),
    predicted_arrival_time: iso(addMinutes(segmentEnd, -predictionError)),
    current_location_lat: segment.start_latitude,
    current_location_lng: segment.start_longitude,
    distance_remaining_km: (Number(segment.distance_km) * (1 + (i % 6) * 0.18)).toFixed(3),
    actual_arrival_time: iso(segmentEnd),
    calculation_method: method,
    confidence_score: Math.max(42, Math.min(97, 58 + quality * 28 - Math.abs(predictionError) * 2 + (['ml', 'hybrid', 'realtime'].includes(method) ? 8 : 0))).toFixed(2),
    context_data: JSON.stringify({
      route_id: context.route_id,
      segment_order: segmentOrder,
      driver_factor: driver.speedFactor,
      bus_factor: bus.speedFactor,
      weather: context.weather_condition,
      traffic: trafficLevel,
      time_bucket: context.time_of_day_category,
    }),
    created_at: iso(addMinutes(predictedAt, 3 / 60)),
  });
}

const aggregateStats = routeSegments.map((segment, index) => {
  const i = index + 1;
  const bucket = ['early_morning', 'morning_peak', 'midday', 'evening_peak', 'night'][i % 5];
  const day = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'][i % 7];
  const weather = ['clear', 'rain', 'heavy_rain', 'fog'][i % 4];
  const averageDuration = Number((Number(segment.baseline_duration_minutes) * durationFactor(trafficFor(bucket, weather, i), weather, bucket, 1, 1)).toFixed(2));
  const stdDev = Number(Math.max(0.8, averageDuration * (0.08 + (i % 5) * 0.01)).toFixed(2));
  return {
    id: uuid(1110000 + i),
    route_segment_id: segment.id,
    time_of_day_category: bucket,
    day_of_week: day,
    weather_condition: weather,
    avg_duration_minutes: averageDuration.toFixed(2),
    median_duration_minutes: (averageDuration - (i % 3) * 0.25).toFixed(2),
    stddev_duration_minutes: stdDev.toFixed(2),
    min_duration_minutes: Math.max(1.5, averageDuration - stdDev * 1.6).toFixed(2),
    max_duration_minutes: (averageDuration + stdDev * 2.2).toFixed(2),
    p95_duration_minutes: (averageDuration + stdDev * 1.7).toFixed(2),
    avg_speed_kmh: (Number(segment.distance_km) / (averageDuration / 60)).toFixed(2),
    sample_count: 18 + ((i * 7) % 96),
    last_trip_date: iso(addMinutes(new Date('2026-05-03T00:00:00Z'), -(i % 21) * 1440)).slice(0, 10),
    avg_data_quality: (0.82 + (i % 15) * 0.01).toFixed(2),
    last_calculated_at: '2026-05-04T02:00:00Z',
    calculation_window_days: 90,
  };
});

const driverProfiles = drivers.map((driver, index) => {
  const i = index + 1;
  const trips = 25 + ((i * 11) % 240);
  return {
    id: uuid(1210000 + i),
    driver_id: driver.id,
    avg_speed_factor: driver.speedFactor.toFixed(3),
    punctuality_score: driver.punctuality.toFixed(2),
    consistency_score: driver.consistency.toFixed(2),
    total_trips_analyzed: trips,
    total_segments_analyzed: trips * 5,
    primary_routes: `{${routes[(i - 1) % routes.length].id}}`,
    last_calculated_at: '2026-05-04T02:00:00Z',
    calculation_window_days: 90,
  };
});
const busProfiles = buses.map((bus, index) => {
  const i = index + 1;
  return {
    id: uuid(1310000 + i),
    bus_id: bus.id,
    avg_speed_factor: bus.speedFactor.toFixed(3),
    reliability_score: bus.reliability.toFixed(2),
    fuel_efficiency_kmpl: (5.8 + (i % 18) * 0.18 + (bus.type === 'AC Luxury' ? -0.4 : 0)).toFixed(2),
    days_since_last_service: 3 + ((i * 5) % 75),
    breakdown_count_90d: bus.reliability < 0.75 ? 2 : bus.reliability < 0.85 ? 1 : 0,
    total_trips_analyzed: 30 + ((i * 13) % 210),
    total_km_traveled: (3500 + ((i * 311) % 38000)).toFixed(2),
    last_calculated_at: '2026-05-04T02:00:00Z',
    calculation_window_days: 90,
  };
});
const loungeStopMetrics = Array.from({ length: 100 }, (_, index) => {
  const i = index + 1;
  const routeNo = ((i - 1) % routes.length) + 1;
  const loungeIndex = [5, 10, 15, 4, 8, 12, 16, 18][(i - 1) % 8];
  const dwell = 7.5 + (i % 9) * 1.15 + ([10, 15].includes(loungeIndex) ? 3 : 0);
  return {
    id: uuid(1410000 + i),
    lounge_id: uuid(410000 + routeNo * 100 + loungeIndex + Math.floor((i - 1) / 40) * 1000),
    master_route_id: routes[routeNo - 1].id,
    average_dwell_time_minutes: dwell.toFixed(2),
    min_dwell_time_minutes: Math.max(3, dwell - 5.5).toFixed(2),
    max_dwell_time_minutes: (dwell + 10 + (i % 6)).toFixed(2),
    peak_hour_dwell_time_minutes: (dwell + 4.2).toFixed(2),
    off_peak_dwell_time_minutes: Math.max(3, dwell - 2.6).toFixed(2),
    average_bookings_per_trip: 2 + ((i * 3) % 18),
    total_stops_recorded: 20 + ((i * 7) % 310),
    last_calculated_at: '2026-05-04T02:00:00Z',
    data_points_count: 20 + ((i * 7) % 310),
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-05-04T02:00:00Z',
  };
});

write('route_segments.csv', routeSegments);
write('trip_contexts.csv', tripContexts);
write('segment_performance_facts.csv', facts);
write('segment_aggregate_stats.csv', aggregateStats);
write('driver_performance_profiles.csv', driverProfiles);
write('bus_performance_profiles.csv', busProfiles);
write('lounge_stop_metrics.csv', loungeStopMetrics);
write('eta_predictions.csv', predictions);
write('segment_historical_performance.csv', historical);

fs.writeFileSync(
  path.join(base, 'README.md'),
  `# ETA Training CSV Sample Pack

These CSV files contain deterministic, synthetic-but-realistic training data for the ETA/ETD model tables in \`backend/migrations/004_create_eta_v3_tables.sql\`.

Every CSV has 100 rows and uses the current project table format: UUIDs, ISO-8601 UTC timestamps, PostgreSQL array text for \`primary_routes\`, and generated columns omitted from import CSVs.

Files: \`route_segments.csv\`, \`trip_contexts.csv\`, \`segment_performance_facts.csv\`, \`segment_aggregate_stats.csv\`, \`driver_performance_profiles.csv\`, \`bus_performance_profiles.csv\`, \`lounge_stop_metrics.csv\`, \`eta_predictions.csv\`, and \`segment_historical_performance.csv\`.

Important: the UUIDs are internally consistent across the CSVs, but they are sample IDs. Replace them with real IDs from your database or seed matching reference rows before importing with foreign keys enabled.

For ETA training, join \`segment_performance_facts\` to \`trip_contexts\`. For ETD, train/predict lounge dwell time from \`lounge_stop_metrics\` and add it to the predicted arrival time.
`,
  'utf8',
);

for (const file of fs.readdirSync(base).filter((name) => name.endsWith('.csv')).sort()) {
  const count = fs.readFileSync(path.join(base, file), 'utf8').trim().split('\n').length - 1;
  console.log(`${file}: ${count}`);
}
