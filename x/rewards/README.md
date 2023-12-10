---
sidebar_position: 1
---

# `x/rewards`

## Abstract

This document specifies the rewards module of Lava Protocol.

The rewards module is responsible for distributing rewards to validators and providers. Rewards are collected from the validators' and providers' pools, respectively. The pools' funds originate from the treasury account - a continuous vesting account that holds all the pre-allocated Lava tokens of the network. It is important to note that the pre-allocated amount is constant, and there will be no minting of new Lava tokens in the future.

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

As per Lava's vision, there is a constant supply of Lava tokens from the chain's genesis. There will be no minting of new tokens in the future like other Cosmos-SDK based chains.

The treasury account holds all of Lava's pre-allocated tokens. It's a continous vesting account which releases tokens continuously over time, linearly, to token pools that are used to reward validators and providers. The vesting period is 4 years minus one month.

### Validators Rewards Pool

In Cosmos-SDK chains, validator get rewards for creating blocks by the Cosmos' distribution module. Each new block new tokens are minted and distributed to all eligible accounts for the new block creation.

In Lava, the Cosmos validators reward distribution mechanism remains as is, but the source of the tokens comes from the rewards module (and not the mint module). The rewards module uses two pools to manage rewards for the validators: `validators_pool` and `validators_block_pool`.

The `validators_pool` is the recepient pool of the vesting from the treasury. In other words, it's continously receiving funds from the treasury account.

The `validators_block_pool` is the pool used to reward validators for block creation. Once a month, the `validators_block_pool` gets month-worth rewards from the `validators_pool`. These funds are used to reward validators for each block. Note that before the `validators_pool` transfers the funds, the `validators_block_pool`'s tokens are burned according to the `LeftOverBurnRate` parameter (see below). Same as Cosmos' mint module, the `validators_block_pool` transters the block rewards to the fee collection account which is used by the distribution module to reward the validators.

Validators block reward is calculated using the following formula:

$$ Reward = \frac{{\text{{validators\_block\_pool\_balance}}} \cdot {\text{bonded\_target\_factor}}}{\text{{remaining\_blocks\_until\_next\_emission}}}$$

Where:
* $\text{validators\_block\_pool\_balance}$ - The remaining balance in the `validators_block_pool`.
* $\text{bonded\_target\_factor}$ - A factor calculated with the module's params. TBD (better explain).
* $\text{remaining\_blocks\_until\_next\_emission}$ - The amount of blocks until the next monthly refill of the `validators_block_pool`.

### Providers Rewards Pool

TBD

## Parameters

The rewards module contains the following parameters:

| Key                | Type            | Default Value |
| ------------------ | --------------- | ------------- |
| MinBondedTarget    | math.LegacyDec  | 0.6           |
| MaxBondedTarget    | math.LegacyDec  | 0.8           |
| LowFactor          | math.LegacyDec  | 0.5           |
| LeftOverBurnRate   | math.LegacyDec  | 1             |

### MinBondedTarget

TBD

### MaxBondedTarget

TBD

### LowFactor

TBD

### LeftOverBurnRate

Value between 0-1 that determines the tokens to burn when refilling the validators block rewards pool. 
LeftOverBurnRate = 0 means there is no burn.