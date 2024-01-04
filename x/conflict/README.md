# `x/conflict`

## Abstract

This document specifies the conflict module of Lava Protocol.

The purpose of the conflict module is to ensure data reliability from providers that offer their services on Lava. To achieve this, Lava allows consumers to report mismatches between different responses to the same request (for deterministic APIs). This way, Lava can identify fraudulent providers and penalize them by confiscating their staked tokens. Consumers are incentivized to do so by Lava, as they receive tokens taken from the fraudulent provider being punished.


## Contents
* [Concepts](#concepts)
    * [Finalization Conflict](#finalization-conflict)
    * [Self Provider Conflict](#self-provider-conflict)
    * [Response Conflict](#response-conflict)
    * [Commit Period](#commit-period)
    * [Reveal Period](#reveal-period)
    * [Conflict Resolve](#Conflict-Resolve)
* [Parameters](#parameters)
* [Queries](#queries)
* [Transactions](#transactions)
* [Events](#events)

## Concepts

### Finalization Conflict
N/A


### Self Provider Conflict
N/A


### Response Conflict
A response conflict occurs when a consumer receives mismatched responses from different providers. In such cases, the consumer is eligible to send a conflict detection message, which includes the relay request and responses from the two providers. This conflict detection message is then validated to ensure that the responses are different, the signatures match the providers and consumer, and that the API is deterministic. If the message is valid, a conflict is opened.

A group of validators is selected as a jury to determine the fraudulent and honest providers. Through an event, the chain announces the conflict voting period and the participating providers. During the voting period, providers need to submit their hashed response + salt to the original relay request. This is done to prevent other providers from cheating or copying their vote. Once the voting period ends, the conflict moves to the reveal state. In this state, providers need to reveal their response + salt, which is then verified and compared to the original responses. After the reveal period ends, the votes are counted, and the provider with the fewest votes is penalized by having a fraction of their staked tokens taken and distributed among all the participants.

### Commit Period

This is the voting period of the conflict, providers that are selected as jury are required to submit their hashed vote.
The hash consisnt of nonce + response hash + provider address

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

Once the reveal period ends votes are than counted to determine the results of the conflict. Votes are weighted by providers stake. 
Providers in the jury that did not vote are punished by jail and slashing, their slashed amount will be added to the conflict reward pool.
For the conflict resolution there needs to be a majority met of votes for Provider A, Provider B or None of them. 
If a majority was not met, conflict reward pool is given to the consumer that reported the conflict.
Once a majority is met providers that voted to the wrong side of the conflict are slashed and frozen, the slashed amount is added to the conflict reward pool.
Now the reward pool is distributed between the comsumer and the providers that voted for the correct provider.

## Parameters

The conflict module contains the following parameters:

| Key                                    | Type                    | Default Value    |
| -------------------------------------- | ----------------------- | -----------------|
| MajorityPercent                        | math.LegacyDec          | 0.95              |
| VoteStartSpan                              | uint64          | 3             |
| VotePeriod                       | uint64          | 2                |
| Rewards                        | rewards                  | N/A                |

`MajorityPercent` majority needed to conclude who is the winner of the conclict
`VoteStartSpan` is the amount of epochs in the past that a consumer can send conflict detection message for
`VotePeriod` is the amount of epochs for each voting state (commit/reveal)
`Rewards` defines how the reward pool tokens will be distributed between all participants

```go
type Rewards struct {
	WinnerRewardPercent github_com_cosmos_cosmos_sdk_types.Dec // the winnig provider of the conflict portion
	ClientRewardPercent github_com_cosmos_cosmos_sdk_types.Dec // the reporter of the conflict portion
	VotersRewardPercent github_com_cosmos_cosmos_sdk_types.Dec // the jury portion
}
```
## Queries

The Conflict module supports the following queries:

| Query             | Arguments         | What it does                                  |
| ----------        | ---------------   | ----------------------------------------------|
| `params`          | none              | show the params of the module                 |
| `consumer-conflicts` | consumer (string)              | shows all conflicts reported and active by the consumer         |
| `list-conflict-vote` | none           | shows all active conflicts                |
| `show-conflict-vote`       | voteID (string)           | shows a specific active conflict                             |

## Transactions

The Conflict module transactions are not ment to be sent by the CLI and are used by the consumer and providers.
for more information look [here](../../proto/lavanet/lava/conflict/tx.proto)

### Events

The plans module has the following events:
| Event             | When it happens       |
| ----------        | --------------- |
| `response_conflict_detection`        | a new conflict has opened (this event includes all of the jury providers) and is entering the commit stage  |
| `conflict_vote_reveal_started`        | conflict has transitioned to reveal state  |
| `conflict_vote_got_commit`        | provider commited his vote  |
| `conflict_vote_got_reveal`        | provider revealed his vote  |
| `conflict_unstake_fraud_voter`        | provider was unstaked due to conflict  |
| `conflict_detection_vote_resolved`        | conflict was succesfully resolved  |
| `conflict_detection_vote_unresolved`        | conflict was not resolved (did not reach majority)  |
| `spec_add`        | a successful addition of a spec  |
