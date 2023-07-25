package types

import (
	"bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils/sigs"
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
	return rf.Exchange.Reply.SigBlocks
}

func (rf RelayFinalization) DataToSign() []byte {
	relaySessionHash := tendermintcrypto.Sha256(rf.Exchange.Request.RelaySession.CalculateHashForFinalization())
	latestBlockBytes := sigs.Encode(uint64(rf.Exchange.Reply.LatestBlock))
	msgParts := [][]byte{
		latestBlockBytes,
		rf.Exchange.Reply.FinalizedBlocksHashes,
		rf.Addr,
		relaySessionHash,
	}
	return bytes.Join(msgParts, nil)
}

func (rf RelayFinalization) HashRounds() int {
	return 1
}
