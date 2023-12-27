# `x/projects`

## Abstract

This document specifies the projects module of Lava Protocol.

The projects module is responsible for managing Lava's consumer projects. When a consumer wants to use Lava, they need to purchase a subscription. Within the subscription, consumers can create projects to utilize Lava and make RPC calls seamlessly.

Note that the plans module is closely connected to the subscription and projects modules. To fully understand how subscriptions, plans, and projects work together, please also refer to their respective READMEs.

## Contents

* [Concepts](#concepts)
  * [Project](#project)
  * [Project Keys](#project-keys)
  * [Badges](#badges)
* [Parameters](#parameters)
* [Queries](#queries)
* [Transactions](#transactions)
  * [Project Policy YAML](#project-policy-yaml)
  * [Project Keys YAML](#project-keys-yaml)
* [Proposals](#proposals)

## Concepts

### Project

Once a consumer purchases a subscription, they can create projects under it. Each project has a unique name (index) within this subscription. A project is defined as follows:

```go
type Project struct {
	Index               string        // project unique index
	Subscription        string        // project associated subscription
	Enabled             bool          // project enabled/disabled
	ProjectKeys         []ProjectKey  // list of project keys
	AdminPolicy         Policy        // project admin policy
	UsedCu              uint64        // project used compute units (CU) per month
	SubscriptionPolicy  Policy        // project subscription policy
	Snapshot            uint64        // project snapshot unique index
}
```

The project index is: `<subscription_owner_address>-<project_name>` (for example: `lava@1p7ezdua4ma90858h7zy9727wmqy53vk0mkdlvg-myProject`). When a consumer buys a subscription, an admin project is automatically generated with the index `<subscription_owner_address>-admin`.

The project is subject to three types of policies: the plan's policy, the subscription's policy, and the admin policy. The plan's policy cannot be changed using transactions (only via a government proposal), while the other two policies can be changed using transactions. The effective policy that the project is subject to is the strictest policy calculated from all three policies. To query for the project's effective policy, use the `Pairing` module's query: `effective-policy` (see [below](#queries)).

Projects are saved in a fixation store (see `FixationStore` module for reference). By using the snapshot ID, we can differentiate between different versions of the project.

For more details regarding the `Policy` struct and its limitations, see [here](https://github.com/lavanet/lava/blob/main/x/plans/README.md#policy). For more details about `ProjectKeys`, see [below](#project-keys).

### Project Keys

A project has a list of accounts (project keys) that can use the project in different ways. There are two kinds of keys: admin key and developer key. A project key is defined as follows:

```go
type ProjectKey struct {
	Key    string  // user lava address
	Kinds  uint32  // key kind
}
```

The `Kinds` enum is defined as follows:

```
enum Kinds {
    ADMIN = 1            // admin key
    DEVELOPER = 2        // developer key
    ADMIN+DEVELOPER = 3  // admin+developer key
}
```

The admin key lets the admin edit the project's admin policy, enable/disable the project and add/delete other project keys. A single user can be admin in multiple projects. The subscription owner is automatically considered an admin in all of the project created under it.

The developer key lets a user to use the project, i.e., make RPC calls and use the project's CU. A single user (consumer) can only be a developer in one project. The project's developer can be viewed as a dapp developer that wants to use Lava to get RPC data for their dapp. To allow end users use their project to get RPC data, the dapp developer will need to use badges. For more details on badges, see [below](#badges).

Note that the admin cannot use the project's CU like a developer, they can only edit the project's properties.

### Badges

Badges are a method to grant compute units (CU) from projects to other users. Given a subscription with projects, a developer key (in a project) can be used to generate a badge key, to be signed by an external badge-server. The badge is ephemeral, limited in time and in CU capacity; It can be used to get pairing with providers, and for requests from providers - which will be charged to the respective project after verification.

A badge is defined as follows:

```go
type Badge struct {
	CuAllocation  uint64  // badge CU limit
	Epoch         uint64  // badge epoch
	Address       string  // badge user address
	LavaChainId   string  // lava chain ID (testnet/mainnet)
	ProjectSig    []byte  // developer key signature (for verification)
	VirtualEpoch  uint64  // used for emergency mode
}
```

The badge's `Epoch` field is the epoch in which the badge is valid. When the epoch changes, the badge becomes invalid. The badge's `VirtualEpoch` is used in emergency mode (see `Downtime` module's README for more details).

When a badge user sends a relay request to the provider, the first request must be accompanied by the signed message (proof of grant, i.e., a valid `ProjectSig` which should be the developer key's signature). Subsequent relays from the same badge don't have to add a valid `ProjectSig`.

## Parameters

The projects module does not contain parameters.

## Queries

The projects module supports the following queries:

| Query        | Arguments       | What it does                                  |
| -------------| --------------------------------------| ----------------------------------------------|
| `info`       | index (string)                        | shows a project's info by index                  |
| `developer`  | developer address (string)            | show a project's info by developer address (registered with a developer key)  |
| `params`   | none                                    | shows the module's parameters                 |

More projects related queries from other modules:

| Query (module)       | Arguments       | What it does                                  |
| -------------| --------------------------------------| ----------------------------------------------|
| `effective-policy` (Pairing)       | spec-id (string), developer/index (string)                        | shows a project's effective policy by project index/developer address and spec index                  |

The spec index argument can be obtained using the `Spec` module's `show-all-chains` query.
 
## Transactions

All the transactions below require setting the `--from` flag and gas related flags. Also, all state changes from these transactions are applied on the start of the next epoch.

The projects module supports the following transactions:

| Transaction      | Arguments       | What it does                                  |
| ---------- | --------------- | ----------------------------------------------|
| `set-policy`     | index (string), policy file path (string)  | sets the admin policy of a project by index (must be sent from the admin/subscription owner)                  |
| `set-subscription-policy`     | indices ([]string), policy file path            | sets the subscription policy of the subscription's projects by their index (must be sent from the subscription owner)  |
| `add-keys`   | index (string), project keys file path (string)            | adds a project key to a project by index                 |
| `del-keys`   | index (string), project keys file path (string)            | deletes a project key from a project by index                 |

Note that the `add-keys` and `del-keys` transactions also support key management with flags, in addition to file input. Refer to the help section of the commands for more details.

To get more details about the policy and project keys YAML file format, see below.

### Project Policy YAML

Both the `set-policy` and `set-subscription-policy` transactions use the same policy YAML format.

Example of a policy YAML file:

```yaml
Policy:
  geolocation_profile: 1  # USC
  total_cu_limit: 10000
  epoch_cu_limit: 100
  max_providers_to_pair: 7
  selected_providers_mode: ALLOWED
  chain_policies:
    - chain_id: ETH1
      requirements:
        - collection:
            api_interface: "YAMLrpc"
            type: "POST"
            add_on: "debug"
    - chain_id: EVMOS
      requirements:
        - collection:
            api_interface: "YAMLrpc"
            type: "POST"
          extensions:
            - "archive"
          mixed: true
        - collection:
            api_interface: "rest"
            type: "GET"
          extensions:
            - "archive"
          mixed: true
        - collection:
            api_interface: "grpc"
            type: ""
          extensions:
            - "archive"
          mixed: true
        - collection:
            api_interface: "tendermintrpc"
            type: ""
          extensions:
            - "archive"
          mixed: true
    - chain_id: "*" # allows all other chains without specifying
```

All fields are not mandatory. An unfilled field will be replaced with a default value or be ignored.

### Project Keys YAML

Both the `add-keys` and `del-keys` transactions use the same project key YAML format.

Example of a project key YAML file:

```yaml
Project-Keys:
  - key: "lava@1xtfqykth53pkt97v955h3lql8zkj2m4s4rq9cr"
    Kinds: 3
  - key: "lava@1r3ernqu6rzp95z92580wae7xpuqwmznk3eqd7w"
    Kinds: 1
```

All fields are mandatory.

## Proposals

The projects module does not support any proposals.
