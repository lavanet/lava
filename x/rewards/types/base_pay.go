package types

import (
	"encoding/binary"
	fmt "fmt"
)

var _ binary.ByteOrder

const (
	// StakeStorageKeyPrefix is the prefix to retrieve all StakeStorage
	BasePayPrefix = "BasePay/"
)

type BasePayIndex struct {
	Provider     string
	ChainID      string
	Subscription string
}

type BasePayWithIndex struct {
	BasePayIndex
	BasePay
}

const (
	serializedFormat = "%s %s %s"
)

func (bp BasePayIndex) String() string {
	return fmt.Sprintf(serializedFormat, bp.Provider, bp.ChainID, bp.Subscription)
}

func BasePayKeyRecover(key string) (bp BasePayIndex) {
	fmt.Sscanf(key, "%s %s %s", &bp.Provider, &bp.ChainID, &bp.Subscription)
	return bp
}
