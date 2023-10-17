package types

import (
	"strconv"
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

// CuTrackerKey encodes a keys using the subscription's consumer address, provider address, relay's chain ID and relay's block
func CuTrackerKey(sub string, provider string, chainID string, relayBlock uint64) string {
	return sub + " " + provider + " " + chainID + " " + strconv.FormatUint(relayBlock, 10)
}

// DecodeCuTrackerKey decodes the CU tracker key
func DecodeCuTrackerKey(key string) (sub string, provider string, chainID string, relayBlock uint64) {
	decodedKey := strings.Split(key, " ")
	relayBlock, err := strconv.ParseUint(decodedKey[3], 10, 64)
	if err != nil {
		return "", "", "", 0
	}
	return decodedKey[0], decodedKey[1], decodedKey[2], relayBlock
}

// CuTrackerTimerKey encodes a key using the subscription's consumer address and its creation block
func CuTrackerTimerKey(sub string, subBlock uint64) string {
	return sub + " " + strconv.FormatUint(subBlock, 10)
}

// DecodeCuTrackerTimerKey decodes the CU tracker timer key. Caller need to check for ""
func DecodeCuTrackerTimerKey(key string) (sub string, subBlock uint64) {
	decodedKey := strings.Split(key, " ")
	subBlock, err := strconv.ParseUint(decodedKey[1], 10, 64)
	if err != nil {
		return "", 0
	}
	return decodedKey[0], subBlock
}
