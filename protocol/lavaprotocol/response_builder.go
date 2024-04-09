package lavaprotocol

import (
	"context"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

func SignRelayResponse(consumerAddress sdk.AccAddress, request pairingtypes.RelayRequest, pkey *btcSecp256k1.PrivateKey, reply *pairingtypes.RelayReply, signDataReliability bool) (*pairingtypes.RelayReply, error) {
	// request is a copy of the original request, but won't modify it
	// update relay request requestedBlock to the provided one in case it was arbitrary
	UpdateRequestedBlock(request.RelayData, reply)
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

	if signDataReliability {
		// update sig blocks signature
		relayFinalization := conflicttypes.NewRelayFinalizationFromRelaySessionAndRelayReply(request.RelaySession, reply, consumerAddress)
		sigBlocks, err := sigs.Sign(pkey, relayFinalization)
		if err != nil {
			return nil, utils.LavaFormatError("failed signing finalization data", err,
				utils.LogAttr("request", request),
				utils.LogAttr("reply", reply),
				utils.LogAttr("userAddr", consumerAddress),
			)
		}
		reply.SigBlocks = sigBlocks
	}
	return reply, nil
}

func VerifyRelayReply(ctx context.Context, reply *pairingtypes.RelayReply, relayRequest *pairingtypes.RelayRequest, addr string) error {
	relayExchange := pairingtypes.NewRelayExchange(*relayRequest, *reply)
	serverKey, err := sigs.RecoverPubKey(relayExchange)
	if err != nil {
		return err
	}
	serverAddr, err := sdk.AccAddressFromHexUnsafe(serverKey.Address().String())
	if err != nil {
		return err
	}
	if serverAddr.String() != addr {
		return utils.LavaFormatError("reply server address mismatch ", ProviderFinalizationDataError,
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("parsed Address", serverAddr.String()),
			utils.LogAttr("expected address", addr),
			utils.LogAttr("requestedBlock", relayRequest.RelayData.RequestBlock),
			utils.LogAttr("latestBlock", reply.GetLatestBlock()),
		)
	}

	return nil
}
