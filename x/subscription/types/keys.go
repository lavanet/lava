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

func CuTrackerKey(sub string, provider string, chainID string) string {
	return sub + " " + provider + " " + chainID
}

func DecodeCuTrackerKey(key string) (sub string, provider string, chainID string) {
	decodedKey := strings.Split(key, " ")
	return decodedKey[0], decodedKey[1], decodedKey[2]
}

func CuTrackerTimerKey(sub string, block uint64) string {
	return sub + " " + strconv.FormatUint(block, 10)
}

// DecodeCuTrackerTimerKey decodes the CU tracker timer key. If there's an error, the returned sub
// is "" -> remember to check it!
func DecodeCuTrackerTimerKey(key string) (sub string, block uint64) {
	decodedKey := strings.Split(key, " ")
	block, err := strconv.ParseUint(decodedKey[1], 10, 64)
	if err != nil {
		return "", 0
	}
	return decodedKey[0], block
}
