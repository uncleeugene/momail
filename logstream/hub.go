package logstream

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"uncleeugene.kz/momail/logutil"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub
	// The websocket connection.
	conn *websocket.Conn
	// Buffered channel of outbound messages.
	send chan []byte
}

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

var LogHub = newHub()

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte, 256), // Buffered to prevent deadlock when Hub logs
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// Run starts the hub's event loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
					// Message sent successfully
				default:
					// Client's send buffer is full, drop message and unregister client
					log.Println(logutil.Warn("LogStream: Client %s send buffer full, unregistering.", client.conn.RemoteAddr()))
					close(client.send)        // Close the channel to signal writePump to exit
					delete(h.clients, client) // Remove client from map
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Write allows the Hub to be used as an io.Writer for the log package.
func (h *Hub) Write(p []byte) (n int, err error) {
	// We must copy the buffer because the log package reuses it.
	buf := make([]byte, len(p))
	copy(buf, p)
	select {
	case h.broadcast <- buf:
	default:
	}
	return len(p), nil
}

// ServeWs handles websocket requests from the peer.
func ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(logutil.Error("LogStream: Failed to upgrade to websocket: %v", err))
		return
	}
	client := &Client{hub: LogHub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	log.Println(logutil.Debug("LogStream: Client %s connected", conn.RemoteAddr()))

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}

// writePump and readPump implementations would go here.
// For a read-only log stream, they can be simplified.
// See the gorilla/websocket chat example for a full implementation.
// For now, we'll omit them for brevity as we are only pushing logs.
func (c *Client) writePump() {
	defer c.conn.Close()
	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			break
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	// Read messages from the client (e.g., pings) and discard them.
	// This loop is essential to detect client disconnects.
	for {
		if _, _, err := c.conn.NextReader(); err != nil {
			log.Println(logutil.Debug("LogStream: Client %s disconnected: %v", c.conn.RemoteAddr(), err))
			break
		}
	}
}
