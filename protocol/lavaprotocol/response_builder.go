package lavaprotocol

import (
	"context"
	"encoding/json"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol/protocolerrors"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/sigs"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
)

func CraftEmptyRPCResponseFromGenericMessage(message rpcInterfaceMessages.GenericMessage) (*rpcInterfaceMessages.RPCResponse, error) {
	createRPCResponse := func(rawId json.RawMessage) (*rpcInterfaceMessages.RPCResponse, error) {
		jsonRpcId, err := rpcInterfaceMessages.IdFromRawMessage(rawId)
		if err != nil {
			return nil, utils.LavaFormatError("failed creating jsonrpc id", err)
		}

		jsonResponse := &rpcInterfaceMessages.RPCResponse{
			JSONRPC: "2.0",
			ID:      jsonRpcId,
			Result:  nil,
			Error:   nil,
		}

		return jsonResponse, nil
	}

	var err error
	var rpcResponse *rpcInterfaceMessages.RPCResponse
	if hasID, ok := message.(interface{ GetID() json.RawMessage }); ok {
		rpcResponse, err = createRPCResponse(hasID.GetID())
		if err != nil {
			return nil, utils.LavaFormatError("failed creating jsonrpc id", err)
		}
	} else {
		rpcResponse, err = createRPCResponse([]byte("1"))
		if err != nil {
			return nil, utils.LavaFormatError("failed creating jsonrpc id", err)
		}
	}

	return rpcResponse, nil
}

func SignRelayResponse(consumerAddress sdk.AccAddress, request pairingtypes.RelayRequest, pkey *btcSecp256k1.PrivateKey, reply *pairingtypes.RelayReply) (*pairingtypes.RelayReply, error) {
	// request is a copy of the original request, but won't modify it
	// update relay request requestedBlock to the provided one in case it was arbitrary
	UpdateRequestedBlock(request.RelayData, reply)

	// Skip signing when configured (e.g., smart router mode) to save CPU/memory
	if SkipRelaySigning {
		return reply, nil
	}

	// Update signature,
	relayExchange := pairingtypes.NewRelayExchange(request, *reply)
	sig, err := sigs.Sign(pkey, relayExchange)
	if err != nil {
		return nil, utils.LavaFormatError("failed signing relay response", err,
			utils.LogAttr("request", request),
			utils.LogAttr("reply", reply),
		)
	}
	reply.Sig = sig

	return reply, nil
}

func VerifyRelayReply(ctx context.Context, reply *pairingtypes.RelayReply, relayRequest *pairingtypes.RelayRequest, addr string) error {
	// Skip verification when signing is disabled (no signature to verify)
	if SkipRelaySigning {
		return nil
	}

	relayExchange := pairingtypes.NewRelayExchange(*relayRequest, *reply)
	serverKey, err := sigs.RecoverPubKey(relayExchange)
	if err != nil {
		return utils.LavaFormatWarning("Relay reply verification failed, RecoverPubKey returned error", err, utils.LogAttr("GUID", ctx))
	}
	serverAddr, err := sdk.AccAddressFromHexUnsafe(serverKey.Address().String())
	if err != nil {
		return utils.LavaFormatWarning("Relay reply verification failed, AccAddressFromHexUnsafe returned error", err, utils.LogAttr("GUID", ctx))
	}
	if serverAddr.String() != addr {
		return utils.LavaFormatError("reply server address mismatch", protocolerrors.ProviderFinalizationDataError,
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("parsedAddress", serverAddr.String()),
			utils.LogAttr("expectedAddress", addr),
			utils.LogAttr("requestedBlock", relayRequest.RelayData.RequestBlock),
			utils.LogAttr("latestBlock", reply.GetLatestBlock()),
		)
	}

	return nil
}
