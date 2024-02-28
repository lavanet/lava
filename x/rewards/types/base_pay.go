package types

import (
	fmt "fmt"
)

const (
	// BasePayPrefix is the prefix to retrieve all BasePay
	BasePayPrefix = "BasePay/"
)

type BasePayIndex struct {
	Provider string
	ChainID  string
}

type BasePayWithIndex struct {
	BasePayIndex
	BasePay
}

const (
	serializedFormat = "%s %s"
)

func (bp BasePayIndex) String() string {
	return fmt.Sprintf(serializedFormat, bp.ChainID, bp.Provider)
}

func BasePayKeyRecover(key string) (bp BasePayIndex) {
	fmt.Sscanf(key, serializedFormat, &bp.ChainID, &bp.Provider)
	return bp
}
