package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// AdjustmentKeyPrefix is the prefix to retrieve all Adjustment
	AdjustmentKeyPrefix = "Adjustment/value/"
)
