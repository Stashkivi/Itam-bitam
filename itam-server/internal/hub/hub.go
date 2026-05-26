// Package hub manages WebSocket connections and broadcasts anomaly events
// from Redis Pub/Sub to all connected frontend clients in an organisation.
package hub

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

const (
	writeTimeout = 10 * time.Second
	pingInterval = 30 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	// In production, validate Origin against an allowlist.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// client is one connected browser session.
type client struct {
	conn  *websocket.Conn
	orgID string
	send  chan []byte
}

// Hub manages all WebSocket clients and a Redis subscriber per org.
type Hub struct {
	mu          sync.RWMutex
	clients     map[*client]struct{}
	byOrg       map[string]map[*client]struct{}
	redisClient *redis.Client
}

// New creates a Hub and starts the Redis Pub/Sub listener.
func New(redisClient *redis.Client) *Hub {
	return &Hub{
		clients:     make(map[*client]struct{}),
		byOrg:       make(map[string]map[*client]struct{}),
		redisClient: redisClient,
	}
}

// Subscribe connects the hub to the Redis channel for orgID and begins
// forwarding messages to all registered clients in that org.
// Each org gets its own goroutine; call once per org on first client connect.
func (h *Hub) Subscribe(ctx context.Context, orgID string) {
	channel := fmt.Sprintf("anomaly:%s", orgID)
	pubsub := h.redisClient.Subscribe(ctx, channel)

	go func() {
		defer pubsub.Close()
		for msg := range pubsub.Channel() {
			h.broadcast(orgID, []byte(msg.Payload))
		}
	}()
}

// Upgrade upgrades an HTTP request to a WebSocket connection and registers
// the client in the hub for the given orgID.
func (h *Hub) Upgrade(w http.ResponseWriter, r *http.Request, orgID string) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	c := &client{
		conn:  conn,
		orgID: orgID,
		send:  make(chan []byte, 256),
	}

	h.register(c)

	go c.writePump()
	go c.readPump(h)

	// Start org subscriber if not already running.
	h.mu.RLock()
	_, exists := h.byOrg[orgID]
	h.mu.RUnlock()
	if !exists {
		h.Subscribe(r.Context(), orgID)
	}

	return nil
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
	if h.byOrg[c.orgID] == nil {
		h.byOrg[c.orgID] = make(map[*client]struct{})
	}
	h.byOrg[c.orgID][c] = struct{}{}
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
	if org := h.byOrg[c.orgID]; org != nil {
		delete(org, c)
		if len(org) == 0 {
			delete(h.byOrg, c.orgID)
		}
	}
	close(c.send)
}

func (h *Hub) broadcast(orgID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.byOrg[orgID] {
		select {
		case c.send <- msg:
		default:
			// Slow client — drop the message to avoid blocking the broadcast.
			log.Printf("[hub] dropped message for slow client org=%s", orgID)
		}
	}
}

// writePump drains the send channel and writes frames to the WebSocket.
func (c *client) writePump() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump discards inbound frames (clients are receive-only) and detects
// disconnects via the read deadline.
func (c *client) readPump(h *Hub) {
	defer func() {
		h.unregister(c)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(pingInterval * 2))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pingInterval * 2))
		return nil
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}
