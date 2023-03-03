package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// PlanKeyPrefix is the prefix to retrieve all Plan
	PlanKeyPrefix = "Plan/value/"
)

// PlanKey returns the store key to retrieve a Plan from the index fields
func PlanKey(
	planIndex string,
) []byte {
	var key []byte

	planIndexBytes := []byte(planIndex)
	key = append(key, planIndexBytes...)
	key = append(key, []byte("/")...)

	return key
}
