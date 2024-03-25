package types

import (
	"fmt"
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

func ProviderEpochCuKey(provider string, chainID string) []byte {
	return []byte(strings.Join([]string{provider, chainID}, " "))
}

func ProviderConsumerEpochCuKey(provider string, project string, chainID string) []byte {
	return []byte(strings.Join([]string{provider, project, chainID}, " "))
}

func DecodeUniqueEpochSessionKey(key string) (provider string, project string, chainID string, sessionID uint64, err error) {
	split := strings.Split(key, " ")
	if len(split) != 4 {
		return "", "", "", 0, fmt.Errorf("invalid UniqueEpochSession key")
	}
	sessionID, err = strconv.ParseUint(split[3], 10, 64)
	if err != nil {
		return "", "", "", 0, err
	}
	return split[0], split[1], split[2], sessionID, nil
}

func DecodeProviderEpochCuKey(key string) (provider string, chainID string, err error) {
	split := strings.Split(key, " ")
	if len(split) != 2 {
		return "", "", fmt.Errorf("invalid ProviderEpochCu key")
	}
	return split[0], split[1], nil
}

func DecodeProviderConsumerEpochCuKey(key string) (provider string, project string, chainID string, err error) {
	split := strings.Split(key, " ")
	if len(split) != 3 {
		return "", "", "", fmt.Errorf("invalid ProviderConsumerEpochCu key")
	}
	return split[0], split[1], split[2], nil
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
