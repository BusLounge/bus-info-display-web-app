# TV Box Setup Guide (Go Sync Agent + Angular Player)

This guide shows exactly how to:
- run the Go sync agent on the TV box
- store AVI files on the TV box automatically
- sync schedules and ads every few minutes
- access the TV box remotely

This guide is written for beginners.

## 1) What runs where

Main backend server (your existing server):
- Runs Go backend API
- Provides live schedule and advertisement data
- Example URL: http://192.168.1.100:8083

TV box (Raspberry Pi):
- Runs Angular TV player (display UI)
- Runs tv-sync-agent-go (downloads data and media)
- Exposes local endpoints on TV box (default port 3000)

## 2) Required folders on TV box

Create these folders on the TV box:
- /home/pi/tv-display/angular-app
- /home/pi/tv-display/tv-sync-agent-go

The sync agent local store will be:
- /home/pi/tv-display/tv-sync-agent-go/local-store

Inside local-store, files are:
- schedule.json
- ads-manifest.json
- media/ (downloaded ad files, including AVI)

## 3) Install software on Raspberry Pi (one time)

Run on Pi terminal:

sudo apt update
sudo apt upgrade -y

# Install Go (if not installed)
sudo apt install -y golang-go

go version

# Install Node 20 for Angular static serving
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

# Install serve and pm2
sudo npm install -g serve pm2

## 4) Copy project artifacts from laptop to Pi

From your development machine:

A) Build Angular app:
- Go to frontend folder
- Build production

Commands:
cd frontend
npx ng build --configuration production

B) Copy Angular build output:
scp -r dist/bus-schedule-lounge/browser/* pi@PI_IP:/home/pi/tv-display/angular-app/

C) Copy Go sync agent:
scp -r tv-sync-agent-go/* pi@PI_IP:/home/pi/tv-display/tv-sync-agent-go/

Replace PI_IP with your TV box IP.

## 5) Configure Go sync agent on Pi

SSH to Pi:
ssh pi@PI_IP

Edit config:
cd /home/pi/tv-display/tv-sync-agent-go
nano config.json

Use this template:

{
  "serverUrl": "http://MAIN_SERVER_IP:8083",
  "loungeId": "YOUR_LOUNGE_UUID",
  "loungeGroup": "YOUR_LOUNGE_GROUP_NAME",
  "tvPurpose": "both",
  "scheduleIntervalCron": "*/5 * * * *",
  "adsIntervalCron": "*/15 * * * *",
  "localBridgePort": 3000,
  "requestTimeoutMs": 10000,
  "downloadTimeoutMs": 120000,
  "storeDir": "/home/pi/tv-display/tv-sync-agent-go/local-store"
}

Important:
- serverUrl must point to the backend instance that serves:
  - /api/departures/lounge/{loungeId}
  - /api/arrivals/lounge/{loungeId}
  - /api/tv/ads/{loungeGroup}
- loungeId is one lounge per TV box.
- loungeGroup is used for ad filtering.
- tvPurpose controls this TV's job:
  - both: sync schedules and ads (default)
  - schedule: sync only departures/arrivals
  - ads: sync only advertisements/media

## 6) Build and run Go sync agent on Pi

cd /home/pi/tv-display/tv-sync-agent-go
go build -o tv-sync-agent ./cmd/agent

Test run:
./tv-sync-agent -config ./config.json

If successful, it will:
- start local bridge on port 3000
- attempt initial sync immediately
- continue periodic sync by intervals

Stop test run with Ctrl+C.

## 7) Start both services with pm2

A) Start Go sync agent:
pm2 start "/home/pi/tv-display/tv-sync-agent-go/tv-sync-agent -config /home/pi/tv-display/tv-sync-agent-go/config.json" --name tv-sync-go

B) Start Angular TV player:
pm2 start "serve -s /home/pi/tv-display/angular-app -l 4200" --name tv-player

C) Save and auto-start on reboot:
pm2 save
pm2 startup

Run the command printed by pm2 startup.

## 8) How AVI files are stored

The Go sync agent downloads media files from advertisement mediaUrl.

Storage location:
- /home/pi/tv-display/tv-sync-agent-go/local-store/media/

How file names are chosen:
- File name is ad ID + extension from media URL.
- If media URL has no extension, agent uses .avi by default.

Examples:
- 5b2...ad1 + .avi => 5b2...ad1.avi
- 7c1...ad2 + .mp4 => 7c1...ad2.mp4

How updates happen:
- Agent computes a hash from media URL + updatedAt.
- If hash changed, file is re-downloaded.
- If ad removed from backend manifest, local file is deleted.

## 9) Time-to-time synchronization logic

Schedule sync:
- Every 5 minutes by default
- Pulls departures and arrivals for one loungeId
- Writes to local-store/schedule.json

Ads sync:
- Every 15 minutes by default
- Pulls TV-ready ad manifest from /api/tv/ads/{loungeGroup}
- Downloads media into local-store/media/
- Writes manifest to local-store/ads-manifest.json

Also at startup:
- Agent runs both syncs immediately once.

## 10) How lounge schedule changes are handled

Per TV box:
- One config.json has one loungeId.
- Agent only fetches schedule for that loungeId.
- So each lounge screen should run its own TV box with its own loungeId.

When backend data changes:
- Next schedule sync cycle updates schedule.json automatically.
- TV display should read local schedule data to show latest updates.

## 11) Local endpoints on TV box

From TV box network:
- http://TV_BOX_IP:3000/local/status
- http://TV_BOX_IP:3000/local/schedule
- http://TV_BOX_IP:3000/local/ads
- http://TV_BOX_IP:3000/local/media/FILENAME.avi

Use status endpoint to check:
- last schedule sync time
- last ads sync time
- last sync errors
- ads count

## 12) Remote access to TV box

Option A: SSH (recommended for commands)
- From laptop:
  ssh pi@PI_IP

Useful commands:
- pm2 list
- pm2 logs tv-sync-go
- pm2 logs tv-player
- pm2 restart tv-sync-go
- pm2 restart tv-player
- sudo reboot

Option B: AnyDesk (remote desktop)
On Pi:
- install AnyDesk
- enable auto start
- note 9-digit AnyDesk address

On laptop:
- install AnyDesk
- connect with Pi AnyDesk address

Option C: VNC (built-in desktop remote)
On Pi:
- sudo raspi-config
- Interface Options -> VNC -> Enable

Then connect from laptop using VNC Viewer.

## 13) Verify everything is working

1) Backend health:
- open http://MAIN_SERVER_IP:8083/health

2) TV sync status:
- open http://TV_BOX_IP:3000/local/status

3) Check local files on Pi:
ls -lah /home/pi/tv-display/tv-sync-agent-go/local-store
ls -lah /home/pi/tv-display/tv-sync-agent-go/local-store/media

4) Check logs:
pm2 logs tv-sync-go --lines 100

## 14) Common issues and quick fixes

Issue: initial sync shows 404
- Cause: wrong serverUrl or wrong backend service on that port.
- Fix: verify API routes exist on target backend URL.

Issue: bind error on port 8083
- Cause: another process already uses backend port.
- Fix: stop duplicate process and restart correct backend.

Issue: no AVI files downloaded
- Cause: ads are not active, wrong loungeGroup, or inaccessible mediaUrl.
- Fix: verify ad status is active and loungeGroup matches exactly.

Issue: TV page not updating live data
- Cause: frontend might still use static mock service.
- Fix: wire TV page to consume local bridge endpoints on port 3000.

## 15) Daily operation checklist

- Backend is running and API reachable.
- TV box is powered and online.
- pm2 list shows tv-sync-go and tv-player as online.
- /local/status shows recent sync times.
- media folder contains latest ad files.

That is the full Go-based TV box setup with automatic AVI storage and periodic synchronization.
