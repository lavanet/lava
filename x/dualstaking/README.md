# `x/dualstaking`

## Abstract

This document specifies the dualstaking module of Lava Protocol.

In the lava blockchain there are two kinds of staking users, the first ones are validators, legacy to cosmos, the second ones are providers.
Providers in lava give services to consumers, they compete for giving services with other providers by staking tokens.
Since a lot of tokens are expected to be staked by providers lava give providers the power to add security to the chain with dualstaking.

This document focuses on the dualstaking technical aspects. For more information on those, please visit Lava's official website.

## Contents
* [Concepts](#concepts)
    * [Delegation](#delegation)
    * [EmptyProvider](#emptyprovider)
    * [Dualstaking](#dualstaking)
        * [Dualstaking](#validatordelegation)
        * [Dualstaking](#validatorunbonding)
        * [Dualstaking](#validatorslashing)
        * [Dualstaking](#providerdelegation)
        * [Dualstaking](#providerunbonding)
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

Dualstaking intoduces to lava provider delegations. provder delegations lets users to delegate their tokens to a specific provider (similar to validators), contribute to their success and claim some of the rewards awarded to the provider.
When a provider is staking tokens he is creating a self-delegation entry.
Whenever a provider recieves rewards, all the delegators will be aligible for a portion of it proportionate to their delegation amount and the commision rate set by the provider.

### EmptyProvider

The empty provider is a place holder for provider delegations that are issued by the staking module. the user can choose to redelegate from this provider to an actual provider.

### Dualstaking

Dualstaking exists to give power to providers the same as validators. whenever a provider stakes tokens the same amount is also staked to a validator.
Dualstaking does this by implementing providers delegators (similar to validators delegators) and using hooks to from the staking module.
For every validator delegation exists a parralel provider delegation and vice versa.
When a provider stakes his tokens, he is a self delegating provider and delegating to a validator with the same amount. This give him gov influence and support a validator of his choosing which in turn boosts the security of the chain (this is also true to providers delegators).
The same way happens with validators, when a validator stakes his tokens (or a delegator), using hooks a delegation is created to an empty provider. Than using the dualstaking module he can redelegate from the empty provider to an actual provider.
Since the module uses hooks to achieve this functionality it does not need to move coins, the staking module does it for him.
For a specific address, at all times the amount of validators delegation is equal to providers delegations.
The following are use cases of the dualstaking module:

#### Validator delegation

1. Call delegate method of the staking module.
2. Hook on create delegation and delegate to the empty provider.

#### Validator unbonding

1. Call unbond method of the staking module.
2. Hook on create delegation and unbond the same amount from the providers delegations uniformaly with priority to the empty provider.

#### Validator slashing

1. Hook on calidator slashing
2. remove the slashed amount uniformaly from the providers delegations

#### Provider delegation

1. Call delegate method of the dualstaking module.
2. Hook on create delegation and delegate to the empty provider.
3. redelegate from the empty provider to the actual provider.

#### Provider unbonding

1. redelegate from the the provider to the empty provider.
2. Call unbond method of the dualstaking module.
3. Hook on create delegation and unbond from empty provider.

### Hooks

Dual staking module uses [staking hooks](keeper/hooks.go) to achieve its functionality.
1. AfterDelegationModified: this hook is called whenever a delegation is changed, whether it is created, or modified (NOT when completly removed). it calculates the difference in providers and validators stake to determine the action of the user (delegation or unbonding) depending on who is higher and than does the same with provider delegation.
    * If provider delegations > validator delegations: user unbonded, uniform unbond from providers delegations (priority to empty provider)
    * If provider delegations < validator delegations: user delegation, delegate to the empty provider
2. BeforeDelegationRemoved: this hook is called when a delegation to a validator is removed (unbonding of all the tokens). uniform unbond from providers delegations
3. BeforeValidatorSlashed: this hook is called when a validator is being slashed. to make sure the balance between validator and provider delegation is kept it uniform unbond from providers delegations the slashed amount. 

### RedelegateFlag

In case of validator redelegation we want to make sure the dual staking module dont take action.
This is with [antehandler](ante/ante_handler.go), whenever a redelegation msg is running, RedelegateFlag is marked true and the hooks will ignore the delegation changes.
Note: The RedelegateFlag is an on memory flag (not on state).

## Parameters

The Dualstaking module does not contain parameters.

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
| `delegate`     | validator-addr provider-addr chain-id amount| delegate to validator and provider the given amount|
| `redelegate`     | src-provider addr src-chain-id dst-provider-addr dst-chain-id  | redelegate provider delegation from source provider to destination provider|
| `unbond`     | validator-addr provider-addr chain-id amount  | undong from validator and provider the given amount                  |
| `claim-rewards`     | optional: provider-addr | claim the rewards from a given provider or all rewards |


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