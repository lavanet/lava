# `x/fixationstore`

## Abstract

This document specifies the `fixationstore` module of Lava Protocol.

This module primarily serves as a utility for other modules in the Lava Protocol. It's not designed for direct user interaction, except for a limited set of queries intended for debugging. As such, it functions as an essential support utility within the protocol's ecosystem.

The fixationstore allows other modules to create fixation stores that manages lists of entries with versions in the store. Fixation entries may change over time, and their versions must be retained on-chain as long as they are referenced. For examples, an older version of a plan is needed as long as the subscription that uses it lives.

## Contents

- [Concepts](#concepts)
  - [Entry](#entry)
    - [Reference Count](#reference-count)
    - [Stale Period](#stale-period)
  - [Usage](#usage)
- [Parameters](#parameters)
- [Queries](#queries)
- [Transactions](#transactions)
- [Proposals](#proposals)

## Concepts

### Entry

A fixated entry version is identified by its index (name) and block (version). The "latest" entry version is the one with the highest block that is not greater than the current (which is retrieved from the context) block height. If an entry version is in the future (with respect to current block height), then it will become the new latest entry when its block is reached.

#### Reference Count

Entry versions maintain reference count (refcount) that determine their lifetime. New entry versions (appended) start with refcount 1. The refcount of the latest version is incremented using the function `GetEntry()`. The refcount of the latest version is decremented when a newer version is appended, or when a future version becomes in effect. References taken with the function `GetEntry()` can be dropped (and the refcount decremented) by using the `PutEntry()` function. The `PutEntry()` function can also be used to cancel/remove a future entry.

#### Stale Period

When an entry's refcount reaches 0, it remains partly visible for a predefined period of blocks (stale-period, currently defined as 240 blocks), and then becomes stale and fully invisible. During the stale-period the `GetEntry()` function will ignore the entry, however the `FindEntry()` function will find it. If the nearest-no-later (then a given block) entry is already stale, the `FindEntry()` function will return not-found. Stale entries eventually get cleaned up automatically.

### Usage

Common fixation store functions:
 - `AppendEntry()` adds a new version of an entry (could be a future version).
 - `ModifyEntry()` updates an existing version of an entry (could be a future version).
 - `GetEntry()` gets the latest (up to current) version of an entry, except if in stale-period. Increases the refcount by 1.
 - `HasEntry()` checks for existence of a specific version of an entry.
 - `FindEntry()` gets the nearest-no-later version of an entry, including if in stale-period. There's no effect on the refcount.
 - `FindEntry2()` same as `FindEntry()`, and also returns the version (block) of the entry. There's no effect on the refcount.
 - `ReadEntry()` gets a specific entry version (stale or not).
 - `PutEntry()` gets the latest (up to current) version of an entry, except if in stale-period. Decreases the refcount by 1.
 - `DelEntry()` deletes and entry and make it invisible to `GetEntry()`. Calls to the `FindEntry()` function for a block beyond that time of deletion (at or later) would fail too. Note, `DelEntry()` will also discard any pending future versions of the entry.

## Parameters

The `fixationstore` module does not contain parameters.

## Queries

The `fixationstore` module supports the following queries:

| Query        | Arguments                           | What it does                                     |
| ------------ | ----------------------------------- | ------------------------------------------------ |
| `all-indices` | store-key (string), prefix (string) | Shows all entry indices of a specific fixation store       |
| `entry`       | store-key (string), prefix (string), key (string), block (uint64) | Shows a specific entry version of a specific fixation store |
| `store-keys` | none                                | Shows all timer store keys and prefixes                      |
| `versions`       | store-key (string), prefix (string), key (string) | Shows all versions of a specific entry of a specific fixation store |

## Transactions

The `fixationstore` module does not support any transaction.

## Proposals

The `fixationstore` module does not support any proposals.