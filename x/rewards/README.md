# `x/rewards`

## Abstract

This document specifies the rewards module of Lava Protocol.

The rewards module is responsible for distributing rewards to validators and providers. Rewards are collected from the validators' and providers' pools, respectively. The pools' funds originate from the treasury account - a module account that holds all the pre-allocated Lava tokens of the network. It is important to note that the pre-allocated amount is constant, and there will be no minting of new Lava tokens in the future.

Please note that this module replaces Cosmos SDK's mint module, which is typically responsible for minting rewards for validators.

## Contents

* [Concepts](#concepts)
    * [The Treasury](#the-treasury)
    * [Validators Rewards Pool](#validators-rewards-pool)
    * [Providers Rewards Pool](#providers-rewards-pool)
* [Parameters](#parameters)
    * [MinBondedTarget](#minbondedtarget)
    * [MaxBondedTarget](#maxbondedtarget)
    * [LowFactor](#lowfactor)
    * [LeftOverBurnRate](#leftoverburnrate)

## Concepts

### The Treasury

As per Lava's vision, there is a constant supply of Lava tokens from the chain's genesis. There will be no minting of new tokens in the future like other Cosmos-SDK based chains. The treasury account holds all of Lava's pre-allocated tokens. The total amount of coins will be $10^9 * 10^6$ ulava.

### Rewards Pools

The rewards module is responsible on distributing rewards for validators that create new blocks, and providers that provide RPC service for their consumers.

To manage the rewards, the module uses two rewards pools: allocation pools and distribution pools. Note that there are two allocation pools and two distribution pools to manage the validators and providers rewards independently.

The allocation pools get some of the treasury account's tokens and hold onto it. Once a month, the allocation pools transfer a fixed amount of funds to the distribution pools. This monthly transfer will last for 4 years, after which the alocation pools' funds will be depleted (the allocation pools lifetime (4 years) is a pre-defined constant in the module's code). Note that before the allocation pools transfer the funds, the distribution pools' tokens are burned according to the `LeftOverBurnRate` parameter (see [below](#leftoverburnrate)).

The distribution pools use the monthly quota of funds to distribute rewards for validators and providers.

#### Validators Rewards

In Cosmos-SDK chains, validators get rewards for creating blocks by the Cosmos' distribution module. For each new block, new tokens are minted and distributed to all the accounts that contributed to the new block creation.

In Lava, the Cosmos validators reward distribution mechanism remains as is, but the source of the tokens comes from the rewards module (and not the mint module). As stated above, the rewards are managed using two pools: an allocation pool and a distribution pool.

The allocation pool of validators, `validators_rewards_allocation_pool`, gets 3% of the tokens of the treasury account.

The distribution pool of validators, `validators_rewards_distribution_pool`, uses its funds to reward validators for proposing new blocks. Same as Cosmos' mint module, the `validators_rewards_distribution_pool` transfers the block rewards to the fee collection account which is used by Cosmos' distribution module to reward the validators.

The validators block reward is calculated using the following formula:

$$ Reward = \frac{{\text{{validators\_distribution\_pool\_balance}}} \cdot {\text{bonded\_target\_factor}}}{\text{{remaining\_blocks\_until\_next\_emission}}}$$

Where:
* $\text{validators\_distribution\_pool\_balance}$ - The remaining balance in the `validators_rewards_distribution_pool`.
* $\text{bonded\_target\_factor}$ - A factor calculated with the module's params (see [below](#bondedtargetfactor)).
* $\text{remaining\_blocks\_until\_next\_emission}$ - The amount of blocks until the next monthly refill of the `validators_rewards_distribution_pool`.

### Providers Rewards Pool

Providers get rewards for their services from the funds used to buy subscriptions (see subscription module readme).

Besides these rewards, the provider also get monthly bonus rewards from a pre-allocated rewards pools. To manage this, like the validators, the providers have an allocation and distribution rewards pools. The bonus rewards exist to provide additional rewards when there is no much demand (few subscriptions).

The total monthly bonus rewards ("total spec payout") are calculated per spec by the following formula:

$$\text{Total spec payout}=\\\min\{RewardsMaxBoost\cdot\text{Total Base Payouts}, \\ \text{Spec Payout Cap}, \\ \max\{0,1.5(\text{Spec Payout Cap})-0.5(\text{Total Base Payouts})\}\}$$

Where:
* $\text{RewardsMaxBoost}$ - A module's parameter (see [below](#maxrewardsboost)).
* $\text{Total Base Payouts}$ - The sum of all base payouts the providers for this spec collected = $\sum_{provider_1.._n} (\text{{provider base rewards}}_i)$
* $\text{Spec Payout Cap}$ - The max value for this spec bonus rewards = $\frac{specStake\cdot specShares}{\sum_i{specStake \cdot specShares}_i}$.
* SpecStake = Total effective stake of providers in this spec.
* SpecShares = Weight factor for the spec (determined in each spec)

The total spec payout is distributed between providers proportional to the rewards they collected from subscriptions throughtout the month. Each provider will get bonus rewards according to the following formula:

$$Provider Bonus Rewards = Total Spec Payout \cdot \frac{\sum_{\text{payment} \; i} (\text{{provider base rewards}}_{i,j} \times \text{{adjustment}}_{i,j})}{\sum_{\text{provider}\;j'}\sum_{\text{payment} \; i}  (\text{{provider base rewards}}_{i,j'}  )}
$$

Where:
* Adjustment - TBD

Note that some of the providers rewards are sent to the community pool (according to the `CommunityTax` parameter, determined by the distribution module) and to the validators block rewards (according to the `ValidatorsSubscriptionParticipation` parameter, see [below](#validatorssubscriptionparticipation)).

The participation fees are calculated according to the following formulas:

$$\text{Validators Participation Fee} = \frac{ValidatorsSubscriptionParticipation}{1-CommunityTax}$$

$$\text{Community Participation Fee} = ValidatorsSubscriptionParticipation + CommunityTax\\ - \text{Validators Participation Fee}$$

## Parameters

The rewards module contains the following parameters:

| Key                                    | Type                    | Default Value    |
| -------------------------------------- | ----------------------- | -----------------|
| MinBondedTarget                        | math.LegacyDec          | 0.6              |
| MaxBondedTarget                        | math.LegacyDec          | 0.8              |
| LowFactor                              | math.LegacyDec          | 0.5              |
| LeftOverBurnRate                       | math.LegacyDec          | 1                |
| MaxRewardsBoost                        | uint64                  | 5                |
| ValidatorsSubscriptionParticipation    | math.LegacyDec          | 0.05             |

### MinBondedTarget

MinBondedTarget is used to calculate the [BondedTargetFactor](#bondedtargetfactor). This is the percentage of stake vs token supply that above it, reduction in validators block rewards is done.

### MaxBondedTarget

MaxBondedTarget is used to calculate the [BondedTargetFactor](#bondedtargetfactor). This is the percentage in which the validators block rewards reduction is capped and reaches LowFactor.

### LowFactor

LowFactor is used to calculate the [BondedTargetFactor](#bondedtargetfactor). This is the linear reduction rate/slope of the validators block rewards.

### LeftOverBurnRate

LeftOverBurnRate determines the percentage of tokens to burn before refilling the distribution rewards pool with the monthly quota (from the allocation pool). 
LeftOverBurnRate = 0 means there is no burn.

### BondedTargetFactor

In Lava we wanted to make validator rewards decrease linearly with the increase in stake. The BondedTargetFactor encapsulates this behaviour.

To calculate the `BondedTargetFactor` see the following formula (note that the staking module's `BondedRatio` parameter is used, which is the fraction of the staking tokens which are currently bonded):

$$\text{BondedTargetFactor}= \\\begin{cases}
1 & \text{$\text{BondRatio} < \text{MinBonded}$}\\
\frac{\text{MaxBonded}-\text{BondRatio}}{\text{MaxBonded}-\text{MinBonded}} + (\text{LowFactor})\frac{\text{BondRatio}-\text{MinBonded}}{\text{MaxBonded}-\text{MinBonded}}  & \text{$\text{MinBonded} < \text{BondRatio} < \text{MaxBonded}$}\\
\text{LowFactor} & \text{$\text{BondRatio} > \text{MaxBonded}$}\\
\end{cases}$$

### MaxRewardsBoost

TBD

### ValidatorsSubscriptionParticipation

ValidatorsSubscriptionParticipation is used to calculate the providers rewards participation fees.