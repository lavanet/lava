# Specifications

### Links

`Spec` protobuf: [https://github.com/lavanet/lava/blob/main/proto/spec/spec.proto](https://github.com/lavanet/lava/blob/main/proto/spec/spec.proto)

`ServiceApi` protobuf: [https://github.com/lavanet/lava/blob/main/proto/spec/service_api.proto](https://github.com/lavanet/lava/blob/main/proto/spec/service_api.proto)

### Overview

A specification (AKA as spec) defines the APIs a provider commits to providing to consumers. An example for a spec can be the Ethereum JSON-RPC spec, defining all supported APIs calls and their compute units (AKA CU).

Lava has many specs and participants can add and modify specs using governance proposals. When adding new blockchains, the first step is to create a spec for it defining the available APIs available on that chain’s RPC.

### Fields

#### Spec ([proto](https://github.com/lavanet/lava/blob/main/proto/spec/spec.proto))

| Field name                          | Description                                                                                                              |
|-------------------------------------|--------------------------------------------------------------------------------------------------------------------------|
| index                               | A unique name for the spec. For example `ETH1`.                                                                          |
| name                                | A non-unique "human" name for the spec. For example, `ethereum mainnet`.                                                |
| imports                             | A list of specs that the current spec inherits from. For example, Polygon uses many RPC APIs of Ethereum. So in the Polygon spec, we'll import `ETH1` to inherit all of its APIs.|
| apis                                | A list of the APIs that are supported in this spec. These are of type `ServiceApi`. See details below.                   |
| enabled                             | True/False to determine whether this spec is active (providers can provide service by it).                              |
| reliability_threshold               | Threshold for VRF to decide when to do a data reliability check (i.e. re-execute query with another provider). Currently set to `268435455` on all specs resulting in a `1/16` ratio.|
| data_reliability_enabled            | True/False for data reliability on/off for this spec.                                                                    |
| block_distance_for_finalized_data   | Blockchains like Ethereum have probabilistic finality, this threshold sets what we expect to be a safe distance from the latest block (In eth it’s 7: i.e. any block bigger in distance than 7 from the latest block we consider final).|
| blocks_in_finalization_proof        | Number of finalized blocks to keep.                                                                                      |
| average_block_time                  | Average block time on this blockchain, used for estimating time of future blocks.                                       |
| allowed_block_lag_for_qos_sync      | Lag used to calculate QoS for providers.                                                                                 |
| block_last_updated                  | The latest block in which the spec was updated.                                                                          |
| min_stake_provider                  | The minimum amount of stake a provider must stake to provide service for the APIs listed in the spec.                    |
| min_stake_client                    | The minimum amount of stake a provider must stake to provide service for the APIs listed in the spec.                    |
| providers_type                      | Can be static/dynamic. More details in the future.                                                                       |


#### Service Apis ([proto](https://github.com/lavanet/lava/blob/main/proto/spec/service_api.proto))

> Every Spec has a list of service apis

| Field                     | Description                                                                                       |
|---------------------------|---------------------------------------------------------------------------------------------------|
| name                      | a unique name of the API.                                                                         |
| block_parsing             | defines how to parse block heights (block number) from this specific API request. See ([ServiceApi proto](https://github.com/lavanet/lava/blob/main/proto/spec/service_api.proto)) for more details.               |
| compute_units             | reflect how much work a provider has to do to service a relay.                                    |
| enabled                   | True/False to determine whether this API is supported by the providers.                           |
| api_interfaces            | Information about this API. It's of type `ApiInterface` (see below).                                                                                                                                               
| parsing *(optional)*      | defines how to parse results for block heights and hashes from this specific API response. This is very important information we use for data reliability. 

##### API interface

| Field                  | Description                                                                                                       |
|------------------------|-------------------------------------------------------------------------------------------------------------------|
| interface              | Name of the interface. For example: `rest, jsonrpc, grpc, tendermintrpc`.                                          |
| type                   | Type of the API: `GET` or `POST`.                                                                                  |
| extra_compute_units    | Amount of extra CU that are added to the total CU used by executing this API.                                      |
| category               | Define the category of API. It's of type `SpecCategory` (see below).                                                                                        |
| overwrite_block_parsing| Defines how to parse results for block heights and hashes from this specific API response. Used for data reliability. |

##### Spec Category

| Field        | Description                                                                                                                            |
|--------------|----------------------------------------------------------------------------------------------------------------------------------------|
| deterministic| If an API is deterministic (executing the API twice in the same block will have the same result), we can run data reliability checks on it. |
| local        | True/False. ?                                                                                                                          |
| subscription | Requires an active connection to a node to get data pushed from a provider.                                                            |
| stateful     | Requires local storage on the provider’s node.                                                                                          |
| hanging_api  | ?                                                                                                                                      |

### How to propose a new spec?

1. Create a proposal JSON file with the desired spec. You can use the explanation above and older specs as reference.

2. propose the new spec with the following command:
```
lavad tx gov submit-proposal spec-add "{JSON_FILE_PATH}" -y --from "{ACCOUNT_NAME}" --gas-adjustment "1.5" --gas "auto" --node "{LAVA_RPC_NODE}"
```
#### Param description (and examples)

`JSON_FILE_PATH` - The path to the proposal JSON file. 

`ACCOUNT_NAME` - The account to be used for the proposal. Example: `alice`

`LAVA_RPC_NODE` - A RPC node for Lava (can be omitted if the current node has joined the Lava network). For example: `https://public-rpc.lavanet.xyz:443/rpc/`

### Spec proposal JSON file example
```json
{
    "proposal": {
        "title": "Add Specs: Optimism",
        "description": "Adding new specification support for relaying Optimism data on Lava",
        "specs": [
            {
                "index": "OPTM",
                "name": "optimism mainnet",
                "enabled": true,
                "imports": [ "ETH1" ],
                "reliability_threshold": 268435455,
                "data_reliability_enabled": true,
                "block_distance_for_finalized_data": 1,
                "blocks_in_finalization_proof": 1,
                "average_block_time": "250",
                "allowed_block_lag_for_qos_sync": "10",
                "min_stake_provider": {
                    "denom": "ulava",
                    "amount": "50000000000"
                },
                "min_stake_client": {
                    "denom": "ulava",
                    "amount": "5000000000"
                },
                "apis": [
                    {
                        "name": "eth_getBlockRange",
                        "block_parsing": {
                            "parser_arg": [
                                "1"
                            ],
                            "parser_func": "PARSE_BY_ARG"
                        },
                        "compute_units": "20",
                        "enabled": true,
                        "api_interfaces": [
                            {
                                "category": {
                                    "deterministic": true,
                                    "local": false,
                                    "subscription": false,
                                    "stateful": 0
                                },
                                "interface": "jsonrpc",
                                "type": "GET",
                                "extra_compute_units": "0"
                            }
                        ]
                    },
                    {
                        "name": "rollup_getInfo",
                        "block_parsing": {
                            "parser_arg": [
                                ""
                            ],
                            "parser_func": "EMPTY"
                        },
                        "compute_units": "10",
                        "enabled": true,
                        "api_interfaces": [
                            {
                                "category": {
                                    "deterministic": false,
                                    "local": true,
                                    "subscription": false,
                                    "stateful": 0
                                },
                                "interface": "jsonrpc",
                                "type": "GET",
                                "extra_compute_units": "0"
                            }
                        ]
                    },
                    {
                        "name": "rollup_gasPrices",
                        "block_parsing": {
                            "parser_arg": [
                                "latest"
                            ],
                            "parser_func": "DEFAULT"
                        },
                        "compute_units": "10",
                        "enabled": true,
                        "api_interfaces": [
                            {
                                "category": {
                                    "deterministic": true,
                                    "local": false,
                                    "subscription": false,
                                    "stateful": 0
                                },
                                "interface": "jsonrpc",
                                "type": "GET",
                                "extra_compute_units": "0"
                            }
                        ]
                    },
                    {
                        "name": "eth_getAccounts",
                        "block_parsing": {
                            "parser_arg": [
                                ""
                            ],
                            "parser_func": "EMPTY"
                        },
                        "compute_units": "10",
                        "enabled": false,
                        "api_interfaces": [
                            {
                                "category": {
                                    "deterministic": true,
                                    "local": false,
                                    "subscription": false,
                                    "stateful": 0
                                },
                                "interface": "jsonrpc",
                                "type": "GET",
                                "extra_compute_units": "0"
                            }
                        ]
                    },
                    {
                        "name": "eth_sendTransaction",
                        "block_parsing": {
                            "parser_arg": [
                                ""
                            ],
                            "parser_func": "EMPTY"
                        },
                        "compute_units": "10",
                        "enabled": false,
                        "api_interfaces": [
                            {
                                "category": {
                                    "deterministic": true,
                                    "local": false,
                                    "subscription": false,
                                    "stateful": 0
                                },
                                "interface": "jsonrpc",
                                "type": "GET",
                                "extra_compute_units": "0"
                            }
                        ]
                    }
                ]
            },
            {
                "index": "OPTMT",
                "name": "optimism goerli testnet",
                "enabled": true,
                "imports": [ "OPTM" ],
                "reliability_threshold": 268435455,
                "data_reliability_enabled": true,
                "block_distance_for_finalized_data": 1,
                "blocks_in_finalization_proof": 1,
                "average_block_time": "250",
                "allowed_block_lag_for_qos_sync": "10",
                "min_stake_provider": {
                    "denom": "ulava",
                    "amount": "50000000000"
                },
                "min_stake_client": {
                    "denom": "ulava",
                    "amount": "5000000000"
                }
            }
        ]
    },
    "deposit": "10000000ulava"
}
```
