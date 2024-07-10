package types

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lavanet/lava/utils"
)

const (
	UniqueEpochSessionPrefix        = "UniqueEpochSession/"
	ProviderEpochCuPrefix           = "ProviderEpochCu/"
	ProviderEpochComplainerCuPrefix = "ProviderEpochComplainerCu/"
	ProviderConsumerEpochCuPrefix   = "ProviderConsumerEpochCu/"
)

func UniqueEpochSessionKey(epoch uint64, provider string, chainID string, project string, sessionID uint64) []byte {
	return append(utils.Serialize(epoch), []byte(strings.Join([]string{provider, chainID, project, strconv.FormatUint(sessionID, 10)}, " "))...)
}

func ProviderEpochCuKey(epoch uint64, provider string, chainID string) []byte {
	return append(utils.Serialize(epoch), []byte(strings.Join([]string{provider, chainID}, " "))...)
}

func ProviderConsumerEpochCuKey(epoch uint64, provider string, project string, chainID string) []byte {
	return append(utils.Serialize(epoch), []byte(strings.Join([]string{provider, project, chainID}, " "))...)
}

func DecodeUniqueEpochSessionKey(key string) (epoch uint64, provider string, chainID string, project string, sessionID uint64, err error) {
	if len(key) < 8 {
		return 0, "", "", "", 0, fmt.Errorf("invalid UniqueEpochSession key: bad structure. key: %s", key)
	}

	split := strings.Split(key[8:], " ")
	if len(split) != 4 {
		return 0, "", "", "", 0, fmt.Errorf("invalid UniqueEpochSession key: bad structure. key: %s", key)
	}
	utils.Deserialize([]byte(key[:8]), &epoch)
	sessionID, err = strconv.ParseUint(split[3], 10, 64)
	if err != nil {
		return 0, "", "", "", 0, fmt.Errorf("invalid UniqueEpochSession key: bad session ID. key: %s", key)
	}
	return epoch, split[0], split[1], split[2], sessionID, nil
}

func DecodeProviderEpochCuKey(key string) (epoch uint64, provider string, chainID string, err error) {
	if len(key) < 8 {
		return 0, "", "", fmt.Errorf("invalid ProviderEpochCu key: bad structure. key: %s", key)
	}
	split := strings.Split(key[8:], " ")
	if len(split) != 2 {
		return 0, "", "", fmt.Errorf("invalid ProviderEpochCu key: bad structure. key: %s", key)
	}
	utils.Deserialize([]byte(key[:8]), &epoch)
	return epoch, split[0], split[1], nil
}

func DecodeProviderConsumerEpochCuKey(key string) (epoch uint64, provider string, project string, chainID string, err error) {
	if len(key) < 8 {
		return 0, "", "", "", fmt.Errorf("invalid ProviderConsumerEpochCu key: bad structure. key: %s", key)
	}
	split := strings.Split(key[8:], " ")
	if len(split) != 3 {
		return 0, "", "", "", fmt.Errorf("invalid ProviderConsumerEpochCu key: bad structure. key: %s", key)
	}

	utils.Deserialize([]byte(key[:8]), &epoch)
	return epoch, split[0], split[1], split[2], nil
}

func UniqueEpochSessionKeyPrefix() []byte {
	return []byte(UniqueEpochSessionPrefix)
}

func ProviderEpochCuKeyPrefix() []byte {
	return []byte(ProviderEpochCuPrefix)
}

func ProviderEpochComplainerCuKeyPrefix() []byte {
	return []byte(ProviderEpochComplainerCuPrefix)
}

func ProviderConsumerEpochCuKeyPrefix() []byte {
	return []byte(ProviderConsumerEpochCuPrefix)
}
