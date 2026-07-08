# Quick Start Guide - TV Sync Agent Frontend

Get up and running with the TV Sync Agent Frontend in 5 minutes!

## Prerequisites

Make sure you have:
- **Node.js** v18+ installed ([Download](https://nodejs.org/))
- **npm** v11+ (comes with Node.js)
- **tv-sync-agent-go** backend running ([Backend Setup](../bus-info-display-web-app/tv-sync-agent-go/README_TV_BOX_SETUP.md))

## Installation (3 steps)

### 1. Navigate to the project
```bash
cd tv-sync-agent-frontend
```

### 2. Install dependencies
```bash
npm install
```

### 3. Start the application
```bash
npm start
```

That's it! 🎉 Open http://localhost:4200 in your browser.

## First-Time Configuration

### Connect to Backend

1. Open `src/environments/environment.development.ts`
2. Update the API URL to match your backend:
   ```typescript
   export const environment = {
     production: false,
     apiUrl: 'http://localhost:8080/api',  // Update this
     wsUrl: 'ws://localhost:8080/ws',
     version: '1.0.0'
   };
   ```

### Start Managing Devices

1. **Select a Device**: Click on any device in the left sidebar
2. **Choose Display Mode**: Select Schedules Only, Ads Only, or Both
3. **Configure Layout**: Choose Split Screen or Full-Screen Alternate
4. **Set Language**: Pick from 7 supported languages
5. **Test Emergency Message**: Send a test message to verify connectivity

## Common Tasks

### Change Display Mode
1. Select device from sidebar
2. Click on desired mode under "Display Mode" section
3. Changes apply immediately

### Send Emergency Message
1. Scroll to "Emergency Message" section
2. Enter your message
3. Select priority level
4. Click "Broadcast Message"

### View Ad Performance
1. Select device
2. Scroll to "Ad Playback Tracking" section
3. Click "View Full Report" for detailed analytics

### Switch Languages
1. Select device
2. Find "Language" section
3. Click on desired language flag

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl/Cmd + D` | Focus device search |
| `Ctrl/Cmd + R` | Refresh current device |
| `Ctrl/Cmd + E` | Focus emergency message |

## Troubleshooting

### App won't start
```bash
# Clear node_modules and reinstall
rm -rf node_modules package-lock.json
npm install
```

### Can't connect to backend
1. Verify backend is running: `curl http://localhost:8080/api/health`
2. Check CORS settings in backend
3. Verify API URL in environment file

### Changes not appearing
1. Hard refresh: `Ctrl+Shift+R` or `Cmd+Shift+R`
2. Clear browser cache
3. Restart dev server: `npm start`

## What's Next?

- 📚 Read the full [README.md](./README.md)
- 🔧 Configure your backend connection
- 🎨 Customize colors and branding
- 📊 Explore the analytics features
- 🌐 Test multi-language support

## Need Help?

- Check the [README](./README.md) for detailed documentation
- Review backend logs for API errors
- Open an issue in the repository

---

Happy managing! 🚀
