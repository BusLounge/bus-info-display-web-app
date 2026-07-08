# Trip Data Integration - Setup Guide

This guide explains how to configure the Trip data integration for displaying real-time trip information in the Route Management section.

## Overview

The application now fetches trip data from a Supabase database (`current_locations` table) and displays it alongside route information in the Route Management page.

## Features Added

1. **Trip Service** - Service to fetch trip data from Supabase
2. **Route Management Enhancement** - Shows active trips for each route
3. **Trip Details Panel** - View detailed information about trips including:
   - Trip progress percentage
   - Current location (lat/lng)
   - Animation status
   - Speed multiplier
   - Point progress (current/total)

## Setup Instructions

### 1. Configure Supabase Credentials

Open the file: `frontend/src/app/core/config/supabase.config.ts`

Replace the placeholder values with your actual Supabase credentials:

```typescript
const supabaseUrl = 'YOUR_SUPABASE_URL'; // e.g., 'https://xxxxx.supabase.co'
const supabaseAnonKey = 'YOUR_SUPABASE_ANON_KEY';
```

**Where to find these values:**
- Log in to your Supabase project at https://supabase.com
- Go to Project Settings > API
- Copy the **Project URL** (supabaseUrl)
- Copy the **anon/public key** (supabaseAnonKey)

### 2. Database Schema

The `current_locations` table should already exist in your Supabase database with the following schema:

```sql
CREATE TABLE current_locations (
  trip_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  route_id UUID NOT NULL REFERENCES master_routes(id) ON DELETE CASCADE,
  route_name VARCHAR(255) NOT NULL,
  current_point_index INTEGER NOT NULL DEFAULT 0,
  current_latitude DECIMAL(10, 8) NOT NULL,
  current_longitude DECIMAL(11, 8) NOT NULL,
  total_points INTEGER NOT NULL,
  speed_multiplier INTEGER NOT NULL DEFAULT 1,
  is_animating BOOLEAN NOT NULL DEFAULT false,
  progress_percentage DECIMAL(5, 2) NOT NULL DEFAULT 0.00,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

If this table doesn't exist, you can create it by running the schema from `polyline-tracker/database-schema.sql`.

### 3. Install Dependencies

The Supabase client has already been installed:

```bash
npm install @supabase/supabase-js
```

### 4. Start the Application

```bash
cd frontend
npm start
```

## Using the Trip Features

### View Trip Information

1. Navigate to **Route Management** page
2. Each route card now shows:
   - **Active Trips count** - Total number of trips for the route
   - **Animating trips** - Number of currently running trips (shown with 🚌 icon)

### View Trip Details

1. Click the **"📍 View Trips"** button on any route card that has active trips
2. A modal panel will open showing:
   - Trip ID
   - Status (Running/Paused)
   - Progress percentage with visual progress bar
   - Current point index out of total points
   - Speed multiplier
   - Current GPS coordinates
   - Creation and update timestamps

### Trip Status Indicators

- **Green border with 🚌 icon** - Trip is actively running/animating
- **Yellow border with ⏸️ icon** - Trip is paused
- **Progress bar** - Visual representation of trip completion

## API Methods Available

The `TripService` provides the following methods:

```typescript
// Get all trips
getAllTrips(): Observable<Trip[]>

// Get trips for a specific route
getTripsByRouteId(routeId: string): Observable<Trip[]>

// Get only active (animating) trips
getActiveTrips(): Observable<Trip[]>

// Get a specific trip by ID
getTripById(tripId: string): Observable<Trip | null>

// Create a new trip
createTrip(trip: Partial<Trip>): Observable<Trip>

// Update a trip
updateTrip(tripId: string, updates: Partial<Trip>): Observable<Trip>

// Delete a trip
deleteTrip(tripId: string): Observable<void>

// Get trip count for a route
getTripCountByRouteId(routeId: string): Observable<number>
```

## Troubleshooting

### Error: "Missing Supabase environment variables"

Make sure you've configured the `supabase.config.ts` file with your actual credentials.

### Error: "Failed to load routes and trips"

1. Check that your Supabase credentials are correct
2. Verify that the `current_locations` table exists
3. Check browser console for detailed error messages
4. Ensure Row Level Security (RLS) policies allow read access to the table

### No trips showing for routes

1. Verify that trip data exists in the `current_locations` table
2. Check that `route_id` in the trips matches the route `id` from `master_routes`
3. Open browser DevTools and check the Network tab for API responses

## Related Files

- **Trip Service**: `frontend/src/app/core/services/trip.service.ts`
- **Supabase Config**: `frontend/src/app/core/config/supabase.config.ts`
- **Route Management Component**: `frontend/src/app/features/route-management/route-management.component.ts`
- **Route Management Template**: `frontend/src/app/features/route-management/route-management.component.html`
- **Route Management Styles**: `frontend/src/app/features/route-management/route-management.component.scss`

## Additional Notes

- The trip data is fetched in real-time when the Route Management page loads
- Both routes and trips are fetched simultaneously using RxJS `forkJoin` for better performance
- The UI updates automatically when trips are added or modified in the database
- Trip progress is displayed with a smooth animated progress bar
