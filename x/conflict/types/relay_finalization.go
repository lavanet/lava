package types

import (
	"bytes"
	"encoding/json"
	fmt "fmt"

	tendermintcrypto "github.com/cometbft/cometbft/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/sigs"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
)

func NewRelayFinalizationFromReplyMetadataAndRelayRequest(reply ReplyMetadata, req pairingtypes.RelayRequest, consumerAddr sdk.AccAddress) RelayFinalization {
	return RelayFinalization{
		FinalizedBlocksHashes: reply.FinalizedBlocksHashes,
		LatestBlock:           reply.LatestBlock,
		ConsumerAddress:       consumerAddr.String(),
		RelaySession:          req.RelaySession,
		SigBlocks:             reply.SigBlocks,
	}
}

func NewRelayFinalizationFromRelaySessionAndRelayReply(relaySession *pairingtypes.RelaySession, relayReply *pairingtypes.RelayReply, consumerAddr sdk.AccAddress) RelayFinalization {
	return RelayFinalization{
		FinalizedBlocksHashes: relayReply.FinalizedBlocksHashes,
		LatestBlock:           relayReply.LatestBlock,
		ConsumerAddress:       consumerAddr.String(),
		RelaySession:          relaySession,
		SigBlocks:             relayReply.SigBlocks,
	}
}

func (rf RelayFinalization) GetSignature() []byte {
	return rf.SigBlocks
}

func (rf RelayFinalization) DataToSign() []byte {
	if rf.RelaySession == nil {
		return nil
	}

	// This is for backward compatibility with old relay finalization messages
	sdkAccAddress, err := sdk.AccAddressFromBech32(rf.ConsumerAddress)
	if err != nil {
		utils.LavaFormatError("failed to convert consumer address to sdk.AccAddress", err)
		return nil
	}

	relaySessionHash := tendermintcrypto.Sha256(rf.RelaySession.CalculateHashForFinalization())
	latestBlockBytes := sigs.EncodeUint64(uint64(rf.LatestBlock))
	msgParts := [][]byte{
		latestBlockBytes,
		rf.FinalizedBlocksHashes,
		sdkAccAddress,
		relaySessionHash,
	}
	return bytes.Join(msgParts, nil)
}

func (rfm RelayFinalization) HashRounds() int {
	return 1
}

func (rf RelayFinalization) Stringify() string {
	consumerAddr := sdk.AccAddress{}
	consumerAddr.Unmarshal([]byte(rf.ConsumerAddress))
	consumerAddrStr := consumerAddr.String()

	finalizedBlocks := map[int64]string{}
	json.Unmarshal(rf.FinalizedBlocksHashes, &finalizedBlocks)

	return fmt.Sprintf("{latestBlock: %v consumerAddress: %v finalizedBlocksHashes: %v relaySessionHash: %v}",
		utils.StrValue(rf.LatestBlock),
		consumerAddrStr,
		utils.StrValue(finalizedBlocks),
		utils.StrValue(rf.RelaySession.CalculateHashForFinalization()),
	)
}
