# `x/plans`

## Abstract

This document specifies the plans module of Lava Protocol.

The plans module is responsible for managing Lava's subscription plans and handling government proposals for adding new ones. When a consumer wants to use Lava, they need to purchase a subscription. Each subscription is associated with a specific plan that sets certain limitations. The plan determines the base limitations of the subscription, such as compute unit (CU) limitations and the maximum number of providers that can be paired in each pairing session, among other things.

Note that the plans module is closely connected to the subscription and projects modules. To fully understand how subscriptions, plans, and projects work together, please also refer to their respective READMEs.

This document is focuses on the plans' technical aspects and does not include current subscription plans and prices. For more information on those, please visit Lava's official website.

## Contents

* [Concepts](#concepts)
  * [Plan](#plan)
  * [Policy](#policy)
    * [Chain Policy](#chain-policy)
    * [Geolocation](#geolocation)
    * [Selected Providers](#selected-providers)
* [Parameters](#parameters)
* [Queries](#queries)
* [Transactions](#transactions)
* [Proposals](#proposals)

## Concepts

### Plan

A plan consists of limitations that are associated with a subscription. A consumer can only use Lava when they purchase a subscription that is subject to the limitations set by the plan. A plan is defined as follows:

```
struct Plan {
    index            string  // plan unique index
    block            uint64  // the epoch that this plan was created
    price            Coin    // plan price (in ulava)
    allow_overuse    bool    // allow CU overuse flag
    overuse_rate     uint64  // price of CU overuse (higher price per CU)
    description      string  // plan description (for humans)
    type             string  // plan type (currently unused)
    annual_discount  uint64  // discount for buying a yearly plan (percentage)
    plan_policy      Policy  // plan's policy
}
```
Note, the `Coin` type is from Cosmos-SDK (`cosmos.base.v1beta1.Coin`).

The plan's limitations mostly lie in its policy. As mentioned above, the plan has other fields like its unique index, price, and more. The plan's “overuse” related fields refers to a scenario where the subscription exceeds the CU limit set by the plan policy. In such cases, if CU overuse is permitted, the price of CU is higher than normal.

### Policy

A policy consists of a set of limitations that apply to a plan, a subscription, or a specific project. Each project opened using a consumer's subscription is subject to the effective policy derived from the plan policy, subscription policy, and project policy. The effective policy is calculated by selecting the strictest limitation from each policy. Overall, the policy affects the pairing between the project and providers and limits the CU that can be used by the consumer.

A policy is defined as follows:

```
struct Policy {
	ChainPolicies          []ChainPolicy            // list of policies per chain
	GeolocationProfile     int32                    // allowed geolocations
	TotalCuLimit           uint64                   // CU usage limit per month
	EpochCuLimit           uint64                   // CU usage limit per epoch
	MaxProvidersToPair     uint64                   // max number of providers to pair
	SelectedProvidersMode  SELECTED_PROVIDERS_MODE  // selected providers mode
	SelectedProviders      []string                 // allow list of selected providers
}
```

See below for more information regarding [chain policies](#chain-policy), [geolocation](#geolocation) and [selected providers](#selected-providers).

A plan will always have a policy, but this is not always the case for subscription or project policies, which can be nil. A plan's policy can only be changed through a government proposal to add a new plan. As for subscription and project policies, they can be modified using transactions defined in the projects module.

#### Chain Policy

A chain policy consists limitations that can be imposed on specific chains (like list of allowed APIs, requirement for advanced APIs, etc.). If a certain chain is not shown in the policy's array of chain policeis, it's regarded as not allowed by the policy (i.e. the consumer won't be able to send relays to this chain).

A chain policy is defined as follows:

```
struct ChainPolicy {
	ChainId       string              // chain's unique index        
	Apis          []string            // allowed APIs
	Requirements  []ChainRequirement 
}
```

A chain requirement is defined as follows:

```
struct ChainRequirement {
	Collection  types.CollectionData
	Extensions  []string             
	Mixed       bool                 
}
```

The `Extensions` field is intended to enable the use of certain APIs to obtain different responses (for a modified price). Currently, the only implemented extension is the `archive` extension, which allows consumers to make archived RPC calls to retrieve data from blocks that are not stored in the chain's memory. If a regular call is made for a block that is not in the chain's memory, an error will occur. In the future, the `compliance` extension will be implemented to enable the rejection of specific APIs.

The `Mixed` field is designed to enable a combination of regular and extension/addon supporting providers. For instance, if the `archive` extension is defined but the `Mixed` field is set to `false`, the consumer's project will only be paired with providers that support the specified extensions and addons. On the other hand, if the `Mixed` field is set to `true`, the consumer's project will also be paired with providers that don't fully support the extenstion/addons.

A chain collection is defined as follows:

```
struct CollectionData {
	ApiInterface  string
	InternalPath  string
	Type          string
	AddOn         string
}
```

The `ApiInterface` field defines the API interface on which the limitations are applied. The available API interfaces for a chain are defined in the chain's spec. Overall, the API interfaces can be: `jsonrpc`, `rest`, `tendermintrpc` and `grpc`.

The `InternalPath` field is utilized for chains that have varying RPC API sets in different internal paths. Avalanche is a prime example of such a chain, consisting of three distinct subchains (or subnets) designed for different applications. For instance, Avalanche's C-Chain is dedicated to smart contracts, while Avalanche's X-Chain facilitates the sending and receiving of funds. For further information on how to define this field, please consult the Avalanche (AVAX) specification.

The `Type` field lets the user define APIs that have different functionalities depending on their type. the valid types are: `GET` and `POST`. An example of such API is Cosmos' `/cosmos/tx/v1beta1/txs` API. If it's sent as a `GET` request, it fetches transactions by event and if it's sent as a `POST` request, it sends a transaction.

The `AddOn` field lets you use additional optional APIs (debug, trace and more). Overall, the add-ons can be: `debug`, `trace` and `convex`.

#### Geolocation

Geolocation profile allows consumers to get paired with providers that support these geolocations, ultimately optimizing latency and improving the quality of service for end-users. The geolocations are defined as a bitmap so the `int32` field in the policy can represent multiple geolocations.

The geolocations are defined as follows:

```
enum Geolocation {
	"GLS" = 0      // Global strict
	"USC" = 1      // US center
	"EU"  = 2      // Europe
	"USE" = 4      // US east
	"USW" = 8      // US west
	"AF"  = 16     // Africa
	"AS"  = 32     // Asia
	"AU"  = 64     // Australia
	"GL"  = 65535  // Global
}
```

The "GLS" geolocation means that the policy is global and not configurable.

#### Selected Providers

The selected providers feature enables consumers to pre-select providers they prefer for their pairing list by implementing an allow-list within the project policy. There several modes that determine this feature's behaviour:

```
enum SELECTED_PROVIDERS_MODE {
    ALLOWED = 0    // no providers restrictions (feature is enabled but no restrictions)
    MIXED = 1      // use the selected providers mixed with randomly chosen providers
    EXCLUSIVE = 2  // use only the selected providers
    DISABLED = 3   // selected providers feature is disabled
}
```

In the policy the user can define the desired mode and the selected providers allow-list.

## Parameters

The plans module does not contain parameters.

## Queries

The plans module supports the following queries:

| Query      | Arguments       | What it does                                  |
| ---------- | --------------- | ----------------------------------------------|
| `info`     | index (string)  | shows a plan's info by index                  |
| `list`     | none            | show the info for all plans (latest version)  |
| `params`   | none            | shows the module's parameters                 |

## Transactions

The plans module does not support any transactions.

## Proposals

The plans module supports the following proposals: `plans-add` and `plans-del`. To send a proposal and vote use the following commands:

```
lavad tx gov submit-legacy-proposal plans-add <proposal_json_1>,<proposal_json_2> --from alice <gas-flags>
lavad tx gov vote <latest_proposal_id> yes --from alice <gas-flags>

OR

lavad tx gov submit-legacy-proposal plans-del <proposal_json_1>,<proposal_json_2> --from alice <gas-flags>
lavad tx gov vote <latest_proposal_id> yes --from alice <gas-flags>
```

A valid `plans-add` JSON proposal format:

```
{
    "proposal": {
        "title": "Add temporary to-delete plan proposal",
        "description": "A proposal of a temporary to-delete plan",
        "plans": [
            {
                "index": "to_delete_plan",
                "description": "This plan has no restrictions",
                "type": "rpc",
                "price": {
                    "denom": "ulava",
                    "amount": "100000"
                },
                "annual_discount_percentage": 20,
                "allow_overuse": true,
                "overuse_rate": 2,
                "plan_policy": {
                    "chain_policies": [
                        {
                            "chain_id": "LAV1",
                            "apis": [
                            ]
                        },
                        {
                            "chain_id": "ETH1",
                            "apis": [
                                "eth_blockNumber",
                                "eth_accounts"
                            ]
                        }
                    ],
                    "geolocation_profile": "AU",
                    "total_cu_limit": 1000000,
                    "epoch_cu_limit": 100000,
                    "max_providers_to_pair": 3,
                    "selected_providers_mode": "MIXED",
                    "selected_providers": [
                        "lava@1wvn4slrf2r7cm92fnqdhvl3x470944uev92squ"
                    ]
                }
            }
        ]
    },
    "deposit": "10000000ulava"
}
```

A valid `plans-del` JSON proposal format:

```
{
    "proposal": {
        "title": "Delete temporary (to-delete) plan proposal",
        "description": "A proposal to delete temporary (to-delete) plan",
        "plans": [
            "to_delete_plan"
        ]
    },
    "deposit": "10000000ulava"
}
```

