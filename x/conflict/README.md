# `x/conflict`

## Abstract

This document specifies the conflict module of Lava Protocol.

The purpose of the conflict module is to ensure data reliability from providers that offer their services on Lava. To achieve this, Lava allows consumers to report mismatches between different responses to the same request (for deterministic APIs). This way, Lava can identify fraudulent providers and penalize them by confiscating their staked tokens. Consumers are incentivized to do so by Lava, as they receive tokens taken from the fraudulent provider being punished.

## Contents

- [Concepts](#concepts)
  - [Response Conflict](#response-conflict)
  - [Finalization Conflict](#finalization-conflict)
    - [Self Provider Conflict](#self-provider-conflict)
    - [Commit Period](#commit-period)
    - [Reveal Period](#reveal-period)
    - [Conflict Resolve](#Conflict-Resolve)
- [Parameters](#parameters)
- [Queries](#queries)
- [Transactions](#transactions)
- [Events](#events)

## Concepts

### Response Conflict

A response conflict occurs when a consumer receives mismatched responses from different providers. In such cases, the consumer is eligible to send a conflict detection message (this is done randomly, determined by the spec reliability threshold field), which includes the relay request and responses from the two providers. This conflict detection message is then validated to ensure that the responses are different, the signatures match the providers and consumer, and that the API is deterministic. If the message is valid, a conflict is opened.

A group of validators is selected as a jury to determine the fraudulent and honest providers. Through an event, the chain announces the conflict voting period and the participating providers. During the voting period, providers need to submit their hashed response + salt to the original relay request. This is done to prevent other providers from cheating or copying their vote. Once the voting period ends, the conflict moves to the reveal state. In this state, providers need to reveal their response + salt, which is then verified and compared to the original responses. After the reveal period ends, the votes are counted, and the provider with the fewest votes, and the jury that voted for him, are penalized by having a fraction of their staked tokens taken and distributed among all the other participants.

### Finalization Conflict

A finalization conflict occurs when a consumer receives mismatched finalized block hashes from different providers. This is similar to a response conflict, except it involves the finalized block hash instead of a response to a request. The process is identical - the consumer submits a conflict detection message, a jury is selected, providers commit/reveal votes, and the provider with the fewest votes is penalized.

### Self Provider Conflict

A finalization conflict occurs when a consumer receives mismatched finalized block hashes from the same provider. This can happen if a provider responds with different finalized block hashes to the same request sent multiple times. In this case, no voting is needed, since the provider contradicted itself. The provider is directly penalized for providing inconsistent responses.

### Commit Period

This is the voting period of the conflict, providers that are selected as jury are required to submit their hashed vote.
The hash consisnt of nonce + response hash + provider address.

```go
type MsgConflictVoteCommit struct {
	Creator string // provider address
	VoteID  string // the conflict unique id
	Hash    []byte // the hashed nonce + response + provider address
}
```

### Reveal Period

Once the commit period ends, the conflict enters the reveal state. At this point, providers are required to submit their full response and nonce. These responses are then validated and compared to the original providers' responses. The result of the vote (Provider A, Provider B, None) is then saved in the state, awaiting the end of the conflict.

```go
type MsgConflictVoteReveal struct {
	Creator string // provider address
	VoteID  string // the conflict unique id
	Nonce   int64  // random nonce
	Hash    []byte // hash of the response
}
```

### Conflict Resolve

Once the reveal period ends, votes are counted to determine the results of the conflict. The votes are weighted based on the providers' stake. Providers on the jury who did not vote will be punished with jail and slashing, and the slashed amount will be added to the conflict reward pool.

To resolve the conflict, a majority of votes must be in favor of Provider A, Provider B, or None of them. If a majority is not reached, the conflict reward pool is given to the consumer who reported the conflict.

Once a majority is reached, providers who voted for the wrong side of the conflict will be slashed and frozen, and the slashed amount will be added to the conflict reward pool. The reward pool is then distributed between the consumer and the providers who voted for the correct provider.
Providers in the jury that did not vote are punished by jail and slashing, their slashed amount will be added to the conflict reward pool.
For the conflict resolution there needs to be a majority met of votes for Provider A, Provider B or None of them.
If a majority was not met, conflict reward pool is given to the consumer that reported the conflict.
Once a majority is met providers that voted to the wrong side of the conflict are slashed and frozen, the slashed amount is added to the conflict reward pool.
Now the reward pool is distributed between the comsumer and the providers that voted for the correct provider.

## Parameters

The conflict module contains the following parameters:

| Key             | Type           | Default Value |
| --------------- | -------------- | ------------- |
| MajorityPercent | math.LegacyDec | 0.95          |
| VoteStartSpan   | uint64         | 3             |
| VotePeriod      | uint64         | 2             |
| Rewards         | rewards        | N/A           |

`MajorityPercent` determines the majority needed to conclude who is the winner of the conflict.

`VoteStartSpan` is the amount of epochs in the past that a consumer can send conflict detection message for.

`VotePeriod` is the number of epochs in the past that a consumer can send a conflict detection message for.
`Rewards` defines how the reward pool tokens will be distributed between all the participants.

```go
type Rewards struct {
	WinnerRewardPercent github_com_cosmos_cosmos_sdk_types.Dec // the conflict's winner portion (provider)
	ClientRewardPercent github_com_cosmos_cosmos_sdk_types.Dec // the conflict's reporter portion
	VotersRewardPercent github_com_cosmos_cosmos_sdk_types.Dec // the jury portion
}
```

## Queries

The Conflict module supports the following queries:

| Query                | Arguments         | What it does                                              |
| -------------------- | ----------------- | --------------------------------------------------------- |
| `params`             | none              | show the params of the module                             |
| `consumer-conflicts` | consumer (string) | shows all the reported and active conflicts by a consumer |
| `list-conflict-vote` | none              | shows all active conflicts                                |
| `show-conflict-vote` | voteID (string)   | shows a specific active conflict                          |

## Transactions

The Conflict module transactions are not meant to be used by the CLI and are used by the consumers and providers.
for more information look [here](../../proto/lavanet/lava/conflict/tx.proto).

### Events

The conflict module has the following events:

| Event                                | When it happens                                                                                               |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------- |
| `response_conflict_detection`        | A new conflict has been opened, which involves all of the jury providers. It is now entering the commit stage |
| `conflict_vote_reveal_started`       | conflict has transitioned to reveal state                                                                     |
| `conflict_vote_got_commit`           | provider commited his vote                                                                                    |
| `conflict_vote_got_reveal`           | provider revealed his vote                                                                                    |
| `conflict_unstake_fraud_voter`       | provider was unstaked due to conflict                                                                         |
| `conflict_detection_vote_resolved`   | conflict was succesfully resolved                                                                             |
| `conflict_detection_vote_unresolved` | conflict was not resolved (did not reach majority)                                                            |
| `conflict_detection_same_provider`   | same provider finalization conflict was detected                                                              |
| `conflict_detection_two_providers`   | finalization conflict between two providers was detected                                                      |
