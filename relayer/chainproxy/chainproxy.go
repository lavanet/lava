package chainproxy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/relayer/chainproxy/rpcclient"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/performance"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	DefaultTimeout            = 5 * time.Second
	ContextUserValueKeyDappID = "dappID"
)

type NodeMessage interface {
	GetServiceApi() *spectypes.ServiceApi
	Send(ctx context.Context, ch chan interface{}) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error)
	RequestedBlock() int64
	GetMsg() interface{}
}

type ChainProxy interface {
	Start(context.Context) error
	GetSentry() *sentry.Sentry
	ParseMsg(string, []byte, string) (NodeMessage, error)
	PortalStart(context.Context, *btcec.PrivateKey, string)
	FetchLatestBlockNum(ctx context.Context) (int64, error)
	FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error)
	GetConsumerSessionManager() *lavasession.ConsumerSessionManager
	SetCache(*performance.Cache)
	GetCache() *performance.Cache
}

func GetChainProxy(nodeUrl string, nConns uint, sentry *sentry.Sentry, pLogs *PortalLogs) (ChainProxy, error) {
	consumerSessionManagerInstance := &lavasession.ConsumerSessionManager{}
	switch sentry.ApiInterface {
	case "jsonrpc":
		return NewJrpcChainProxy(nodeUrl, nConns, sentry, consumerSessionManagerInstance, pLogs), nil
	case "tendermintrpc":
		return NewtendermintRpcChainProxy(nodeUrl, nConns, sentry, consumerSessionManagerInstance, pLogs), nil
	case "rest":
		return NewRestChainProxy(nodeUrl, sentry, consumerSessionManagerInstance, pLogs), nil
	}
	return nil, fmt.Errorf("chain proxy for apiInterface (%s) not found", sentry.ApiInterface)
}

func VerifyRelayReply(reply *pairingtypes.RelayReply, relayRequest *pairingtypes.RelayRequest, addr string, comparesHashes bool) error {

	serverKey, err := sigs.RecoverPubKeyFromRelayReply(reply, relayRequest)
	if err != nil {
		return err
	}
	serverAddr, err := sdk.AccAddressFromHex(serverKey.Address().String())
	if err != nil {
		return err
	}
	if serverAddr.String() != addr {
		return fmt.Errorf("server address mismatch in reply (%s) (%s)", serverAddr.String(), addr)
	}

	if comparesHashes {
		strAdd, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			return err
		}
		serverKey, err = sigs.RecoverPubKeyFromResponseFinalizationData(reply, relayRequest, strAdd)
		if err != nil {
			return err
		}

		serverAddr, err = sdk.AccAddressFromHex(serverKey.Address().String())
		if err != nil {
			return err
		}

		if serverAddr.String() != strAdd.String() {
			return fmt.Errorf("server address mismatch in reply sigblocks (%s) (%s)", serverAddr.String(), strAdd.String())
		}
	}
	return nil
}

// Client requests and queries
func SendRelay(
	ctx context.Context,
	cp ChainProxy,
	privKey *btcec.PrivateKey,
	url string,
	req string,
	connectionType string,
	dappID string,
) (*pairingtypes.RelayReply, *pairingtypes.Relayer_RelaySubscribeClient, error) {

	// Unmarshal request
	nodeMsg, err := cp.ParseMsg(url, []byte(req), connectionType)
	if err != nil {
		return nil, nil, err
	}
	isSubscription := nodeMsg.GetServiceApi().Category.Subscription
	blockHeight := int64(-1) //to sync reliability blockHeight in case it changes
	requestedBlock := int64(0)

	// Get Session. we get session here so we can use the epoch in the callbacks
	singleConsumerSession, epoch, providerPublicAddress, reportedProviders, err := cp.GetConsumerSessionManager().GetSession(ctx, nodeMsg.GetServiceApi().ComputeUnits, nil)
	if err != nil {
		return nil, nil, err
	}
	// consumerSession is locked here.

	callback_send_relay := func(consumerSession *lavasession.SingleConsumerSession) (*pairingtypes.RelayReply, *pairingtypes.Relayer_RelaySubscribeClient, *pairingtypes.RelayRequest, time.Duration, bool, error) {
		//client session is locked here
		fromCache := false
		blockHeight = int64(epoch) // epochs heights only

		// we need to apply CuSum and relay number that we plan to add in  the relay request. even if we didn't yet apply them to the consumerSession.
		relayRequest := &pairingtypes.RelayRequest{
			Provider:              providerPublicAddress,
			ConnectionType:        connectionType,
			ApiUrl:                url,
			Data:                  []byte(req),
			SessionId:             uint64(consumerSession.SessionId),
			ChainID:               cp.GetSentry().ChainID,
			CuSum:                 consumerSession.CuSum + consumerSession.LatestRelayCu, // add the latestRelayCu which will be applied when session is returned properly
			BlockHeight:           blockHeight,
			RelayNum:              consumerSession.RelayNum + lavasession.RelayNumberIncrement, // increment the relay number. which will be applied when session is returned properly
			RequestBlock:          nodeMsg.RequestedBlock(),
			QoSReport:             consumerSession.QoSInfo.LastQoSReport,
			DataReliability:       nil,
			UnresponsiveProviders: reportedProviders,
		}
		sig, err := sigs.SignRelay(privKey, *relayRequest)
		if err != nil {
			return nil, nil, nil, 0, fromCache, err
		}
		relayRequest.Sig = sig
		c := *consumerSession.Endpoint.Client

		connectCtx, cancel := context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()

		var replyServer pairingtypes.Relayer_RelaySubscribeClient
		var reply *pairingtypes.RelayReply

		relaySentTime := time.Now()
		if isSubscription {
			replyServer, err = c.RelaySubscribe(ctx, relayRequest)
		} else {
			cache := cp.GetCache()
			reply, err = cache.GetEntry(ctx, relayRequest, cp.GetSentry().ApiInterface, nil, cp.GetSentry().ChainID, false) // caching in the portal doesn't care about hashes, and we don't have data on finalization yet
			if err != nil || reply == nil {
				if performance.NotConnectedError.Is(err) {
					utils.LavaFormatError("cache not connected", err, nil)
				}
				reply, err = c.Relay(connectCtx, relayRequest)
			} else {
				// Info was fetched from cache, so we need to change the state
				fromCache = true
				// so we can return here, no need to update anything and calculate as this info was fetched from the cache
				return reply, nil, relayRequest, 0, fromCache, nil
			}
		}
		currentLatency := time.Since(relaySentTime)
		if err != nil {
			return nil, nil, nil, 0, fromCache, err
		}

		if !isSubscription {
			//update relay request requestedBlock to the provided one in case it was arbitrary
			sentry.UpdateRequestedBlock(relayRequest, reply)
			finalized := cp.GetSentry().IsFinalizedBlock(relayRequest.RequestBlock, reply.LatestBlock)
			err = VerifyRelayReply(reply, relayRequest, providerPublicAddress, cp.GetSentry().GetSpecComparesHashes())
			if err != nil {
				return nil, nil, nil, 0, fromCache, err
			}
			cache := cp.GetCache()
			// TODO: response sanity, check its under an expected format add that format to spec
			cache.SetEntry(ctx, relayRequest, cp.GetSentry().ApiInterface, nil, cp.GetSentry().ChainID, dappID, reply, finalized) // caching in the portal doesn't care about hashes
			return reply, nil, relayRequest, currentLatency, fromCache, nil
		}
		// isSubscription
		return reply, &replyServer, relayRequest, currentLatency, fromCache, nil
	}

	callback_send_reliability := func(consumerSession *lavasession.SingleConsumerSession, dataReliability *pairingtypes.VRFData, providerAddress string) (*pairingtypes.RelayReply, *pairingtypes.RelayRequest, time.Duration, error) {
		//client session is locked here
		sentry := cp.GetSentry()
		if blockHeight < 0 {
			return nil, nil, 0, fmt.Errorf("expected callback_send_relay to be called first and set blockHeight")
		}

		relayRequest := &pairingtypes.RelayRequest{
			Provider:              providerAddress,
			ApiUrl:                url,
			Data:                  []byte(req),
			SessionId:             lavasession.DataReliabilitySessionId, //sessionID for reliability is 0
			ChainID:               sentry.ChainID,
			CuSum:                 lavasession.DataReliabilityCuSum, // consumerSession.CuSum == 0
			BlockHeight:           blockHeight,
			RelayNum:              0, // consumerSession.RelayNum == 0
			RequestBlock:          requestedBlock,
			QoSReport:             nil,
			DataReliability:       dataReliability,
			ConnectionType:        connectionType,
			UnresponsiveProviders: reportedProviders,
		}

		sig, err := sigs.SignRelay(privKey, *relayRequest)
		if err != nil {
			return nil, nil, 0, err
		}
		relayRequest.Sig = sig

		sig, err = sigs.SignVRFData(privKey, relayRequest.DataReliability)
		if err != nil {
			return nil, nil, 0, err
		}
		relayRequest.DataReliability.Sig = sig
		c := *consumerSession.Endpoint.Client
		relaySentTime := time.Now()
		reply, err := c.Relay(ctx, relayRequest)
		if err != nil {
			return nil, nil, 0, err
		}
		currentLatency := time.Since(relaySentTime)
		err = VerifyRelayReply(reply, relayRequest, providerAddress, cp.GetSentry().GetSpecComparesHashes())
		if err != nil {
			return nil, nil, 0, err
		}

		return reply, relayRequest, currentLatency, nil
	}

	reply, replyServer, relayLatency, isCachedResult, firstSessionError := cp.GetSentry().SendRelay(ctx, singleConsumerSession, epoch, providerPublicAddress, callback_send_relay, callback_send_reliability, nodeMsg.GetServiceApi().Category)
	if firstSessionError != nil {
		// on session failure here
		errReport := cp.GetConsumerSessionManager().OnSessionFailure(singleConsumerSession, firstSessionError)
		if errReport != nil {
			return nil, nil, fmt.Errorf("original error: %v, onSessionFailure: %v", firstSessionError, errReport)
		}
		if lavasession.SendRelayError.Is(firstSessionError) {
			// Retry
			originalProviderAddress := providerPublicAddress
			singleConsumerSession, epoch, providerPublicAddress, reportedProviders, err = cp.GetConsumerSessionManager().GetSessionFromAllExcept(ctx, map[string]struct{}{providerPublicAddress: {}}, nodeMsg.GetServiceApi().ComputeUnits, epoch)
			if err != nil {
				return nil, nil, utils.LavaFormatError("relay_retry_attempt - Failed to get a second session from a different provider", nil, &map[string]string{"Original Error": firstSessionError.Error(), "GetSessionFromAllExcept Error": err.Error(), "ChainID": cp.GetSentry().ChainID, "Original_Provider_Address": originalProviderAddress})
			}
			var secondSessionError error
			reply, replyServer, relayLatency, isCachedResult, secondSessionError = cp.GetSentry().SendRelay(ctx, singleConsumerSession, epoch, providerPublicAddress, callback_send_relay, callback_send_reliability, nodeMsg.GetServiceApi().Category)
			if secondSessionError != nil {
				errReport = cp.GetConsumerSessionManager().OnSessionFailure(singleConsumerSession, secondSessionError)
				if errReport != nil {
					return nil, nil, fmt.Errorf("original error: %v, onSessionFailure: %v", firstSessionError, errReport)
				}
				// compare error1 with error2
				if secondSessionError.Error() != firstSessionError.Error() {
					return nil, nil, utils.LavaFormatError("relay_retry_attempt - Received two different errors from different providers", nil, &map[string]string{"firstSessionError": firstSessionError.Error(), "secondSessionError": secondSessionError.Error(), "firstProviderAddr": originalProviderAddress, "secondProviderAddr": providerPublicAddress})
				} else {
					// if both errors are the same, just return the first error.
					return nil, nil, firstSessionError
				}
			}
			// retry attempt succeeded! can continue normally
		} else {
			return nil, nil, firstSessionError
		}
	}
	if !isSubscription {
		if isCachedResult {
			err = cp.GetConsumerSessionManager().OnSessionUnUsed(singleConsumerSession)
			return reply, replyServer, err
		}
		latestBlock := reply.LatestBlock
		expectedBH, numOfProviders := cp.GetSentry().ExpectedBlockHeight()
		err = cp.GetConsumerSessionManager().OnSessionDone(singleConsumerSession, epoch, latestBlock, nodeMsg.GetServiceApi().ComputeUnits, relayLatency, expectedBH, numOfProviders, cp.GetSentry().GetProvidersCount()) // session done successfully
	} else {
		err = cp.GetConsumerSessionManager().OnSessionDoneIncreaseRelayAndCu(singleConsumerSession) // session done successfully
	}
	if reply.Data == nil && err == nil {
		return nil, nil, utils.LavaFormatError("invalid handling of an error reply Data is nil & error is nil", nil, nil)
	}
	return reply, replyServer, err
}

func ConstructFiberCallbackWithDappIDExtraction(callbackToBeCalled fiber.Handler) fiber.Handler {
	webSocketCallback := callbackToBeCalled
	handler := func(c *fiber.Ctx) error {
		dappID := ""
		if len(c.Route().Params) > 1 {
			dappID = c.Route().Params[1]
			dappID = strings.Replace(dappID, "*", "", -1)
		}
		c.Context().SetUserValue(ContextUserValueKeyDappID, dappID) //this sets a user value in context and this is given to the callback
		return webSocketCallback(c)                                 //uses external dappID
	}
	return handler
}
func ExtractDappIDFromWebsocketConnection(c *websocket.Conn) string {
	dappIDLocal := c.Locals(ContextUserValueKeyDappID)
	if dappID, ok := dappIDLocal.(string); ok {
		//zeroallocation policy for fiber.Ctx
		buffer := make([]byte, len(dappID))
		copy(buffer, dappID)
		return string(buffer)
	}
	return "NoDappID"
}

func ExtractDappIDFromFiberContext(c *fiber.Ctx) (dappID string) {
	if len(c.Route().Params) > 1 {
		dappID = c.Route().Params[1]
		dappID = strings.Replace(dappID, "*", "", -1)
		return
	}
	return "NoDappID"
}
