package types

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

const (
	UniqueEpochSessionPrefix        = "UniqueEpochSession/"
	ProviderEpochCuPrefix           = "ProviderEpochCu/"
	ProviderEpochComplainerCuPrefix = "ProviderEpochComplainerCu/"
	ProviderConsumerEpochCuPrefix   = "ProviderConsumerEpochCu/"
)

func EncodeBlock(block uint64) []byte {
	encodedKey := make([]byte, 8)
	binary.BigEndian.PutUint64(encodedKey[0:8], block)
	return encodedKey
}

func DecodeBlock(encodedKey []byte) uint64 {
	return binary.BigEndian.Uint64(encodedKey[0:8])
}

func UniqueEpochSessionKey(epoch uint64, provider string, chainID string, project string, sessionID uint64) []byte {
	return []byte(strings.Join([]string{string(EncodeBlock(epoch)), provider, chainID, project, strconv.FormatUint(sessionID, 10)}, " "))
}

func ProviderEpochCuKey(epoch uint64, provider string, chainID string) []byte {
	return []byte(strings.Join([]string{string(EncodeBlock(epoch)), provider, chainID}, " "))
}

func ProviderConsumerEpochCuKey(epoch uint64, provider string, project string, chainID string) []byte {
	return []byte(strings.Join([]string{string(EncodeBlock(epoch)), provider, project, chainID}, " "))
}

func DecodeUniqueEpochSessionKey(key string) (epoch uint64, provider string, chainID string, project string, sessionID uint64, err error) {
	if len(key) < 9 {
		return 0, "", "", "", 0, fmt.Errorf("invalid UniqueEpochSession key: bad structure. key: %s", key)
	}

	split := strings.Split(key[9:], " ")
	if len(split) != 4 {
		return 0, "", "", "", 0, fmt.Errorf("invalid UniqueEpochSession key: bad structure. key: %s", key)
	}
	epoch = DecodeBlock([]byte(key[0:8]))
	sessionID, err = strconv.ParseUint(split[4], 10, 64)
	if err != nil {
		return 0, "", "", "", 0, fmt.Errorf("invalid UniqueEpochSession key: bad session ID. key: %s", key)
	}
	return epoch, split[1], split[2], split[3], sessionID, nil
}

func DecodeProviderEpochCuKey(key string) (epoch uint64, provider string, chainID string, err error) {
	if len(key) < 9 {
		return 0, "", "", fmt.Errorf("invalid ProviderEpochCu key: bad structure. key: %s", key)
	}
	split := strings.Split(key[9:], " ")
	if len(split) != 2 {
		return 0, "", "", fmt.Errorf("invalid ProviderEpochCu key: bad structure. key: %s", key)
	}
	epoch = DecodeBlock([]byte(key[0:8]))
	return epoch, split[1], split[2], nil
}

func DecodeProviderConsumerEpochCuKey(key string) (epoch uint64, provider string, project string, chainID string, err error) {
	if len(key) < 9 {
		return 0, "", "", "", fmt.Errorf("invalid ProviderConsumerEpochCu key: bad structure. key: %s", key)
	}
	split := strings.Split(key[9:], " ")
	if len(split) != 3 {
		return 0, "", "", "", fmt.Errorf("invalid ProviderConsumerEpochCu key: bad structure. key: %s", key)
	}
	epoch = DecodeBlock([]byte(key[0:8]))
	return epoch, split[1], split[2], split[3], nil
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
