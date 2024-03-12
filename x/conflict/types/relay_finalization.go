package types

import (
	tendermintcrypto "github.com/cometbft/cometbft/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

func NewRelayFinalization(relaySession *pairingtypes.RelaySession, relayReply *pairingtypes.RelayReply, addr sdk.AccAddress, blockDistanceToFinalization int64) RelayFinalization {
	return RelayFinalization{
		RelaySessionHash:            tendermintcrypto.Sha256(relaySession.CalculateHashForFinalization()),
		FinalizedBlocksHashes:       relayReply.FinalizedBlocksHashes,
		LatestBlock:                 relayReply.LatestBlock,
		Sig:                         relayReply.SigBlocks,
		ConsumerAddress:             string(addr.Bytes()),
		BlockDistanceToFinalization: blockDistanceToFinalization,
		SpecId:                      relaySession.SpecId,
		Epoch:                       relaySession.Epoch,
	}
}

func (rf RelayFinalization) GetSignature() []byte {
	return rf.Sig
}

func (rf RelayFinalization) DataToSign() []byte {
	latestBlockBytes := sigs.EncodeUint64(uint64(rf.LatestBlock))
	blockDistanceToFinalizationBytes := sigs.EncodeUint64(uint64(rf.BlockDistanceToFinalization))
	epochBytes := sigs.EncodeUint64(uint64(rf.Epoch))
	msgParts := [][]byte{
		latestBlockBytes,
		rf.FinalizedBlocksHashes,
		[]byte(rf.ConsumerAddress),
		rf.RelaySessionHash,
		blockDistanceToFinalizationBytes,
		[]byte(rf.SpecId),
		epochBytes,
	}
	return sigs.Join(msgParts)
}

func (rf RelayFinalization) HashRounds() int {
	return 1
}
