package types

import (
	"strings"
)

const (
	// ModuleName defines the module name
	ModuleName = "subscription"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for slashing
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_subscription"

	// prefix for the subscription fixation store
	SubsFixationPrefix = "subs-fs"

	// prefix for the subscription timer store
	SubsTimerPrefix = "subs-ts"

	// prefix for the CU tracker fixation store
	CuTrackerFixationPrefix = "cu-tracker-fs"

	// prefix for the CU tracker timer store
	CuTrackerTimerPrefix = "cu-tracker-ts"
)

// CuTrackerKey encodes a keys using the subscription's consumer address, provider address and the relay's chain ID
func CuTrackerKey(sub string, provider string, chainID string) string {
	return sub + " " + provider + " " + chainID
}

// DecodeCuTrackerKey decodes the CU tracker key
func DecodeCuTrackerKey(key string) (sub string, provider string, chainID string) {
	decodedKey := strings.Split(key, " ")
	return decodedKey[0], decodedKey[1], decodedKey[2]
}
