package types

import (
	"strconv"
	"strings"
)

const (
	UniqueEpochSessionPrefix      = "UniqueEpochSession/"
	ProviderEpochCuPrefix         = "ProviderEpochCu/"
	ProviderConsumerEpochCuPrefix = "ProviderConsumerEpochCu/"
)

func UniqueEpochSessionKey(provider string, project string, chainID string, sessionID uint64) []byte {
	return []byte(strings.Join([]string{provider, project, chainID, strconv.FormatUint(sessionID, 10)}, " "))
}

func ProviderEpochCuKey(provider string) []byte {
	return []byte(provider)
}

func ProviderConsumerEpochCuKey(provider string, project string) []byte {
	return []byte(strings.Join([]string{provider, project}, " "))
}

func UniqueEpochSessionKeyPrefix(epoch uint64) []byte {
	return []byte(UniqueEpochSessionPrefix + strconv.FormatUint(epoch, 10) + "/")
}

func ProviderEpochCuKeyPrefix(epoch uint64) []byte {
	return []byte(ProviderEpochCuPrefix + strconv.FormatUint(epoch, 10) + "/")
}

func ProviderConsumerEpochCuKeyPrefix(epoch uint64) []byte {
	return []byte(ProviderConsumerEpochCuPrefix + strconv.FormatUint(epoch, 10) + "/")
}
