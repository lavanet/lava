package types

import (
	"strings"
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

	// prefix for the delegations fixation store
	DelegationPrefix = "delegation-fs"

	// prefix for the delegators fixation store
	DelegatorPrefix = "delegator-fs"

	// prefix for the unbonding timer store
	UnbondingPrefix = "unbonding-ts"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

// DelegationKey returns the key/prefix for the Delegation entry in fixation store.
// Using " " (space) as spearator is safe because Bech32 forbids its use as part of
// the address (and is the only visible character that can be safely used).
// (reference https://en.bitcoin.it/wiki/BIP_0173#Specification)
func DelegationKey(delegator, provider, chainID string) string {
	return provider + "  " + chainID + " " + delegator
}

func DelegationKeyDecode(prefix string) (delegator, provider, chainID string) {
	split := strings.Split(prefix, " ")
	return split[2], split[0], split[1]
}

// DelegatorKey returns the key/prefix for the Delegator entry in fixation store.
func DelegatorKey(delegator string) string {
	return delegator
}

func DelegatorKeyDecode(prefix string) (delegator string) {
	return prefix
}
