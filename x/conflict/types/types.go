package types

import (
	tendermintcrypto "github.com/cometbft/cometbft/crypto"
	"github.com/lavanet/lava/utils/sigs"
)

const (
	StateCommit = 0
	StateReveal = 1
)

// vote status for each voter
const (
	NoVote             = 0
	Commit             = 1
	Provider0          = 2
	Provider1          = 3
	NoneOfTheProviders = 4
)

const (
	ConflictVoteRevealEventName            = "conflict_vote_reveal_started"
	ConflictDetectionReceivedEventName     = "conflict_detection_received"
	ConflictVoteDetectionEventName         = "response_conflict_detection"
	ConflictVoteResolvedEventName          = "conflict_detection_vote_resolved"
	ConflictVoteUnresolvedEventName        = "conflict_detection_vote_unresolved"
	ConflictVoteGotCommitEventName         = "conflict_vote_got_commit"
	ConflictVoteGotRevealEventName         = "conflict_vote_got_reveal"
	ConflictUnstakeFraudVoterEventName     = "conflict_unstake_fraud_voter"
	ConflictDetectionSameProviderEventName = "conflict_detection_same_provider"
	ConflictDetectionTwoProvidersEventName = "conflict_detection_two_providers"
)

// unstake description
const (
	UnstakeDescriptionFraudVote = "fraud provider found in conflict detection"
)

func CommitVoteData(nonce int64, dataHash []byte, providerAddress string) []byte {
	commitData := sigs.EncodeUint64(uint64(nonce))
	commitData = append(commitData, dataHash...)
	commitData = append(commitData, []byte(providerAddress)...)
	commitDataHash := tendermintcrypto.Sha256(commitData)
	return commitDataHash
}
