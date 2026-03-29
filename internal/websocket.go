package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		// In production, you should validate the origin
		return true
	},
}

// WebSocketHub manages all WebSocket connections
type WebSocketHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mutex      sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

// Run starts the WebSocket hub
func (hub *WebSocketHub) Run() {
	for {
		select {
		case client := <-hub.register:
			hub.mutex.Lock()
			hub.clients[client] = true
			hub.mutex.Unlock()
			log.Printf("📡 WebSocket client connected. Total clients: %d", len(hub.clients))

		case client := <-hub.unregister:
			hub.mutex.Lock()
			if _, ok := hub.clients[client]; ok {
				delete(hub.clients, client)
				client.Close()
				log.Printf("📡 WebSocket client disconnected. Total clients: %d", len(hub.clients))
			}
			hub.mutex.Unlock()

		case message := <-hub.broadcast:
			hub.mutex.RLock()
			for client := range hub.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Printf("Error sending to client: %v", err)
					client.Close()
					hub.mutex.RUnlock()
					hub.unregister <- client
					hub.mutex.RLock()
				}
			}
			hub.mutex.RUnlock()
		}
	}
}

// BroadcastEvent sends an event to all connected clients
func (hub *WebSocketHub) BroadcastEvent(eventType string, data interface{}) {
	event := map[string]interface{}{
		"type":      eventType,
		"data":      data,
		"timestamp": time.Now().Unix(),
	}

	message, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling event: %v", err)
		return
	}

	hub.broadcast <- message
}

// HandleWebSocket handles WebSocket connections
func (hub *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	hub.register <- conn

	// Send initial state
	go hub.sendInitialState(conn)

	// Keep connection alive and handle incoming messages
	go func() {
		defer func() {
			hub.unregister <- conn
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				break
			}
			// Handle incoming messages if needed
		}
	}()
}

// sendInitialState sends the current system state to a newly connected client
func (hub *WebSocketHub) sendInitialState(conn *websocket.Conn) {
	initialState := map[string]interface{}{
		"type": "initial_state",
		"data": map[string]interface{}{
			"connected": true,
			"message":   "Connected to Priority Fabric Transaction Gateway",
		},
		"timestamp": time.Now().Unix(),
	}

	message, err := json.Marshal(initialState)
	if err != nil {
		log.Printf("Error marshaling initial state: %v", err)
		return
	}

	err = conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		log.Printf("Error sending initial state: %v", err)
	}
}

// WebSocket event types
const (
	EventTxSubmitted      = "tx_submitted"
	EventBatchCountdown   = "batch_countdown"
	EventBatchStarted     = "batch_started"
	EventBatchCompleted   = "batch_completed"
	EventTxStatusChange   = "tx_status_change"
	EventMempoolStats     = "mempool_stats"
	EventBatcherStats     = "batcher_stats"
)
