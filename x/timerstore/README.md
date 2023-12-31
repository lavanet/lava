# `x/timerstore`

## Abstract

This document specifies the `timerstore` module of Lava Protocol.

This module primarily serves as a utility for other modules in the Lava Protocol. It's not designed for direct user interaction, except for a limited set of queries intended for debugging. As such, it functions as an essential support utility within the protocol's ecosystem.

The `timerstore` allows other modules to create timers, that triggers callback functions.  
A timer defined in this module can be one of two types: BlockHeight or BlockTime.  
The BlockHeight timers operate based on block counts, while BlockTime timers are measured in seconds.
The callback function can be triggered either at BeginBlock or EndBlock.

## Contents

- [Concepts](#concepts)
  - [Creating the TimerStore](#creating-the-timerstore)
    - [TimerStore Object](#timerstore-object)
    - [Initialization Methods](#initialization-methods)
    - [Setting Callbacks](#setting-callbacks)
  - [Using the TimerStore](#using-the-timerstore)
    - [Overview of Timer Usage](#overview-of-timer-usage)
    - [Timer Types](#timer-types)
    - [Adding Timers to TimerStore](#adding-timers-to-timerstore)
    - [Timer Lifecycle](#timer-lifecycle)
- [Parameters](#parameters)
- [Queries](#queries)
- [Transactions](#transactions)
- [Proposals](#proposals)

## Concepts

### Creating the TimerStore

#### TimerStore Object

The core of this module is the `TimerStore` object, encapsulating all essential logic:

```go
type Keeper struct {
	timerStoresBegin []*timerstoretypes.TimerStore
	timerStoresEnd   []*timerstoretypes.TimerStore
	cdc              codec.BinaryCodec
}
```

This structure reflects the dual nature of timer triggers: at the beginning or end of a block.

#### Initialization Methods

To initialize a `TimerStore`, use `NewTimerStoreBeginBlock` or `NewTimerStoreEndBlock`. These methods set up the store to trigger timers at either the start or end of a block, respectively.

#### Setting Callbacks

Callbacks are set using `WithCallbackByBlockHeight` for block-height-based timers or `WithCallbackByBlockTime` for time-based timers. These methods link the specified callback function to the timer, ensuring it's executed at the right moment.

**Note:** The following code snippet demonstrates a common pattern used in the initialization of a timer in the codebase. This approach typically involves creating a `TimerStore` instance and immediately configuring its callback function, streamlining the setup process:

```go
callbackFunction := func(ctx sdk.Context, key, data []byte) {
	// callback logic here
}

timerStore := timerStoreKeeper.NewTimerStoreBeginBlock(storeKey, timerPrefix).
	WithCallbackByBlockTime(callbackFunction)
```

### Using the TimerStore

#### Overview of Timer Usage

After establishing the `TimerStore`, it's used to create and manage individual timers. This section delves into the operational aspect of the `TimerStore`.

#### Timer Types

There are two types of timers:

- `BlockHeight` timers trigger based on the count of blocks.
- `BlockTime` timers operate based on elapsed time in seconds.

Each type serves different use cases, offering flexibility in scheduling tasks.

#### Adding Timers to TimerStore

Timers are added to the `TimerStore` via `AddTimerByBlockHeight` and `AddTimerByBlockTime`. These functions create timers that will trigger either at a specified block height or after a set time:

```go
// Add a BlockHeight timer
AddTimerByBlockHeight(ctx sdk.Context, block uint64, key, data []byte)

// Add a BlockTime timer
AddTimerByBlockTime(ctx sdk.Context, timestamp uint64, key, data []byte)

```

#### Timer Lifecycle

Once set, a timer remains active until triggered, after which it's automatically deleted. For recurring events, it's necessary to create a new timer each time.

## Parameters

The `timerstore` module does not contain parameters.

## Queries

The `timerstore` module supports the following queries:

| Query        | Arguments                           | What it does                                     |
| ------------ | ----------------------------------- | ------------------------------------------------ |
| `all-timers` | store-key (string), prefix (string) | Shows all timers of a specific timer store       |
| `next`       | store-key (string), prefix (string) | Shows the next timeout of a specific timer store |
| `store-keys` | none                                | Shows all timer store keys                       |

## Transactions

The `timerstore` module does not support any transaction.

## Proposals

The `timerstore` module does not support any proposals.
