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
  - [TimerStore](#timerstore)
  - [BeginBlock & EndBlock](#beginblock--endblock)
  - [BlockHeight & BlockTime](#blockheight--blocktime)
  - [Trigger](#trigger)
- [Parameters](#parameters)
- [Queries](#queries)
- [Transactions](#transactions)
- [Proposals](#proposals)

## Concepts

### TimerStore

The `TimerStore` object is the main entity of this module, and holds all of its logic.
The `timerstore` module's keeper is pretty straight forward:

```go
type Keeper struct {
	timerStoresBegin []*timerstoretypes.TimerStore
	timerStoresEnd   []*timerstoretypes.TimerStore
	cdc              codec.BinaryCodec
}
```

Whenever a module creates a new timer, it will be stored in `timerStoresBegin` or in `timerStoresEnd`, depending on the function that was used to create the timer.

### BeginBlock & EndBlock

A module can decide whether to trigger a certain timer at the start of the block or at the end of it.  
The function `NewTimerStoreBeginBlock` will create a new `TimerStore` that will trigger at the `BeginBlock` when the time is right, and the function `NewTimerStoreEndBlock` will create a new `TimerStore` that will trigger at the `EndBlock` when the time is right.

### BlockHeight & BlockTime

After calling the function `NewTimerStoreBeginBlock` or `NewTimerStoreEndBlock` the calling procedure can use the returned `TimerStore` object to set the callback of the timer. The callback has to be of the signature `func(ctx sdk.Context, key, data []byte)`, more on that in the next section.

To set the callback, the procedure needs to call `WithCallbackByBlockHeight` or `WithCallbackByBlockTime`.
The `WithCallbackByBlockHeight` function will add the callback to the `TimerStore` object, under the `BlockHeight` kind, and the `WithCallbackByBlockTime` function will add the callback to the `TimerStore` object, under the `BlockTime` kind.

The `BlockHeight` callbacks, can be triggered at a given block, and the `BlockTime` callback can be triggered at the first block that it's time is past the given time.

### Trigger

As stated in the previous section, timers can be one of two kinds: `BlockHeight` or `BlockTime`.  
Once a callback is created in the TimerStore, a procedure can then create a new timer corresponding to its kind.

For `BlockHeight` callbacks, the function `AddTimerByBlockHeight(ctx sdk.Context, block uint64, key, data []byte)` will create a timer that will trigger at `block`, with the given `key` and `data`.

For `BlockTime` callbacks, the function `AddTimerByBlockTime(ctx sdk.Context, timestamp uint64, key, data []byte)` will create a timer that will trigger at the first block, which has a time that passed the given `timestamp`, with the given `key` and `data`.

Once the timer is triggered, it is deleted. So for a recurring timer, a new timer is need to be created every time it's triggered.

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
