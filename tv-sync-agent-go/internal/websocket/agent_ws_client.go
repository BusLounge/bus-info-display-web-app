package websocket

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	gorilla "github.com/gorilla/websocket"

	syncagent "tv-sync-agent-go/internal/sync"
)

const (
	updatesPath    = "/api/tv/updates"
	readLimit      = 4096
	initialBackoff = time.Second
	maxBackoff     = 30 * time.Second
)

type Notification struct {
	Type      string    `json:"type"`
	LoungeID  string    `json:"loungeId"`
	Resource  string    `json:"resource"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AgentWSClient struct {
	url      string
	loungeID string
	agent    SnapshotRefresher
	dialer   *gorilla.Dialer
	headers  http.Header
}

type SnapshotRefresher interface {
	SyncSchedule(context.Context) error
	SyncBroadcasts(context.Context) error
	SyncAds(context.Context) error
	SyncLoungeAds(context.Context) error
}

func NewAgentWSClient(serverURL, loungeID string, agent SnapshotRefresher) (*AgentWSClient, error) {
	return NewAgentWSClientWithHeaders(serverURL, loungeID, agent, nil)
}

func NewAgentWSClientWithHeaders(serverURL, loungeID string, agent SnapshotRefresher, headers http.Header) (*AgentWSClient, error) {
	if strings.TrimSpace(loungeID) == "" {
		return nil, errors.New("loungeID is required")
	}
	if agent == nil {
		return nil, errors.New("snapshot refresher is required")
	}

	endpoint, err := makeWebSocketURL(serverURL, loungeID)
	if err != nil {
		return nil, err
	}
	return &AgentWSClient{
		url:      endpoint,
		loungeID: strings.TrimSpace(loungeID),
		agent:    agent,
		dialer:   gorilla.DefaultDialer,
		headers:  headers.Clone(),
	}, nil
}

func (c *AgentWSClient) Run(ctx context.Context) {
	backoff := initialBackoff
	for {
		if ctx.Err() != nil {
			return
		}

		if err := c.runConnection(ctx); err != nil && ctx.Err() == nil {
			log.Printf("websocket connection ended: %v", err)
		}
		if !sleepContext(ctx, backoff) {
			return
		}
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func (c *AgentWSClient) runConnection(ctx context.Context) error {
	conn, response, err := c.dialer.DialContext(ctx, c.url, c.headers)
	if err != nil {
		if response != nil {
			return fmt.Errorf("dial websocket: %w (HTTP %s)", err, response.Status)
		}
		return fmt.Errorf("dial websocket: %w", err)
	}
	defer conn.Close()

	stopClose := context.AfterFunc(ctx, func() {
		_ = conn.Close()
	})
	defer stopClose()

	refreshCtx, cancelRefresh := context.WithCancel(ctx)
	refreshQueue := make(chan string, 16)
	refreshDone := make(chan struct{})
	go func() {
		defer close(refreshDone)
		for {
			select {
			case <-refreshCtx.Done():
				return
			case resource, ok := <-refreshQueue:
				if !ok {
					return
				}
				if err := c.refresh(refreshCtx, resource); err != nil {
					log.Printf("refresh snapshots for websocket resource %q: %v", resource, err)
				}
			}
		}
	}()
	defer func() {
		cancelRefresh()
		close(refreshQueue)
		<-refreshDone
	}()

	conn.SetReadLimit(readLimit)
	if err := conn.SetReadDeadline(time.Now().Add(70 * time.Second)); err != nil {
		return fmt.Errorf("set websocket read deadline: %w", err)
	}
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	})
	conn.SetPingHandler(func(appData string) error {
		if err := conn.SetReadDeadline(time.Now().Add(70 * time.Second)); err != nil {
			return err
		}
		return conn.WriteControl(
			gorilla.PongMessage,
			[]byte(appData),
			time.Now().Add(10*time.Second),
		)
	})

	for {
		var notification Notification
		if err := conn.ReadJSON(&notification); err != nil {
			return fmt.Errorf("read websocket notification: %w", err)
		}
		if notification.Type != "update" || notification.LoungeID != c.loungeID {
			log.Printf("ignoring invalid or out-of-scope websocket notification: type=%q loungeId=%q",
				notification.Type, notification.LoungeID)
			continue
		}
		select {
		case refreshQueue <- notification.Resource:
		default:
			log.Printf("dropping websocket resource %q because the refresh queue is full", notification.Resource)
		}
	}
}

func (c *AgentWSClient) refresh(ctx context.Context, resource string) error {
	switch strings.ToLower(strings.TrimSpace(resource)) {
	case "schedule", "eta":
		return c.agent.SyncSchedule(ctx)
	case "broadcast":
		return c.agent.SyncBroadcasts(ctx)
	case "ads":
		return c.agent.SyncAds(ctx)
	case "lounge_ads":
		return c.agent.SyncLoungeAds(ctx)
	case "all":
		return errors.Join(
			c.agent.SyncSchedule(ctx),
			c.agent.SyncBroadcasts(ctx),
			c.agent.SyncAds(ctx),
			c.agent.SyncLoungeAds(ctx),
		)
	default:
		return fmt.Errorf("unsupported websocket resource %q", resource)
	}
}

func makeWebSocketURL(serverURL, loungeID string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(serverURL))
	if err != nil {
		return "", fmt.Errorf("parse server URL: %w", err)
	}
	if u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errors.New("server URL must use http or https and include a host")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("server URL must not contain a query or fragment")
	}

	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	}
	u.Path = path.Join("/", strings.TrimSuffix(u.Path, "/"), strings.TrimPrefix(updatesPath, "/"))
	query := u.Query()
	query.Set("loungeId", loungeID)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func sleepContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

var _ SnapshotRefresher = (*syncagent.Agent)(nil)
