package chainlib

import (
	"context"
	"strconv"
	"sync"
	"time"

	gojson "github.com/goccy/go-json"
	"github.com/gofiber/websocket/v2"
	formatter "github.com/lavanet/lava/ecosystem/cache/format"
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

func (cwm *ConsumerWebsocketManager) ListenToMessages() {
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

	for {
		startTime := time.Now()
		msgSeed := logger.GetMessageSeed()

		utils.LavaFormatTrace("listening for new message from the websocket")

		if messageType, msg, err = websocketConn.ReadMessage(); err != nil {
			utils.LavaFormatTrace("error reading msg from the websocket, probably websocket was closed by the user", utils.LogAttr("err", err))
			formatterMsg := logger.AnalyzeWebSocketErrorAndGetFormattedMessage(websocketConn.LocalAddr().String(), err, msgSeed, msg, cwm.apiInterface, time.Since(startTime))
			if formatterMsg != nil {
				websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: formatterMsg}
			}
			break
		}

		dappID, ok := websocketConn.Locals("dapp-id").(string)
		if !ok {
			// Log and remove the analyze
			formatterMsg := logger.AnalyzeWebSocketErrorAndGetFormattedMessage(websocketConn.LocalAddr().String(), nil, msgSeed, []byte("Unable to extract dappID"), cwm.apiInterface, time.Since(startTime))
			if formatterMsg != nil {
				websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: formatterMsg}
			}
		}

		msgSeed = strconv.FormatUint(guid, 10)
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

		chainMessage, directiveHeaders, relayRequestData, err := cwm.relaySender.ParseRelay(webSocketCtx, "", string(msg), cwm.connectionType, dappID, userIp, metricsData, nil)
		if err != nil {
			formatterMsg := logger.AnalyzeWebSocketErrorAndGetFormattedMessage(websocketConn.LocalAddr().String(), utils.LavaFormatError("could not parse message", err), msgSeed, msg, cwm.apiInterface, time.Since(startTime))
			if formatterMsg != nil {
				websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: formatterMsg}
			}
			continue
		}

		// check whether its a normal relay / unsubscribe / unsubscribe_all otherwise its a subscription flow.
		if !IsFunctionTagOfType(chainMessage, spectypes.FUNCTION_TAG_SUBSCRIBE) {
			if IsFunctionTagOfType(chainMessage, spectypes.FUNCTION_TAG_UNSUBSCRIBE) {
				err := cwm.consumerWsSubscriptionManager.Unsubscribe(webSocketCtx, chainMessage, directiveHeaders, relayRequestData, dappID, userIp, metricsData)
				if err != nil {
					utils.LavaFormatWarning("error unsubscribing from subscription", err, utils.LogAttr("GUID", webSocketCtx))
					if err == common.SubscriptionNotFoundError {
						msgData, err := gojson.Marshal(common.JsonRpcSubscriptionNotFoundError)
						if err != nil {
							continue
						}

						websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: msgData}
					}
				}
				continue
			} else if IsFunctionTagOfType(chainMessage, spectypes.FUNCTION_TAG_UNSUBSCRIBE_ALL) {
				err := cwm.consumerWsSubscriptionManager.UnsubscribeAll(webSocketCtx, dappID, userIp, metricsData)
				if err != nil {
					utils.LavaFormatWarning("error unsubscribing from all subscription", err, utils.LogAttr("GUID", webSocketCtx))
				}
				continue
			} else {
				// Normal relay over websocket. (not subscription related)
				relayResult, err := cwm.relaySender.SendParsedRelay(webSocketCtx, dappID, userIp, metricsData, chainMessage, directiveHeaders, relayRequestData)
				if err != nil {
					formatterMsg := logger.AnalyzeWebSocketErrorAndGetFormattedMessage(websocketConn.LocalAddr().String(), utils.LavaFormatError("could not send parsed relay", err), msgSeed, msg, cwm.apiInterface, time.Since(startTime))
					if formatterMsg != nil {
						websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: formatterMsg}
						continue
					}
				}

				relayResultReply := relayResult.GetReply()
				if relayResultReply != nil {
					// No need to verify signature since this is already happening inside the SendParsedRelay flow
					websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: relayResult.GetReply().Data}
					continue
				}
				utils.LavaFormatError("Relay result is nil over websocket normal request flow, should not happen", err, utils.LogAttr("messageType", messageType))
			}
		}

		// Subscription flow
		inputFormatter, outputFormatter := formatter.FormatterForRelayRequestAndResponse(relayRequestData.ApiInterface) // we use this to preserve the original jsonrpc id
		inputFormatter(relayRequestData.Data)                                                                           // set the extracted jsonrpc id

		reply, subscriptionMsgsChan, err := cwm.consumerWsSubscriptionManager.StartSubscription(webSocketCtx, chainMessage, directiveHeaders, relayRequestData, dappID, userIp, metricsData)
		if err != nil {
			utils.LavaFormatWarning("StartSubscription returned an error", err,
				utils.LogAttr("GUID", webSocketCtx),
				utils.LogAttr("dappID", dappID),
				utils.LogAttr("userIp", userIp),
				utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
			)

			formatterMsg := logger.AnalyzeWebSocketErrorAndGetFormattedMessage(websocketConn.LocalAddr().String(), utils.LavaFormatError("could not start subscription", err), msgSeed, msg, cwm.apiInterface, time.Since(startTime))
			if formatterMsg != nil {
				websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: formatterMsg} // No need to use outputFormatter here since we are sending an error
				continue
			}

			// Handle the case when the error is a method not found error
			if common.APINotSupportedError.Is(err) {
				msgData, err := gojson.Marshal(common.JsonRpcMethodNotFoundError)
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
					utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
				)

				for subscriptionMsgReply := range subscriptionMsgsChan {
					websocketConnWriteChan <- webSocketMsgWithType{messageType: messageType, msg: outputFormatter(subscriptionMsgReply.Data)}
				}

				utils.LavaFormatTrace("subscriptionMsgsChan was closed",
					utils.LogAttr("GUID", webSocketCtx),
					utils.LogAttr("dappID", dappID),
					utils.LogAttr("userIp", userIp),
					utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
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
