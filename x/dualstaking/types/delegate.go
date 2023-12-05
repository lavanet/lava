package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/slices"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

func NewDelegation(delegator, provider, chainID string, blockTime time.Time) Delegation {
	return Delegation{
		Delegator: delegator,
		Provider:  provider,
		ChainID:   chainID,
		Amount:    sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.ZeroInt()),
		Timestamp: utils.NextMonth(blockTime).UTC().Unix(),
	}
}

func (delegation *Delegation) AddAmount(amount sdk.Coin) {
	delegation.Amount = delegation.Amount.Add(amount)
}

func (delegation *Delegation) SubAmount(amount sdk.Coin) {
	delegation.Amount = delegation.Amount.Sub(amount)
}

func (delegation *Delegation) IsZero() bool {
	return delegation.Amount.IsZero()
}

func (delegation *Delegation) Equal(other *Delegation) bool {
	if delegation.Delegator != other.Delegator ||
		delegation.Provider != other.Provider ||
		delegation.ChainID != other.ChainID ||
		!delegation.Amount.IsEqual(other.Amount) {
		return false
	}
	return true
}

func (delegation *Delegation) IsFirstMonthPassed(currentTimestamp int64) bool {
	return delegation.Timestamp <= currentTimestamp
}

func NewDelegator(delegator, provider string) Delegator {
	return Delegator{
		Providers: []string{provider},
	}
}

func (delegator *Delegator) AddProvider(provider string) {
	if !slices.Contains(delegator.Providers, provider) {
		delegator.Providers = append(delegator.Providers, provider)
	}
}

func (delegator *Delegator) DelProvider(provider string) {
	if slices.Contains(delegator.Providers, provider) {
		delegator.Providers, _ = slices.Remove(delegator.Providers, provider)
	}
}

func (delegator *Delegator) IsEmpty() bool {
	return len(delegator.Providers) == 0
}
