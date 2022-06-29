package types

import (
	"encoding/binary"

	tendermintcrypto "github.com/tendermint/tendermint/crypto"
)

const (
	StateCommit = 0
	StateReveal = 1
)

const (
	NoVote    = 0
	Commit    = 1
	Provider0 = 2
	Provider1 = 3
	None      = 4
)

const (
	ConflictVoteRevealEventName = "conflict_vote_reveal"
)

func CommitVoteData(nonce int64, dataHash []byte) []byte {
	nonceBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(nonceBytes, uint64(nonce))
	commitData := append(nonceBytes, dataHash...)
	commitDataHash := tendermintcrypto.Sha256(commitData)
	return commitDataHash
}
