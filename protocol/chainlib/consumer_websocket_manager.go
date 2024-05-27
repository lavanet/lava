package chainlib

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
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
	mutex                         sync.Mutex
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
		mutex:                         sync.Mutex{},
	}

	return cwm
}

func (cwm *ConsumerWebsocketManager) ListenForMessages() {
	utils.LavaFormatTrace("consumer websocket manager started")
	defer utils.LavaFormatTrace("consumer websocket manager stopped")
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
	webSocketCtx = utils.WithUniqueIdentifier(webSocketCtx, guid)
	defer cancelWebSocketCtx() // In case there's a problem make sure to cancel the connection

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

	for {
		startTime := time.Now()
		msgSeed := logger.GetMessageSeed()

		utils.LavaFormatTrace("listening for new message from the websocket")

		if messageType, msg, err = websocketConn.ReadMessage(); err != nil {
			utils.LavaFormatDebug("error reading msg from the websocket", utils.LogAttr("err", err))
			logger.AnalyzeWebSocketErrorAndWriteMessage(websocketConn, messageType, err, msgSeed, msg, cwm.apiInterface, time.Since(startTime))
			break
		}

		dappID, ok := websocketConn.Locals("dapp-id").(string)
		if !ok {
			// Log and remove the analyze
			logger.AnalyzeWebSocketErrorAndWriteMessage(websocketConn, messageType, nil, msgSeed, []byte("Unable to extract dappID"), cwm.apiInterface, time.Since(startTime))
		}

		msgSeed = strconv.FormatUint(guid, 10)
		consumerIp := websocketConn.RemoteAddr().String()

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

		refererMatch, ok := websocketConn.Locals(cwm.refererMatchString).(string)
		metricsData := metrics.NewRelayAnalytics(dappID, cwm.chainId, cwm.apiInterface)

		chainMessage, directiveHeaders, relayRequestData, err := cwm.relaySender.ParseRelay(webSocketCtx, "", string(msg), cwm.connectionType, dappID, consumerIp, metricsData, nil)
		if err != nil {
			logger.AnalyzeWebSocketErrorAndWriteMessage(websocketConn, messageType, utils.LavaFormatError("could not parse message", err), msgSeed, msg, cwm.apiInterface, time.Since(startTime))
			continue
		}

		if !IsOfFunctionType(chainMessage, spectypes.FUNCTION_TAG_SUBSCRIBE) {
			if IsOfFunctionType(chainMessage, spectypes.FUNCTION_TAG_UNSUBSCRIBE) {
				err := cwm.consumerWsSubscriptionManager.Unsubscribe(webSocketCtx, chainMessage, directiveHeaders, relayRequestData, dappID, consumerIp, metricsData)
				if err != nil {
					utils.LavaFormatWarning("error unsubscribing from subscription", err, utils.LogAttr("GUID", webSocketCtx))
				}
				continue
			}

			if IsOfFunctionType(chainMessage, spectypes.FUNCTION_TAG_UNSUBSCRIBE_ALL) {
				err := cwm.consumerWsSubscriptionManager.UnsubscribeAll(webSocketCtx, chainMessage, directiveHeaders, relayRequestData, dappID, consumerIp, metricsData)
				if err != nil {
					utils.LavaFormatWarning("error unsubscribing from all subscription", err, utils.LogAttr("GUID", webSocketCtx))
				}
				continue
			}

			// One shot relay
			relayResult, err := cwm.relaySender.SendParsedRelay(webSocketCtx, dappID, consumerIp, metricsData, chainMessage, directiveHeaders, relayRequestData)
			if err != nil {
				logger.AnalyzeWebSocketErrorAndWriteMessage(websocketConn, messageType, utils.LavaFormatError("could not send parsed relay", err), msgSeed, msg, cwm.apiInterface, time.Since(startTime))
			}

			// No need to verify signature since this is already happening inside the SendParsedRelay flow
			websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: relayResult.GetReply().Data}
			continue
		}

		// Subscription flow
		reply, websocketSubMsgsChan, err := cwm.consumerWsSubscriptionManager.StartSubscription(webSocketCtx, chainMessage, directiveHeaders, relayRequestData, dappID, consumerIp, metricsData)

		go func() {
			for reply := range websocketSubMsgsChan {
				websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: reply.Data}
			}
		}()

		if ok && refererMatch != "" && cwm.refererData != nil && err == nil {
			go cwm.refererData.SendReferer(refererMatch, cwm.chainId, string(msg), nil, websocketConn)
		}

		go logger.AddMetricForWebSocket(metricsData, err, websocketConn)

		if err != nil {
			logger.AnalyzeWebSocketErrorAndWriteMessage(websocketConn, messageType, err, msgSeed, msg, "tendermint", time.Since(startTime))

			// Handle the case when the error is a method not found error
			if common.APINotSupportedError.Is(err) {
				msgData, err := json.Marshal(common.JsonRpcMethodNotFoundError)
				if err != nil {
					continue
				}

				websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: msgData}
			}

			// TODO: Write error to the user

			continue
		}

		if reply != nil {
			websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: reply.Data}

			logger.LogRequestAndResponse("jsonrpc ws msg", false, "ws", websocketConn.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, time.Since(startTime), nil)
		}
	}
}
