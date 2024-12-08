package chainlib

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/utils"
)

// WebsocketConnection defines the interface for websocket connections
type WebsocketConnection interface {
	// Add only the methods you need to mock
	RemoteAddr() net.Addr
	Locals(key string) interface{}
	WriteMessage(messageType int, data []byte) error
}

// Will limit a certain amount of connections per IP
type WebsocketConnectionLimiter struct {
	ipToNumberOfActiveConnections map[string]int64
	lock                          sync.RWMutex
}

func (wcl *WebsocketConnectionLimiter) HandleFiberRateLimitFlags(c *fiber.Ctx) {
	userAgent := c.Get(fiber.HeaderUserAgent)
	// Store the User-Agent in locals for later use
	c.Locals(fiber.HeaderUserAgent, userAgent)

	forwardedFor := c.Get(common.IP_FORWARDING_HEADER_NAME)
	if forwardedFor == "" {
		// If not present, fallback to c.IP() which retrieves the real IP
		forwardedFor = c.IP()
	}
	// Store the X-Forwarded-For or real IP in the context
	c.Locals(common.IP_FORWARDING_HEADER_NAME, forwardedFor)

	rateLimitString := c.Get(WebSocketRateLimitHeader)
	rateLimit, err := strconv.ParseInt(rateLimitString, 10, 64)
	if err != nil {
		rateLimit = 0
	}
	c.Locals(WebSocketRateLimitHeader, rateLimit)

	connectionLimitString := c.Get(WebSocketOpenConnectionsLimitHeader)
	connectionLimit, err := strconv.ParseInt(connectionLimitString, 10, 64)
	if err != nil {
		connectionLimit = 0
	}
	c.Locals(WebSocketOpenConnectionsLimitHeader, connectionLimit)
}

func (wcl *WebsocketConnectionLimiter) getConnectionLimit(websocketConn WebsocketConnection) int64 {
	connectionLimitHeaderValue, ok := websocketConn.Locals(WebSocketOpenConnectionsLimitHeader).(int64)
	if !ok || connectionLimitHeaderValue < 0 {
		connectionLimitHeaderValue = 0
	}
	// Do not allow header to overwrite flag value if its set.
	if MaximumNumberOfParallelWebsocketConnectionsPerIp > 0 && connectionLimitHeaderValue > MaximumNumberOfParallelWebsocketConnectionsPerIp {
		return MaximumNumberOfParallelWebsocketConnectionsPerIp
	}
	// Return the larger of the global limit (if set) or the header value
	return utils.Max(MaximumNumberOfParallelWebsocketConnectionsPerIp, connectionLimitHeaderValue)
}

func (wcl *WebsocketConnectionLimiter) CanOpenConnection(websocketConn WebsocketConnection) (bool, func()) {
	// Check which connection limit is higher and use that.
	connectionLimit := wcl.getConnectionLimit(websocketConn)
	decreaseIpConnectionCallback := func() {}
	if connectionLimit > 0 { // 0 is disabled.
		ipForwardedInterface := websocketConn.Locals(common.IP_FORWARDING_HEADER_NAME)
		ipForwarded, assertionSuccessful := ipForwardedInterface.(string)
		if !assertionSuccessful {
			ipForwarded = ""
		}
		ip := websocketConn.RemoteAddr().String()
		userAgent, assertionSuccessful := websocketConn.Locals("User-Agent").(string)
		if !assertionSuccessful {
			userAgent = ""
		}
		key := wcl.getKey(ip, ipForwarded, userAgent)

		// Check current connections before incrementing
		currentConnections := wcl.getCurrentAmountOfConnections(key)
		// If already at or exceeding limit, deny the connection
		if currentConnections >= connectionLimit {
			websocketConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, fmt.Sprintf("Too Many Open Connections, limited to %d", connectionLimit)))
			return false, decreaseIpConnectionCallback
		}
		// If under limit, increment and return cleanup function
		wcl.addIpConnection(key)
		decreaseIpConnectionCallback = func() { wcl.decreaseIpConnection(key) }
	}
	return true, decreaseIpConnectionCallback
}

func (wcl *WebsocketConnectionLimiter) getCurrentAmountOfConnections(key string) int64 {
	wcl.lock.RLock()
	defer wcl.lock.RUnlock()
	return wcl.ipToNumberOfActiveConnections[key]
}

func (wcl *WebsocketConnectionLimiter) addIpConnection(key string) {
	wcl.lock.Lock()
	defer wcl.lock.Unlock()
	// wether it exists or not we add 1.
	wcl.ipToNumberOfActiveConnections[key] += 1
}

func (wcl *WebsocketConnectionLimiter) decreaseIpConnection(key string) {
	wcl.lock.Lock()
	defer wcl.lock.Unlock()
	// it must exist as we dont get here without adding it prior
	wcl.ipToNumberOfActiveConnections[key] -= 1
	if wcl.ipToNumberOfActiveConnections[key] == 0 {
		delete(wcl.ipToNumberOfActiveConnections, key)
	}
}

func (wcl *WebsocketConnectionLimiter) getKey(ip string, forwardedIp string, userAgent string) string {
	returnedKey := ""
	ipOriginal := net.ParseIP(ip)
	if ipOriginal != nil {
		returnedKey = ipOriginal.String()
	} else {
		ipPart, _, err := net.SplitHostPort(ip)
		if err == nil {
			returnedKey = ipPart
		}
	}
	ips := strings.Split(forwardedIp, ",")
	for _, ipStr := range ips {
		ipParsed := net.ParseIP(strings.TrimSpace(ipStr))
		if ipParsed != nil {
			returnedKey += SEP + ipParsed.String()
		} else {
			ipPart, _, err := net.SplitHostPort(ipStr)
			if err == nil {
				returnedKey += SEP + ipPart
			}
		}
	}
	returnedKey += SEP + userAgent
	return returnedKey
}
