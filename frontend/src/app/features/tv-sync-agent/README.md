# TV Sync Agent - Feature Documentation

## Overview

The TV Sync Agent is a comprehensive, modern, and professional frontend interface for managing lounge TV displays. It provides real-time control over content display, device management, emergency messaging, and billing tracking.

## Features

### 1. **Display Mode Configuration**
Control what content appears on your TV screens:
- **Schedules Only**: Display only bus schedules
- **Ads Only**: Display only advertisements
- **Both**: Show both schedules and advertisements

### 2. **Layout Options**
Choose how content is arranged:
- **Split Screen**: Display schedules on one side and ads on the other
- **Full Screen Alternate**: Alternate between full-screen schedules and ads

### 3. **Multi-Language Support**
Support for 10+ languages:
- English, Español, Français, Deutsch
- 中文, 日本語, हिन्दी, العربية
- Português, Русский

### 4. **Emergency Messaging**
Broadcast urgent messages to all displays:
- Customizable message text
- Priority levels (Low, Medium, High, Critical)
- Custom background and text colors
- Configurable display duration
- Real-time preview before activation

### 5. **Device Management**
Monitor and control all connected TV devices:
- Real-time device status (Online, Offline, Syncing, Error)
- Last sync timestamp
- Device location and IP address
- Firmware version and screen resolution
- System metrics (uptime, CPU, memory usage)
- Remote control actions (Refresh, Reboot)

### 6. **Ad Playback Tracking**
Complete billing and analytics:
- Track every ad playback event
- Record play duration and completion status
- Device-level tracking
- Automatic billing report generation
- Export reports in PDF, CSV, or Excel format

### 7. **Dashboard Analytics**
Real-time statistics at a glance:
- Online/Offline device counts
- Total ads played (7-day rolling)
- Last sync time
- System health overview

## File Structure

```
frontend/src/app/
├── core/
│   ├── models/
│   │   └── tv-sync-agent.model.ts      # TypeScript interfaces
│   └── services/
│       └── tv-sync-agent.service.ts    # API integration service
└── features/
    └── tv-sync-agent/
        ├── tv-sync-agent.component.ts    # Component logic
        ├── tv-sync-agent.component.html  # Template
        └── tv-sync-agent.component.scss  # Styles
```

## Usage

### Accessing the TV Sync Agent

Navigate to: `http://your-domain/tv-sync-agent`

### Quick Start

1. **Select a Lounge**: Use the dropdown in the header
2. **Configure Display Mode**: Go to "Display Settings" tab
3. **Choose Layout**: Select split-screen or full-screen alternate
4. **Set Language**: Pick from 10+ supported languages
5. **Monitor Devices**: Check device status in "Devices" tab
6. **Emergency Messages**: Create urgent alerts in "Emergency" tab
7. **View Billing**: Generate reports in "Billing" tab

## API Endpoints

The TV Sync Agent requires the following backend API endpoints:

### Configuration
- `GET /tv-sync/config/:loungeId` - Get agent configuration
- `POST /tv-sync/config` - Save agent configuration
- `PUT /tv-sync/config/:loungeId` - Update agent configuration

### Device Management
- `GET /tv-sync/device/:deviceId/status` - Get device status
- `GET /tv-sync/devices` - Get all devices
- `PUT /tv-sync/device/:deviceId/status` - Update device status
- `POST /tv-sync/device/:deviceId/command` - Send remote command
- `POST /tv-sync/device/:deviceId/screenshot` - Capture screenshot

### Emergency Messaging
- `POST /tv-sync/emergency/:loungeId` - Set emergency message
- `DELETE /tv-sync/emergency/:loungeId` - Clear emergency message
- `GET /tv-sync/emergency/:loungeId` - Get current emergency message

### Content
- `GET /tv-sync/schedules/:loungeId` - Get schedules for display
- `GET /tv-sync/ads/:loungeId` - Get ads for display

### Ad Tracking & Billing
- `POST /tv-sync/tracking/playback` - Track ad playback
- `GET /tv-sync/tracking/playback/:loungeId` - Get playback history
- `POST /tv-sync/billing/report` - Generate billing report
- `GET /tv-sync/billing/report/download` - Download billing report

### Sync Operations
- `POST /tv-sync/sync/:loungeId` - Force sync
- `GET /tv-sync/validate/:loungeId` - Validate content

## TypeScript Interfaces

### Main Interfaces

```typescript
interface TVSyncAgentConfig {
  id?: string;
  loungeId: string;
  loungeName: string;
  deviceId?: string;
  displayMode: DisplayMode;
  emergencyMessage?: EmergencyMessage;
  language: string;
  layoutOption: LayoutOption;
  adPlaybackTracking: AdPlaybackTracking[];
  deviceStatus: DeviceStatus;
}

interface DeviceStatus {
  deviceId: string;
  status: 'online' | 'offline' | 'syncing' | 'error';
  lastSyncTime?: string;
  location?: DeviceLocation;
  ipAddress?: string;
  firmwareVersion?: string;
  uptime?: number;
  memoryUsage?: number;
  cpuUsage?: number;
}

interface EmergencyMessage {
  enabled: boolean;
  message: string;
  priority: 'low' | 'medium' | 'high' | 'critical';
  backgroundColor?: string;
  textColor?: string;
  displayDuration?: number;
}
```

See `tv-sync-agent.model.ts` for complete interface definitions.

## Styling & Theme

The TV Sync Agent uses a modern, minimalistic design with:
- **Primary Colors**: Purple gradient (#667eea to #764ba2)
- **Typography**: Inter font family
- **Components**: Rounded corners, subtle shadows, smooth transitions
- **Responsive**: Mobile-friendly layout
- **Accessibility**: High contrast, WCAG compliant

## Component Architecture

### State Management
Uses Angular Signals for reactive state:
- `lounges` - Available lounges
- `devices` - Connected devices
- `config` - Current configuration
- `emergencyMessage` - Active emergency message
- `adPlaybackHistory` - Recent ad playbacks

### Auto-Refresh
- Polls for updates every 30 seconds
- Maintains fresh device status
- Updates statistics in real-time

### Error Handling
- User-friendly error notifications
- Graceful degradation on API failures
- Loading states for better UX

## Future Enhancements

Potential features for future releases:
- Live video preview of TV displays
- Custom scheduling for content rotation
- Advanced analytics and reporting
- Multi-lounge bulk operations
- Mobile app for remote management
- Push notifications for device alerts
- A/B testing for advertisement effectiveness

## Browser Support

- Chrome/Edge (latest 2 versions)
- Firefox (latest 2 versions)
- Safari (latest 2 versions)
- Mobile browsers (iOS Safari, Chrome Mobile)

## Dependencies

- Angular 21+
- RxJS 7.8+
- TypeScript 5.9+
- Modern CSS with SCSS

## License

Part of the Bus Info Display Web Application

---

For questions or support, please contact the development team.
