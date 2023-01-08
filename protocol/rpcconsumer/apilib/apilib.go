package apilib

import (
	"context"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/relayer/lavasession"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

func NewApiParser(apiInterface string) (apiParser APIParser, err error) {
	switch apiInterface {
	case spectypes.APIInterfaceJsonRPC:
		return NewJrpcAPIParser()
	case spectypes.APIInterfaceTendermintRPC:
		return NewTendermintRpcAPIParser()
	case spectypes.APIInterfaceRest:
		return NewRestAPIParser()
	case spectypes.APIInterfaceGrpc:
		return NewGrpcAPIParser()
	}
	return nil, fmt.Errorf("apiParser for apiInterface (%s) not found", apiInterface)
}

func NewApiListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, apiPArser APIParser, relaySender RelaySender) (APIListener, error) {
	switch listenEndpoint.ApiInterface {
	case spectypes.APIInterfaceJsonRPC:
		return NewJrpcAPIListener(), nil
	case spectypes.APIInterfaceTendermintRPC:
		return NewTendermintRpcAPIListener(), nil
	case spectypes.APIInterfaceRest:
		return NewRestAPIListener(), nil
	case spectypes.APIInterfaceGrpc:
		return NewGrpcAPIListener(), nil
	}
	return nil, fmt.Errorf("apiListener for apiInterface (%s) not found", listenEndpoint.ApiInterface)
}

// this is an interface for parsing and generating messages of the supported APIType
// it checks for the existence of the method in the spec, and formats the message
type APIParser interface {
	ParseMsg(url string, data []byte, connectionType string) (APIMessage, error)
	SetSpec(spec spectypes.Spec)
}

type APIMessage interface {
	GetServiceApi() *spectypes.ServiceApi
	GetInterface() *spectypes.ApiInterface
	RequestedBlock() int64
}

type RelaySender interface {
	SendRelay(
		ctx context.Context,
		privKey *btcec.PrivateKey,
		url string,
		req string,
		connectionType string,
		dappID string,
	) (*pairingtypes.RelayReply, *pairingtypes.Relayer_RelaySubscribeClient, error)
}

type APIListener interface {
	Serve()
}
