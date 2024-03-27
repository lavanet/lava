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

func UniqueEpochSessionKey(epoch uint64, provider string, chainID string, project string, sessionID uint64) []byte {
	return []byte(strings.Join([]string{strconv.FormatUint(epoch, 10), provider, chainID, project, strconv.FormatUint(sessionID, 10)}, " "))
}

func ProviderEpochCuKey(epoch uint64, provider string, chainID string) []byte {
	return []byte(strings.Join([]string{strconv.FormatUint(epoch, 10), provider, chainID}, " "))
}

func ProviderConsumerEpochCuKey(epoch uint64, provider string, project string, chainID string) []byte {
	return []byte(strings.Join([]string{strconv.FormatUint(epoch, 10), provider, project, chainID}, " "))
}

func DecodeUniqueEpochSessionKey(key string) (epoch uint64, provider string, chainID string, project string, sessionID uint64, err error) {
	split := strings.Split(key, " ")
	if len(split) != 5 {
		return 0, "", "", "", 0, fmt.Errorf("invalid UniqueEpochSession key: bad structure. key: %s", key)
	}
	epoch, err = strconv.ParseUint(split[0], 10, 64)
	if err != nil {
		return 0, "", "", "", 0, fmt.Errorf("invalid UniqueEpochSession key: bad epoch. key: %s", key)
	}
	sessionID, err = strconv.ParseUint(split[4], 10, 64)
	if err != nil {
		return 0, "", "", "", 0, fmt.Errorf("invalid UniqueEpochSession key: bad session ID. key: %s", key)
	}
	return epoch, split[1], split[2], split[3], sessionID, nil
}

func DecodeProviderEpochCuKey(key string) (epoch uint64, provider string, chainID string, err error) {
	split := strings.Split(key, " ")
	if len(split) != 3 {
		return 0, "", "", fmt.Errorf("invalid ProviderEpochCu key: bad structure. key: %s", key)
	}
	epoch, err = strconv.ParseUint(split[0], 10, 64)
	if err != nil {
		return 0, "", "", fmt.Errorf("invalid ProviderEpochCu key: bad epoch. key: %s", key)
	}
	return epoch, split[1], split[2], nil
}

func DecodeProviderConsumerEpochCuKey(key string) (epoch uint64, provider string, project string, chainID string, err error) {
	split := strings.Split(key, " ")
	if len(split) != 4 {
		return 0, "", "", "", fmt.Errorf("invalid ProviderConsumerEpochCu key: bad structure. key: %s", key)
	}
	epoch, err = strconv.ParseUint(split[0], 10, 64)
	if err != nil {
		return 0, "", "", "", fmt.Errorf("invalid ProviderConsumerEpochCu key: bad epoch. key: %s", key)
	}
	return epoch, split[1], split[2], split[3], nil
}

func UniqueEpochSessionKeyPrefix() []byte {
	return []byte(UniqueEpochSessionPrefix)
}

func ProviderEpochCuKeyPrefix() []byte {
	return []byte(ProviderEpochCuPrefix)
}

func ProviderConsumerEpochCuKeyPrefix() []byte {
	return []byte(ProviderConsumerEpochCuPrefix)
}
