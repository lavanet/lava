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

// Will limit a certain amount of connections per IP
type WebsocketConnectionLimiter struct {
	ipToNumberOfActiveConnections map[string]int64
	lock                          sync.RWMutex
}

func (wcl *WebsocketConnectionLimiter) handleFiberRateLimitFlags(c *fiber.Ctx) {
	userAgent := c.Get("User-Agent")
	utils.LavaFormatDebug("User-Agent", utils.LogAttr("userAgent", userAgent))
	// Store the User-Agent in locals for later use
	c.Locals("User-Agent", userAgent)

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

func (wcl *WebsocketConnectionLimiter) getConnectionLimit(websocketConn *websocket.Conn) int64 {
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

func (wcl *WebsocketConnectionLimiter) canOpenConnection(websocketConn *websocket.Conn) (bool, func()) {
	// Check which connection limit is higher and use that.
	connectionLimit := wcl.getConnectionLimit(websocketConn)
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
		numberOfActiveConnections := wcl.addIpConnectionAndGetCurrentAmount(key)

		if numberOfActiveConnections > connectionLimit {
			websocketConn.WriteMessage(1, []byte(fmt.Sprintf("Too Many Open Connections, limited to %d", connectionLimit)))
			return false, func() { wcl.decreaseIpConnection(key) }
		}
	}
	return true, func() {}
}

func (wcl *WebsocketConnectionLimiter) addIpConnectionAndGetCurrentAmount(ip string) int64 {
	wcl.lock.Lock()
	defer wcl.lock.Unlock()
	// wether it exists or not we add 1.
	wcl.ipToNumberOfActiveConnections[ip] += 1
	return wcl.ipToNumberOfActiveConnections[ip]
}

func (wcl *WebsocketConnectionLimiter) decreaseIpConnection(ip string) {
	wcl.lock.Lock()
	defer wcl.lock.Unlock()
	// wether it exists or not we add 1.
	wcl.ipToNumberOfActiveConnections[ip] -= 1
	if wcl.ipToNumberOfActiveConnections[ip] == 0 {
		delete(wcl.ipToNumberOfActiveConnections, ip)
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
