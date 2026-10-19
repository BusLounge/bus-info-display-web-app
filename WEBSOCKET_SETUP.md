# Optional WebSocket Update Notifications

This optional layer sends small `update` notifications to connected TV sync
agents. The notification is only a trigger: the agent fetches the current data
over the existing REST APIs and atomically refreshes its local JSON snapshots.
Keep the existing periodic sync running as the offline/reconnect recovery path.

## Files and dependency

The implementation is isolated in:

- `backend/internal/websocket/hub.go`
- `backend/internal/websocket/client.go`
- `tv-sync-agent-go/internal/websocket/agent_ws_client.go`

No existing application files are changed by this optional feature. Both Go
modules need Gorilla WebSocket before building these packages:

```powershell
Set-Location backend
go get github.com/gorilla/websocket

Set-Location ..\tv-sync-agent-go
go get github.com/gorilla/websocket
```

These commands update each module's dependency manifest and checksum file.
They are intentionally not run as part of adding the standalone source files.

## Register the backend endpoint

In `backend/cmd/server/main.go`, import the new package with an alias:

```go
wsupdates "bus-schedule-lounge/internal/websocket"
```

After creating the Gorilla mux router, create the hub and register it on the
API router. Do this before the server starts:

```go
updatesHub := wsupdates.NewHub(func(r *http.Request, loungeID string) bool {
    // Replace with the deployment's actual device-token validation and
    // lounge authorization. Return false until the connection is authorized.
    return validateTVSyncAgent(r, loungeID)
})
api.Handle("/tv/updates", updatesHub)
```

`validateTVSyncAgent` above is an integration placeholder, not a function
provided by this module. Connect it to the same device credential/authorization
policy used for the TV agent. Do not expose an unauthenticated production
endpoint. Gorilla's default origin check allows same-origin browser clients;
the TV sync agent is a non-browser client and does not send an `Origin` header.
If a browser client is added later, configure and review origin policy
explicitly. Use `wss://` when connections cross an untrusted network.

The hub accepts `GET /api/tv/updates?loungeId=<lounge-id>` and delivers
notifications only to clients subscribed to that lounge. It supports these
resource names:

| Resource | Agent action |
| --- | --- |
| `schedule` or `eta` | Refresh the schedule snapshot |
| `broadcast` | Refresh the broadcasts snapshot |
| `ads` | Refresh the ad manifest |
| `lounge_ads` | Refresh the lounge-ad snapshot |
| `all` | Refresh each enabled snapshot using the existing agent methods |

## Publish after successful data changes

The hub does not inspect database writes or infer when data changes. After a
successful schedule/ETA or broadcast mutation, publish the corresponding
notification from the existing backend handler/service integration point:

```go
if _, err := updatesHub.Broadcast(wsupdates.Notification{
    LoungeID: loungeID,
    Resource: wsupdates.ResourceSchedule, // or ResourceETA, ResourceBroadcast
}); err != nil {
    log.Printf("publish TV update notification: %v", err)
}
```

Only publish after persistence succeeds. For changes affecting multiple
lounges, publish once per affected lounge. A zero client count is normal: it
means no sync agent was connected at that moment. The subsequent periodic
sync still catches the change.

## Enable the agent client

The agent client accepts the existing sync agent as its snapshot refresher.
In `tv-sync-agent-go/cmd/agent/main.go`, add an import:

```go
wsupdates "tv-sync-agent-go/internal/websocket"
```

After creating `a := syncagent.New(cfg)`, construct and run the optional client
with the same cancellable context already used for the periodic agent:

```go
updatesClient, err := wsupdates.NewAgentWSClient(cfg.ServerURL, cfg.LoungeID, a)
if err != nil {
    log.Fatalf("configure websocket updates: %v", err)
}
go updatesClient.Run(ctx)
```

If the backend authorizer requires a bearer token or another HTTP header, use
`NewAgentWSClientWithHeaders` and supply an `http.Header` populated from the
deployment's existing secret/configuration mechanism. Do not hard-code
credentials in source or `config.json`.

This client reconnects with capped exponential backoff. It only triggers the
existing `SyncSchedule`, `SyncBroadcasts`, `SyncAds`, and `SyncLoungeAds`
methods; it does not replace the polling loop, local bridge, or JSON files.
Connection and refresh errors are logged, and the next periodic sync remains
the fallback if an event or connection is missed.

## Test and operate

1. Add Gorilla WebSocket to both modules as shown above.
2. Register the backend route and provide a real lounge/device authorizer.
3. Start the backend and agent, then verify the agent's connection logs.
4. Publish a test notification for the configured lounge and confirm the
   matching `local-store` JSON snapshot's `updatedAt` changes.
5. Stop the backend temporarily; verify the agent reconnects and the existing
   periodic sync/local snapshots continue to work.

Notifications are transient, not a durable event queue. Do not use this
channel as the sole delivery path for critical operational updates.
