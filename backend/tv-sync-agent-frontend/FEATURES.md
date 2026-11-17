# Features Documentation

## Dashboard Overview

The TV Sync Agent Frontend provides a comprehensive management dashboard for controlling digital signage displays across multiple locations.

---

## 1. Device Management 🖥️

### Real-Time Status Monitoring
- **Online**: Device is connected and syncing
- **Offline**: Device is not reachable
- **Syncing**: Device is currently updating content
- **Error**: Device encountered an issue

### Device Information Display
Each device shows:
- Device name (e.g., TV-Display-001)
- Lounge name (e.g., Premium Lounge A)
- Location (e.g., Terminal 1, Gate 5)
- IP address and MAC address
- Last sync time (e.g., "5m ago", "2h ago")
- Connection status with color indicators

### Multi-Device Support
- View all devices in a scrollable sidebar
- Click any device to view/edit its configuration
- Status indicators update in real-time
- Quick device selection with visual feedback

---

## 2. Display Mode Configuration 📺

Choose what content appears on your TV displays:

### Schedules Only 📅
- Shows only bus/transport schedules
- Ideal for information kiosks
- No commercial content
- Focuses on departure/arrival times

### Advertisements Only 📺
- Displays promotional content exclusively
- Perfect for advertising-focused displays
- Maximizes ad revenue
- Continuous ad playback

### Both (Hybrid) 🔀
- Combines schedules and advertisements
- Balances information and marketing
- Configurable with layout options
- Most popular mode

**How to Change:**
1. Select a device
2. Navigate to "Display Mode" section
3. Click your preferred mode
4. Changes apply instantly

---

## 3. Layout Configuration 🖼️

*Available only when Display Mode is set to "Both (Hybrid)"*

### Split Screen ⬜⬜
- Divides screen into two sections
- Left side: Schedules
- Right side: Advertisements
- Simultaneous display
- Best for large displays

**Use Cases:**
- Airport terminals
- Bus stations
- Train stations
- Large waiting areas

### Full Screen Alternate 🔄
- Shows content in full screen
- Alternates between schedules and ads
- Timer-based switching
- Better visual impact

**Use Cases:**
- Smaller displays
- Focused content delivery
- High-engagement scenarios
- Premium advertising

**Configuration:**
1. Set display mode to "Both"
2. Navigate to "Layout Mode" section
3. Select Split Screen or Full Screen Alternate
4. Layout switches immediately

---

## 4. Multi-Language Support 🌍

Support for 7 languages to accommodate diverse audiences:

| Language | Native Name | Flag |
|----------|------------|------|
| English | English | 🇺🇸 |
| Spanish | Español | 🇪🇸 |
| French | Français | 🇫🇷 |
| German | Deutsch | 🇩🇪 |
| Chinese | 中文 | 🇨🇳 |
| Japanese | 日本語 | 🇯🇵 |
| Arabic | العربية | 🇸🇦 |

**Features:**
- One-click language switching
- Applies to all on-screen text
- Schedule times remain consistent
- Right-to-left support for Arabic

**How to Change:**
1. Select device
2. Scroll to "Language" section
3. Click desired language flag
4. TV display updates immediately

---

## 5. Emergency Messaging 🚨

Broadcast urgent announcements to displays instantly.

### Priority Levels

| Priority | Color | Use Case |
|----------|-------|----------|
| 🔵 Low | Blue | General announcements, reminders |
| 🟡 Medium | Yellow | Important notices, delays |
| 🟠 High | Orange | Urgent updates, gate changes |
| 🔴 Critical | Red | Emergencies, evacuations |

### Features
- Real-time message preview
- Color-coded display based on priority
- Custom message text
- Easy activation/deactivation
- Overrides current content when active

### Sending a Message
1. Select target device
2. Navigate to "Emergency Message" section
3. Enter message text
4. Select priority level
5. Preview the message
6. Click "Broadcast Message"

### Clearing a Message
- Click "Clear Message" button
- Message removed immediately
- Display returns to normal mode

**Best Practices:**
- Keep messages concise (under 100 characters)
- Use appropriate priority levels
- Test with "Low" priority first
- Clear messages when resolved

---

## 6. Advertisement Tracking & Billing 💰

Monitor ad performance and generate billing reports.

### Metrics Tracked
- **Total Impressions**: Number of times ad was displayed
- **Playback Time**: Total minutes ad was shown
- **Completion Rate**: Percentage of full ad views
- **Billing Amount**: Revenue generated per ad

### Dashboard Statistics
Each device shows:
- Total impressions today
- Total billing amount
- Average completion rate
- Performance trends

### Generating Reports
1. Select device
2. Navigate to "Ad Playback Tracking"
3. Click "View Full Report"
4. Select date range
5. Export as PDF or CSV

### Report Contents
- Device information
- Date range
- Per-ad breakdown
- Total impressions
- Total revenue
- Completion statistics
- Performance graphs

**Use Cases:**
- Client billing
- Performance analysis
- ROI calculation
- Campaign optimization

---

## 7. Device Information ℹ️

Comprehensive device details at a glance:

### Connection Status
- Current status with color indicator
- Status history
- Uptime statistics

### Network Information
- IP address
- MAC address
- Connection type
- Network quality

### Sync Information
- Last successful sync
- Sync frequency
- Sync history
- Error logs

### Location Details
- Physical location
- Terminal/gate information
- GPS coordinates (if available)
- Timezone

### Timestamps
- Device creation date
- Last configuration update
- Last status change

---

## 8. Responsive Design 📱

The dashboard adapts to different screen sizes:

### Desktop (Optimal)
- Full sidebar navigation
- Multi-column layout
- All features visible
- Keyboard shortcuts enabled

### Tablet
- Collapsible sidebar
- Two-column layout
- Touch-optimized controls
- Swipe gestures

### Mobile
- Bottom navigation
- Single-column layout
- Simplified interface
- Essential features only

---

## 9. Real-Time Updates ⚡

The dashboard updates automatically:

### Auto-Refresh
- Device status: Every 30 seconds
- Sync times: Every minute
- Ad statistics: Every 5 minutes
- Emergency messages: Instant

### WebSocket Support
- Real-time device status changes
- Instant message delivery
- Live sync progress
- Connection monitoring

### Manual Refresh
- Click "Refresh" button on any device
- Pull-to-refresh on mobile
- Keyboard shortcut: `Ctrl/Cmd + R`

---

## 10. Accessibility ♿

Built with accessibility in mind:

### Keyboard Navigation
- Tab through all controls
- Enter to activate buttons
- Arrow keys for selection
- Escape to close dialogs

### Screen Reader Support
- ARIA labels on all controls
- Semantic HTML structure
- Descriptive alt text
- Status announcements

### Visual Accessibility
- High contrast colors
- Clear status indicators
- Large touch targets
- Readable fonts (16px minimum)

### Color Blind Friendly
- Not relying solely on color
- Text labels with icons
- Pattern differentiation
- Status symbols

---

## Coming Soon 🚀

Features in development:

- **Bulk Operations**: Configure multiple devices at once
- **Scheduling**: Program display mode changes
- **Analytics Dashboard**: Advanced performance metrics
- **User Roles**: Admin, manager, viewer permissions
- **Dark Mode**: Reduce eye strain
- **Notifications**: Desktop/mobile alerts
- **Schedule Editor**: Create custom schedules
- **Ad Manager**: Upload and manage advertisements

---

## Tips & Tricks 💡

### Power User Tips
1. Use keyboard shortcuts for faster navigation
2. Bookmark frequently used devices
3. Set up multiple browser windows for multi-site monitoring
4. Use the search function for large device fleets

### Best Practices
1. Check device status daily
2. Test emergency messages regularly
3. Review ad performance weekly
4. Update device locations as needed
5. Keep firmware up to date

### Performance Optimization
1. Close unused browser tabs
2. Refresh page if UI becomes sluggish
3. Use Chrome or Edge for best performance
4. Enable hardware acceleration

---

For more information, see the [README](./README.md) or [Quick Start Guide](./QUICK_START.md).
