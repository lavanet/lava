package chainlib

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/goccy/go-json"
	"github.com/gofiber/websocket/v2"
	formatter "github.com/lavanet/lava/v4/ecosystem/cache/format"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/metrics"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/rand"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"github.com/tidwall/gjson"
)

var (
	WebSocketRateLimit   = -1               // rate limit requests per second on websocket connection
	WebSocketBanDuration = time.Duration(0) // once rate limit is reached, will not allow new incoming message for a duration
	MaxIdleTimeInSeconds = int64(20 * 60)   // 20 minutes of idle time will disconnect the websocket connection
)

const (
	WebSocketRateLimitHeader            = "x-lava-websocket-rate-limit"
	WebSocketOpenConnectionsLimitHeader = "x-lava-websocket-open-connections-limit"
)

type ConsumerWebsocketManager struct {
	websocketConn                 *websocket.Conn
	rpcConsumerLogs               *metrics.RPCConsumerLogs
	cmdFlags                      common.ConsumerCmdFlags
	refererMatchString            string
	relayMsgLogMaxChars           int
	chainId                       string
	apiInterface                  string
	connectionType                string
	refererData                   *RefererData
	relaySender                   RelaySender
	consumerWsSubscriptionManager *ConsumerWSSubscriptionManager
	WebsocketConnectionUID        string
	headerRateLimit               uint64
}

type ConsumerWebsocketManagerOptions struct {
	WebsocketConn                 *websocket.Conn
	RpcConsumerLogs               *metrics.RPCConsumerLogs
	RefererMatchString            string
	CmdFlags                      common.ConsumerCmdFlags
	RelayMsgLogMaxChars           int
	ChainID                       string
	ApiInterface                  string
	ConnectionType                string
	RefererData                   *RefererData
	RelaySender                   RelaySender
	ConsumerWsSubscriptionManager *ConsumerWSSubscriptionManager
	WebsocketConnectionUID        string
	headerRateLimit               uint64
}

func NewConsumerWebsocketManager(options ConsumerWebsocketManagerOptions) *ConsumerWebsocketManager {
	cwm := &ConsumerWebsocketManager{
		websocketConn:                 options.WebsocketConn,
		relaySender:                   options.RelaySender,
		rpcConsumerLogs:               options.RpcConsumerLogs,
		cmdFlags:                      options.CmdFlags,
		refererMatchString:            options.RefererMatchString,
		relayMsgLogMaxChars:           options.RelayMsgLogMaxChars,
		chainId:                       options.ChainID,
		apiInterface:                  options.ApiInterface,
		connectionType:                options.ConnectionType,
		refererData:                   options.RefererData,
		consumerWsSubscriptionManager: options.ConsumerWsSubscriptionManager,
		WebsocketConnectionUID:        options.WebsocketConnectionUID,
		headerRateLimit:               options.headerRateLimit,
	}
	return cwm
}

func (cwm *ConsumerWebsocketManager) GetWebSocketConnectionUniqueId(dappId, userIp string) string {
	return dappId + "__" + userIp + "__" + cwm.WebsocketConnectionUID
}

func (cwm *ConsumerWebsocketManager) handleRateLimitReached(inpData []byte) ([]byte, error) {
	rateLimitError := common.JsonRpcRateLimitError
	id := 0
	result := gjson.GetBytes(inpData, "id")
	switch result.Type {
	case gjson.Number:
		id = int(result.Int())
	case gjson.String:
		idParsed, err := strconv.Atoi(result.Raw)
		if err == nil {
			id = idParsed
		}
	}
	rateLimitError.Id = id
	bytesRateLimitError, err := json.Marshal(rateLimitError)
	if err != nil {
		return []byte{}, utils.LavaFormatError("failed marshalling jsonrpc rate limit error", err)
	}
	return bytesRateLimitError, nil
}

func (cwm *ConsumerWebsocketManager) ListenToMessages() {
	// adding metrics for how many active connections we have.
	cwm.rpcConsumerLogs.SetWebSocketConnectionActive(cwm.chainId, cwm.apiInterface, true)
	defer cwm.rpcConsumerLogs.SetWebSocketConnectionActive(cwm.chainId, cwm.apiInterface, false)

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

	websocketConn := cwm.websocketConn
	logger := cwm.rpcConsumerLogs

	webSocketCtx, cancelWebSocketCtx := context.WithCancel(context.Background())
	guid := utils.GenerateUniqueIdentifier()
	guidString := strconv.FormatUint(guid, 10)
	webSocketCtx = utils.WithUniqueIdentifier(webSocketCtx, guid)
	utils.LavaFormatDebug("consumer websocket manager started", utils.LogAttr("GUID", webSocketCtx))
	defer func() {
		cancelWebSocketCtx() // In case there's a problem make sure to cancel the connection
		utils.LavaFormatDebug("consumer websocket manager stopped", utils.LogAttr("GUID", webSocketCtx))
	}()

	go func() {
		for msg := range websocketConnWriteChan {
			select {
			case <-webSocketCtx.Done():
				utils.LavaFormatTrace("websocket's context cancelled", utils.LogAttr("GUID", webSocketCtx))
				return
			default:
				err := cwm.websocketConn.WriteMessage(msg.messageType, msg.msg)
				if err != nil {
					utils.LavaFormatTrace("error writing msg to the websocket")
					return
				}
			}
		}
	}()

	// set up a routine to check for rate limits or idle time
	idleFor := atomic.Int64{}
	idleFor.Store(time.Now().Unix())
	requestsPerSecond := &atomic.Uint64{}
	go func() {
		if WebSocketRateLimit <= 0 && cwm.headerRateLimit <= 0 && MaxIdleTimeInSeconds <= 0 {
			return
		}
		ticker := time.NewTicker(time.Second) // rate limit per second.
		defer ticker.Stop()
		for {
			select {
			case <-webSocketCtx.Done():
				utils.LavaFormatDebug("ctx done in time checker")
				return
			case <-ticker.C:
				if MaxIdleTimeInSeconds > 0 {
					utils.LavaFormatDebug("checking idle time", utils.LogAttr("idleFor", idleFor.Load()), utils.LogAttr("maxIdleTime", MaxIdleTimeInSeconds), utils.LogAttr("now", time.Now().Unix()))
					idleDuration := idleFor.Load() + MaxIdleTimeInSeconds
					if time.Now().Unix() > idleDuration {
						websocketConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, fmt.Sprintf("Connection idle for too long, closing connection. Idle time: %d", idleDuration)))
						return
					}
				}
				if cwm.headerRateLimit > 0 || WebSocketRateLimit > 0 {
					// check if rate limit reached, and ban is required
					currentRequestsPerSecondLoad := requestsPerSecond.Load()
					if WebSocketBanDuration > 0 && (currentRequestsPerSecondLoad > cwm.headerRateLimit || currentRequestsPerSecondLoad > uint64(WebSocketRateLimit)) {
						// wait the ban duration before resetting the store.
						select {
						case <-webSocketCtx.Done():
							return
						case <-time.After(WebSocketBanDuration): // just continue
						}
					}
					requestsPerSecond.Store(0)
				}
			}
		}
	}()

	for {
		idleFor.Store(time.Now().Unix())
		startTime := time.Now()
		msgSeed := guidString + "_" + strconv.Itoa(rand.Intn(10000000000)) // use message seed with original guid and new int

		utils.LavaFormatTrace("listening for new message from the websocket")

		if messageType, msg, err = websocketConn.ReadMessage(); err != nil {
			utils.LavaFormatTrace("error reading msg from the websocket, probably websocket was closed by the user", utils.LogAttr("err", err))
			formatterMsg := logger.AnalyzeWebSocketErrorAndGetFormattedMessage(websocketConn.LocalAddr().String(), err, msgSeed, msg, cwm.apiInterface, time.Since(startTime))
			if formatterMsg != nil {
				websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: formatterMsg}
			}
			break
		}

		// Check rate limit is met
		currentRequestsPerSecond := requestsPerSecond.Add(1)
		if (cwm.headerRateLimit > 0 && currentRequestsPerSecond > cwm.headerRateLimit) ||
			(WebSocketRateLimit > 0 && currentRequestsPerSecond > uint64(WebSocketRateLimit)) {
			rateLimitResponse, err := cwm.handleRateLimitReached(msg)
			if err == nil {
				websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: rateLimitResponse}
			}
			continue
		}

		dappID, ok := websocketConn.Locals("dapp-id").(string)
		if !ok {
			// Log and remove the analyze
			formatterMsg := logger.AnalyzeWebSocketErrorAndGetFormattedMessage(websocketConn.LocalAddr().String(), nil, msgSeed, []byte("Unable to extract dappID"), cwm.apiInterface, time.Since(startTime))
			if formatterMsg != nil {
				websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: formatterMsg}
			}
		}

		userIp := websocketConn.RemoteAddr().String()

		logFormattedMsg := string(msg)
		if !cwm.cmdFlags.DebugRelays {
			logFormattedMsg = utils.FormatLongString(logFormattedMsg, cwm.relayMsgLogMaxChars)
		}

		utils.LavaFormatDebug("ws in <<<",
			utils.LogAttr("seed", msgSeed),
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("msg", logFormattedMsg),
			utils.LogAttr("dappID", dappID),
		)

		metricsData := metrics.NewRelayAnalytics(dappID, cwm.chainId, cwm.apiInterface)

		protocolMessage, err := cwm.relaySender.ParseRelay(webSocketCtx, "", string(msg), cwm.connectionType, dappID, userIp, nil)
		if err != nil {
			utils.LavaFormatDebug("ws manager could not parse message", utils.LogAttr("message", msg), utils.LogAttr("err", err))
			formatterMsg := logger.AnalyzeWebSocketErrorAndGetFormattedMessage(websocketConn.LocalAddr().String(), err, msgSeed, msg, cwm.apiInterface, time.Since(startTime))
			if formatterMsg != nil {
				websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: formatterMsg}
			}
			continue
		}

		// check whether it's a normal relay / unsubscribe / unsubscribe_all otherwise its a subscription flow.
		if !IsFunctionTagOfType(protocolMessage, spectypes.FUNCTION_TAG_SUBSCRIBE) {
			if IsFunctionTagOfType(protocolMessage, spectypes.FUNCTION_TAG_UNSUBSCRIBE) {
				err := cwm.consumerWsSubscriptionManager.Unsubscribe(webSocketCtx, protocolMessage, dappID, userIp, cwm.WebsocketConnectionUID, metricsData)
				if err != nil {
					utils.LavaFormatWarning("error unsubscribing from subscription", err, utils.LogAttr("GUID", webSocketCtx))
					if err == common.SubscriptionNotFoundError {
						msgData, err := json.Marshal(common.JsonRpcSubscriptionNotFoundError)
						if err != nil {
							continue
						}
						websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: msgData}
					}
				}
				continue
			} else if IsFunctionTagOfType(protocolMessage, spectypes.FUNCTION_TAG_UNSUBSCRIBE_ALL) {
				err := cwm.consumerWsSubscriptionManager.UnsubscribeAll(webSocketCtx, dappID, userIp, cwm.WebsocketConnectionUID, metricsData)
				if err != nil {
					utils.LavaFormatWarning("error unsubscribing from all subscription", err, utils.LogAttr("GUID", webSocketCtx))
				}
				continue
			} else {
				// Normal relay over websocket. (not subscription related)
				relayResult, err := cwm.relaySender.SendParsedRelay(webSocketCtx, metricsData, protocolMessage)
				if err != nil {
					formatterMsg := logger.AnalyzeWebSocketErrorAndGetFormattedMessage(websocketConn.LocalAddr().String(), utils.LavaFormatError("could not send parsed relay", err), msgSeed, msg, cwm.apiInterface, time.Since(startTime))
					if formatterMsg != nil {
						websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: formatterMsg}
					}
					continue
				}

				relayResultReply := relayResult.GetReply()
				if relayResultReply != nil {
					// No need to verify signature since this is already happening inside the SendParsedRelay flow
					websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: relayResult.GetReply().Data}
				} else {
					utils.LavaFormatError("Relay result is nil over websocket normal request flow, should not happen", err, utils.LogAttr("messageType", messageType))
				}
				continue
			}
		}

		// Subscription flow
		inputFormatter, outputFormatter := formatter.FormatterForRelayRequestAndResponse(protocolMessage.GetApiCollection().CollectionData.ApiInterface) // we use this to preserve the original jsonrpc id
		inputFormatter(protocolMessage.RelayPrivateData().Data)                                                                                          // set the extracted jsonrpc id

		reply, subscriptionMsgsChan, err := cwm.consumerWsSubscriptionManager.StartSubscription(webSocketCtx, protocolMessage, dappID, userIp, cwm.WebsocketConnectionUID, metricsData)
		if err != nil {
			utils.LavaFormatWarning("StartSubscription returned an error", err,
				utils.LogAttr("GUID", webSocketCtx),
				utils.LogAttr("dappID", dappID),
				utils.LogAttr("userIp", userIp),
				utils.LogAttr("params", protocolMessage.GetRPCMessage().GetParams()),
			)

			formatterMsg := logger.AnalyzeWebSocketErrorAndGetFormattedMessage(websocketConn.LocalAddr().String(), utils.LavaFormatError("could not start subscription", err), msgSeed, msg, cwm.apiInterface, time.Since(startTime))
			if formatterMsg != nil {
				websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: formatterMsg} // No need to use outputFormatter here since we are sending an error
				continue
			}

			// Handle the case when the error is a method not found error
			if common.APINotSupportedError.Is(err) {
				msgData, err := json.Marshal(common.JsonRpcMethodNotFoundError)
				if err != nil {
					continue
				}

				websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: outputFormatter(msgData)}
				continue
			}
			continue
		}

		if subscriptionMsgsChan != nil { // if == nil, it means that we already have an active subscription running on this query
			go func() {
				utils.LavaFormatTrace("created go routine for new websocketSubMsgsChan",
					utils.LogAttr("GUID", webSocketCtx),
					utils.LogAttr("dappID", dappID),
					utils.LogAttr("userIp", userIp),
					utils.LogAttr("params", protocolMessage.GetRPCMessage().GetParams()),
				)

				for subscriptionMsgReply := range subscriptionMsgsChan {
					idleFor.Store(time.Now().Unix())
					websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: outputFormatter(subscriptionMsgReply.Data)}
				}

				utils.LavaFormatTrace("subscriptionMsgsChan was closed",
					utils.LogAttr("GUID", webSocketCtx),
					utils.LogAttr("dappID", dappID),
					utils.LogAttr("userIp", userIp),
					utils.LogAttr("params", protocolMessage.GetRPCMessage().GetParams()),
				)
			}()
		}

		refererMatch, referrerMatchCastedSuccessfully := websocketConn.Locals(cwm.refererMatchString).(string)
		if referrerMatchCastedSuccessfully && refererMatch != "" && cwm.refererData != nil {
			go cwm.refererData.SendReferer(refererMatch, cwm.chainId, string(msg), websocketConn.RemoteAddr().String(), nil, websocketConn)
		}

		go logger.AddMetricForWebSocket(metricsData, err, websocketConn)

		if reply != nil {
			reply.Data = outputFormatter(reply.Data) // use that id for the reply

			websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: reply.Data}

			logger.LogRequestAndResponse("jsonrpc ws msg", false, "ws", websocketConn.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, time.Since(startTime), nil)
		}
	}
}
