package types

import (
	"cosmossdk.io/collections"
)

const (
	// ModuleName defines the module name
	ModuleName = "dualstaking"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_dualstaking"

	// DisableDualstakingHooks prefix
	DisableDualstakingHookPrefix = "disable-dualstaking-hooks"

	// SlashedValidators prefix
	SlashedValidatorsPrefix = "slashed-validators"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

func DelegationKey(provider, delegator string) collections.Pair[string, string] {
	return collections.Join(provider, delegator)
}

var (
	DelegationsPrefix = collections.NewPrefix([]byte("Delegations/"))
	RewardPrefix      = collections.NewPrefix([]byte("Rewards/"))
)
