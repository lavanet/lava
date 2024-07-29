package types

import (
	fmt "fmt"
	"strings"

	regmath "math"

	"cosmossdk.io/math"
	"github.com/lavanet/lava/utils"
)

var (
	StakeEntriesPrefix        = []byte("StakeEntries/")
	StakeEntriesCurrentPrefix = []byte("StakeEntriesCurrent/")
)

func StakeEntryKey(epoch uint64, chainID string, stake math.Int, provider string) []byte {
	key := append(utils.SerializeBigEndian(epoch), []byte(" "+chainID+" ")...)
	key = append(key, utils.SerializeBigEndian(stake.Uint64())...)
	key = append(key, []byte(" "+provider)...)
	return key
}

func StakeEntryKeyPrefixEpochChainId(epoch uint64, chainID string) []byte {
	return append(utils.SerializeBigEndian(epoch), []byte(" "+chainID+" ")...)
}

func StakeEntryKeyCurrent(chainID string, provider string) []byte {
	return []byte(strings.Join([]string{chainID, provider}, " "))
}

func ExtractEpochFromStakeEntryKey(key string) (epoch uint64, err error) {
	if len(key) < 8 {
		return 0, fmt.Errorf("ExtractEpochFromStakeEntryKey: invalid StakeEntryKey, bad structure. key: %s", key)
	}

	utils.DeserializeBigEndian([]byte(key[:8]), &epoch)
	return epoch, nil
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
