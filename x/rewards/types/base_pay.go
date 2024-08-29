package types

import (
	fmt "fmt"
)

const (
	// BasePayPrefix is the prefix to retrieve all BasePay
	BasePayPrefix = "BasePay/"
)

const (
	serializedFormat = "%s %s"
)

func (bp BasePayWithIndex) Index() string {
	return fmt.Sprintf(serializedFormat, bp.ChainId, bp.Provider)
}

func BasePayKeyRecover(key string) (bp BasePayWithIndex) {
	fmt.Sscanf(key, serializedFormat, &bp.ChainId, &bp.Provider)
	return bp
}
