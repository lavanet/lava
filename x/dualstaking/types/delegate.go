package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/lavaslices"
)

func NewDelegation(delegator, provider, chainID string, blockTime time.Time, tokenDenom string) Delegation {
	return Delegation{
		Delegator: delegator,
		Provider:  provider,
		ChainID:   chainID,
		Amount:    sdk.NewCoin(tokenDenom, sdk.ZeroInt()),
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

func (delegator *Delegator) AddProvider(provider string) bool {
	contains := lavaslices.Contains(delegator.Providers, provider)
	if !contains {
		delegator.Providers = append(delegator.Providers, provider)
	}
	return contains
}

func (delegator *Delegator) DelProvider(provider string) {
	if lavaslices.Contains(delegator.Providers, provider) {
		delegator.Providers, _ = lavaslices.Remove(delegator.Providers, provider)
	}
}

func (delegator *Delegator) IsEmpty() bool {
	return len(delegator.Providers) == 0
}
