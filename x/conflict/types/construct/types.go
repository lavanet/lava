package construct

import (
	"github.com/lavanet/lava/utils/sigs"
	"github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

func ConstructReplyMetadata(reply *pairingtypes.RelayReply, req *pairingtypes.RelayRequest) *types.ReplyMetadata {
	if reply == nil || req == nil {
		return nil
	}

	relayExchange := pairingtypes.NewRelayExchange(*req, *reply)

	allDataHash := relayExchange.DataToSign()
	for i := 0; i < relayExchange.HashRounds(); i++ {
		allDataHash = sigs.HashMsg(allDataHash)
	}

	res := &types.ReplyMetadata{
		HashAllDataHash:       allDataHash,
		Sig:                   reply.Sig,
		LatestBlock:           reply.LatestBlock,
		FinalizedBlocksHashes: reply.FinalizedBlocksHashes,
		SigBlocks:             reply.SigBlocks,
	}
	return res
}

func ConstructConflictRelayData(reply *pairingtypes.RelayReply, req *pairingtypes.RelayRequest) *types.ConflictRelayData {
	return &types.ConflictRelayData{
		Request: req,
		Reply:   ConstructReplyMetadata(reply, req),
	}
}
