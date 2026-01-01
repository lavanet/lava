package chainlib

import (
	"context"
	"encoding/json"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/rand"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
)

// RestWebsocketManager handles plain JSON WebSocket communication for REST APIs
// Unlike ConsumerWebsocketManager which handles JSON-RPC, this sends plain JSON as-is
type RestWebsocketManager struct {
	websocketConn          *websocket.Conn
	rpcConsumerLogs        *metrics.RPCConsumerLogs
	cmdFlags               common.ConsumerCmdFlags
	relayMsgLogMaxChars    int
	chainId                string
	apiInterface           string
	refererData            *RefererData
	relaySender            RelaySender
	WebsocketConnectionUID string
	headerRateLimit        uint64
}

type RestWebsocketManagerOptions struct {
	WebsocketConn          *websocket.Conn
	RpcConsumerLogs        *metrics.RPCConsumerLogs
	CmdFlags               common.ConsumerCmdFlags
	RelayMsgLogMaxChars    int
	ChainID                string
	ApiInterface           string
	RefererData            *RefererData
	RelaySender            RelaySender
	WebsocketConnectionUID string
	HeaderRateLimit        uint64
}

func NewRestWebsocketManager(options RestWebsocketManagerOptions) *RestWebsocketManager {
	return &RestWebsocketManager{
		websocketConn:          options.WebsocketConn,
		relaySender:            options.RelaySender,
		rpcConsumerLogs:        options.RpcConsumerLogs,
		cmdFlags:               options.CmdFlags,
		relayMsgLogMaxChars:    options.RelayMsgLogMaxChars,
		chainId:                options.ChainID,
		apiInterface:           options.ApiInterface,
		refererData:            options.RefererData,
		WebsocketConnectionUID: options.WebsocketConnectionUID,
		headerRateLimit:        options.HeaderRateLimit,
	}
}

func (rwm *RestWebsocketManager) ListenToMessages() {
	// Adding metrics for active connections
	rwm.rpcConsumerLogs.SetWebSocketConnectionActive(rwm.chainId, rwm.apiInterface, true)
	defer rwm.rpcConsumerLogs.SetWebSocketConnectionActive(rwm.chainId, rwm.apiInterface, false)

	var (
		messageType int
		msg         []byte
		err         error
	)

	type webSocketMsgWithType struct {
		messageType int
		msg         []byte
	}

	websocketConnWriteChan := make(chan webSocketMsgWithType)

	websocketConn := rwm.websocketConn
	logger := rwm.rpcConsumerLogs

	webSocketCtx, cancelWebSocketCtx := context.WithCancel(context.Background())
	guid := utils.GenerateUniqueIdentifier()
	guidString := strconv.FormatUint(guid, 10)
	webSocketCtx = utils.WithUniqueIdentifier(webSocketCtx, guid)
	utils.LavaFormatDebug("REST websocket manager started", utils.LogAttr("GUID", webSocketCtx))
	defer func() {
		cancelWebSocketCtx()
		utils.LavaFormatDebug("REST websocket manager stopped", utils.LogAttr("GUID", webSocketCtx))
	}()

	// Writer goroutine
	go func() {
		for msg := range websocketConnWriteChan {
			select {
			case <-webSocketCtx.Done():
				utils.LavaFormatTrace("REST websocket context cancelled", utils.LogAttr("GUID", webSocketCtx))
				return
			default:
				err := rwm.websocketConn.WriteMessage(msg.messageType, msg.msg)
				if err != nil {
					utils.LavaFormatTrace("error writing msg to REST websocket")
					return
				}
			}
		}
	}()

	// Rate limiting and idle time checking
	idleFor := atomic.Int64{}
	idleFor.Store(time.Now().Unix())
	requestsPerSecond := &atomic.Uint64{}
	go func() {
		if WebSocketRateLimit <= 0 && rwm.headerRateLimit <= 0 && MaxIdleTimeInSeconds <= 0 {
			return
		}
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-webSocketCtx.Done():
				return
			case <-ticker.C:
				if MaxIdleTimeInSeconds > 0 {
					idleDuration := idleFor.Load() + MaxIdleTimeInSeconds
					if time.Now().Unix() > idleDuration {
						websocketConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Connection idle for too long"))
						return
					}
				}
				if rwm.headerRateLimit > 0 || WebSocketRateLimit > 0 {
					currentRequestsPerSecondLoad := requestsPerSecond.Load()
					if WebSocketBanDuration > 0 && (currentRequestsPerSecondLoad > rwm.headerRateLimit || currentRequestsPerSecondLoad > uint64(WebSocketRateLimit)) {
						select {
						case <-webSocketCtx.Done():
							return
						case <-time.After(WebSocketBanDuration):
						}
					}
					requestsPerSecond.Store(0)
				}
			}
		}
	}()

	// Main message loop
	for {
		idleFor.Store(time.Now().Unix())
		startTime := time.Now()
		msgSeed := guidString + "_" + strconv.Itoa(rand.Intn(10000000000))

		utils.LavaFormatTrace("REST websocket listening for new message")

		if messageType, msg, err = websocketConn.ReadMessage(); err != nil {
			utils.LavaFormatTrace("error reading from REST websocket", utils.LogAttr("err", err))
			break
		}

		// Check rate limit
		currentRequestsPerSecond := requestsPerSecond.Add(1)
		if (rwm.headerRateLimit > 0 && currentRequestsPerSecond > rwm.headerRateLimit) ||
			(WebSocketRateLimit > 0 && currentRequestsPerSecond > uint64(WebSocketRateLimit)) {
			// Send rate limit error as plain JSON
			rateLimitErr := []byte(`{"error": "rate limit exceeded"}`)
			websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: rateLimitErr}
			continue
		}

		dappID, ok := websocketConn.Locals("dapp-id").(string)
		if !ok {
			dappID = "DefaultDappID"
		}

		userIp := websocketConn.RemoteAddr().String()

		logFormattedMsg := string(msg)
		if !rwm.cmdFlags.DebugRelays {
			logFormattedMsg = utils.FormatLongString(logFormattedMsg, rwm.relayMsgLogMaxChars)
		}

		utils.LavaFormatDebug("REST ws in <<<",
			utils.LogAttr("seed", msgSeed),
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("msg", logFormattedMsg),
			utils.LogAttr("dappID", dappID),
		)

		metricsData := metrics.NewRelayAnalytics(dappID, rwm.chainId, rwm.apiInterface)

		// For REST WebSocket, extract the "command" or "method" field from the JSON
		// and use it as the API name (similar to how JSON-RPC uses "method")
		// This supports XRP/Ripple-style WebSocket which uses {"command": "server_info"}
		// The API name matches the spec (e.g., "server_info" not "/server_info")
		apiName := ""
		var msgObj map[string]interface{}
		if err := json.Unmarshal(msg, &msgObj); err == nil {
			// Try "command" first (XRP/Ripple style), then "method" (JSON-RPC style without envelope)
			if cmd, ok := msgObj["command"].(string); ok && cmd != "" {
				apiName = cmd
			} else if method, ok := msgObj["method"].(string); ok && method != "" {
				apiName = method
			}
		}

		if apiName == "" {
			utils.LavaFormatDebug("REST ws message missing command/method field", utils.LogAttr("message", msg))
			errResponse := []byte(`{"error": "missing command or method field in request"}`)
			websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: errResponse}
			continue
		}

		// Add "websocket" extension so the provider routes to the WebSocket proxy
		// This is passed via the lava-extension header in metadata
		wsMetadata := []pairingtypes.Metadata{
			{Name: common.EXTENSION_OVERRIDE_HEADER_NAME, Value: WebSocketExtension},
		}

		// Use "WEBSOCKET" connection type to signal REST parser to use the API name directly
		// (like JSON-RPC uses method name) instead of treating it as a URL path
		protocolMessage, err := rwm.relaySender.ParseRelay(webSocketCtx, apiName, string(msg), "WEBSOCKET", dappID, userIp, wsMetadata)
		if err != nil {
			utils.LavaFormatDebug("REST ws manager could not parse message", utils.LogAttr("message", msg), utils.LogAttr("apiName", apiName), utils.LogAttr("err", err))
			// Return error as plain JSON (not JSON-RPC format)
			errResponse := []byte(`{"error": "` + err.Error() + `"}`)
			websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: errResponse}
			continue
		}

		// Send the relay
		relayResult, err := rwm.relaySender.SendParsedRelay(webSocketCtx, metricsData, protocolMessage)
		if err != nil {
			utils.LavaFormatDebug("REST ws relay failed", utils.LogAttr("err", err))
			errResponse := []byte(`{"error": "` + err.Error() + `"}`)
			websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: errResponse}
			continue
		}

		relayResultReply := relayResult.GetReply()
		if relayResultReply != nil {
			websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: relayResult.GetReply().Data}
		} else {
			utils.LavaFormatError("REST ws relay result is nil", nil)
		}

		go logger.AddMetricForWebSocket(metricsData, err, websocketConn)
		logger.LogRequestAndResponse("rest ws msg", false, "ws", websocketConn.LocalAddr().String(), string(msg), string(relayResultReply.Data), msgSeed, time.Since(startTime), nil)
	}
}
