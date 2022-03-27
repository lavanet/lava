package chainproxy

import (
	"context"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/relayer/sigs"
	servicertypes "github.com/lavanet/lava/x/servicer/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type NodeMessage interface {
	GetServiceApi() *spectypes.ServiceApi
	Send(ctx context.Context) (*servicertypes.RelayReply, error)
}

type ChainProxy interface {
	Start(context.Context) error
	GetSentry() *sentry.Sentry
	ParseMsg([]byte) (NodeMessage, error)
	PortalStart(context.Context, *btcec.PrivateKey, string)
}

func GetChainProxy(specId uint64, nodeUrl string, nConns uint, sentry *sentry.Sentry) (ChainProxy, error) {
	switch specId {
	case 0, 1:
		return NewEthereumChainProxy(nodeUrl, nConns, sentry), nil
	case 2:
		return NewCosmosChainProxy(nodeUrl, sentry), nil
	}
	return nil, fmt.Errorf("chain proxy for chain id (%d) not found", specId)
}

func SendRelay(
	ctx context.Context,
	cp ChainProxy,
	privKey *btcec.PrivateKey,
	req string,
) (*servicertypes.RelayReply, error) {

	//
	// Unmarshal request
	nodeMsg, err := cp.ParseMsg([]byte(req))
	if err != nil {
		return nil, err
	}

	//
	//
	reply, err := cp.GetSentry().SendRelay(ctx, func(clientSession *sentry.ClientSession) (*servicertypes.RelayReply, error) {
		clientSession.CuSum += nodeMsg.GetServiceApi().ComputeUnits

		relayRequest := &servicertypes.RelayRequest{
			Servicer:    clientSession.Client.Acc,
			Data:        []byte(req),
			SessionId:   uint64(clientSession.SessionId),
			SpecId:      uint32(cp.GetSentry().SpecId),
			CuSum:       clientSession.CuSum,
			BlockHeight: cp.GetSentry().GetBlockHeight(),
		}

		sig, err := sigs.SignRelay(privKey, []byte(relayRequest.String()))
		if err != nil {
			return nil, err
		}
		relayRequest.Sig = sig

		c := *clientSession.Client.Client
		reply, err := c.Relay(ctx, relayRequest)
		if err != nil {
			return nil, err
		}
		serverKey, err := sigs.RecoverPubKeyFromRelayReply(reply)
		if err != nil {
			return nil, err
		}
		serverAddr, err := sdk.AccAddressFromHex(serverKey.Address().String())
		if err != nil {
			return nil, err
		}
		if serverAddr.String() != clientSession.Client.Acc {
			return nil, fmt.Errorf("server address mismatch in reply (%s) (%s)", serverAddr.String(), clientSession.Client.Acc)
		}

		return reply, nil
	})

	return reply, err
}
