# `x/dualstaking`

## Abstract

This document specifies the dualstaking module of Lava Protocol.

In the Lava blockchain there are two kinds of staking users, the first ones are validators, legacy to cosmos, the second ones are providers.
Validators play a role in the consensus mechanism, while providers offer services to consumers and compete with other providers by staking tokens.
Since a lot of tokens are expected to be staked by providers, to enhance the security of the chain, Lava lets providers to participate in the consensus via dualstaking. 
Dualstaking makes this happen by "duplicating" delegations, for each validator delegation a parallel provider delegation will be created for the delegator, As a result, providers gain power in the consensus, influencing governance and block creation.


## Contents
* [Concepts](#concepts)
    * [Delegation](#delegation)
    * [Empty Provider](#empty-provider)
    * [Dualstaking](#dualstaking)
        * [Validator Delegation](#validator-delegation)
        * [Validator Unbonding](#validator-unbonding)
        * [Validator Slashing](#validator-slashing)
        * [Provider Delegation](#provider-delegation)
        * [Provider Unbonding](#provider-unbonding)
    * [Hooks](#hooks)
    * [RedelegateFlag](#redelegateflag)
    * [Rewards](#rewards)
* [Parameters](#parameters)
* [Queries](#queries)
* [Transactions](#transactions)
* [Proposals](#proposals)
* [Events](#events)

## Concepts

### Delegation

Dualstaking introduces provider delegations to the Lava network. Provider delegations allow users to delegate their tokens to a specific provider, similar to validators, in order to contribute to their success and claim a portion of the rewards awarded to the provider.
When a provider stakes tokens, they create a self-delegation entry. Whenever a provider receives rewards, all delegators are eligible for a portion of the rewards based on their delegation amount and the commission rate set by the provider.

### Empty Provider

The empty provider is a place holder for provider delegations that are issued by the staking module. 
To support the functionality of the legacy Staking module, when a user delegates to a validator (it can't define the provider to delegate to in the legacy message), the dual staking module will delegate the same amount to the empty provider.
The user can than choose to redelegate from the empty provider to an actual provider.

### Dualstaking

Dualstaking exists to give power to providers in the same way as validators. Whenever a provider stakes tokens, an equal amount is also staked to a validator.
Dualstaking achieves this by implementing provider delegators (similar to validator delegators) and using hooks from the staking module. For every validator delegation, there exists a parallel provider delegation and vice versa.

When a provider stakes their tokens, they are both self-delegating as a provider and delegating to a validator with the same amount. This gives them governance influence and supports a validator of their choosing, which in turn boosts the security of the chain (the same applies to provider delegators). Please note, when a provider stakes, their stake amount must exceed the minimum stake specified in the corresponding specification. If a provider stakes an amount lower than this minimum, they are automatically frozen. The lowest boundary is the minimum self-delegation, a parameter of the dualstaking module. If the provider attempts to stake an amount below the minimum self-delegation, the stake transaction fails. Furthermore, if the provider tries to modify their existing stake entry and falls below the minimum self-delegation threshold, their stake entry is automatically removed (unstaked).

The same process happens with validators. When a validator stakes their tokens (or a delegator), a delegation is created to an empty provider using hooks. Then, using the dualstaking module, they can redelegate from the empty provider to an actual provider.
Since the module utilizes hooks to achieve this functionality, there is no need to move coins by the module, the staking module handles it automatically.
For a specific address, the amount of validator delegation is always equal to the amount of provider delegations.
The following are use cases of the dualstaking module:

#### Validator delegation

1. Call delegate method of the staking module.
2. Hook on create delegation and delegate to the empty provider.

#### Validator unbonding

1. Call unbond method of the staking module.
2. Hook on create delegation and unbond the same amount from the providers delegations uniformaly with priority to the empty provider.

#### Validator slashing

1. Hook on calidator slashing.
2. Remove the slashed amount uniformaly from the providers delegations.

#### Provider delegation

1. Call delegate method of the dualstaking module.
2. Hook on create delegation and delegate to the empty provider.
3. Redelegate from the empty provider to the actual provider.

#### Provider unbonding

1. Redelegate from the the provider to the empty provider.
2. Call unbond method of the dualstaking module.
3. Hook on create delegation and unbond from empty provider.

### Hooks

Dual staking module uses [staking hooks](keeper/hooks.go) to achieve its functionality.
1. AfterDelegationModified: this hook is called whenever a delegation is changed, whether it is created, or modified (NOT when completely removed). it calculates the difference in providers and validators stake to determine the action of the user (delegation or unbonding) depending on who is higher and than does the same with provider delegation.
    * If provider delegations > validator delegations: user unbonded, uniform unbond from providers delegations (priority to empty provider).
    * If provider delegations < validator delegations: user delegation, delegate to the empty provider.
2. BeforeDelegationRemoved: this hook is called when a delegation to a validator is removed (unbonding of all the tokens). uniform unbond from providers delegations
3. BeforeValidatorSlashed: this hook is called when a validator is being slashed. to make sure the balance between validator and provider delegation is kept it uniform unbond from providers delegations the slashed amount. 

### RedelegateFlag

To prevent the dual staking module from taking action in the case of validator redelegation, we utilize the [antehandler](ante/ante_handler.go). When a redelegation message is being processed, the RedelegateFlag is set to true, and the hooks will disregard any delegation changes. It is important to note that the RedelegateFlag is stored in memory and not in the chain’s state.

## Parameters

The dualstaking parameters:

| Key                                    | Type                    | Default Value    |
| -------------------------------------- | ----------------------- | -----------------|
| MinSelfDelegation                            | uint64                  | 100LAVA(=100000000ulava)               |

`MinSelfDelegation` determines the minimum amount of stake when a provider self delegates.

## Queries

The Dualstaking module supports the following queries:

| Query             | Arguments         | What it does                                  |
| ----------        | ---------------   | ----------------------------------------------|
| `params`          | none              | show the params of the module                 |
| `delegator-providers` | delegator address              | shows the providers that the delegator address is delegated to         |
| `provider-delegators` | provider address           | shows  all the providers delegators              |
| `delegator-rewards`       | delegator address           | shows all the claimable rewards of the delegator                             |

## Transactions

The Dualstaking module supports the following transactions:

| Transaction      | Arguments       | What it does                                  |
| ---------- | --------------- | ----------------------------------------------|
| `delegate`     | validator-addr(string) provider-addr (string) chain-id (string) amount (coin)| delegate to validator and provider the given amount|
| `redelegate`     | src-provider-addr (string) src-chain-id (string) dst-provider-addr (string) dst-chain-id (string) amount (coin)| redelegate provider delegation from source provider to destination provider|
| `unbond`     | validator-addr (string) provider-addr (string) chain-id (string) amount (coin) | undong from validator and provider the given amount                  |
| `claim-rewards`     | optional: provider-addr (string)| claim the rewards from a given provider or all rewards |


## Proposals

The Dualstaking module does not have proposals.

### Events

The Dualstaking module has the following events:
| Event             | When it happens       |
| ----------        | --------------- |
| `delegate_to_provider`        | a successful provider delegation  |
| `unbond_from_provider`     | a successful provider delegation unbond   |
| `redelegate_between_providers`    | a successful provider redelegation|
| `delegator_claim_rewards`    | a successful provider delegator reward claim|
| `contributor_rewards`    | spec contributor got new rewards|
| `validator_slash`    | validator slashed happened, providers slashed accordingly|