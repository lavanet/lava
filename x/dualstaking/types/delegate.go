package types

import (
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/utils/lavaslices"
)

var DelegationIndexPrefix = collections.NewPrefix(1)

type DelegationIndexes struct {
	ReverseIndex *indexes.ReversePair[string, string, Delegation]
}

func (a DelegationIndexes) IndexesList() []collections.Index[collections.Pair[string, string], Delegation] {
	return []collections.Index[collections.Pair[string, string], Delegation]{a.ReverseIndex}
}

func NewDelegationIndexes(sb *collections.SchemaBuilder) DelegationIndexes {
	return DelegationIndexes{
		ReverseIndex: indexes.NewReversePair[Delegation](
			sb, DelegationIndexPrefix, "delegation_by_provider_delegator",
			collections.PairKeyCodec(collections.StringKey, collections.StringKey),
		),
	}
}

func NewDelegation(delegator, provider string, blockTime time.Time, tokenDenom string) Delegation {
	return Delegation{
		Delegator: delegator,
		Provider:  provider,
		Amount:    sdk.NewCoin(tokenDenom, math.ZeroInt()),
		Timestamp: blockTime.AddDate(0, 0, 7).UTC().Unix(),
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
		!delegation.Amount.IsEqual(other.Amount) {
		return false
	}
	return true
}

func (delegation *Delegation) IsFirstWeekPassed(currentTimestamp int64) bool {
	// this is a temporary code to reduce the time to 1 week instead of month, will be changed in the gradual delegation increase feature.
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
