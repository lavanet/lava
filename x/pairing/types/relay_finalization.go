package types

import (
	"bytes"
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
)

type RelayFinalization struct {
	Exchange RelayExchange
	Addr     sdk.AccAddress
}

func NewRelayFinalization(exch RelayExchange, addr sdk.AccAddress) RelayFinalization {
	return RelayFinalization{Exchange: exch, Addr: addr}
}

func (rf RelayFinalization) GetSignature() []byte {
	return rf.Exchange.GetSignature()
}

func (rf RelayFinalization) PrepareForSignature() []byte {
	relaySessionHash := tendermintcrypto.Sha256(rf.Exchange.Request.RelaySession.CalculateHashForFinalization())
	latestBlockBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(latestBlockBytes, uint64(rf.Exchange.Reply.LatestBlock))
	return bytes.Join([][]byte{latestBlockBytes, rf.Exchange.Reply.FinalizedBlocksHashes, rf.Addr, relaySessionHash}, nil)
}

func (rf RelayFinalization) HashCount() int {
	return 1
}
