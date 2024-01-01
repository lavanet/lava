# `x/downtime`

## Abstract

This document specifies the downtime module of Lava Protocol.

The downtime module is responsible handling provider rewards in case of a chain halt. Lava protocol lets consumers send relays and providers to accept them even if blocks are not advancing because of consensus failure. When the blockchain is back to normal the providers should be able to receive rewards on the relays they served. The downtime is also used to determine the validators' block rewards.

## Contents
* [Parameters](#parameters)
  * [DowntimeDuration](#downtimeduration)
  * [EpochDuration](#epochduration)
* [Queries](#queries)
* [Transactions](#transactions)
* [Proposals](#proposals)

## Parameters

The downtime module contains the following parameters:

| Key                                    | Type                    | Default Value    |
| -------------------------------------- | ----------------------- | -----------------|
| DowntimeDuration                        | time.Duration          | 5min              |
| EpochDuration                        | time.Duration          | 30min              |

### DowntimeDuration

DowntimeDuration defines the minimum time elapsed between blocks that we consider the chain to be down.

### EpochDuration

EpochDuration defines an estimation of the time elapsed between epochs.

## Queries

The downtime module supports the following queries:

| Query      | Arguments       | What it does                                  |
| ---------- | --------------- | ----------------------------------------------|
| `downtime`     | block (uint64)  | shows downtime at the given epoch (the epoch of the block)                 |
| `params`   | none            | shows the module's parameters                 |

## Transactions

The downtime module does not support any transactions.

## Proposals

The downtime module does not support any proposals.
