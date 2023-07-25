package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
)

type RelayFinalizationMetaData struct {
	MetaData ReplyMetadata
	Request  pairingtypes.RelayRequest
	Addr     sdk.AccAddress
}

func NewRelayFinalizationMetaData(meta ReplyMetadata, req pairingtypes.RelayRequest, addr sdk.AccAddress) RelayFinalizationMetaData {
	return RelayFinalizationMetaData{MetaData: meta, Request: req, Addr: addr}
}

func (rfm RelayFinalizationMetaData) GetSignature() []byte {
	return rfm.MetaData.SigBlocks
}

func (rfm RelayFinalizationMetaData) DataToSign() []byte {
	relaySessionHash := tendermintcrypto.Sha256(rfm.Request.RelaySession.CalculateHashForFinalization())
	latestBlockBytes := sigs.EncodeUint64(uint64(rfm.MetaData.LatestBlock))
	msgParts := [][]byte{
		latestBlockBytes,
		rfm.MetaData.FinalizedBlocksHashes,
		rfm.Addr,
		relaySessionHash,
	}
	return sigs.Join(msgParts)
}

func (rfm RelayFinalizationMetaData) HashRounds() int {
	return 1
}
