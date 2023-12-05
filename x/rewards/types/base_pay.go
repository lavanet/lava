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
	return fmt.Sprintf(serializedFormat, bp.ChainID, bp.Provider, bp.Subscription)
}

func BasePayKeyRecover(key string) (bp BasePayIndex) {
	fmt.Sscanf(key, "%s %s %s", &bp.ChainID, &bp.Provider, &bp.Subscription)
	return bp
}
