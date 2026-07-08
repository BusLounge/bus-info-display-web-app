const { app, BrowserWindow, ipcMain } = require('electron');
const path = require('path');
const fs = require('fs');
const { spawn, exec } = require('child_process');
const api = require('./api');
const express = require('express');

let expressApp;
let expressServer;
const EXPRESS_PORT = 4200;

function startExpressServer() {
    expressApp = express();
    // Serve static files from 'www' (our packaged Angular app)
    const wwwPath = path.join(__dirname, 'www');
    expressApp.use(express.static(wwwPath));

    // Handle SPA routing by redirecting all other requests to index.html
    expressApp.get('*', (req, res) => {
        res.sendFile(path.join(wwwPath, 'index.html'));
    });

    expressServer = expressApp.listen(EXPRESS_PORT, () => {
        console.log(`Express server running on http://localhost:${EXPRESS_PORT}`);
    });
}

function stopExpressServer() {
    if (expressServer) {
        expressServer.close();
    }
}

let mainWindow;
let adminWindow;
let loginWindow;
let deviceId;
let pollingInterval;
let goProcess;

const DISPLAY_RESOLUTION_PRESETS = {
  '1280x720': { width: 1280, height: 720 },
  '1920x1080': { width: 1920, height: 1080 },
  '3840x2160': { width: 3840, height: 2160 },
  '7680x4320': { width: 7680, height: 4320 },
};

function normalizeSyncFrequencySeconds(value) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return 30;
  }
  return Math.max(5, Math.floor(parsed));
}

function normalizeDisplayResolution(value) {
  const resolution = String(value || '').trim();
  return DISPLAY_RESOLUTION_PRESETS[resolution] ? resolution : '1920x1080';
}

function normalizeDisplayOrientation(value) {
  const v = String(value || '').toLowerCase().trim();
  if (v === 'portrait' || v === 'portrait-primary') return 'portrait';
  return 'landscape';
}

function getConfigPath() {
    const isPackaged = app.isPackaged;
    if (isPackaged || fs.existsSync(path.join(__dirname, 'bin', 'agent.exe'))) {
        return path.join(app.getPath('userData'), 'config.json');
    }
    return path.join(__dirname, '..', 'tv-sync-agent-go', 'config.json');
}


function stopGoProcess() {
    return new Promise((resolve) => {
        if (!goProcess || goProcess.killed) {
            return resolve();
        }

        console.log(`Stopping Go backend process with PID: ${goProcess.pid}...`);
        
        if (process.platform === "win32") {
            exec(`taskkill /PID ${goProcess.pid} /T /F`, (err, stdout, stderr) => {
                if (err) {
                    console.error(`Failed to kill process on Windows: ${err}`);
                }
                console.log('Go backend process stopped on Windows.');
                goProcess = null;
                // Add a small delay for the OS to release the port
                setTimeout(resolve, 1000);
            });
        } else {
            // For macOS and Linux
            goProcess.kill('SIGINT');
            goProcess.on('close', () => {
                console.log('Go backend process stopped.');
                goProcess = null;
                setTimeout(resolve, 1000);
            });
        }
    });
}


function startGoProcess() {
    const isPackaged = app.isPackaged;
    
    if (isPackaged || fs.existsSync(path.join(__dirname, 'bin', 'agent.exe'))) {
        // Run the bundled executable
        const agentExe = path.join(__dirname, 'bin', 'agent.exe');
        const userDataPath = app.getPath('userData');
        const configPath = getConfigPath();
        
        console.log(`Starting compiled Go backend process: ${agentExe}`);
        console.log(`Using config path: ${configPath}`);
        console.log(`Working directory: ${userDataPath}`);

        // Ensure default config exists
        if (!fs.existsSync(configPath)) {
            const defaultConfig = {
                loungeId: "",
                loungeGroup: "",
                language: "en",
                displayMode: "both",
                layoutMode: "split-screen",
                storeDir: "./local-store",
                apiBaseUrl: "http://18.140.238.163:4000/api",
                syncFrequencySeconds: 30,
                displayResolution: "1920x1080",
                displayOrientation: "landscape"
            };
            fs.writeFileSync(configPath, JSON.stringify(defaultConfig, null, 2), 'utf8');
        }
        
        goProcess = spawn(agentExe, ['-config', configPath], { cwd: userDataPath });
    } else {
        // Fallback to go run for development
        const agentPath = path.join(__dirname, '..', 'tv-sync-agent-go');
        const configPath = getConfigPath();
        console.log('Starting Go backend process (dev mode)...');
        goProcess = exec(`go run cmd/agent/main.go -config "${configPath}"`, { cwd: agentPath });
    }

    if (goProcess.stdout) goProcess.stdout.on('data', (data) => console.log(`Go Backend: ${data.toString()}`));
    if (goProcess.stderr) goProcess.stderr.on('data', (data) => console.error(`Go Backend Error: ${data.toString()}`));
    
    goProcess.on('close', (code) => {
        if (code !== 0 && code !== null) {
            console.error(`Go backend process exited with code ${code}`);
        }
        goProcess = null;
    });
}


function createMainWindow() {
  mainWindow = new BrowserWindow({
    width: 800,
    height: 600,
    kiosk: true,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      nodeIntegration: false,
      contextIsolation: true
    }
  });

  mainWindow.loadFile('index.html');

  mainWindow.on('closed', function () {
    mainWindow = null;
    if (pollingInterval) {
      clearInterval(pollingInterval);
    }
  });
}

function createAdminWindow() {
  adminWindow = new BrowserWindow({
    width: 500,
    height: 600,
    webPreferences: {
        preload: path.join(__dirname, 'preload.js'),
        contextIsolation: true,
        nodeIntegration: false
    }
  });

  adminWindow.loadFile('admin.html');

  adminWindow.on('closed', function () {
    adminWindow = null;
  });
}

function createLoginWindow() {
    loginWindow = new BrowserWindow({
      width: 400,
      height: 500,
      webPreferences: {
          preload: path.join(__dirname, 'preload.js'),
          contextIsolation: true,
          nodeIntegration: false
      }
    });
  
    loginWindow.loadFile('login.html');
  
    loginWindow.on('closed', function () {
      loginWindow = null;
    });
  }

app.on('ready', () => {
    // We start the express server when app is ready
    if (fs.existsSync(path.join(__dirname, 'www'))) {
        startExpressServer();
    }
    createMainWindow();
});
app.on('window-all-closed', function () {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('will-quit', () => {
    stopGoProcess();
    stopExpressServer();
});

app.on('activate', function () {
  if (mainWindow === null) {
    createMainWindow();
  }
});

ipcMain.on('open-admin', () => {
    if (!loginWindow && !adminWindow) {
        createLoginWindow();
    }
});

ipcMain.on('login', (event, { username, password }) => {
    if (username === 'admin' && password === 'password') {
        createAdminWindow();
        if (loginWindow) {
            loginWindow.close();
        }
    } else {
        event.sender.send('login-result', 'Invalid credentials');
    }
});

ipcMain.handle('bridge:get', async (event, endpoint) => {
    const configPath = getConfigPath();
    let config = {};
    try {
        const data = await fs.promises.readFile(configPath, 'utf8');
        config = JSON.parse(data);
    } catch (err) {
        console.error('Could not read tv-sync-agent-go config for bridge:get', err);
    }

    switch (endpoint) {
      case 'status':
        return {
            language: config.language || 'en',
            displayMode: config.displayMode || 'both',
            layoutMode: config.layoutMode || 'split-screen',
            syncFrequencySeconds: Number(config.syncFrequencySeconds) > 0 ? Number(config.syncFrequencySeconds) : 30,
            displayResolution: normalizeDisplayResolution(config.displayResolution),
            displayOrientation: normalizeDisplayOrientation(config.displayOrientation),
            broadcastsEnabled: true, // You may want to make this dynamic
            lastBroadcastSync: new Date().toISOString(),
        };
      // Add cases for 'schedule', 'ads', etc. as you build them out
      // For now, return empty data to prevent errors
      case 'schedule':
        return { loungeId: config.loungeId, departuresRaw: { departures: [] }, arrivalsRaw: { arrivals: [] } };
      case 'ads':
        return { items: [] };
      case 'lounge-ads':
        return { items: [] };
      case 'broadcasts':
        return { items: [] };
      default:
        return {};
    }
  });

async function pollForUpdates() {
  if (!deviceId || !mainWindow) return;

  try {
    const [status, config] = await Promise.all([
      api.getDeviceStatus(deviceId),
      api.getDeviceConfig(deviceId)
    ]);

    if (mainWindow) {
      mainWindow.webContents.send('device-update', { status, config });
    }
  } catch (error) {
    console.error('Error polling for updates:', error);
  }
}

ipcMain.on('admin-credentials', async (event, credentials) => {
    console.log('Admin credentials received:', credentials);

    const configPath = getConfigPath();
    
    try {
        let config = {};
        if (fs.existsSync(configPath)) {
            const data = await fs.promises.readFile(configPath, 'utf8');
            config = JSON.parse(data);
        } else {
            // Provide default base config
            config = {
                storeDir: "./local-store",
                apiBaseUrl: "http://18.140.238.163:4000/api"
            };
        }
        
      const syncFrequencySeconds = normalizeSyncFrequencySeconds(credentials.syncFrequencySeconds);
      const displayResolution = normalizeDisplayResolution(credentials.displayResolution);
      const displayOrientation = normalizeDisplayOrientation(credentials.displayOrientation);
        
        config.loungeId = credentials.loungeId;
        config.loungeGroup = credentials.loungeName;
        config.language = credentials.language;
        config.displayMode = credentials.displayMode;
        config.layoutMode = credentials.layoutMode;
      config.syncFrequencySeconds = syncFrequencySeconds;
      config.displayResolution = displayResolution;
      config.displayOrientation = displayOrientation;

        await fs.promises.writeFile(configPath, JSON.stringify(config, null, 2), 'utf8');
        console.log('Successfully updated config.json.');

        await stopGoProcess();
        
        startGoProcess();

        if(mainWindow) {
            deviceId = credentials.loungeId;
            const { loungeId, loungeName, displayMode, layoutMode, language } = credentials;
            const url = `http://localhost:4200/bids-display?loungeId=${encodeURIComponent(loungeId)}&loungeName=${encodeURIComponent(loungeName)}&kiosk=true&displayMode=${encodeURIComponent(displayMode)}&layoutMode=${encodeURIComponent(layoutMode)}&language=${encodeURIComponent(language)}&syncFrequencySeconds=${encodeURIComponent(syncFrequencySeconds)}&displayResolution=${encodeURIComponent(displayResolution)}&displayOrientation=${encodeURIComponent(displayOrientation)}`;
            
            // Add a delay to ensure the Go server is ready before loading the URL
            setTimeout(() => {
                mainWindow.loadURL(url);
        
                if (pollingInterval) {
                  clearInterval(pollingInterval);
                }

                pollingInterval = setInterval(pollForUpdates, syncFrequencySeconds * 1000);
                pollForUpdates();
            }, 3000); // 3-second delay
        }
    
        if(adminWindow) {
            adminWindow.close();
        }

    } catch (err) {
        console.error('An error occurred during admin setup:', err);
    }
});
