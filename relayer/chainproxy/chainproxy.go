package chainproxy

import (
	"context"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/chainproxy/rpcclient"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/relayer/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	DefaultTimeout = 5 * time.Second
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
}

func GetChainProxy(nodeUrl string, nConns uint, sentry *sentry.Sentry) (ChainProxy, error) {
	consumerSessionManagerInstance := &lavasession.ConsumerSessionManager{}
	switch sentry.ApiInterface {
	case "jsonrpc":
		return NewJrpcChainProxy(nodeUrl, nConns, sentry, consumerSessionManagerInstance), nil
	case "tendermintrpc":
		return NewtendermintRpcChainProxy(nodeUrl, nConns, sentry, consumerSessionManagerInstance), nil
	case "rest":
		return NewRestChainProxy(nodeUrl, sentry, consumerSessionManagerInstance), nil
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

	callback_send_relay := func(consumerSession *lavasession.SingleConsumerSession) (*pairingtypes.RelayReply, *pairingtypes.Relayer_RelaySubscribeClient, *pairingtypes.RelayRequest, time.Duration, error) {
		//client session is locked here
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
			return nil, nil, nil, 0, err
		}
		relayRequest.Sig = sig
		c := *consumerSession.Endpoint.Client

		relaySentTime := time.Now()
		connectCtx, cancel := context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()

		var replyServer pairingtypes.Relayer_RelaySubscribeClient
		var reply *pairingtypes.RelayReply

		if isSubscription {
			replyServer, err = c.RelaySubscribe(ctx, relayRequest)
		} else {
			reply, err = c.Relay(connectCtx, relayRequest)
		}
		currentLatency := time.Since(relaySentTime)
		if err != nil {
			return nil, nil, nil, 0, err
		}

		if !isSubscription {
			//update relay request requestedBlock to the provided one in case it was arbitrary
			sentry.UpdateRequestedBlock(relayRequest, reply)
			err = VerifyRelayReply(reply, relayRequest, providerPublicAddress, cp.GetSentry().GetSpecComparesHashes())
			if err != nil {
				return nil, nil, nil, 0, err
			}
			return reply, nil, relayRequest, currentLatency, nil
		}
		// isSubscription
		return reply, &replyServer, relayRequest, currentLatency, nil
	}

	callback_send_reliability := func(consumerSession *lavasession.SingleConsumerSession, dataReliability *pairingtypes.VRFData, providerAddress string) (*pairingtypes.RelayReply, *pairingtypes.RelayRequest, error) {
		//client session is locked here
		sentry := cp.GetSentry()
		if blockHeight < 0 {
			return nil, nil, fmt.Errorf("expected callback_send_relay to be called first and set blockHeight")
		}

		relayRequest := &pairingtypes.RelayRequest{
			Provider:              providerAddress,
			ApiUrl:                url,
			Data:                  []byte(req),
			SessionId:             uint64(0), //sessionID for reliability is 0
			ChainID:               sentry.ChainID,
			CuSum:                 consumerSession.CuSum,
			BlockHeight:           blockHeight,
			RelayNum:              consumerSession.RelayNum,
			RequestBlock:          requestedBlock,
			QoSReport:             nil,
			DataReliability:       dataReliability,
			ConnectionType:        connectionType,
			UnresponsiveProviders: reportedProviders,
		}

		sig, err := sigs.SignRelay(privKey, *relayRequest)
		if err != nil {
			return nil, nil, err
		}
		relayRequest.Sig = sig

		sig, err = sigs.SignVRFData(privKey, relayRequest.DataReliability)
		if err != nil {
			return nil, nil, err
		}
		relayRequest.DataReliability.Sig = sig
		c := *consumerSession.Endpoint.Client
		reply, err := c.Relay(ctx, relayRequest)
		if err != nil {
			return nil, nil, err
		}

		err = VerifyRelayReply(reply, relayRequest, providerAddress, cp.GetSentry().GetSpecComparesHashes())
		if err != nil {
			return nil, nil, err
		}

		return reply, relayRequest, nil
	}

	reply, replyServer, relayLatency, err := cp.GetSentry().SendRelay(ctx, singleConsumerSession, epoch, providerPublicAddress, callback_send_relay, callback_send_reliability, nodeMsg.GetServiceApi().Category)
	if err != nil {
		// on session failure here
		errReport := cp.GetConsumerSessionManager().OnSessionFailure(singleConsumerSession, err)
		if errReport != nil {
			return nil, nil, fmt.Errorf("original error: %v, onSessionFailure: %v", err, errReport)
		}
		if lavasession.SendRelayError.Is(err) {
			// TODO send again?
		}
		return nil, nil, err
	}
	if !isSubscription {
		latestBlock := reply.LatestBlock
		expectedBH, numOfProviders := cp.GetSentry().ExpectedBlockHeight()
		err = cp.GetConsumerSessionManager().OnSessionDone(singleConsumerSession, epoch, latestBlock, nodeMsg.GetServiceApi().ComputeUnits, relayLatency, expectedBH, numOfProviders, cp.GetSentry().GetProvidersCount()) // session done successfully
	} else {
		err = cp.GetConsumerSessionManager().OnSessionDoneWithoutQoSChanges(singleConsumerSession) // session done successfully
	}
	return reply, replyServer, err
}
