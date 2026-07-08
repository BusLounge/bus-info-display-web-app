# TV Sync Agent Frontend

A modern, minimalistic, and professional web application for managing TV displays in lounges, terminals, and waiting areas. This frontend provides a comprehensive dashboard for configuring display modes, managing devices, broadcasting emergency messages, and tracking advertisement performance.

## Features

### 🖥️ **Device Management**
- Real-time device status monitoring (Online, Offline, Syncing, Error)
- Device information display (IP address, MAC address, location)
- Last sync time tracking
- Multi-device support with easy device selection

### 📺 **Display Mode Configuration**
- **Schedules Only**: Display bus/transport schedules exclusively
- **Advertisements Only**: Show only promotional content
- **Both (Hybrid)**: Combine schedules and advertisements

### 🖼️ **Layout Options**
- **Split Screen**: Display schedules and ads side-by-side
- **Full Screen Alternate**: Switch between full-screen schedules and ads

### 🌍 **Multi-Language Support**
Support for multiple languages with easy switching:
- English 🇺🇸
- Español 🇪🇸
- Français 🇫🇷
- Deutsch 🇩🇪
- 中文 🇨🇳
- 日本語 🇯🇵
- العربية 🇸🇦

### 🚨 **Emergency Messaging**
- Broadcast urgent announcements to one or more displays
- Priority levels: Low, Medium, High, Critical
- Color-coded message display based on priority
- Real-time message preview
- Easy activation and deactivation

### 💰 **Advertisement Tracking & Billing**
- Track ad impressions and playback metrics
- Monitor completion rates
- Generate billing reports
- View performance statistics per device

## Technology Stack

- **Framework**: Angular 21.1.0
- **Language**: TypeScript
- **Styling**: SCSS with modern gradient design
- **State Management**: Angular Signals (reactive state)
- **Architecture**: Standalone components

## Project Structure

```
tv-sync-agent-frontend/
├── src/
│   ├── app/
│   │   ├── core/
│   │   │   ├── models/           # Data models and interfaces
│   │   │   │   ├── device.model.ts
│   │   │   │   ├── ad-tracking.model.ts
│   │   │   │   └── index.ts
│   │   │   └── services/         # Core services
│   │   │       ├── device.service.ts
│   │   │       ├── emergency-message.service.ts
│   │   │       ├── ad-tracking.service.ts
│   │   │       └── index.ts
│   │   ├── features/
│   │   │   └── dashboard/        # Main dashboard feature
│   │   │       ├── dashboard.component.ts
│   │   │       ├── dashboard.component.html
│   │   │       └── dashboard.component.scss
│   │   ├── shared/               # Shared components, directives, pipes
│   │   ├── app.routes.ts         # Application routing
│   │   └── app.ts                # Root component
│   └── styles.scss               # Global styles
└── package.json
```

## Installation

### Prerequisites
- Node.js (v18 or higher)
- npm (v11 or higher)

### Steps

1. **Navigate to the project directory**:
   ```bash
   cd tv-sync-agent-frontend
   ```

2. **Install dependencies**:
   ```bash
   npm install
   ```

3. **Start the development server**:
   ```bash
   npm start
   ```

4. **Open your browser** and navigate to:
   ```
   http://localhost:4200
   ```

## Usage

### Dashboard Overview

The dashboard is organized into several sections:

#### 1. **Device Sidebar**
- View all registered TV devices
- See real-time status indicators
- Click on a device to configure it

#### 2. **Display Mode Selection**
Choose how content is displayed:
- Schedules only for information-focused displays
- Ads only for promotional displays
- Both for hybrid displays

#### 3. **Layout Configuration**
Available when "Both" mode is selected:
- Split screen for simultaneous display
- Full-screen alternate for timed switching

#### 4. **Language Settings**
Select the display language for the TV interface

#### 5. **Emergency Messaging**
- Enter urgent message text
- Select priority level
- Broadcast to selected device(s)
- Preview message before sending

#### 6. **Device Information**
View detailed device metrics:
- Connection status
- Network information
- Sync history
- Location details

#### 7. **Ad Tracking**
Monitor advertisement performance:
- Total impressions
- Billing calculations
- Completion rates

## API Integration

This frontend is designed to work with the `tv-sync-agent-go` backend. To integrate:

1. Update the API base URL in your environment configuration
2. Ensure the backend is running and accessible
3. Configure CORS settings if needed

### Example API Service Setup

```typescript
// Create src/app/core/services/api.service.ts
import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  private apiUrl = 'http://localhost:8080/api'; // Your Go backend URL

  constructor(private http: HttpClient) {}

  // Add API methods here
}
```

## Development

### Running Tests
```bash
npm test
```

### Building for Production
```bash
npm run build
```

The build artifacts will be stored in the `dist/` directory.

### Code Formatting
This project uses Prettier for code formatting:
```bash
npm run format
```

## Design Philosophy

### Modern & Minimalistic
- Clean, uncluttered interface
- Intuitive navigation
- Focus on essential information

### Professional
- Consistent design language
- High-quality visual hierarchy
- Responsive layout

### Interactive
- Real-time updates
- Smooth transitions
- Immediate visual feedback

### Color Scheme
- Primary gradient: Purple to violet (`#667eea` to `#764ba2`)
- Status colors:
  - Online: Green (`#10b981`)
  - Offline: Red (`#ef4444`)
  - Syncing: Yellow (`#f59e0b`)
- Emergency message colors based on priority

## Responsive Design

The application is fully responsive and works on:
- Desktop computers (optimal experience)
- Tablets
- Mobile devices (simplified layout)

## Future Enhancements

- [ ] Real-time WebSocket updates for device status
- [ ] Advanced analytics dashboard
- [ ] Schedule editor
- [ ] Advertisement upload and management
- [ ] User authentication and role-based access
- [ ] Export reports (PDF, CSV)
- [ ] Device grouping and bulk operations
- [ ] Notification system
- [ ] Dark mode support

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Open a Pull Request

## License

This project is part of the BusInfo system.

## Support

For issues or questions, please create an issue in the repository or contact the development team.

---

**Built with ❤️ using Angular 21**
