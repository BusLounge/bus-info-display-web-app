# Route Management Feature

## Overview
The Route Management section allows you to create, edit, and manage bus routes with interactive map-based polyline editing - exactly like the polyline-tracker project but integrated into the Angular bus-info-display application.

## Features

### 1. Route List View
- **View All Routes**: Display all routes with their details (route number, name, origin, destination, distance, duration)
- **Search**: Filter routes by name, number, origin city, or destination city
- **Quick Actions**: 
  - Edit route segments on map
  - Activate/Deactivate routes
  - Delete routes
- **Create New Route**: Navigate to the editor to create a new route

### 2. Interactive Map Editor
The route editor provides a full-featured map interface for creating and editing route polylines:

#### Edit Modes (4 Modes)
1. **➕ Add Mode** (Default)
   - Click on map to add points to the end of the route
   - Click on markers to delete individual points
   - Points are appended sequentially

2. **➕➖ Insert Mode**
   - Click near any line segment to insert a point between existing points
   - Green midpoint markers show where insertions will occur
   - Smart segment detection finds the nearest line segment
   - Perfect for refining existing routes

3. **✋ Move Mode**
   - Drag and drop markers to reposition points
   - Real-time polyline updates as you drag
   - Ideal for fine-tuning point positions

4. **🔲 Select Mode**
   - Click markers to select multiple points
   - Delete multiple points at once
   - Selected points are highlighted in blue

### 3. Route Details Form
- **Route Name**: Descriptive name for the route
- **Origin City**: Starting city
- **Destination City**: Ending city
- **Distance**: Auto-calculated from points (in kilometers using Haversine formula)
- **Duration**: Estimated travel time in minutes
- **Active Status**: Enable/disable the route

### 4. Points Management
- **Points List**: View all coordinates with latitude/longitude
- **Highlight Points**: Click any point in the list to center map on it
- **Delete Individual Points**: Remove specific points
- **Delete Selected**: Bulk delete selected points (in Select mode)
- **Clear All**: Remove all points and start over
- **Focus All**: Fit map bounds to show all points

### 5. Real-World Map Integration
- **OpenStreetMap**: Uses real-world map data
- **Search Cities**: Navigate to real-world locations by zooming and panning
- **Zoom Controls**: Built-in zoom in/out functionality
- **Auto-fit Bounds**: Automatically centers map on route points

## Technical Implementation

### Polyline Encoding/Decoding
The system uses Google's Polyline Algorithm (precision 5) for efficient storage:

```typescript
// Decode: Database → Display
const points = decodePolyline(encodedPolyline);

// Encode: Display → Database
const encodedPolyline = encodePolyline(points);
```

### Distance Calculation
Uses the Haversine formula for accurate distance calculation between GPS coordinates:

```typescript
export function calculateDistance(point1: LatLng, point2: LatLng): number {
  // Calculate great-circle distance between two points on Earth
  // Returns distance in kilometers
}
```

### Insert Algorithm
Uses perpendicular distance calculation to find the nearest line segment:

```typescript
export function distanceToSegment(
  point: LatLng,
  segmentStart: LatLng,
  segmentEnd: LatLng
): number {
  // Calculate minimum distance from point to line segment
  // Uses vector projection for accuracy
}
```

## Backend API Endpoints

### Routes API
```
GET    /api/routes              - Get all routes
GET    /api/routes/{id}         - Get single route by ID
POST   /api/routes              - Create new route
PUT    /api/routes/{id}         - Update existing route
DELETE /api/routes/{id}         - Delete route
```

### Request/Response Format
```typescript
interface MasterRoute {
  idx?: number;
  id?: string;
  route_number: string;
  route_name: string;
  origin_city: string;
  destination_city: string;
  total_distance_km: string;
  estimated_duration_minutes: number;
  encoded_polyline: string;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}
```

## Database Schema
Routes are stored in the `master_routes` table:

```sql
CREATE TABLE master_routes (
  idx SERIAL PRIMARY KEY,
  id UUID DEFAULT uuid_generate_v4(),
  route_number VARCHAR(50) NOT NULL,
  route_name VARCHAR(255) NOT NULL,
  origin_city VARCHAR(255) NOT NULL,
  destination_city VARCHAR(255) NOT NULL,
  total_distance_km VARCHAR(50),
  estimated_duration_minutes INTEGER,
  encoded_polyline VARCHAR(10000) NOT NULL,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);
```

## Usage Guide

### Creating a New Route
1. Navigate to **Route Management** from the header menu
2. Click **"➕ Create New Route"**
3. Fill in route details (name, origin, destination, duration)
4. Click on the map to add coordinate points
5. Use different edit modes to refine the route:
   - **Add**: Extend the route
   - **Insert**: Add points between existing ones
   - **Move**: Reposition points by dragging
   - **Select**: Multi-select for bulk deletion
6. Review the points list to ensure accuracy
7. Click **"💾 Create Route"** to save

### Editing an Existing Route
1. Go to Route Management
2. Click **"✏️ Edit"** on any route card
3. The editor loads with existing points displayed
4. Modify the route using edit modes
5. Update route details if needed
6. Click **"💾 Update Route"** to save changes

### Managing Routes
- **Search**: Use the search bar to filter routes
- **Activate/Deactivate**: Toggle route status with the button
- **Delete**: Remove routes you no longer need

## Components

### Angular Components
- `RouteManagementComponent`: Main list view
- `RouteEditorComponent`: Interactive map editor

### Services
- `RouteService`: Angular service for API calls
- `polyline.utils.ts`: Utility functions for polyline operations

### Backend Components
- `RouteHandler`: HTTP request handlers
- `RouteService`: Business logic
- `RouteRepository`: Database operations
- `MasterRoute`: Data model

## Dependencies

### Frontend
- `leaflet`: Map library
- `@types/leaflet`: TypeScript definitions
- `@googlemaps/polyline-codec`: Polyline encoding/decoding

### Backend
- `gorilla/mux`: HTTP router
- PostgreSQL database

## Comparison with Polyline-Tracker

This implementation mirrors the polyline-tracker functionality:

| Feature | Polyline-Tracker | Route Management |
|---------|------------------|------------------|
| Framework | React + Vite | Angular |
| Map Library | Leaflet | Leaflet |
| Add Points | ✅ | ✅ |
| Insert Between | ✅ | ✅ |
| Move Points | ✅ | ✅ |
| Select Mode | ✅ | ✅ |
| Real-world Map | ✅ | ✅ |
| Polyline Encoding | ✅ | ✅ |
| Backend Database | Supabase | PostgreSQL |
| Edit Modes | 4 modes | 4 modes |

## Tips

1. **Start with major waypoints**: Add key cities/intersections first, then refine with Insert mode
2. **Use Move mode for precision**: Fine-tune point positions by dragging
3. **Zoom in for accuracy**: Get closer to add precise points
4. **Test routes**: Create a trip animation (if integrated) to verify the route
5. **Save frequently**: Update routes after major changes

## Troubleshooting

**Q: Map doesn't display**
- Check that Leaflet CSS is loaded
- Verify `angular.json` includes `node_modules/leaflet/dist/leaflet.css`

**Q: Points not adding**
- Ensure edit mode is correct (Add or Insert)
- Check console for JavaScript errors

**Q: Distance calculation seems wrong**
- Distance uses Haversine formula (great-circle distance)
- Results are in kilometers

**Q: Can't save route**
- Ensure at least 2 points are added
- Fill in all required fields (name, origin, destination)

---

Created to provide comprehensive route management with real-world map integration for the Smart Transit System.
