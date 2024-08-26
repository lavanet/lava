# `x/epochstorage`

## Abstract

This document specifies the epochstorage module of Lava Protocol.

Lava blocks are divided into bulk of blocks called epochs. epoch length is defines as a parameter of the module.
The module keeps track of the epochs in memory, epochs that need to be deleted and anchoring data to each epoch, mainly providers stakes.
Note that the module will make sure that any changes will be applied only in the next epoch.

## Contents
* [Concepts](#concepts)
  * [Epoch](#epoch)
    * [EpochDetails](#epochdetails)
  * [FixatedParams](#fixatedparams)
  * [StakeEntry](#stakeentry)
	* [StakeEntry Storage](#stakeentry-storage)
* [Parameters](#parameters)
* [Queries](#queries)
* [Transactions](#transactions)
* [Proposals](#proposals)
* [Events](#events)

## Concepts

### Epoch

An epoch is defined as a batch of blocks. Within each epoch, the data that influences pairing remains constant. For instance, the providers and their data remain unchanged across all blocks within the epoch. Data can only be modified in the subsequent epoch.

#### EpochDetails

This object shows the details of the current epoch, the last epoch in memory and the last deleted epochs.

```go
type EpochDetails struct {
	StartBlock    uint64   // start of the current epoch
	EarliestStart uint64   // earliest epoch that is saved on memory
	DeletedEpochs []uint64 // list of deleted epochs (updates each epoch)
}
```

### FixatedParams

Some parameters on the chain can affect the results of the pairing list. Since the pairing list needs to remain the same throughout the epoch, the chain saves the necessary parameters for each epoch. This is achieved through fixated parameters. Each module with a parameter that affects the pairing list should register the parameter as a fixated parameter using the following method of the keeper:

```go
AddFixationRegistry(fixationKey string, getParamFunction func(sdk.Context) any) 
```

By doing this, the module will be able to track [param change proposals](../spec/proposal_handler.go) and save any changes to the relevant param as a fixated param in the store. The fixated param is responsible for keeping track of param changes, saving different versions if they are registered, and deleting old versions when they are no longer needed (as determined by the EpochsToSave parameter).
fixated param is in charge of tracking param changes, save their different versions if they are registered and delete old versions when they are not needed anymore (determined by EpochsToSave parameter).

A fixation parameter is saved on store as:

```go

type FixatedParams struct {
	Index         string // the index of the param
	Parameter     []byte // serialized data of the param
	FixationBlock uint64 // the block at which the param has changed
}
```

This is done in the [BeginBlock method of the module](keeper/fixated_params.go)

### StakeEntry

The stake entry is a struct that contains all the information of a provider.

```go
type StakeEntry struct {
	Stake              types.Coin // the providers stake amount (self delegation)
	Vault              string     // the lava address of the provider's vault which holds most of its funds
	Address            string     // the lava address of the provider is used to run and operate the provider process
	StakeAppliedBlock  uint64     // the block at which the provider is included in the pairing list
	Endpoints          []Endpoint // the endpoints of the provider
	Geolocation        int32      // the geolocation this provider supports
	Chain              string     // the chain ID on which the provider is staked on
	Moniker            string     // free string description
	DelegateTotal      types.Coin // total delegation to the provider (without self delegation)
	DelegateLimit      types.Coin // delegation total limit
	DelegateCommission uint64     // commission from delegation rewards
}
```

The provider entity utilizes two different addresses: the operator address ("address" field in the StakeEntry protobuf) and the vault address. The operator address is used when running the provider process, while the vault address holds the provider's funds. This separation enhances security by allowing the user to store the vault address' private key on a different machine from the one running the visible provider process.

Despite the provider being operated via the operator address, all rewards are directed to the vault address, which holds the provider's funds. Most provider-related transactions can be executed using the operator address, except for actions like staking/unstaking, changing delegation limit and commission, and claiming rewards. These actions can only be done by the vault address.

It's important to note that the operator address is used for the pairing mechanism and relay payments. Also, note that specifying a vault address when staking a provider is optional. By default, the same address is used for both operating the provider and holding its funds.

Geolocation are bit flags that indicate all the geolocations that the provider supports, this is the sum of all endpoints geolocations.
for more about [geolocation](../../proto/lavanet/lava/plans/plan.proto).

For more information about delegation, go to dualstaking [README.md](../dualstaking/README.md).

#### StakeEntry Storage

Stake entries are kept in two dedicated KV stores. One store keeps the current stake entries and the other keeps past stake entries (by epoch).

The current stake entries store is updated when a provider stakes itself to a chain (using the pairing module). This will make it be in the providers list of the next epoch.

When a new epoch starts, the epochstorage module will create a copy of the current stake entries and save it in the past stake entries store with their matching epoch. Then, every outdated stake entry is deleted (determined by the param EpochsToSave).

### EndPoint

Endpoint is a struct that describes how to connect and receive services from the provider for the specific interface and geolocation

```go
type Endpoint struct {
	IPPORT        string   // the ip and port where the consumer can access the provider
	Geolocation   int32    // the geolocation of the endpoint
	Addons        []string // the supported addons of the endpoint
	ApiInterfaces []string // the supported interfaces by the enpoint
	Extensions    []string // the supported extensions of the endpoint
}
```

for more info on `Addons` and `Extensions` look at the spec module [README.md](../spec/README.md).

## Parameters

The epochstorage parameters:

| Key                                    | Type                    | Default Value    |
| -------------------------------------- | ----------------------- | -----------------|
| EpochBlocks                            | uint64                  | 30               |
| EpochsToSave                           | uint64                  | 20               |
| LatestParamChange                      | uint64                  | N/A               |

`EpochBlocks` determines the amount of blocks for each epoch.

`EpochsToSave` determines how many epochs are saved on the chain at a given moment.

`LatestParamChange` saves the last time a fixated param has changed.

## Queries

The epochstorage module supports the following queries:

| Query                 | Arguments         | What it does                                  |
| ----------            | ---------------   | ----------------------------------------------|
| `show-epoch-details`  | none              | the current epoch details                     |
| `params`              | none              | show the params of the module                 |
| `list-fixated-params` | none              | list of all fixated params indices            |
| `show-fixated-params` | chainid           | a specific fixated param                      |
| `list-stake-storage`  | chainid           | list of all stake storages indices            |
| `show-stake-storage`  | chainid           | show a specific stake storage                 |

## Transactions

The epochstorage module does not support any transactions.

## Proposals

The epochstorage module does not have gov proposals.


### Events

The plans module has the following events:
| Event                     | When it happens       |
| ----------                | --------------- |
| `new_epoch`               | a new epoch has started |
| `earliest_epoch`          | new earliest epoch block   |
| `fixated_params_change`   | a new fixated param has changes|
| `fixated_params_clean`    | a fixated param is ooutdated and deleted|