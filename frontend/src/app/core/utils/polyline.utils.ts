import { decode, encode } from '@googlemaps/polyline-codec';

export interface LatLng {
  lat: number;
  lng: number;
}

/**
 * Decodes Google encoded polyline to array of lat/lng coordinates
 */
export function decodePolyline(encodedPolyline: string): LatLng[] {
  // Handle escaped backslashes from database
  const cleanedPolyline = encodedPolyline.replace(/\\\\/g, '\\');
  const decoded = decode(cleanedPolyline, 5);
  return decoded.map(([lat, lng]) => ({ lat, lng }));
}

/**
 * Encodes array of lat/lng coordinates to Google encoded polyline
 */
export function encodePolyline(points: LatLng[]): string {
  const coordinates: [number, number][] = points.map(p => [p.lat, p.lng]);
  return encode(coordinates, 5);
}

/**
 * Calculates time between each point based on total duration
 */
export function calculateTimePerPoint(
  totalDurationMinutes: number,
  numberOfPoints: number
): number {
  // Return time in milliseconds per point
  return (totalDurationMinutes * 60 * 1000) / numberOfPoints;
}

/**
 * Format time in hours and minutes
 */
export function formatDuration(minutes: number): string {
  const hours = Math.floor(minutes / 60);
  const mins = Math.round(minutes % 60);
  
  if (hours > 0) {
    return `${hours}h ${mins}m`;
  }
  return `${mins}m`;
}

/**
 * Calculate distance from point to line segment
 */
export function distanceToSegment(
  point: LatLng,
  segmentStart: LatLng,
  segmentEnd: LatLng
): number {
  const x = point.lng;
  const y = point.lat;
  const x1 = segmentStart.lng;
  const y1 = segmentStart.lat;
  const x2 = segmentEnd.lng;
  const y2 = segmentEnd.lat;

  const A = x - x1;
  const B = y - y1;
  const C = x2 - x1;
  const D = y2 - y1;

  const dot = A * C + B * D;
  const lenSq = C * C + D * D;
  let param = -1;

  if (lenSq !== 0) {
    param = dot / lenSq;
  }

  let xx, yy;

  if (param < 0) {
    xx = x1;
    yy = y1;
  } else if (param > 1) {
    xx = x2;
    yy = y2;
  } else {
    xx = x1 + param * C;
    yy = y1 + param * D;
  }

  const dx = x - xx;
  const dy = y - yy;
  return Math.sqrt(dx * dx + dy * dy);
}

/**
 * Calculate distance between two points in kilometers
 */
export function calculateDistance(point1: LatLng, point2: LatLng): number {
  const R = 6371; // Earth's radius in km
  const dLat = toRad(point2.lat - point1.lat);
  const dLng = toRad(point2.lng - point1.lng);
  
  const a = Math.sin(dLat / 2) * Math.sin(dLat / 2) +
    Math.cos(toRad(point1.lat)) * Math.cos(toRad(point2.lat)) *
    Math.sin(dLng / 2) * Math.sin(dLng / 2);
  
  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
  return R * c;
}

function toRad(degrees: number): number {
  return degrees * (Math.PI / 180);
}

/**
 * Calculate bearing between two points in degrees
 */
export function calculateBearing(from: LatLng, to: LatLng): number {
  const dLng = toRad(to.lng - from.lng);
  const y = Math.sin(dLng) * Math.cos(toRad(to.lat));
  const x = Math.cos(toRad(from.lat)) * Math.sin(toRad(to.lat)) -
    Math.sin(toRad(from.lat)) * Math.cos(toRad(to.lat)) * Math.cos(dLng);
  const bearing = Math.atan2(y, x);
  return (toDeg(bearing) + 360) % 360;
}

function toDeg(radians: number): number {
  return radians * (180 / Math.PI);
}

/**
 * Format bearing as compass direction
 */
export function bearingToDirection(bearing: number): string {
  const directions = ['N', 'NNE', 'NE', 'ENE', 'E', 'ESE', 'SE', 'SSE',
    'S', 'SSW', 'SW', 'WSW', 'W', 'WNW', 'NW', 'NNW'];
  const index = Math.round(bearing / 22.5) % 16;
  return directions[index];
}

export interface RouteSegment {
  index: number;
  from: LatLng;
  to: LatLng;
  distance: number;
  bearing: number;
  bearingDirection: string;
  distanceFormatted: string;
}

/**
 * Generate segments from polyline points
 */
export function generateSegments(points: LatLng[]): RouteSegment[] {
  const segments: RouteSegment[] = [];
  
  for (let i = 0; i < points.length - 1; i++) {
    const from = points[i];
    const to = points[i + 1];
    const distance = calculateDistance(from, to);
    const bearing = calculateBearing(from, to);
    
    segments.push({
      index: i + 1,
      from,
      to,
      distance,
      bearing,
      bearingDirection: bearingToDirection(bearing),
      distanceFormatted: distance.toFixed(2)
    });
  }
  
  return segments;
}
