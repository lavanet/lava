package types

import (
	"strings"

	regmath "math"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/math"
)

var (
	StakeEntriesPrefix                = collections.NewPrefix([]byte("StakeEntries/"))
	StakeEntriesCurrentPrefix         = collections.NewPrefix([]byte("StakeEntriesCurrent/"))
	EpochChainIdProviderIndexesPrefix = collections.NewPrefix([]byte("EpochChainIdProviderIndexes/"))
)

// EpochChainIdProviderIndexes defines a secondary unique index for the keeper's stakeEntries indexed map
// Normally, a stake entry can be accessed with the primary key: [epoch, chainID, stake, address]
// The new set of indexes, EpochChainIdProviderIndexes, allows accessing the stake entries with [epoch, chainID, address]
type EpochChainIdProviderIndexes struct {
	Index *indexes.Unique[collections.Triple[uint64, string, string], collections.Triple[uint64, string, collections.Pair[uint64, string]], StakeEntry]
}

func (e EpochChainIdProviderIndexes) IndexesList() []collections.Index[collections.Triple[uint64, string, collections.Pair[uint64, string]], StakeEntry] {
	return []collections.Index[collections.Triple[uint64, string, collections.Pair[uint64, string]], StakeEntry]{e.Index}
}

func NewEpochChainIdProviderIndexes(sb *collections.SchemaBuilder) EpochChainIdProviderIndexes {
	return EpochChainIdProviderIndexes{
		Index: indexes.NewUnique(sb, EpochChainIdProviderIndexesPrefix, "stake_entry_by_epoch_chain_address",
			collections.TripleKeyCodec(collections.Uint64Key, collections.StringKey, collections.StringKey),
			collections.TripleKeyCodec(collections.Uint64Key, collections.StringKey,
				collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)),
			func(pk collections.Triple[uint64, string, collections.Pair[uint64, string]], _ StakeEntry) (collections.Triple[uint64, string, string], error) {
				return collections.Join3(pk.K1(), pk.K2(), pk.K3().K2()), nil
			},
		),
	}
}

func StakeEntryKeyCurrent(chainID string, provider string) []byte {
	return []byte(strings.Join([]string{chainID, provider}, " "))
}

// StakeEntry methods

func (se StakeEntry) EffectiveStake() math.Int {
	effective := se.Stake.Amount
	if se.DelegateLimit.Amount.LT(se.DelegateTotal.Amount) {
		effective = effective.Add(se.DelegateLimit.Amount)
	} else {
		effective = effective.Add(se.DelegateTotal.Amount)
	}
	return effective
}

// Frozen provider block const
const FROZEN_BLOCK = regmath.MaxInt64

func (stakeEntry *StakeEntry) Freeze() {
	stakeEntry.StakeAppliedBlock = FROZEN_BLOCK
}

func (stakeEntry *StakeEntry) UnFreeze(currentBlock uint64) {
	stakeEntry.StakeAppliedBlock = currentBlock
}

func (stakeEntry *StakeEntry) IsFrozen() bool {
	return stakeEntry.StakeAppliedBlock == FROZEN_BLOCK
}

func (stakeEntry *StakeEntry) IsJailed(time int64) bool {
	return stakeEntry.JailEndTime > time
}

func (stakeEntry *StakeEntry) IsAddressVaultAndNotProvider(address string) bool {
	return address != stakeEntry.Address && address == stakeEntry.Vault
}

func (stakeEntry *StakeEntry) IsAddressVaultOrProvider(address string) bool {
	return address == stakeEntry.Address || address == stakeEntry.Vault
}
