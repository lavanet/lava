package types

import (
	tendermintcrypto "github.com/cometbft/cometbft/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils/sigs"
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
	latestBlockBytes := sigs.EncodeUint64(uint64(rf.Exchange.Reply.LatestBlock))
	msgParts := [][]byte{
		latestBlockBytes,
		rf.Exchange.Reply.FinalizedBlocksHashes,
		rf.Addr,
		relaySessionHash,
	}
	return sigs.Join(msgParts)
}

func (rf RelayFinalization) HashRounds() int {
	return 1
}
