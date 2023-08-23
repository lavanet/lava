package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils/slices"
)

func NewDelegation(delegator, provider, chainID string) Delegation {
	return Delegation{
		Delegator: delegator,
		Provider:  provider,
		ChainID:   chainID,
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
