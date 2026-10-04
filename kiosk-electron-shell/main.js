const { app, BrowserWindow, ipcMain } = require('electron');
const path = require('path');
const fs = require('fs');
const http = require('http');
const { spawn, execFile } = require('child_process');
const api = require('./api');
const express = require('express');

let expressApp;
let expressServer;
const EXPRESS_PORT = 4200;

function getAppResourcePath(...segments) {
  const resourceRoot = app.isPackaged ? process.resourcesPath : __dirname;
  return path.join(resourceRoot, ...segments);
}

function getLocalScheduleSnapshot(port) {
  return new Promise((resolve, reject) => {
    const request = http.get({
      hostname: '127.0.0.1',
      port,
      path: '/local/schedule',
      timeout: 3000
    }, (response) => {
      let body = '';
      response.setEncoding('utf8');
      response.on('data', (chunk) => { body += chunk; });
      response.on('end', () => {
        if (response.statusCode < 200 || response.statusCode >= 300) {
          reject(new Error(`Local schedule endpoint returned HTTP ${response.statusCode}`));
          return;
        }

        try {
          resolve(JSON.parse(body));
        } catch (error) {
          reject(error);
        }
      });
    });

    request.on('timeout', () => request.destroy(new Error('Local schedule endpoint timed out')));
    request.on('error', reject);
  });
}

async function readAgentSnapshot(filename) {
  const configPath = getConfigPath();
  const configData = await fs.promises.readFile(configPath, 'utf8');
  const config = JSON.parse(configData);
  const storeDir = typeof config.storeDir === 'string' ? config.storeDir : './local-store';
  const snapshotPath = path.resolve(path.dirname(configPath), storeDir, filename);
  const snapshotData = await fs.promises.readFile(snapshotPath, 'utf8');
  return JSON.parse(snapshotData);
}

function startExpressServer() {
    expressApp = express();
    // Serve static files from 'www' (our packaged Angular app)
  const wwwPath = getAppResourcePath('www');
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
  if (!expressServer) return Promise.resolve();

  const server = expressServer;
  expressServer = null;
  return new Promise((resolve) => {
    server.close((error) => {
      if (error) {
        console.error('Failed to stop local Express server:', error);
      } else {
        console.log('Local Express server stopped.');
      }
      resolve();
    });
  });
}

let mainWindow;
let adminWindow;
let loginWindow;
let deviceId;
let pollingInterval;
let goProcess;
let isQuitting = false;

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
  const child = goProcess;
  if (!child) return Promise.resolve();

  console.log(`Stopping Go backend process with PID: ${child.pid || 'unknown'}...`);

  return new Promise((resolve) => {
    let settled = false;
    let killTimer;

    const finish = () => {
      if (settled) return;
      settled = true;
      clearTimeout(killTimer);
      if (goProcess === child) goProcess = null;
      console.log('Go backend process stopped.');
      resolve();
    };

    const forceKill = () => {
      if (child.exitCode !== null || child.signalCode !== null) {
        finish();
        return;
      }

      console.warn('Go backend did not stop in time; forcing termination.');
      try {
        child.kill('SIGKILL');
      } catch (error) {
        console.error('Failed to force-stop Go backend:', error);
        finish();
        return;
      }
      killTimer = setTimeout(finish, 3000);
    };

    child.once('close', finish);
    if (child.exitCode !== null || child.signalCode !== null) {
      finish();
      return;
    }

    if (process.platform === 'win32' && child.pid) {
      execFile('taskkill', ['/PID', String(child.pid), '/T', '/F'], (error) => {
        if (settled) return;
        if (error) {
          console.error('Windows taskkill failed; trying child-process termination:', error);
        }
        killTimer = setTimeout(forceKill, 3000);
      });
    } else {
      try {
        child.kill('SIGTERM');
      } catch (error) {
        console.error('Failed to signal Go backend:', error);
      }
      killTimer = setTimeout(forceKill, 5000);
    }
  });
}


function startGoProcess() {
  if (goProcess && goProcess.exitCode === null && goProcess.signalCode === null) {
    console.log(`Go backend is already running with PID: ${goProcess.pid || 'unknown'}.`);
    return;
  }

  const agentExe = getAppResourcePath('bin', 'agent.exe');
  if (app.isPackaged || fs.existsSync(agentExe)) {
    if (!fs.existsSync(agentExe)) {
      console.error(`Bundled Go backend executable was not found: ${agentExe}`);
      return;
    }

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
        
        try {
          goProcess = spawn(agentExe, ['-config', configPath], { cwd: userDataPath });
        } catch (error) {
          console.error('Failed to start bundled Go backend:', error);
          goProcess = null;
          return;
        }
    } else {
        // Fallback to go run for development
        const agentPath = path.join(__dirname, '..', 'tv-sync-agent-go');
        const configPath = getConfigPath();
        console.log('Starting Go backend process (dev mode)...');
        try {
          goProcess = spawn('go', ['run', 'cmd/agent/main.go', '-config', configPath], { cwd: agentPath });
        } catch (error) {
          console.error('Failed to start Go backend in development mode:', error);
          goProcess = null;
          return;
        }
    }

    const child = goProcess;
    if (child.stdout) child.stdout.on('data', (data) => console.log(`Go Backend: ${data.toString()}`));
    if (child.stderr) child.stderr.on('data', (data) => console.error(`Go Backend Error: ${data.toString()}`));

    child.on('error', (error) => {
        console.error('Go backend process could not be started:', error);
    });
    
    child.on('close', (code) => {
        if (code !== 0 && code !== null) {
            console.error(`Go backend process exited with code ${code}`);
        } else {
          console.log(`Go backend process exited with code ${code}.`);
        }
        if (goProcess === child) goProcess = null;
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

  const angularIndexPath = getAppResourcePath('www', 'index.html');

  if (fs.existsSync(angularIndexPath)) {
    mainWindow.loadURL('http://localhost:4200/bids-display');
  } else {
    mainWindow.loadFile('index.html');
  }

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
  if (fs.existsSync(getAppResourcePath('www'))) {
        startExpressServer();
    }
  startGoProcess();
    createMainWindow();
});
app.on('window-all-closed', function () {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('before-quit', (event) => {
  if (isQuitting) return;

  event.preventDefault();
  isQuitting = true;

  Promise.all([stopGoProcess(), stopExpressServer()])
    .catch((error) => console.error('Error during application shutdown:', error))
    .finally(() => app.quit());
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
      case 'schedule': {
        const localBridgePort = Number(config.localBridgePort) || 3000;
        try {
          return await getLocalScheduleSnapshot(localBridgePort);
        } catch (bridgeError) {
          const storeDir = typeof config.storeDir === 'string' ? config.storeDir : './local-store';
          const snapshotPath = path.resolve(path.dirname(configPath), storeDir, 'schedule.json');
          try {
            const snapshot = await fs.promises.readFile(snapshotPath, 'utf8');
            return JSON.parse(snapshot);
          } catch (cacheError) {
            console.error('Could not load the local schedule endpoint or cached snapshot:', bridgeError, cacheError);
            throw new Error('Local schedule data is unavailable');
          }
        }
      }
      case 'ads':
        return readAgentSnapshot('ads-manifest.json');
      case 'lounge-ads':
        return readAgentSnapshot('lounge-ads.json');
      case 'broadcasts':
        return readAgentSnapshot('broadcasts.json');
      default:
        return {};
    }
  });

ipcMain.handle('bridge:get-ads', () => readAgentSnapshot('ads-manifest.json'));
ipcMain.handle('bridge:get-broadcasts', () => readAgentSnapshot('broadcasts.json'));
ipcMain.handle('bridge:get-lounge-ads', () => readAgentSnapshot('lounge-ads.json'));

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
