const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('electron', {
  submitCredentials: (credentials) => ipcRenderer.send('admin-credentials', credentials),
  openAdmin: () => ipcRenderer.send('open-admin'),
  login: (credentials) => ipcRenderer.send('login', credentials),
  onLoginResult: (callback) => ipcRenderer.on('login-result', (_event, value) => callback(value)),
  onDeviceUpdate: (callback) => ipcRenderer.on('device-update', (_event, value) => callback(value)),
});

contextBridge.exposeInMainWorld('electronAPI', {
    invoke: (channel, ...args) => ipcRenderer.invoke(channel, ...args)
});
