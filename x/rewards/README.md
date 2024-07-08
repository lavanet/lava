# `x/rewards`

## Abstract

This document specifies the rewards module of Lava Protocol.

The rewards module is responsible for distributing rewards to validators and providers. Rewards are collected from the validators' and providers' pools, respectively. The pools' funds originate from the treasury account - a module account that holds all the pre-allocated Lava tokens of the network. It is important to note that the pre-allocated amount is constant, and there will be no minting of new Lava tokens in the future.

The rewards module also operates the Incentivized Providers RPC program (IPRPC). This program distributes bonus rewards to providers servicing specific chains. The IPRPC rewards are funded by the chains' DAO and other contributors.

Please note that this module replaces Cosmos SDK's mint module, which is typically responsible for minting rewards for validators.

## Contents

* [Concepts](#concepts)
  * [The Treasury](#the-treasury)
  * [Rewards Pools](#rewards-pools)
    * [Validators Rewards Pool](#validators-rewards-pool)
    * [Providers Rewards Pool](#providers-rewards-pool)
  * [IPRPC](#iprpc)
    * [IPRPC over IBC](#iprpc-over-ibc)
* [Parameters](#parameters)
  * [MinBondedTarget](#minbondedtarget)
  * [MaxBondedTarget](#maxbondedtarget)
  * [LowFactor](#lowfactor)
  * [LeftOverBurnRate](#leftoverburnrate)
  * [MaxRewardsBoost](#maxrewardsboost)
  * [ValidatorsSubscriptionParticipation](#validatorssubscriptionparticipation)
  * [IbcIprpcExpiration](#ibciprpcexpiration)
* [Queries](#queries)
* [Transactions](#transactions)
* [Proposals](#proposals)

## Concepts

### The Treasury

As per Lava's vision, there is a constant supply of Lava tokens from the chain's genesis. There will be no minting of new tokens in the future like other Cosmos-SDK based chains. The treasury account holds all of Lava's pre-allocated tokens. The total amount of coins will be $10^9 * 10^6$ ulava.

### Rewards Pools

The rewards module is responsible on distributing rewards for validators that create new blocks, and providers that provide RPC service for their consumers.

To manage the rewards, the module uses two rewards pools: allocation pools and distribution pools. Note that there are two allocation pools and two distribution pools to manage the validators and providers rewards independently.

The allocation pools get some of the treasury account's tokens and hold onto it. Once a month, the allocation pools transfer a fixed amount of funds to the distribution pools. This monthly transfer will last for 4 years, after which the allocation pools' funds will be depleted (the allocation pools lifetime (4 years) is a pre-defined constant in the module's code). Note that before the allocation pools transfer the funds, the distribution pools' tokens are burned according to the `LeftOverBurnRate` parameter (see [below](#leftoverburnrate)).

The distribution pools use the monthly quota of funds to distribute rewards for validators and providers.

#### Validators Rewards Pool

In Cosmos-SDK chains, validators get rewards for creating blocks by the Cosmos' distribution module. For each new block, new tokens are minted and distributed to all the accounts that contributed to the new block creation.

In Lava, the Cosmos validators reward distribution mechanism remains as is, but the source of the tokens comes from the rewards module (and not the mint module). As stated above, the rewards are managed using two pools: an allocation pool and a distribution pool.

The allocation pool of validators, `validators_rewards_allocation_pool`, gets 3% of the tokens of the treasury account.

The distribution pool of validators, `validators_rewards_distribution_pool`, uses its funds to reward validators for proposing new blocks. Same as Cosmos' mint module, the `validators_rewards_distribution_pool` transfers the block rewards to the fee collection account which is used by Cosmos' distribution module to reward the validators.

The validators block reward is calculated using the following formula:

```math
Reward = \frac{{\text{{validators\_distribution\_pool\_balance}}} \cdot {\text{bonded\_target\_factor}}}{\text{{remaining\_blocks\_until\_next\_emission}}}
```
Where:
* $`\text{validators\_distribution\_pool\_balance}`$ - The remaining balance in the `validators_rewards_distribution_pool`.
* $`\text{bonded\_target\_factor}`$ - A factor calculated with the module's params (see [below](#bondedtargetfactor)).
* $`\text{remaining\_blocks\_until\_next\_emission}`$ - The amount of blocks until the next monthly refill of the `validators_rewards_distribution_pool`.

#### Providers Rewards Pool

Providers get rewards for their services from the funds used to buy subscriptions (see subscription module readme).

Besides these rewards, the provider also get monthly bonus rewards from a pre-allocated rewards pools. To manage this, like the validators, the providers have an allocation and distribution rewards pools. The bonus rewards exist to provide additional rewards when there is no much demand (few subscriptions).

The total monthly bonus rewards ("total spec payout") are calculated per spec by the following formula:

```math
\text{Total spec payout}=\\\min\{MaxRewardsBoost\cdot\text{Total Base Payouts}, \\ \text{Spec Payout Cap}, \\ \max\{0,1.5(\text{Spec Payout Cap})-0.5(\text{Total Base Payouts})\}\}
```

Where:
* $`\text{MaxRewardsBoost}`$ - A module's parameter (see [below](#maxrewardsboost)).
* $`\text{Total Base Payouts}`$ - The sum of all base payouts the providers for this spec collected = $`\sum_{provider_1.._n} (\text{{provider base rewards}}_i)`$
* $`\text{Spec Payout Cap}`$ - The max value for this spec bonus rewards = $`\frac{specStake\cdot specShares}{\sum_i{specStake \cdot specShares}_i}`$.
* SpecStake = Total effective stake of providers in this spec.
* SpecShares = Weight factor for the spec (determined in each spec)

The total spec payout is distributed between providers proportional to the rewards they collected from subscriptions throughout the month. Each provider will get bonus rewards according to the following formula:

```math
Provider Bonus Rewards = Total Spec Payout \cdot \frac{\sum_{\text{payment} \; i} (\text{{provider base rewards}}_{i,j} \times \text{{adjustment}}_{i,j})}{\sum_{\text{provider}\;j'}\sum_{\text{payment} \; i}  (\text{{provider base rewards}}_{i,j'}  )}
```

Note that some of the providers rewards are sent to the community pool (according to the `CommunityTax` parameter, determined by the distribution module) and to the validators block rewards (according to the `ValidatorsSubscriptionParticipation` parameter, see [below](#validatorssubscriptionparticipation)). For more details about the `adjustment`, see [below](#adjustment).

The participation fees are calculated according to the following formulas:

```math
\text{Validators Participation Fee} = \frac{ValidatorsSubscriptionParticipation}{1-CommunityTax}
```

```math
\text{Community Participation Fee} = ValidatorsSubscriptionParticipation + CommunityTax\\ - \text{Validators Participation Fee}
```

#### Adjustment

Adjustment is a security mechanism that prevents collusion and abuse of rewards boost. It calculates the distribution of usage across multiple providers for each consumer. The more scattered a consumer's usage is across providers, the higher the bonus rewards they can receive. The adjustment value is always smaller or equal to one and can reduce the bonus rewards by up to $\frac{1}{MaxRewardsBoost}$.

The adjustment is calculated per epoch using a weighted average based on usage. It considers the consumer's CU (compute units) used with a specific provider relative to the total CU the consumer used. The formula is as follows:

```math
\text{epochAdjustment}_\text{consumet i, provider j}=\\\min\{1,\;\;\;minAdjustment\cdot(\frac{\sum_{\text{provider }k}{CU_\text{i,k}}}{CU _\text{i,j}})\}
```

```math
\text{monthlyAdjustment}_{i,k} = \frac{\sum_{\text{epoch}\; t }\left(CU_\text{i}(t)\cdot \text{epochAdjustment}_{i,k}(t)\right)}{\sum_{\text{epoch}\; t}CU_{i}(t)}
```

### IPRPC

IPRPC (Incentivized Providers RPC) is a program designed to motivate providers to offer services for specific chains using Lava. Providers supplying RPC data for chain X will receive additional rewards from the IPRPC pool each month. This pool is funded by chain X's DAO and other contributors.

The IPRPC pools will hold both Lava's native tokens and IBC wrapped tokens. Providers' rewards will be based on the amount of CU they serviced during the month relative to other providers' serviced CU. Only the CU from eligible subscriptions will be counted. IPRPC eligible subscriptions are determined via a government proposal.

Users have the freedom to fund the pool with any token of their choice, for any supported chain and set the emission schedule for their funds. For instance, a user can fund the IPRPC pool for the Osmosis chain with 300ulava, which will be dispersed over a period of three months. It's important to note that the IPRPC pool will start receiving funds from the beginning of the upcoming month.

In order to fund the IPRPC pool, the user's funds must cover the monthly minimum IPRPC cost, a parameter determined by governance. This minimum IPRPC cost will be subtracted from the monthly emission.

#### IPRPC over IBC

To ease the funding process of funding the IPRPC pool, users can send an `ibc-transfer` transaction to fund the IPRPC pool directly from the source chain. For example, using IPRPC over IBC allows funding the IPRPC pool by sending a single `ibc-transfer` to an Osmosis node.

To make the `ibc-transfer` transaction fund the IPRPC pool, it must contain a custom memo with the following format:

```json
{
  "iprpc": {
    "creator": "alice",
    "spec": "ETH1",
    "duration": "3"
  }
}
```

The creator field can be any name, the spec field must be an on-chain spec ID, and the duration should be the funding period (in months). Users can use the `generate-ibc-iprpc-tx` helper command with the `--memo-only` flag to generate a valid IPRPC memo. If the flag is not included, this command creates a valid `ibc-transfer` transaction JSON with a custom memo. The only remaining step is to sign and send the transaction.

As mentioned above, funding the IPRPC pool requires a fee derived from the monthly minimum IPRPC cost. Because of that, all IPRPC over IBC requests are considered a pending IPRPC fund request until the fee is paid. To view the current pending requests, users can use the `pending-ibc-iprpc-funds` query and cover the minimum cost using the `cover-ibc-iprpc-fund-cost` transaction. The expiration of pending IPRPC fund requests is dictated by the `IbcIprpcExpiration`, which equals the expiration period in months.

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
| IbcIprpcExpiration                     | uint64                  | 3                |

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

```math
\text{BondedTargetFactor}= \\\begin{cases}
1 & \text{$\text{BondRatio} < \text{MinBonded}$}\\
\frac{\text{MaxBonded}-\text{BondRatio}}{\text{MaxBonded}-\text{MinBonded}} + (\text{LowFactor})\frac{\text{BondRatio}-\text{MinBonded}}{\text{MaxBonded}-\text{MinBonded}}  & \text{$\text{MinBonded} < \text{BondRatio} < \text{MaxBonded}$}\\
\text{LowFactor} & \text{$\text{BondRatio} > \text{MaxBonded}$}\\
\end{cases}
```

### MaxRewardsBoost

MaxRewardsBoost is a multiplier that determines the maximum bonus for provider rewards from subscriptions.

### ValidatorsSubscriptionParticipation

ValidatorsSubscriptionParticipation is used to calculate the providers rewards participation fees.

### IbcIprpcExpiration

IbcIprpcExpiration determines the pending IPRPC over IBC fund requests expiration in months.

## Queries

The rewards module supports the following queries:

| Query      | Arguments       | What it does                                  |
| ---------- | --------------- | ----------------------------------------------|
| `block-reward`     | none  | show the block reward for validators                  |
| `pools`     | none            | show the info of the distribution pools  |
| `params`   | none            | shows the module's parameters                 |
| `show-iprpc-data`   | none            | shows the IPRPC data that includes the minimum IPRPC cost and a list of IPRPC eligible subscriptions                 |
| `iprpc-provider-reward`   | provider (string)            | shows the estimated IPRPC rewards for a specific provider (relative to its serviced CU) for the upcoming monthly emission                 |
| `iprpc-spec-rewards`   | spec (string, optional)            | shows a specific spec's IPRPC rewards (for the entire period). If no spec is given, all IPRPC rewards are shown                 |
| `generate-ibc-iprpc-tx`   | spec (string), duration (uint64, in months), amount (sdk.Coin, optional), src-port (string, optional), src-channel (string, optional), --from (string, mandatory), --node (string, mandatory), --memo-only (optional)            | generates an `ibc-transfer` transaction JSON that can be used to fund the IPRPC pool over IBC. To generate only the custom memo that triggers the IPRPC funding, use the `--memo-only` flag.                 |
| `pending-ibc-iprpc-funds`   | filter (string, optional)            | Lists the pending IPRPC over IBC fund requests. Use the optional filter to filter by index, creator, or spec. Each entry is shown with the minimum IPRPC cost that needs to be paid to apply it.                  |

Note, use the provider's address for the `iprpc-provider-reward` query. For more information on the provider's two addresses (regular and vault) see the epochstorage module's [README.md](../epochstorage/README.md).

## Transactions

All the transactions below require setting the `--from` flag and gas related flags.

The rewards module supports the following transactions:

| Transaction      | Arguments       | What it does                                  |
| ---------- | --------------- | ----------------------------------------------|
| `fund-iprpc`     | spec (string), duration (uint64, in months), coins (sdk.Coins, for example: `100ulava,50ibctoken`)  | fund the IPRPC pool to a specific spec with ulava or IBC wrapped tokens. The tokens will be vested for `duration` months.                  |
| `cover-ibc-iprpc-fund-cost`     | index (uint64)  | cover the costs of a pending IPRPC over IBC fund request by index. The required minimum IPRPC cost is automatically paid by the transaction sender.                  |

Please note that the coins specified in the `fund-iprpc` transaction is the monthly emission fund. For instance, if you specify a fund of 100ulava for a duration of three months, providers will receive 100ulava each month, not 33ulava.

Also, the coins must include `duration*min_iprpc_cost` of ulava tokens (`min_iprpc_cost` is shown with the `show-iprpc-data` query).

## Proposals

The rewards module supports the `set-iprpc-data` proposal which set the minimum IPRPC cost and the IPRPC eligible subscriptions.

To send a proposal and vote use the following commands:

```
lavad tx gov submit-legacy-proposal set-iprpc-data <deposit> --min-cost 0ulava --add-subscriptions addr1,addr2 --remove-subscriptions addr3,addr4 --from alice <gas-flags>

lavad tx gov vote <latest_proposal_id> yes --from alice <gas-flags>
```

## Events

The rewards module has the following events:

| Event      | When it happens       |
| ---------- | --------------- |
| `distribution_pools_refill`     | a successful distribution rewards pools refill   |
| `provider_bonus_rewards`     | a successful distribution of provider bonus rewards   |
| `iprpc_pool_emmission`     | a successful distribution of IPRPC bonus rewards   |
| `iprpc_pool_emmission`     | a successful distribution of IPRPC bonus rewards   |
| `set_iprpc_data`     | a successful setting of IPRPC data   |
| `fund_iprpc`     | a successful funding of the IPRPC pool   |
| `transfer_iprpc_reward_to_next_month`     | a successful transfer of the current month's IPRPC reward to the next month. Happens when there are no providers eligible for IPRPC rewards in the current month   |
| `pending_ibc_iprpc_fund_created`     | a successful IPRPC over IBC fund request which creates a pending fund request entry on-chain   |
| `cover_ibc_iprpc_fund_cost`     | a successful cost coverage of a pending IPRPC fund request that applies the request and funds the IPRPC pool   |
| `expired_pending_ibc_iprpc_fund_removed`     | a successful removal of an expired pending IPRPC fund request   |