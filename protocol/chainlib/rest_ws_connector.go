package chainlib

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/utils"
)

const (
	restWsReadTimeout      = 30 * time.Second
	restWsWriteTimeout     = 10 * time.Second
	restWsHandshakeTimeout = 10 * time.Second // Timeout for initial WebSocket handshake
)

// RestWsConnection wraps a websocket connection for REST-style JSON communication
// Mirrors the rpcclient.Client interface where applicable
type RestWsConnection struct {
	conn    *websocket.Conn
	mu      sync.Mutex
	headers map[string]string
}

// SetHeader sets a header value for the connection (mirrors rpcclient.Client.SetHeader)
func (rwc *RestWsConnection) SetHeader(key, value string) {
	rwc.mu.Lock()
	defer rwc.mu.Unlock()
	if rwc.headers == nil {
		rwc.headers = make(map[string]string)
	}
	if value == "" {
		delete(rwc.headers, key)
	} else {
		rwc.headers[key] = value
	}
}

// DelHeader deletes a header (mirrors rpcclient.Client.DelHeader)
func (rwc *RestWsConnection) DelHeader(key string) {
	rwc.mu.Lock()
	defer rwc.mu.Unlock()
	if rwc.headers != nil {
		delete(rwc.headers, key)
	}
}

// SendAndReceive sends a plain JSON message and waits for a response
// This is the REST WebSocket equivalent of rpcclient.Client.CallContext
func (rwc *RestWsConnection) SendAndReceive(ctx context.Context, msg []byte) ([]byte, error) {
	rwc.mu.Lock()
	defer rwc.mu.Unlock()

	// Set write deadline
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(restWsWriteTimeout)
	}
	if err := rwc.conn.SetWriteDeadline(deadline); err != nil {
		return nil, utils.LavaFormatError("failed to set write deadline", err)
	}

	// Send the message as text
	if err := rwc.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		return nil, utils.LavaFormatError("failed to write websocket message", err)
	}

	// Set read deadline
	readDeadline, ok := ctx.Deadline()
	if !ok {
		readDeadline = time.Now().Add(restWsReadTimeout)
	}
	if err := rwc.conn.SetReadDeadline(readDeadline); err != nil {
		return nil, utils.LavaFormatError("failed to set read deadline", err)
	}

	// Read the response
	messageType, response, err := rwc.conn.ReadMessage()
	if err != nil {
		return nil, utils.LavaFormatError("failed to read websocket response", err)
	}

	if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
		return nil, utils.LavaFormatError("unexpected websocket message type", nil, utils.LogAttr("messageType", messageType))
	}

	return response, nil
}

// Close closes the websocket connection
func (rwc *RestWsConnection) Close() error {
	return rwc.conn.Close()
}

// RestWsConnector manages a pool of websocket connections for REST-style JSON communication
type RestWsConnector struct {
	lock          sync.RWMutex
	freeClients   []*RestWsConnection
	usedClients   int64
	nodeUrl       common.NodeUrl
	hashedNodeUrl string
	nConns        uint
}

// NewRestWsConnector creates a new REST WebSocket connector
// Mirrors chainproxy.NewConnector pattern
func NewRestWsConnector(ctx context.Context, nConns uint, nodeUrl common.NodeUrl) (*RestWsConnector, error) {
	connector := &RestWsConnector{
		freeClients:   make([]*RestWsConnection, 0, nConns),
		nodeUrl:       nodeUrl,
		hashedNodeUrl: chainproxy.HashURL(nodeUrl.Url),
		nConns:        nConns,
	}

	// Create the first connection to validate connectivity
	// Use a generous timeout for the initial connection
	initCtx, cancel := context.WithTimeout(ctx, restWsHandshakeTimeout*2)
	defer cancel()

	wsConn, err := connector.createConnection(initCtx)
	if err != nil {
		return nil, utils.LavaFormatError("failed to create initial REST websocket connection", err, utils.LogAttr("url", nodeUrl.UrlStr()))
	}
	connector.addClient(wsConn)

	// Create additional connections asynchronously (same pattern as chainproxy.Connector)
	go connector.addClientsAsync(ctx, nConns-1)

	return connector, nil
}

func (c *RestWsConnector) createConnection(ctx context.Context) (*RestWsConnection, error) {
	// Build the websocket URL with auth path (e.g., API key in path)
	wsUrl := c.nodeUrl.AuthConfig.AddAuthPath(c.nodeUrl.Url)

	// Create dialer with proper timeout for WebSocket handshake
	dialer := websocket.Dialer{
		HandshakeTimeout: restWsHandshakeTimeout,
	}

	// Create a timeout context for the connection attempt
	connectCtx, cancel := context.WithTimeout(ctx, restWsHandshakeTimeout)
	defer cancel()

	// Build auth headers (same as REST HTTP does with SetAuthHeaders)
	var headers http.Header
	if len(c.nodeUrl.AuthConfig.AuthHeaders) > 0 {
		headers = make(http.Header)
		for key, value := range c.nodeUrl.AuthConfig.AuthHeaders {
			headers.Set(key, value)
		}
	}

	// Connect to the websocket with auth headers
	conn, _, err := dialer.DialContext(connectCtx, wsUrl, headers)
	if err != nil {
		return nil, err
	}

	return &RestWsConnection{conn: conn}, nil
}

func (c *RestWsConnector) addClient(client *RestWsConnection) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.freeClients = append(c.freeClients, client)
}

func (c *RestWsConnector) addClientsAsync(ctx context.Context, count uint) {
	for i := uint(0); i < count; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		wsConn, err := c.createConnection(ctx)
		if err != nil {
			utils.LavaFormatWarning("failed to create REST websocket connection", err)
			continue
		}
		c.addClient(wsConn)
	}

	utils.LavaFormatInfo("Finished adding REST WebSocket clients",
		utils.LogAttr("free clients", len(c.freeClients)),
		utils.LogAttr("url", c.nodeUrl.UrlStr()),
	)
}

// GetUrlHash returns the hashed node URL
func (c *RestWsConnector) GetUrlHash() string {
	return c.hashedNodeUrl
}

// GetWsConnection gets a websocket connection from the pool
// Mirrors Connector.GetRpc interface
func (c *RestWsConnector) GetWsConnection(ctx context.Context, block bool) (*RestWsConnection, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	numberOfFreeClients := len(c.freeClients)

	// If running low on connections, create more asynchronously (same pattern as Connector)
	if numberOfFreeClients <= int(c.usedClients) {
		go c.increaseNumberOfClients(ctx)
	}

	if numberOfFreeClients == 0 {
		if !block {
			return nil, ErrNoWebSocketConnections
		}
		// Block until a connection is available (same pattern as Connector.GetRpc)
		for {
			c.lock.Unlock()
			go c.increaseNumberOfClients(ctx)
			time.Sleep(50 * time.Millisecond)
			c.lock.Lock()
			numberOfFreeClients = len(c.freeClients)
			if numberOfFreeClients != 0 {
				break
			}
		}
	}

	// Get a connection from the pool
	ret := c.freeClients[0]
	c.freeClients = c.freeClients[1:]
	c.usedClients++

	return ret, nil
}

// ReturnWsConnection returns a websocket connection to the pool
func (c *RestWsConnector) ReturnWsConnection(conn *RestWsConnection) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.usedClients--
	if len(c.freeClients) > (int(c.usedClients) + int(c.nConns)) {
		conn.Close() // close connection
		return       // return without appending back to decrease idle connections
	}
	c.freeClients = append(c.freeClients, conn)
}

func (c *RestWsConnector) increaseNumberOfClients(ctx context.Context) {
	timeoutCtx, cancel := context.WithTimeout(ctx, common.AverageWorldLatency*2)
	defer cancel()

	wsConn, err := c.createConnection(timeoutCtx)
	if err != nil {
		utils.LavaFormatDebug("failed to increase REST websocket connections", utils.LogAttr("err", err))
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	c.freeClients = append(c.freeClients, wsConn)
}

// Close closes all connections in the pool
func (c *RestWsConnector) Close() {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, client := range c.freeClients {
		client.Close()
	}
	c.freeClients = []*RestWsConnection{}
}

var ErrNoWebSocketConnections = errors.New("no websocket connections available")
