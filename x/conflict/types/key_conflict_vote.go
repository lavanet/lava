package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// ConflictVoteKeyPrefix is the prefix to retrieve all ConflictVote
	ConflictVoteKeyPrefix = "ConflictVote/value/"
)

// ConflictVoteKey returns the store key to retrieve a ConflictVote from the index fields
func ConflictVoteKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
