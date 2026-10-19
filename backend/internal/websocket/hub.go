package websocket

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	gorilla "github.com/gorilla/websocket"
)

const (
	ResourceSchedule  = "schedule"
	ResourceETA       = "eta"
	ResourceBroadcast = "broadcast"
	ResourceAds       = "ads"
	ResourceLoungeAds = "lounge_ads"
	ResourceAll       = "all"
)

var (
	ErrInvalidNotification = errors.New("invalid websocket notification")
)

type Notification struct {
	Type      string    `json:"type"`
	LoungeID  string    `json:"loungeId"`
	Resource  string    `json:"resource"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Authorizer func(*http.Request, string) bool

type Hub struct {
	mu         sync.Mutex
	clients    map[*Client]struct{}
	authorizer Authorizer
	upgrader   gorilla.Upgrader
}

func NewHub(authorizer Authorizer) *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		authorizer: authorizer,
		upgrader: gorilla.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	loungeID := strings.TrimSpace(r.URL.Query().Get("loungeId"))
	if loungeID == "" {
		http.Error(w, "loungeId is required", http.StatusBadRequest)
		return
	}
	if h.authorizer == nil {
		http.Error(w, "websocket authorization is not configured", http.StatusServiceUnavailable)
		return
	}
	if !h.authorizer(r, loungeID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{
		hub:      h,
		conn:     conn,
		loungeID: loungeID,
		send:     make(chan []byte, 16),
	}
	h.register(client)

	go client.writePump()
	client.readPump()
}

func (h *Hub) Broadcast(notification Notification) (int, error) {
	notification.Type = "update"
	notification.LoungeID = strings.TrimSpace(notification.LoungeID)
	notification.Resource = strings.ToLower(strings.TrimSpace(notification.Resource))
	if notification.LoungeID == "" || !validResource(notification.Resource) {
		return 0, ErrInvalidNotification
	}
	if notification.UpdatedAt.IsZero() {
		notification.UpdatedAt = time.Now().UTC()
	}

	payload, err := json.Marshal(notification)
	if err != nil {
		return 0, err
	}

	delivered := 0
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		if client.loungeID != notification.LoungeID {
			continue
		}

		select {
		case client.send <- payload:
			delivered++
		default:
			delete(h.clients, client)
			close(client.send)
		}
	}
	return delivered, nil
}

func (h *Hub) register(client *Client) {
	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) unregister(client *Client) {
	h.mu.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
	h.mu.Unlock()
}

func validResource(resource string) bool {
	switch resource {
	case ResourceSchedule, ResourceETA, ResourceBroadcast, ResourceAds, ResourceLoungeAds, ResourceAll:
		return true
	default:
		return false
	}
}
