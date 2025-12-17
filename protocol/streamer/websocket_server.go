package streamer

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lavanet/lava/v5/utils"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (configure CORS properly in production)
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WebSocketServer handles WebSocket connections for real-time streaming
type WebSocketServer struct {
	config          *StreamerConfig
	subscriptionMgr *SubscriptionManager
	metrics         *StreamerMetrics
	connections     map[string]*websocket.Conn
	mu              sync.RWMutex
	eventChan       chan *StreamEvent
	server          *http.Server
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(config *StreamerConfig, subMgr *SubscriptionManager, metrics *StreamerMetrics) *WebSocketServer {
	return &WebSocketServer{
		config:          config,
		subscriptionMgr: subMgr,
		metrics:         metrics,
		connections:     make(map[string]*websocket.Conn),
		eventChan:       make(chan *StreamEvent, config.MaxEventsBufferSize),
	}
}

// Start starts the WebSocket server
func (ws *WebSocketServer) Start(ctx context.Context) error {
	if !ws.config.EnableWebSocket {
		utils.LavaFormatInfo("WebSocket server disabled")
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc(ws.config.WebSocketPath, ws.handleWebSocket)
	mux.HandleFunc("/subscribe", ws.handleSubscribe)
	mux.HandleFunc("/unsubscribe", ws.handleUnsubscribe)
	mux.HandleFunc("/health", ws.handleHealth)

	ws.server = &http.Server{
		Addr:    ws.config.WebSocketAddr,
		Handler: mux,
	}

	utils.LavaFormatInfo("WebSocket server starting",
		utils.LogAttr("address", ws.config.WebSocketAddr),
		utils.LogAttr("path", ws.config.WebSocketPath),
	)

	// Start event broadcaster
	go ws.broadcastEvents(ctx)

	// Start ping routine
	go ws.pingConnections(ctx)

	go func() {
		if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.LavaFormatError("WebSocket server error", err)
		}
	}()

	return nil
}

// Stop stops the WebSocket server
func (ws *WebSocketServer) Stop(ctx context.Context) error {
	if ws.server == nil {
		return nil
	}

	utils.LavaFormatInfo("Stopping WebSocket server")
	return ws.server.Shutdown(ctx)
}

// handleWebSocket handles WebSocket connections
func (ws *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check max connections
	ws.mu.RLock()
	connCount := len(ws.connections)
	ws.mu.RUnlock()

	if connCount >= ws.config.MaxWebSocketClients {
		http.Error(w, "Max connections reached", http.StatusServiceUnavailable)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		utils.LavaFormatError("Failed to upgrade connection", err)
		return
	}

	clientID := r.RemoteAddr

	ws.mu.Lock()
	ws.connections[clientID] = conn
	ws.mu.Unlock()

	utils.LavaFormatInfo("New WebSocket connection",
		utils.LogAttr("clientID", clientID),
	)

	// Handle connection
	go ws.handleConnection(clientID, conn)
}

// handleConnection handles a single WebSocket connection
func (ws *WebSocketServer) handleConnection(clientID string, conn *websocket.Conn) {
	defer func() {
		ws.mu.Lock()
		delete(ws.connections, clientID)
		ws.mu.Unlock()
		conn.Close()

		utils.LavaFormatInfo("WebSocket connection closed",
			utils.LogAttr("clientID", clientID),
		)
	}()

	// Read messages from client (subscription requests, etc.)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				utils.LavaFormatWarning("WebSocket error", err)
			}
			break
		}

		// Handle client messages (subscriptions, etc.)
		ws.handleClientMessage(clientID, message, conn)
	}
}

// handleClientMessage handles messages from WebSocket clients
func (ws *WebSocketServer) handleClientMessage(clientID string, message []byte, conn *websocket.Conn) {
	var req struct {
		Action  string       `json:"action"` // subscribe, unsubscribe, ping
		Filters *EventFilter `json:"filters,omitempty"`
		SubID   string       `json:"subscription_id,omitempty"`
	}

	if err := json.Unmarshal(message, &req); err != nil {
		ws.sendError(conn, "Invalid request format")
		return
	}

	switch req.Action {
	case "subscribe":
		sub, err := ws.subscriptionMgr.Subscribe(clientID, req.Filters, nil)
		if err != nil {
			ws.sendError(conn, "Failed to create subscription")
			return
		}

		// Associate WebSocket with subscription
		ws.subscriptionMgr.UpdateWebSocket(sub.ID, conn)

		response := map[string]interface{}{
			"action":          "subscribed",
			"subscription_id": sub.ID,
			"client_id":       clientID,
		}
		ws.sendMessage(conn, response)

	case "unsubscribe":
		if req.SubID == "" {
			ws.sendError(conn, "subscription_id required")
			return
		}

		if err := ws.subscriptionMgr.Unsubscribe(req.SubID); err != nil {
			ws.sendError(conn, "Failed to unsubscribe")
			return
		}

		response := map[string]interface{}{
			"action":          "unsubscribed",
			"subscription_id": req.SubID,
		}
		ws.sendMessage(conn, response)

	case "ping":
		ws.sendMessage(conn, map[string]string{"action": "pong"})

	default:
		ws.sendError(conn, "Unknown action")
	}
}

// SendEvent sends an event to the WebSocket channel
func (ws *WebSocketServer) SendEvent(event *StreamEvent) {
	select {
	case ws.eventChan <- event:
	default:
		utils.LavaFormatWarning("WebSocket event buffer full, dropping event", nil,
			utils.LogAttr("eventType", event.Type),
		)
	}
}

// broadcastEvents broadcasts events to matching subscriptions
func (ws *WebSocketServer) broadcastEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-ws.eventChan:
			ws.broadcastEvent(event)
		}
	}
}

// broadcastEvent broadcasts a single event to matching subscribers
func (ws *WebSocketServer) broadcastEvent(event *StreamEvent) {
	matches := ws.subscriptionMgr.MatchEvent(event)

	for _, sub := range matches {
		if sub.WebSocket == nil {
			continue
		}

		conn, ok := sub.WebSocket.(*websocket.Conn)
		if !ok {
			continue
		}

		// Send event to client
		if err := ws.sendMessage(conn, event); err != nil {
			utils.LavaFormatWarning("Failed to send event to client", err,
				utils.LogAttr("subscriptionID", sub.ID),
			)
			continue
		}

		ws.subscriptionMgr.IncrementEventCount(sub.ID)
	}
}

// pingConnections sends periodic pings to keep connections alive
func (ws *WebSocketServer) pingConnections(ctx context.Context) {
	ticker := time.NewTicker(ws.config.WebSocketPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ws.mu.RLock()
			conns := make(map[string]*websocket.Conn)
			for id, conn := range ws.connections {
				conns[id] = conn
			}
			ws.mu.RUnlock()

			for id, conn := range conns {
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					utils.LavaFormatDebug("Ping failed, closing connection",
						utils.LogAttr("clientID", id),
					)
					conn.Close()
				}
			}
		}
	}
}

// HTTP handlers

func (ws *WebSocketServer) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ClientID string         `json:"client_id"`
		Filters  *EventFilter   `json:"filters"`
		Webhook  *WebhookConfig `json:"webhook,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.ClientID == "" {
		req.ClientID = r.RemoteAddr
	}

	sub, err := ws.subscriptionMgr.Subscribe(req.ClientID, req.Filters, req.Webhook)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

func (ws *WebSocketServer) handleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SubscriptionID string `json:"subscription_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := ws.subscriptionMgr.Unsubscribe(req.SubscriptionID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "unsubscribed"})
}

func (ws *WebSocketServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "ok",
		"connections": len(ws.connections),
		"time":        time.Now().Format(time.RFC3339),
	})
}

// Helper methods

func (ws *WebSocketServer) sendMessage(conn *websocket.Conn, data interface{}) error {
	return conn.WriteJSON(data)
}

func (ws *WebSocketServer) sendError(conn *websocket.Conn, message string) {
	conn.WriteJSON(map[string]string{
		"error": message,
	})
}

// GetConnectionCount returns the number of active WebSocket connections
func (ws *WebSocketServer) GetConnectionCount() int {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return len(ws.connections)
}


