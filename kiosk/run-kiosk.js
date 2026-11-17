#!/usr/bin/env node
/*
 * Runs the kiosk for local testing on macOS or Windows, then puts the kiosk
 * folder back the way it was.
 *
 *   node run-kiosk.js            start the kiosk using run-kiosk.conf
 *   node run-kiosk.js --cleanup  only restore files left over from a run that was interrupted
 *
 * Stop the kiosk with Ctrl+C in this terminal. The script then stops every
 * process it started and restores the files the run changed:
 *   - tv-sync-agent-go/config.json   (lounge and server settings are written for the run)
 *   - tv-sync-agent-go/local-store/  (schedule, ads and media downloaded by the agent)
 *   - local-store/                   (media the agent mirrors here)
 *   - kiosk-electron-shell/bin/agent.exe (moved aside so Electron runs the agent from source)
 * Files are restored to how they were when the script started, not to git.
 */
'use strict';

const fs = require('fs');
const net = require('net');
const path = require('path');
const http = require('http');
const { spawn, spawnSync } = require('child_process');

const KIOSK_DIR = __dirname;
const CONF_PATH = path.join(KIOSK_DIR, 'run-kiosk.conf');
const BACKUP_DIR = path.join(KIOSK_DIR, '.run-kiosk-backup');
const MANIFEST_PATH = path.join(BACKUP_DIR, 'manifest.json');

const SHELL_DIR = path.join(KIOSK_DIR, 'kiosk-electron-shell');
const FRONTEND_DIR = path.join(KIOSK_DIR, 'kiosk-frontend');
const AGENT_DIR = path.join(KIOSK_DIR, 'tv-sync-agent-go');
const AGENT_CONFIG = path.join(AGENT_DIR, 'config.json');
const AGENT_EXE = path.join(SHELL_DIR, 'bin', 'agent.exe');
const WWW_DIR = path.join(SHELL_DIR, 'www');

const DISPLAY_PORT = 4200;
const BRIDGE_PORT = 3001;
const OS_PLATFORMS = { mac: 'darwin', windows: 'win32' };

let osName = null; // set from the config
const children = [];
let shuttingDown = false;

function log(message) {
  console.log(`[run-kiosk] ${message}`);
}

function fail(message) {
  console.error(`[run-kiosk] ERROR: ${message}`);
  process.exit(1);
}

// ---------------------------------------------------------------- config

function readConfig() {
  if (!fs.existsSync(CONF_PATH)) fail(`Settings file not found: ${CONF_PATH}`);

  const config = {};
  for (const rawLine of fs.readFileSync(CONF_PATH, 'utf8').split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line || line.startsWith('#')) continue;
    const colon = line.indexOf(':');
    if (colon === -1) fail(`Bad line in run-kiosk.conf (expected "key: value"): ${line}`);
    const key = line.slice(0, colon).trim();
    let value = line.slice(colon + 1).trim();
    // Strip an inline comment that starts with " #"
    const hash = value.search(/\s#/);
    if (hash !== -1) value = value.slice(0, hash).trim();
    config[key] = value.replace(/^["']|["']$/g, '');
  }

  config.os = String(config.os || '').toLowerCase();
  config.mode = String(config.mode || 'electron').toLowerCase();
  config.rebuildDisplay = String(config.rebuildDisplay).toLowerCase() === 'true';
  config.openBrowser = String(config.openBrowser).toLowerCase() !== 'false';
  config.serverUrl = String(config.serverUrl || 'http://localhost:8083').replace(/\/+$/, '');

  if (!OS_PLATFORMS[config.os]) fail(`"os" must be mac or windows (got "${config.os}").`);
  if (process.platform !== OS_PLATFORMS[config.os]) {
    const actual = { darwin: 'macOS', win32: 'Windows' }[process.platform] || process.platform;
    fail(`run-kiosk.conf says "os: ${config.os}" but this computer is ${actual}. Change "os" in run-kiosk.conf.`);
  }
  if (!['electron', 'browser'].includes(config.mode)) {
    fail(`"mode" must be electron or browser (got "${config.mode}").`);
  }
  if (!config.loungeId) fail('"loungeId" is required in run-kiosk.conf.');
  if (!config.loungeName) fail('"loungeName" is required in run-kiosk.conf.');
  return config;
}

// ---------------------------------------------------------------- processes

function npmCommand() {
  return osName === 'windows' ? 'npm.cmd' : 'npm';
}

// Runs a command to completion (used for installs and builds).
function runSync(command, args, cwd) {
  log(`${path.relative(KIOSK_DIR, cwd) || '.'}> ${command} ${args.join(' ')}`);
  const result = spawnSync(command, args, {
    cwd,
    stdio: 'inherit',
    shell: osName === 'windows', // .cmd files need a shell on Windows
  });
  if (result.error) fail(`Could not run ${command}: ${result.error.message}`);
  if (result.status !== 0) fail(`${command} ${args.join(' ')} failed (exit code ${result.status}).`);
}

// Starts a long-running process. On macOS it gets its own process group so the
// whole tree (npm -> electron -> go run -> agent) can be stopped together.
function start(name, command, args, cwd) {
  log(`starting ${name}: ${command} ${args.join(' ')}`);
  // Some tools (for example VS Code extensions) set ELECTRON_RUN_AS_NODE, which
  // makes Electron run as plain Node and the kiosk app crash on startup.
  const env = { ...process.env };
  delete env.ELECTRON_RUN_AS_NODE;
  const child = spawn(command, args, {
    cwd,
    env,
    stdio: 'inherit',
    shell: osName === 'windows',
    detached: osName === 'mac',
  });
  child.name = name;
  child.on('error', (error) => {
    console.error(`[run-kiosk] ${name} could not start: ${error.message}`);
    shutdown(1);
  });
  child.on('exit', (code, signal) => {
    child.exited = true;
    if (!shuttingDown) {
      log(`${name} stopped (${signal || `exit code ${code}`}).`);
      shutdown(code || 0);
    }
  });
  children.push(child);
  return child;
}

function waitForExit(child, timeoutMs) {
  return new Promise((resolve) => {
    if (child.exited || child.exitCode !== null || child.signalCode !== null) return resolve(true);
    const timer = setTimeout(() => resolve(false), timeoutMs);
    child.once('exit', () => {
      clearTimeout(timer);
      resolve(true);
    });
  });
}

// True while any process in the child's process group is still running (macOS).
function groupAlive(pid) {
  try {
    process.kill(-pid, 0);
    return true;
  } catch {
    return false;
  }
}

async function waitForGroupExit(pid, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    if (!groupAlive(pid)) return true;
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  return !groupAlive(pid);
}

async function stopChild(child) {
  if (!child.pid) return;
  if (osName === 'windows') {
    if (child.exited) return;
    log(`stopping ${child.name}...`);
    // /T stops the whole tree (npm, electron, go, agent), /F forces it.
    spawnSync('taskkill', ['/PID', String(child.pid), '/T', '/F'], { stdio: 'ignore' });
    await waitForExit(child, 5000);
    return;
  }
  // macOS: stop the whole process group, even if the process we started has
  // already exited (for example after "Exit Kiosk"), so nothing it started is left behind.
  if (!groupAlive(child.pid)) return;
  log(`stopping ${child.name}...`);
  try {
    process.kill(-child.pid, 'SIGTERM');
  } catch {
    return;
  }
  if (!(await waitForGroupExit(child.pid, 5000))) {
    try {
      process.kill(-child.pid, 'SIGKILL');
    } catch {
      /* already gone */
    }
    await waitForGroupExit(child.pid, 2000);
  }
}

async function warnAboutBusyPorts() {
  for (const port of [DISPLAY_PORT, BRIDGE_PORT]) {
    if (await anyoneListening(port)) {
      log(`WARNING: something is still using port ${port}. Stop it before the next run (see docs/QUICKSTART.md, Troubleshooting).`);
    }
  }
}

// ---------------------------------------------------------------- checks

function commandExists(command) {
  const finder = osName === 'windows' ? 'where' : 'which';
  return spawnSync(finder, [command], { stdio: 'ignore' }).status === 0;
}

// Dev servers may listen on IPv4 (127.0.0.1) or IPv6 (::1) only, so check both.
const LOCAL_HOSTS = ['127.0.0.1', '::1'];

function canConnect(port, host) {
  return new Promise((resolve) => {
    const socket = net.connect({ port, host, timeout: 1000 });
    const done = (result) => {
      socket.destroy();
      resolve(result);
    };
    socket.once('connect', () => done(true));
    socket.once('timeout', () => done(false));
    socket.once('error', () => done(false));
  });
}

async function anyoneListening(port) {
  const results = await Promise.all(LOCAL_HOSTS.map((host) => canConnect(port, host)));
  return results.some(Boolean);
}

async function portIsFree(port) {
  if (await anyoneListening(port)) return false;
  return new Promise((resolve) => {
    const server = net.createServer();
    server.once('error', () => resolve(false));
    server.once('listening', () => server.close(() => resolve(true)));
    server.listen(port);
  });
}

function checkBackend(serverUrl) {
  return new Promise((resolve) => {
    const request = http.get(`${serverUrl}/health`, { timeout: 3000 }, (response) => {
      response.resume();
      resolve(response.statusCode === 200);
    });
    request.on('timeout', () => request.destroy());
    request.on('error', () => resolve(false));
  });
}

async function waitForPort(port, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    if (shuttingDown) return false;
    if (await anyoneListening(port)) return true;
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  return false;
}

// ---------------------------------------------------------------- backup / restore

function backupPaths(config) {
  // www/ is build output (not in the repository), so a rebuild is kept after the run.
  return [
    AGENT_CONFIG,
    path.join(AGENT_DIR, 'local-store'),
    path.join(KIOSK_DIR, 'local-store'),
  ];
}

function copyPath(from, to) {
  fs.mkdirSync(path.dirname(to), { recursive: true });
  fs.cpSync(from, to, { recursive: true });
}

function removePath(target) {
  fs.rmSync(target, { recursive: true, force: true, maxRetries: 10, retryDelay: 300 });
}

function backup(config) {
  fs.mkdirSync(BACKUP_DIR, { recursive: true });
  const manifest = { createdAt: new Date().toISOString(), entries: [], movedAgentExe: false };

  backupPaths(config).forEach((original, index) => {
    const relative = path.relative(KIOSK_DIR, original);
    const existed = fs.existsSync(original);
    const copy = path.join(BACKUP_DIR, `item-${index}`);
    if (existed) copyPath(original, copy);
    manifest.entries.push({ path: relative, existed, copy: existed ? path.basename(copy) : null });
  });

  // Electron runs bin/agent.exe if it exists (and then ignores the agent's config.json),
  // so move it aside for the run.
  if (config.mode === 'electron' && fs.existsSync(AGENT_EXE)) {
    fs.renameSync(AGENT_EXE, path.join(BACKUP_DIR, 'agent.exe'));
    manifest.movedAgentExe = true;
  }

  fs.writeFileSync(MANIFEST_PATH, JSON.stringify(manifest, null, 2));
  log(`backed up ${manifest.entries.length} paths to ${path.relative(KIOSK_DIR, BACKUP_DIR)}`);
}

function restore() {
  if (!fs.existsSync(MANIFEST_PATH)) {
    if (fs.existsSync(BACKUP_DIR)) removePath(BACKUP_DIR);
    return false;
  }
  const manifest = JSON.parse(fs.readFileSync(MANIFEST_PATH, 'utf8'));

  for (const entry of manifest.entries) {
    const target = path.join(KIOSK_DIR, entry.path);
    removePath(target);
    if (entry.existed) copyPath(path.join(BACKUP_DIR, entry.copy), target);
    log(`restored ${entry.path}`);
  }

  if (manifest.movedAgentExe) {
    fs.mkdirSync(path.dirname(AGENT_EXE), { recursive: true });
    fs.renameSync(path.join(BACKUP_DIR, 'agent.exe'), AGENT_EXE);
    log('restored kiosk-electron-shell/bin/agent.exe');
  }

  removePath(BACKUP_DIR);
  log('kiosk folder is back to how it was before the run.');
  return true;
}

// ---------------------------------------------------------------- run steps

function writeAgentConfig(config) {
  const agentConfig = JSON.parse(fs.readFileSync(AGENT_CONFIG, 'utf8'));
  agentConfig.loungeId = config.loungeId;
  agentConfig.loungeGroup = config.loungeName;
  agentConfig.serverUrl = config.serverUrl;
  fs.writeFileSync(AGENT_CONFIG, JSON.stringify(agentConfig, null, 2));
  log(`lounge for this run: ${config.loungeName} (${config.loungeId})`);
}

function installIfMissing(dir) {
  if (!fs.existsSync(path.join(dir, 'node_modules'))) {
    // npm ci installs exactly what package-lock.json says and does not rewrite it.
    runSync(npmCommand(), ['ci', '--no-audit', '--no-fund'], dir);
  }
}

function rebuildDisplay() {
  installIfMissing(FRONTEND_DIR);
  runSync(npmCommand(), ['run', 'build'], FRONTEND_DIR);
  const built = path.join(FRONTEND_DIR, 'dist', 'kiosk-display', 'browser');
  if (!fs.existsSync(built)) fail(`Build output not found: ${built}`);
  removePath(WWW_DIR);
  copyPath(built, WWW_DIR);
  log('kiosk-electron-shell/www built from kiosk-frontend.');
}

function displayUrl(config) {
  const params = new URLSearchParams({
    loungeId: config.loungeId,
    loungeName: config.loungeName,
    kiosk: 'true',
  });
  return `http://localhost:${DISPLAY_PORT}/bids-display?${params.toString()}`;
}

function openInBrowser(url) {
  if (osName === 'windows') {
    spawn('cmd', ['/c', 'start', '""', url], { stdio: 'ignore', detached: true }).unref();
  } else {
    spawn('open', [url], { stdio: 'ignore', detached: true }).unref();
  }
}

async function runElectron(config) {
  // www/ is build output and not in the repository, so build it on the first run.
  const displayMissing = !fs.existsSync(path.join(WWW_DIR, 'index.html'));
  if (displayMissing) log('kiosk-electron-shell/www is missing; building the kiosk display first.');
  if (config.rebuildDisplay || displayMissing) rebuildDisplay();
  installIfMissing(SHELL_DIR);
  start('kiosk app (Electron)', npmCommand(), ['start'], SHELL_DIR);
  log('The kiosk window takes over the screen and hides the cursor.');
  log('To stop it, use "Exit Kiosk" in the admin panel (below), or press Ctrl+C in this terminal.');
  log('Kiosk admin panel: Ctrl+Shift+A, login admin / password. "Exit Kiosk" there closes the app.');
}

async function runBrowser(config) {
  installIfMissing(FRONTEND_DIR);
  start('sync agent', 'go', ['run', './cmd/agent', '-config', 'config.json'], AGENT_DIR);
  start('display (ng serve)', npmCommand(), ['start', '--', '--port', String(DISPLAY_PORT)], FRONTEND_DIR);

  const url = displayUrl(config);
  if (await waitForPort(DISPLAY_PORT, 120000)) {
    log(`display: ${url}`);
    log(`agent data: http://localhost:${BRIDGE_PORT}/local/schedule`);
    if (config.openBrowser) openInBrowser(url);
  } else {
    log(`display did not start on port ${DISPLAY_PORT} within 2 minutes; check the output above.`);
  }
  log('Press Ctrl+C to stop.');
}

async function shutdown(exitCode) {
  if (shuttingDown) return;
  shuttingDown = true;
  console.log('');
  log('shutting down...');
  for (const child of children) await stopChild(child);
  await warnAboutBusyPorts();
  try {
    restore();
  } catch (error) {
    console.error(`[run-kiosk] Cleanup failed: ${error.message}`);
    console.error('[run-kiosk] Run "node run-kiosk.js --cleanup" to try again.');
    process.exit(1);
  }
  process.exit(exitCode);
}

async function main() {
  const config = readConfig();
  osName = config.os;

  if (process.argv.includes('--cleanup')) {
    if (!restore()) log('nothing to clean up.');
    return;
  }

  // A previous run that was killed before it could clean up leaves a backup behind.
  if (fs.existsSync(BACKUP_DIR)) {
    log('found files left over from an interrupted run; restoring them first.');
    restore();
  }

  if (!commandExists('go')) fail('Go is not installed or not on PATH (https://go.dev/dl/).');
  if (!commandExists('npm')) fail('Node.js/npm is not installed or not on PATH (https://nodejs.org/).');

  for (const port of [DISPLAY_PORT, BRIDGE_PORT]) {
    if (!(await portIsFree(port))) {
      fail(`Port ${port} is already in use. Stop whatever is using it (see docs/QUICKSTART.md, Troubleshooting) and try again.`);
    }
  }

  if (!(await checkBackend(config.serverUrl))) {
    log(`WARNING: the backend at ${config.serverUrl} is not responding. The display will be empty until it is running.`);
  }

  for (const signal of ['SIGINT', 'SIGTERM', 'SIGHUP']) {
    process.on(signal, () => shutdown(0));
  }

  backup(config);
  try {
    writeAgentConfig(config);
    if (config.mode === 'electron') await runElectron(config);
    else await runBrowser(config);
  } catch (error) {
    console.error(`[run-kiosk] ${error.message}`);
    await shutdown(1);
  }
}

process.on('exit', () => {
  // Last resort if the script exits without going through shutdown().
  if (!shuttingDown && fs.existsSync(MANIFEST_PATH)) {
    try {
      restore();
    } catch {
      /* reported by the next run */
    }
  }
});

main();
