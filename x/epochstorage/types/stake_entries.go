package types

import (
	"strings"

	regmath "math"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
)

var (
	StakeEntriesPrefix        = collections.NewPrefix([]byte("StakeEntries/"))
	StakeEntriesCurrentPrefix = collections.NewPrefix([]byte("StakeEntriesCurrent/"))
)

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
