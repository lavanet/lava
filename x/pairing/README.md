# `x/pairing`

## Abstract

This document specifies the pairing module of Lava Protocol.

The pairing module is responsible for handling providers staking, providers freezing, calculating the providers pairing list for consumers, punishing unresponsive providers and handling providers relay payment requests.

The Pairing Engine is a sophisticated algorithmic mechanism designed to match consumers with the most appropriate service providers, considering a wide array of inputs and constraints. It upholds the network's principles of fairness, security, and quality of service, while ensuring each consumer's experience is personalized and in line with their specific requirements.

The pairing module is one of Lava's core modules and is closely connected to the subscription, dualstaking and epochstorage modules. To fully understand they all work together, please also refer to their respective READMEs.

## Contents

* [Concepts](#concepts)
  * [Providers](#providers)
    * [Stake](#stake)
    * [Unstake](#unstake)
    * [Freeze](#freeze)
  * [Pairing](#pairing)
    * [Filters](#filters)
    * [Scores](#scores)
    * [Quality Of Service](#quality-of-service)
	  * [Reputation](#reputation)
	  * [Passable QoS](#passable-qos)
    * [Pairing Verification](#pairing-verification)
    * [Unresponsiveness](#unresponsiveness)
    * [Static Providers](#static-providers)
  * [Payments](#payments)
    * [CU Tracking](#cu-tracking)
    * [Providers Payment](#providers-payment)
    * [Storage](#storage)
* [Parameters](#parameters)
  * [QoSWeight](#qosweight)
  * [EpochBlocksOverlap](#epochblocksoverlap)
  * [RecommendedEpochNumToCollectPayment](#recommendedepochnumtocollectpayment)
* [Queries](#queries)
* [Transactions](#transactions)
* [Proposals](#proposals)
* [Events](#events)

## Concepts

### Providers

Providers are entities that have access to blockchains and want to monetize that access by providing services to consumers. Providers stake tokens under a geolocation and supported specifications (e.g. Ethereum JSON-RPC in U.S. East), once active they must provide service to consumers. Providers run the lava process and the desired blockchain or service (e.g. Ethereum JSON-RPC) they are providing access for.

Note, a provider stakes its funds using its vault address and runs the Lava provider process using the its provider address. All the provider's rewards are sent to its vault address. In case a provider address is defined (which is different from the vault address),
it is recommended to let the provider use the vault address' funds for gas fees with the appropriate flag (see below).

#### Stake

When a provider stakes, a new stake entry is created on-chain. A stake entry is defined as follows:

```go
type StakeEntry struct {
	Stake              types.Coin // the providers stake amount (self delegation)
	Vault              string     // the lava address of the provider's vault which holds most of its funds
	Address           string      // the lava address of the provider entity's which is used to run and operate the provider process
	StakeAppliedBlock  uint64     // the block at which the provider is included in the pairing list
	Endpoints          []Endpoint // the endpoints of the provider
	Geolocation        int32      // the geolocation this provider supports
	Chain              string     // the chain ID on which the provider staked on
	Moniker            string     // free string description
	DelegateTotal      types.Coin // total delegation to the provider (without self delegation)
	DelegateLimit      types.Coin // delegation total limit
	DelegateCommission uint64     // commission from delegation rewards
	Jails              uint64     // number of times the provider has been jailed
	JailTime           int64      // the end of the jail time, after which the provider can return to service
}
```

To bolster security, a provider entity is now associated with two addresses: the vault address and the provider address. If the provider entity doesn't specify the provider address when staking, it defaults to being the same as the vault address. When a provider entity stakes, the account from which the funds originate is considered the vault address. This address is utilized to hold the provider entity's funds and to receive rewards from the provider entity's service. Any other actions performed by the provider entity utilize the provider entity's provider address. The provider address can perform all actions except for staking/unstaking, modifying stake-related fields in the provider entity's stake entry, and claiming rewards. To let the provider address use the vault's funds for gas fees, use the `--grant-provider-gas-fees-auth`. The only transactions that are funded by the vault are: `relay-payment`, `freeze`, `unfreeze`, `modify-provider`, `detection` (conflict module), `conflict-vote-commit` and `conflict-vote-reveal`. When executing any of these transactions using the CLI with the provider entity, use the `--fee-granter` flag to specify the vault address which will pay for the gas fees. It's important to note that once an provider address is registered through a provider entity's staking, it cannot stake on the same chain again.

Note, the `Coin` type is from Cosmos-SDK (`cosmos.base.v1beta1.Coin`). A provider can accept delegations to increase its effective stake, which increases its chances of being selected in the pairing process. The provider can also set a delegation limit, which determines the maximum value of delegations they can accept. This limit is in place to prevent delegators from increasing the provider's effective stake to a level where the provider is overwhelmed with more consumers than they can handle in the pairing process. For more details about delegations, refer to the dualstaking module README.

* `DelegateCommission` and `DelegateLimit` changes for existing providers are limited as follows: limitations are applied only for providers that have delegations. limitations are on decreasing `DelegateLimit` and/or increasing `DelegateCommission`, limits are changes up to 1% of the original value and once per 24H.

An provider's endpoint is defined as follows:

```go
type Endpoint struct {
	IPPORT        string    // IP + port
	Geolocation   int32     // supported geolocation
	Addons        []string  // list of supported add-ons
	ApiInterfaces []string  // list of supported API interfaces
	Extensions    []string  // list of supported extensions
}
```

A consumer sends requests to a provider's endpoint to communicate with them. The aggregated geolocations of all the provider's endpoints should be equal to the geolocation field present in the provider's stake entry. Other than that, the `Addons`, `ApiInterfaces` and `Extensions` specify the "extra" features the provider supports. To get more details about those, see the spec module README.

Stake entries' storage is managed by the epochstorage module. For more details, see its README.

#### Unstake

A provider can unstake and retrieve their coins. When a provider unstakes, they are removed from the pairing list starting from the next epoch. After a specified number of blocks called `UnstakeHoldBlocks` (a parameter of the epochstorage module), the provider is eligible to receive their coins back.

#### Freeze

Freeze Mode enables the Provider to temporarily suspend their node's operation during maintenance to avoid bad Quality of Service (QoS). Freeze/Unfreeze is applied on the next Epoch. The Provider can initiate multiple Freeze actions with one command.

### Pairing

The Pairing Engine is a core component of the Lava Network, responsible for connecting consumers with the most suitable service providers. It operates on a complex array of inputs, including the strictest policies defined at the plan, subscription, and project levels. These policies set the boundaries for service provisioning, ensuring that consumers' specific requirements are met while adhering to the network's overarching rules.

The Pairing Engine takes into account a diverse range of parameters to make informed matchmaking decisions. This includes a list of available providers, each with their unique service offerings, preferred geolocation, available addons and extensions, and additional factors. Importantly, this pairing mechanism is deterministic, and recalculates the pairing list every epoch.

Furthermore, the Pairing Engine has the ability to filter out irrelevant providers, such as those that may be frozen or jailed due to non-compliance with network rules. Additionally, it factors in the stake held by each provider, giving those with higher stakes a greater chance of being selected.

The selection process is also enriched by pseudorandomization, enhancing the security and unpredictability of the pairings. A changing seed, derived from the Lava blockchain hashes, is introduced to the selection algorithm. This seed is unique per consumer, achieved by incorporating the consumer's address as a "salt," which ensures that each consumer's pairings are distinct and less susceptible to prediction or manipulation.

Finally, providers are assigned scores based on QoS metrics, including latency, availability, and synchronization. This scoring system ensures that the Pairing Engine selects providers that best match the specific QoS requirements of the consumer's cluster. Providers with superior QoS scores receive preference in the pairing process.

#### Filters

When calculating the pairing list of providers for a consumer, filters are applied to exclude unwanted providers. These providers may be frozen, jailed, not supporting the required geolocation, and more. All filters are initialized according to the effective (strictest) policy of the consumer's project.

The current pairing filters are:

| Filter            | What it does                                  |
| ---------- | ----------------------------------------------|
| `Freeze`      | excludes frozen providers                   |
| `Add-on`      | excludes providers not supporting required add-ons                   |
| `Geolocation`      | excludes providers not supporting required geolocations                   |
| `Selected providers`      | excludes providers that are not in the policy's selected providers allow-list                   |

Some filters support a "mix" behavior, where the filter does not exclude all providers, but instead creates a mix of providers that pass the filter and providers that do not pass the filter.

All filters are implementing the following interface:

```go
type Filter interface {
	Filter(ctx sdk.Context, providers []epochstoragetypes.StakeEntry, currentEpoch uint64) []bool
	InitFilter(strictestPolicy planstypes.Policy) bool // return if filter is usable (by the policy)
	IsMix() bool
}
```

#### Scores

The scoring mechanism is used to select providers during the pairing process. This mechanism is applied after the full list of providers has been filtered using the pairing filters.

The pairing scoring process involves the following steps:
  1. Collect pairing requirements and strategy from the policy.
  2. Generate pairing slots with requirements (one slot per provider).
  3. Compute the pairing score of each provider with respect to each slot.
  4. Pick a provider for each slot with a pseudo-random weighted choice.

Pairing requirements describe the policy-imposed requirements for paired providers. Those include geolocation constraints, stake amount, and expectations regarding QoS ranking of selected providers. Pairing requirements must satisfy the `ScoreReq` interface:

```go
type ScoreReq interface {
	Init(policy planstypes.Policy) bool
	Score(score PairingScore) math.Uint // calculates the score of a provider for a specific score requirement
	GetName() string
	Equal(other ScoreReq) bool
	GetReqForSlot(policy planstypes.Policy, slotIdx int) ScoreReq // returns the score requirement for a slot given policy limitations
}
```

A pairing slot represents a single provider slot in the pairing list (The number of slots for pairing is defined by the policy). Each pairing slot holds a set of pairing requirements (a pairing slot may repeat). For example, a policy may state that the pairing list has 6 slots, and providers should be located in Asia and Europe. This can be satisfied with a pairing list that has 3 (identical) slots that require providers in Asia and 3 (identical) slots that require providers in Europe. A pairing slot is defined as follows:

```go
type PairingSlot struct {
	Reqs  map[string]ScoreReq // example: "geoReq": geo Score req object with EU requirement
	Index int
}
```

The `Reqs` map holds pairing requirements names that point to ScoreReq object. Continuing the example from above, slot A's map can contain: `"geoReq": GeoReq{geo: "EU"}` and slot B's map can contain: `"geoReq": GeoReq{geo: "AS"}` (the map is filled according to `GetReqForSlot()`'s output).

A pairing score describes the suitability of a provider for a pairing slot (under a given strategy). The score depends on the slot's requirements: for example, given a slot which requires geolocation in Asia, a provider in Asia will generally get higher score than one in Europe. The score is calculated for each <provider, slot> combination.

A pairing score is defined as follows:

```go
type PairingScore struct {
	Provider            *StakeEntry
	Score               math.Uint  // total score according to all the pairing requirements
	ScoreComponents     map[string]math.Uint  // score components by pairing requirements
	SkipForSelection    bool
	SlotFiltering       map[int]struct{} // slot indexes here are skipped
	QosExcellenceReport pairingtypes.QualityOfServiceReport
}
```

Finally, we calculate the score of each provider for a specific slot and select a provider using a pseudo-random weighted choice. It should be noted that a provider that best meets the policy requirements will have a higher score and therefore a greater chance of being selected. Additionally, a provider's stake is a factor in the scoring process. Thus, a provider with a large stake will also have a higher chance of being chosen. Consequently, providers with a significant stake are more likely to be selected. However, the score mechanism also encourages providers to excel in other metrics in order to attract more consumers for pairing.

#### Quality Of Service

##### Reputation

The Lava Network places a strong emphasis on delivering exceptional Quality of Service (QoS) to its consumers. To ensure this, consumers actively participate in monitoring and customizing their QoS excellence metrics. They gauge provider performance by measuring latency in provider responses relative to a benchmark, assessing data freshness in comparison to the fastest provider, and evaluating the percentage of error or timeout responses in the availability metric. These scores are diligently recorded and sent on-chain alongside the relay proofs of service, creating a transparent and accountable system. The provider's performance metric is called "Reputation". Higher reputation indicates higher QoS scores.

To further enhance the integrity of the QoS scores, updates are aggregated across all consumers in a manner that safeguards against false reports. Negative reports are weighted by usage, meaning that a consumer must actively use and pay a provider to diminish their QoS score. This mechanism discourages users from artificially lowering a provider's score.

The Reputation metric only affect pairings and is aggregated over time with a decay function that favors the latest data, meaning providers can improve, and those providers that their service fails will be impacted to affect fewer users. This approach ensures that the reputation system remains dynamic and responsive, benefiting providers striving to enhance their services while minimizing the impact of service failures on a broader scale.

##### Passable QoS

In the Lava Network, alongside the comprehensive Reputation metric (which is calculated using QoS excellence reports), there exists an additional metric known as Passable QoS. Unlike Reputation, which offers a broad range of values, Passable QoS operates on a binary scale, either assigning a value of 0 or 1, averaged over relays. This metric simplifies the evaluation of service quality to a binary determination, indicating whether a relay meets the Passable QoS threshold, meaning it provides a level of service deemed acceptable for use.

The Passable QoS score directly influences the total payout for a specific payment; however, it's important to note that only 50% of the payout is exposed to this metric (can be changed via governance). This allocation ensures a balance between incentivizing excellent service and discouraging poor performance.

#### Pairing Verification

The calculation of the pairing list is deterministic and depends on the current epoch. The pseudo-random factor only applies when pairing a specific provider with a consumer. Therefore, the pairing list can be recalculated using the consumer's address, block, and chain ID.

Pairing verification is used by the provider to determine whether to offer service for a consumer request. If the provider decides to serve a consumer that fails pairing verification, it will not receive payment for the service it provided.

#### Unresponsiveness

Providers can get punished for being unresponsive to consumer requests. If a provider wishes to stop getting paired with consumers for any reason to avoid getting punished, it can freeze itself. Currently, the punishment for being unresponsive is jailing.

When a consumer is getting paired with a provider, it sends requests for service. If provider A is unresponsive after a few tries, the consumer switches to another provider from its pairing list, provider B, and send requests to it. When communicatting with provider B, the consumer appends the address of provider A to its request, thus adding the current request's CU to provider A's "complainers CU" counter.

Every epoch start, the amount of complainers CU is compared with the amount of serviced CU of each provider across a few epochs back. If the complainers CU is higher, the provider is considered unresponsive and gets punished. The number of epochs back is determined by the recommendedEpochNumToCollectPayment parameter

#### Jail

If a provider is down and users report it, the provider will be jailed.
The first 2 instances of jailing are temporary, lasting 1 hour each, and will be automatically removed.
After 2 consecutive jailings, the provider will be jailed for 24 hours and set to a 'frozen' state. To resume activity, the provider must send an 'unfreeze' transaction after the jail time has ended.

#### Static Providers

Static providers are Lava chain providers that offer services to any consumer without relying on pairing. This feature allows new consumers to communicate with the Lava chain without a centralized provider. For example, when a new consumer wants to start using Lava, it needs to obtain its pairing list from a Lava node. However, since it initially does not have a list of providers to communicate with, it can use the static providers list to obtain its initial pairing list.

To clarify, a consumer cannot randomly find a Lava provider's endpoint and receive service from it because a provider will only serve consumers whose pairing with it can be verified. Therefore, the consumer must obtain the providers' pairing list from a Lava node.

Note that unstaking static providers will have their coins returned after a longer period of time compared to regular ("dynamic") providers due to their special status.

### Payments

The Lava network connects blockchain data providers with consumers who need access to that data. Once a pairing is established, a consumer can send a message to the provider to request a specific blockchain query. The provider then sends a relay request to the blockchain, retrieves the required information, and forwards it to the consumer. After completing the requested service, the provider sends a relay payment transaction to the Lava network to receive payment.

Here is the workflow for Lava's payment mechanism:

1. The provider sends a relay payment transaction, which is signed by the consumer.
2. The reported CU (compute units) from the relay payment transaction is tracked.
3. Upon a subscription's month expiry, all the providers' tracked CU are reviewed, and payments are sent for their accumulated services.

#### CU Tracking

The goal of CU tracking is to keep records of serviced CUs by the providers to a specific subscription, in order to determine the amount of payment they should receive at the end of the month of said subscription. When a relay payment transaction occurs, the number of CUs associated with each relay is counted and saved under the provider who initiated the transaction. At the end of each subscription end of month, the CU tracker is reset for all providers that serviced this subscription.

#### Providers Payment

Providers receive payments once a month. These payments are funded by the funds that consumers paid for their subscriptions. The amount of coins to be sent to each provider depends on the number of tracked CUs they had over the previous month. The payment is calculated using the following formula:

```math
Payment = subscription\_price * \frac{provider\_tracked\_CU}{total\_CU\_used\_by\_subscription}
```

Explaining the formula, the provider receives a portion of the subscription price. This portion is calculated by dividing the amount of CU that the provider serviced for the consumer by the total amount of CU that the consumer used, which may involve multiple providers. It is important to note that the total CU and tracked CU are counted separately for each chain, such as ETH and STRK.

The rewards for providers can increase if the consumer exceeds the allotted CU for their subscription. For more detailed information, please refer to the README of the subscription module.

Additionally, it should be noted that the providers' rewards are influenced by factors such as adjustment factor, community/validators' contributions and bonus rewards. For further information, please consult the README of the rewards module.

#### Storage

Successful provider payments are stored on-chain. This is done to avoid double spend attacks (in which a provider tries to get paid twice for the same relay) and to track CU for the current epoch (and limit it by the policy restrictions).

Payments are stored using the following objects:

```go
// holds all the payments of a specific epoch
type EpochPayments struct {
	Index                      string   
	ProviderPaymentStorageKeys []string 
}
```

```go
// holds all the payments of a specific epoch for a specific provider
type ProviderPaymentStorage struct {
	Index                                  string   
	Epoch                                  uint64   
	UniquePaymentStorageClientProviderKeys []string  // list of payments
	ComplainersTotalCu                     uint64    // amount of complainers CU for unresponsiveness
}
```

```go
// holds a payment for a specific provider in a specific epoch
type UniquePaymentStorageClientProvider struct {
	Index  string 
	Block  uint64 
	UsedCU uint64 
}
```

## Parameters

The pairing module contains the following parameters:

| Key                                    | Type                    | Default Value    |
| -------------------------------------- | ----------------------- | -----------------|
| QoSWeight                        | math.LegacyDec          | 0.5              |
| EpochBlocksOverlap                              | uint64          | 5              |
| RecommendedEpochNumToCollectPayment    | uint64          | 3             |

### QoSWeight

QoSWeight determines the weight of passable QoS score in the provider payment calculation. For example, say that a provider has a lower QoS than the passable QoS, the QoS weight determines how much of the provider's payment will be deducted from not passing the minimum QoS (QoS weight = 40% --> 40% of the provider's payment is deducted).

### EpochBlocksOverlap

EpochBlocksOverlap is the number of blocks a consumer waits before interacting with a provider from a new pairing list to let providers that are behind the latest block to catch up with the chain.

### RecommendedEpochNumToCollectPayment

RecommendedEpochNumToCollectPayment is the recommended max number of epochs for providers to claim payments. It's also used for determining unresponsiveness.

### ReputationVarianceStabilizationPeriod

ReputationVarianceStabilizationPeriod is the period in which reputation reports are not truncated due to variance.

### ReputationLatencyOverSyncFactor

ReputationLatencyOverSyncFactor is the factor that decreases the reputation's sync report influence when calculating the reputation score.

### ReputationHalfLifeFactor

ReputationHalfLifeFactor is the half life factor that determines the degradation in old reputation samples (when the reputation is updated).

### ReputationRelayFailureCost

ReputationRelayFailureCost is the cost (in seconds) for a failed relay sent from the consumer to the provider. This is part of the reputation report calculation.

## Queries

The pairing module supports the following queries:

| Query      | Arguments       | What it does                                  |
| ---------- | --------------- | ----------------------------------------------|
| `account-info`     | address (string)  | detailed summary of a Lava account                   |
| `effective-policy`     |  chain-id (string), project developer/index (string) |  shows a project's effective policy                 |
| `get-pairing`     | chain-id (string), consumer (string)  | shows a consumer's providers pairing list                  |
| `list-epoch-payments`     | none  | show all epochPayment objects                  |
| `list-provider-payment-storage`     | none  | show all providerPaymentStorage objects                 |
| `list-unique-payment-storage-client-provider`     | none  | show all uniquePaymentStorageClientProvider objects                 |
| `provider-monthly-payout`     | provider (string)  |  show the current monthly payout for a specific provider                 |
| `providers`     | chain-id (string)  | show all the providers staked on a specific chain                  |
| `sdk-pairing`     | none  | query used by Lava-SDK to get all the required pairing info                  |
| `show-epoch-payments`     | index (string)  | show an epochPayment object by index                  |
| `show-provider-payment-storage`     | index (string)  | show a providerPaymentStorage object by index                  |
| `show-unique-payment-storage-client-provider`     | index (string)  | show an uniquePaymentStorageClientProvider object by index                  |
| `static-providers-list`     | chain-id (string)  | show the list of static providers for a specific chain                  |
| `subscription-monthly-payout`     | consumer (string)  |  show the current monthly payout for a specific consumer                 |
| `user-entry`     | consumer (string), chain-id (string), block (uint64)  |  show the remaining allowed CU for the current epoch for a consumer                 |
| `verify-pairing`     | chain-id (string), consumer (string), provider (string), block (uint64)  | verify the provider was in the consumer's pairing list on a specific block                  |
| `params`   | none            | shows the module's parameters                 |

For more details on payment storage objects (`epochPayment`, `providerPaymentStorage`, `uniquePaymentStorageClientProvider`), see the `EpochStorage` module's README.

## Transactions

All the transactions below require setting the `--from` flag and gas related flags (other required flags of specific commands will be shown in the TX arguments). Also, state changes from these transactions are applied at different times, depending on the transaction. Some changes take effect immediately, while others may have a delay before they are applied.

The pairing module supports the following transactions:

| Transaction      | Arguments       | What it does                                  |
| ---------- | --------------- | ----------------------------------------------|
| `bulk-stake-provider`     | chain-ids ([]string), amount (Coin), endpoints ([]Endpoint), geolocation (int32), {repeat args for another bulk}, validator (string, optional), --provider-moniker (string)  | stake provider in multiple chains with multiple endpoints with one command                  |
| `freeze`     | chain-ids ([]string)  | freeze a provider in multiple chains                  |
| `modify-provider`     | chain-id (string)  | modify a provider's stake entry (use the TX optional flags)                  |
| `relay-payment`     | chain-id (string) | automatically generated TX used by a provider to request payment for their service                  | 
| `simulate-relay-payment`     | consumer-key (string), chainId (string)  | simulate a relay payment TX                  |
| `stake-provider`     | chain-id (string), amount (Coin), endpoints ([]Endpoint), geolocation (int32), validator (string, optional), --provider-moniker (string) --grant-provider-gas-fees-auth (bool)| stake a provider in a chain with multiple endpoints                 |
| `unfreeze`     | chain-ids ([]string)  | unfreeze a provider in multiple chains                  |
| `unstake-provider`     | chain-ids ([]string), validator (string, optional)  | unstake a provider from multiple chains                  |

Note, the `Coin` type is from Cosmos-SDK (`cosmos.base.v1beta1.Coin`). From the CLI, use `100ulava` to assign a `Coin` argument. The `Endpoint` type defines a provider endpoint. From the CLI, use "my-provider-grpc-addr.com:9090,1" for one endpoint (includes the endpoint's URL+port and the endpoint's geolocation). When it comes to staking-related transactions, the geolocation argument should encompass the geolocations of all the endpoints combined.

Moreover, using `--grant-provider-gas-fees-auth` flag when staking a provider entity will let the provider address use the vault address funds for gas fees. When sending a TX, the provider address should add the `--fee-granter` flag to specify the vault's account as the payer of gas fees.

## Proposals

The pairing module supports the provider unstake proposal, which is designed to address malicious providers and compel them to unstake from the chain.

To send the proposal, use the following commands:

```
lavad tx gov submit-legacy-proposal unstake <proposal_json_1>,<proposal_json_2> --from alice <gas-flags>
lavad tx gov vote <latest_proposal_id> yes --from alice <gas-flags>
```

A valid `unstake` JSON proposal format:

```json
{
    "proposal": {
        "title": "Unstake proposal",
        "description": "A proposal for unstaking providers",
        "providers_info": [
            {
                "provider": "lava@1q0q74n8hgme7hrmhrvjgfq2dzhchjulqr526y7",
                "chain_id": "*"
            },
            {
                "provider": "lava@1rx5g4j65ztzu29pv5wrn4s2rcjkhkhcxztplv9",
                "chain_id": "ETH1"
            }
        ]
    },
    "deposit": "10000000ulava"
}
```

## Events

The pairing module has the following events:

| Event      | When it happens       |
| ---------- | --------------- |
| `stake_new_provider`     | a successful provider stake   |
| `stake_update_provider`     | a successful provider stake entry modification  |
| `provider_unstake_commit`     | a successful provider unstake (before receiving the funds back)   |
| `relay_payment`     | a successful relay payment   |
| `provider_reported`     | a successful provider report for unresponsiveness   |
| `provider_latest_block_report`     | a successful report of latest block of a provider   |
| `rejected_cu`     | a successful relay payment that rejected some of the relays   |
| `unstake_gov_proposal`     | a successful unstake gov proposal   |