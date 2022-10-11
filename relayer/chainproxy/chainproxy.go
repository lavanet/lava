package chainproxy

import (
	"context"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	Send(ctx context.Context) (*pairingtypes.RelayReply, error)
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
}

func GetChainProxy(nodeUrl string, nConns uint, sentry *sentry.Sentry) (ChainProxy, error) {
	switch sentry.ApiInterface {
	case "jsonrpc":
		return NewJrpcChainProxy(nodeUrl, nConns, sentry), nil
	case "tendermintrpc":
		return NewtendermintRpcChainProxy(nodeUrl, nConns, sentry), nil
	case "rest":
		return NewRestChainProxy(nodeUrl, sentry), nil
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
) (*pairingtypes.RelayReply, error) {

	//
	// Unmarshal request
	nodeMsg, err := cp.ParseMsg(url, []byte(req), connectionType)
	if err != nil {
		return nil, err
	}
	blockHeight := int64(-1) //to sync reliability blockHeight in case it changes
	requestedBlock := int64(0)
	callback_send_relay := func(clientSession *sentry.ClientSession, unresponsiveProviders []byte) (*pairingtypes.RelayReply, *pairingtypes.RelayRequest, error) {
		//client session is locked here
		serviceApiComputeUnits := nodeMsg.GetServiceApi().ComputeUnits
		err := VerifyComputeUnits(clientSession, serviceApiComputeUnits)
		if err != nil {
			return nil, nil, err
		}
		blockHeight = int64(clientSession.Client.GetPairingEpoch()) // epochs heights only

		relayRequest := &pairingtypes.RelayRequest{
			Provider:              clientSession.Client.Acc,
			ConnectionType:        connectionType,
			ApiUrl:                url,
			Data:                  []byte(req),
			SessionId:             uint64(clientSession.SessionId),
			ChainID:               cp.GetSentry().ChainID,
			CuSum:                 clientSession.CuSum + serviceApiComputeUnits, // we increase the cusum by the compute units of the service.
			BlockHeight:           blockHeight,
			RelayNum:              clientSession.RelayNum + 1, // we increase the relayNum by 1.
			RequestBlock:          nodeMsg.RequestedBlock(),
			QoSReport:             clientSession.QoSInfo.LastQoSReport,
			DataReliability:       nil,
			UnresponsiveProviders: unresponsiveProviders,
		}

		sig, err := sigs.SignRelay(privKey, *relayRequest)
		if err != nil {
			return nil, nil, err
		}
		relayRequest.Sig = sig
		c := *clientSession.Endpoint.Client

		relaySentTime := time.Now()
		clientSession.QoSInfo.TotalRelays++
		connectCtx, cancel := context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()

		reply, err := c.Relay(connectCtx, relayRequest)

		if err != nil {
			if err.Error() == context.DeadlineExceeded.Error() {
				clientSession.QoSInfo.ConsecutiveTimeOut++
			}
			return nil, nil, err
		}

		// After reply was received without an error and we got the message successfully, we can increase the CU.
		ApplyComputeUnits(clientSession, serviceApiComputeUnits)

		currentLatency := time.Since(relaySentTime)
		clientSession.QoSInfo.ConsecutiveTimeOut = 0
		clientSession.QoSInfo.AnsweredRelays++

		//update relay request requestedBlock to the provided one in case it was arbitrary
		sentry.UpdateRequestedBlock(relayRequest, reply)
		requestedBlock = relayRequest.RequestBlock

		err = VerifyRelayReply(reply, relayRequest, clientSession.Client.Acc, cp.GetSentry().GetSpecComparesHashes())
		if err != nil {
			return nil, nil, err
		}

		expectedBH, numOfProviders := cp.GetSentry().ExpecedBlockHeight()
		clientSession.CalculateQoS(serviceApiComputeUnits, currentLatency, expectedBH-reply.LatestBlock, numOfProviders, int64(cp.GetSentry().GetProvidersCount()))

		return reply, relayRequest, nil
	}
	callback_send_reliability := func(clientSession *sentry.ClientSession, dataReliability *pairingtypes.VRFData, unresponsiveProviders []byte) (*pairingtypes.RelayReply, *pairingtypes.RelayRequest, error) {
		//client session is locked here
		sentry := cp.GetSentry()
		if blockHeight < 0 {
			return nil, nil, fmt.Errorf("expected callback_send_relay to be called first and set blockHeight")
		}

		relayRequest := &pairingtypes.RelayRequest{
			Provider:              clientSession.Client.Acc,
			ApiUrl:                url,
			Data:                  []byte(req),
			SessionId:             uint64(0), //sessionID for reliability is 0
			ChainID:               sentry.ChainID,
			CuSum:                 clientSession.CuSum,
			BlockHeight:           blockHeight,
			RelayNum:              clientSession.RelayNum,
			RequestBlock:          requestedBlock,
			QoSReport:             nil,
			DataReliability:       dataReliability,
			ConnectionType:        connectionType,
			UnresponsiveProviders: unresponsiveProviders,
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
		c := *clientSession.Endpoint.Client
		reply, err := c.Relay(ctx, relayRequest)
		if err != nil {
			return nil, nil, err
		}

		err = VerifyRelayReply(reply, relayRequest, clientSession.Client.Acc, cp.GetSentry().GetSpecComparesHashes())
		if err != nil {
			return nil, nil, err
		}

		return reply, relayRequest, nil
	}
	//
	//
	reply, err := cp.GetSentry().SendRelay(ctx, callback_send_relay, callback_send_reliability, nodeMsg.GetServiceApi().Category)

	return reply, err
}

// After a successful reply we can apply the cu.
func ApplyComputeUnits(clientSession *sentry.ClientSession, apiCu uint64) {
	clientSession.Client.SessionsLock.Lock()
	defer clientSession.Client.SessionsLock.Unlock()

	clientSession.CuSum += apiCu
	clientSession.Client.UsedComputeUnits += apiCu
	clientSession.RelayNum += 1

	return
}

// Verify we have enough CU to send the request
func VerifyComputeUnits(clientSession *sentry.ClientSession, apiCu uint64) error {
	clientSession.Client.SessionsLock.Lock()
	defer clientSession.Client.SessionsLock.Unlock()

	if clientSession.Client.UsedComputeUnits+apiCu > clientSession.Client.MaxComputeUnits {
		return fmt.Errorf("used all the available compute units")
	}
	return nil
}
