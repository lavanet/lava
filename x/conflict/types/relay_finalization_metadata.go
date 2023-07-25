package types

import (
	"bytes"
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
	latestBlockBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(latestBlockBytes, uint64(rfm.MetaData.LatestBlock))
	return bytes.Join([][]byte{latestBlockBytes, rfm.MetaData.FinalizedBlocksHashes, rfm.Addr, relaySessionHash}, nil)
}

func (rfm RelayFinalizationMetaData) HashCount() int {
	return 1
}
