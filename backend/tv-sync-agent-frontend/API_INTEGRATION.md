# API Integration Guide

This guide helps you integrate the TV Sync Agent Frontend with your backend API.

## Backend Requirements

The frontend expects a REST API with the following endpoints:

---

## API Endpoints

### Base URL
```
Development: http://localhost:8080/api
Production: https://your-domain.com/api
```

### Authentication
If your API requires authentication, implement an HTTP interceptor:

```typescript
// src/app/core/interceptors/auth.interceptor.ts
import { HttpInterceptorFn } from '@angular/common/http';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const token = localStorage.getItem('auth_token');

  if (token) {
    const cloned = req.clone({
      headers: req.headers.set('Authorization', `Bearer ${token}`)
    });
    return next(cloned);
  }

  return next(req);
};
```

---

## Device Endpoints

### Get All Devices
```http
GET /api/devices
```

**Response:**
```json
[
  {
    "id": "device-001",
    "name": "TV-Display-001",
    "loungeName": "Premium Lounge A",
    "status": "online",
    "location": "Terminal 1, Gate 5",
    "lastSyncTime": "2024-03-24T10:30:00Z",
    "ipAddress": "192.168.1.100",
    "macAddress": "00:1B:44:11:3A:B7",
    "displayMode": "both",
    "layoutMode": "split_screen",
    "language": "en",
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-03-24T10:30:00Z"
  }
]
```

### Get Single Device
```http
GET /api/devices/{deviceId}
```

**Response:** Same as single device object above

### Update Device Configuration
```http
PATCH /api/devices/{deviceId}/config
```

**Request Body:**
```json
{
  "displayMode": "both",
  "layoutMode": "split_screen",
  "language": "es"
}
```

**Response:** Updated device object

### Refresh Device Status
```http
POST /api/devices/{deviceId}/refresh
```

**Response:**
```json
{
  "id": "device-001",
  "status": "online",
  "lastSyncTime": "2024-03-24T10:35:00Z"
}
```

---

## Emergency Message Endpoints

### Send Emergency Message
```http
POST /api/devices/{deviceId}/emergency-message
```

**Request Body:**
```json
{
  "message": "Gate change: Flight AA123 now departing from Gate 15",
  "priority": "high",
  "isActive": true,
  "backgroundColor": "#f97316",
  "textColor": "#ffffff"
}
```

**Response:**
```json
{
  "id": "em_123456",
  "message": "Gate change: Flight AA123 now departing from Gate 15",
  "priority": "high",
  "isActive": true,
  "startTime": "2024-03-24T10:40:00Z",
  "backgroundColor": "#f97316",
  "textColor": "#ffffff"
}
```

### Clear Emergency Message
```http
DELETE /api/devices/{deviceId}/emergency-message
```

**Response:** `204 No Content`

---

## Advertisement Tracking Endpoints

### Get Ad Playbacks
```http
GET /api/devices/{deviceId}/ad-playbacks?startDate=2024-03-01&endDate=2024-03-24
```

**Query Parameters:**
- `startDate` (optional): ISO 8601 date string
- `endDate` (optional): ISO 8601 date string

**Response:**
```json
[
  {
    "id": "playback-001",
    "deviceId": "device-001",
    "adId": "ad-123",
    "adName": "Coffee Shop Special",
    "adDuration": 30,
    "playbackStartTime": "2024-03-24T09:00:00Z",
    "playbackEndTime": "2024-03-24T09:00:30Z",
    "status": "completed",
    "impressions": 45,
    "completionRate": 98,
    "billingAmount": 12.50
  }
]
```

### Generate Billing Report
```http
POST /api/devices/{deviceId}/billing-report
```

**Request Body:**
```json
{
  "startDate": "2024-03-01T00:00:00Z",
  "endDate": "2024-03-24T23:59:59Z"
}
```

**Response:**
```json
{
  "deviceId": "device-001",
  "deviceName": "TV-Display-001",
  "totalImpressions": 1250,
  "totalPlaybackTime": 37500,
  "totalBillingAmount": 425.50,
  "reportPeriod": {
    "startDate": "2024-03-01T00:00:00Z",
    "endDate": "2024-03-24T23:59:59Z"
  },
  "ads": [
    {
      "adId": "ad-123",
      "adName": "Coffee Shop Special",
      "totalImpressions": 450,
      "totalPlaybackTime": 13500,
      "averageCompletionRate": 97.5,
      "billingAmount": 150.25
    }
  ]
}
```

---

## Health Check Endpoint

### Health Check
```http
GET /api/health
```

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "timestamp": "2024-03-24T10:45:00Z"
}
```

---

## Data Models

### Device Status Enum
```typescript
enum DeviceStatus {
  ONLINE = 'online',
  OFFLINE = 'offline',
  SYNCING = 'syncing',
  ERROR = 'error'
}
```

### Display Mode Enum
```typescript
enum DisplayMode {
  SCHEDULES_ONLY = 'schedules_only',
  ADS_ONLY = 'ads_only',
  BOTH = 'both'
}
```

### Layout Mode Enum
```typescript
enum LayoutMode {
  SPLIT_SCREEN = 'split_screen',
  FULL_SCREEN_ALTERNATE = 'full_screen_alternate'
}
```

### Emergency Priority
```typescript
type Priority = 'low' | 'medium' | 'high' | 'critical';
```

### Playback Status
```typescript
enum PlaybackStatus {
  PLAYING = 'playing',
  COMPLETED = 'completed',
  SKIPPED = 'skipped',
  ERROR = 'error'
}
```

---

## WebSocket Integration (Optional)

For real-time updates, implement WebSocket support:

### WebSocket Connection
```
ws://localhost:8080/ws
```

### Message Format
```json
{
  "type": "device_status_change",
  "deviceId": "device-001",
  "status": "online",
  "timestamp": "2024-03-24T10:50:00Z"
}
```

### Event Types
- `device_status_change`: Device went online/offline
- `device_config_update`: Configuration changed
- `emergency_message`: Emergency message sent/cleared
- `sync_complete`: Device finished syncing
- `ad_playback_start`: Ad started playing
- `ad_playback_end`: Ad finished playing

### Frontend WebSocket Service
```typescript
// src/app/core/services/websocket.service.ts
import { Injectable } from '@angular/core';
import { Observable, Subject } from 'rxjs';
import { environment } from '../../../environments/environment.development';

@Injectable({
  providedIn: 'root'
})
export class WebSocketService {
  private ws: WebSocket | null = null;
  private messages$ = new Subject<any>();

  connect(): Observable<any> {
    if (!this.ws) {
      this.ws = new WebSocket(environment.wsUrl);

      this.ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        this.messages$.next(data);
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
      };

      this.ws.onclose = () => {
        console.log('WebSocket connection closed');
        this.ws = null;
        // Implement reconnection logic here
      };
    }

    return this.messages$.asObservable();
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}
```

---

## Error Handling

### Standard Error Response
```json
{
  "error": {
    "code": "DEVICE_NOT_FOUND",
    "message": "Device with ID 'device-001' not found",
    "statusCode": 404
  }
}
```

### HTTP Status Codes
- `200 OK`: Successful request
- `201 Created`: Resource created successfully
- `204 No Content`: Successful deletion
- `400 Bad Request`: Invalid request data
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

---

## CORS Configuration

Ensure your backend allows requests from the frontend:

### Go (Gin) Example
```go
func CORSMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
        c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }

        c.Next()
    }
}
```

---

## Testing the API

### Using curl
```bash
# Get all devices
curl http://localhost:8080/api/devices

# Update device config
curl -X PATCH http://localhost:8080/api/devices/device-001/config \
  -H "Content-Type: application/json" \
  -d '{"displayMode":"both","language":"es"}'

# Send emergency message
curl -X POST http://localhost:8080/api/devices/device-001/emergency-message \
  -H "Content-Type: application/json" \
  -d '{"message":"Test message","priority":"medium","isActive":true}'
```

### Using Postman
Import the endpoints above into Postman for easier testing.

---

## Environment Configuration

### Development
```typescript
// src/environments/environment.development.ts
export const environment = {
  production: false,
  apiUrl: 'http://localhost:8080/api',
  wsUrl: 'ws://localhost:8080/ws',
  version: '1.0.0'
};
```

### Production
```typescript
// src/environments/environment.ts
export const environment = {
  production: true,
  apiUrl: 'https://api.yourdomain.com/api',
  wsUrl: 'wss://api.yourdomain.com/ws',
  version: '1.0.0'
};
```

---

## Backend Implementation Example (Go)

Here's a simple example using Go and Gin:

```go
package main

import (
    "github.com/gin-gonic/gin"
    "net/http"
)

type Device struct {
    ID           string `json:"id"`
    Name         string `json:"name"`
    LoungeName   string `json:"loungeName"`
    Status       string `json:"status"`
    Location     string `json:"location"`
    DisplayMode  string `json:"displayMode"`
    LayoutMode   string `json:"layoutMode"`
    Language     string `json:"language"`
    // ... other fields
}

func main() {
    r := gin.Default()
    r.Use(CORSMiddleware())

    api := r.Group("/api")
    {
        api.GET("/devices", getDevices)
        api.GET("/devices/:id", getDevice)
        api.PATCH("/devices/:id/config", updateDeviceConfig)
        api.POST("/devices/:id/refresh", refreshDevice)
        api.POST("/devices/:id/emergency-message", sendEmergencyMessage)
        api.DELETE("/devices/:id/emergency-message", clearEmergencyMessage)
        api.GET("/devices/:id/ad-playbacks", getAdPlaybacks)
        api.POST("/devices/:id/billing-report", generateBillingReport)
        api.GET("/health", healthCheck)
    }

    r.Run(":8080")
}

func getDevices(c *gin.Context) {
    // Implementation
    devices := []Device{
        // ... sample devices
    }
    c.JSON(http.StatusOK, devices)
}

// ... implement other handlers
```

---

## Next Steps

1. ✅ Implement the required endpoints in your backend
2. ✅ Configure CORS properly
3. ✅ Update environment files with your API URL
4. ✅ Test each endpoint individually
5. ✅ Implement WebSocket support (optional)
6. ✅ Add authentication if needed
7. ✅ Deploy and test in production

For more help, see:
- [README.md](./README.md)
- [QUICK_START.md](./QUICK_START.md)
- [FEATURES.md](./FEATURES.md)
