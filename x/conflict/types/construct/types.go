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
	allDataHash := sigs.HashMsg(relayExchange.DataToSign())
	res := &types.ReplyMetadata{
		HashAllDataHash:       sigs.HashMsg(allDataHash),
		Sig:                   reply.Sig,
		LatestBlock:           reply.LatestBlock,
		FinalizedBlocksHashes: reply.FinalizedBlocksHashes,
		SigBlocks:             reply.SigBlocks,
	}
	return res
}

func ConstructConflictRelayData(reply *pairingtypes.RelayReply, req *pairingtypes.RelayRequest) *types.ConflictRelayData {
	return &types.ConflictRelayData{Reply: ConstructReplyMetadata(reply, req), Request: req}
}
