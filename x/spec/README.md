# `x/spec`

## Abstract

This document specifies the spec module of Lava Protocol.

A specification, also known as a spec, defines the APIs that a provider commits to providing to consumers. An example of a spec is the Ethereum JSON-RPC spec, which defines the properties of the chain, all supported API calls and their compute units, also known as CUs.
Lava has multiple specs, and participants can add and modify specs using governance proposals. When adding a new blockchain, the first step to support a new chain in lava is to create a spec for it.

This document focuses on the specs' technical aspects and does not include current chain support in Lava. For more information on those, please visit Lava's official website.

## Contents
* [Concepts](#concepts)
  * [Spec](#spec)
  * [ApiCollections](#apicollection)
  * [CollectionData](#collectiondata)
  * [Extension](#extension)
  * [Api](#api)
    * [SpecCategory](#speccategory)
    * [BlockParsing](#blockparsing)
  * [ParseDirective](#parsedirective)
  * [Verification](#verification)
  * [Header](#header)
  * [Import](#import)
* [Parameters](#parameters)
* [Queries](#queries)
* [Transactions](#transactions)
* [Proposals](#proposals)
* [Events](#events)

## Concepts

### Spec

A chain spec consists of general properties of the chain and a list of interfaces it supports. To simplify the creation and maintenance of specs, they can import APIs from another spec. For example, the X testnet spec can import from the X mainnet spec, eliminating the need to redefine all of the interfaces.
For more instructions on how to build a spec visit: https://github.com/lavanet/lava/blob/main/cookbook/README.md


```go
type Spec struct {
	Index                         string                // chain unique index                           
	Name                          string                // description string of the spec
	Enabled                       bool                  // spec enabled/disable 
	AverageBlockTime              int64                 // average block time of the chain in msec
	MinStakeProvider              Coin                  // min stake for a provider to be active in the chain
	ProvidersTypes                Spec_ProvidersTypes   // determines if the spec is for lava or chains 
	Imports                       []string              // list of chains to import ApiCollections from
	ApiCollections                []*ApiCollection      // list of ApiCollections that defines all the interfaces and APIs of the chain
	Contributor                   []string              // list of contributors (public lava address {lava@...})
	ContributorPercentage         *Dec                  // the percentage of coins the contributors will get from each reward a provider get
	Shares                        uint64                // factor for bonus rewards at the end of the month (see rewards module)
    AllowedBlockLagForQosSync     int64                 // defines the accepted blocks a provider can be behind the chain without QOS degradation
	BlockLastUpdated              uint64                // the last block this spec was updated on chain
    ReliabilityThreshold          uint32                // this determines the probability of data reliability checks by the consumer
	DataReliabilityEnabled        bool                  // enables/disables data reliability for the chain
	BlockDistanceForFinalizedData uint32                // number of finalized blocks a provider keeps for data reliability             
	BlocksInFinalizationProof     uint32                // number of blocks for finalization              
}
```
`Coin` type is from Cosmos-SDK (`cosmos.base.v1beta1.Coin`).
`Dec` type is from Cosmos-SDK math (`cosmossdk.io/math`).

A `Contributor` is a member of the Lava community who can earn token commissions by maintaining specs on Lava.

### ApiCollection

ApiCollection is a struct that defines an interface, such as REST, JSON, etc., along with all of its APIs and properties.

```go
type ApiCollection struct {
	Enabled         bool                // enables/disables the collection
	CollectionData  CollectionData      // defines the properties of the collection, also acts as a unique key
	Apis            []*Api              // list of api's in the collection
	Headers         []*Header           // list of headers supported by the interface and their behaviour
	InheritanceApis []*CollectionData   // list of other ApiCollection to inherite from
	ParseDirectives []*ParseDirective   // list of parsing instructions of specific api's
	Extensions      []*Extension        // list of extensions that providers can support in addition to the basic behaviour (for example, archive node)
	Verifications   []*Verification     // list of verifications that providers must pass to make sure they provide full functionality
}
```

### CollectionData

CollectionData defines the api properties and acts as a unique key for the [api collection](#apicollection).

```go
type CollectionData struct {
	ApiInterface string         // defines the connection interface (rest/json/grpc etc...)
	InternalPath string         // defines internal path of the node for this specific ApiCollection
	Type         string         // type of api (POST/GET)
	AddOn        string         // optional APIs support
}
```

The `ApiInterface` field defines the API interface on which the limitations are applied. The available API interfaces for a chain are defined in the chain's spec. Overall, the API interfaces can be: `jsonrpc`, `rest`, `tendermintrpc` and `grpc`.

The `InternalPath` field is utilized for chains that have varying RPC API sets in different internal paths. Avalanche is a prime example of such a chain, consisting of three distinct subchains (or subnets) designed for different applications. For instance, Avalanche's C-Chain is dedicated to smart contracts, while Avalanche's X-Chain facilitates the sending and receiving of funds. For further information on how to define this field, please consult the Avalanche (AVAX) specification.

The `Type` field lets the user define APIs that have different functionalities depending on their type. the valid types are: `GET` and `POST`. An example of such API is Cosmos' `/cosmos/tx/v1beta1/txs` API. If it's sent as a `GET` request, it fetches transactions by event and if it's sent as a `POST` request, it sends a transaction.

The `AddOn` field lets you use additional optional APIs like debug, trace etc. 

### Extension

this field defines an extension for the api collection.

```go
type Extension struct {
	Name         string  // name of the extension (archive)
	CuMultiplier float32 // cu factor for supporting this extension
	Rule         *Rule   // describes the rules when this extension is active
}
```

### Api

Api define a specific api in the api collection.

```go
type Api struct {
	Enabled           bool          // enable/disable the api
	Name              string        // api name
	ComputeUnits      uint64        // the amount of cu of this api (can be defined as the "price" of using this api)
	ExtraComputeUnits uint64        // not used
	Category          SpecCategory  // defines the property of the api
	BlockParsing      BlockParser   // specify how to parse the block from the api request
	TimeoutMs         uint64        // specifies the timeout expected for the api (mseconds)
}
```

example of an api definition:
```json
    {
        "name": "eth_getBlockByNumber",
        "block_parsing": {
            "parser_arg": [
                "0"
            ],
            "parser_func": "PARSE_BY_ARG"
        },
        "compute_units": 20,
        "enabled": true,
        "category": {
            "deterministic": true,
            "local": false,
            "subscription": false,
            "stateful": 0
        },
        "extra_compute_units": 0
    },
```

### SpecCategory

This struct defines properties of an api.

```go
type SpecCategory struct {
	Deterministic bool   // if this api have the same response across nodes
	Local         bool   // specific to the local node (like node info query)
	Subscription  bool   // subscription base api
	Stateful      uint32 // true for transaction APIs
	HangingApi    bool   // marks this api with longer timeout
}
```

### BlockParsing

This struct defines how to extract the block number from the api request.

```go
type BlockParser struct {
	ParserArg    []string    // describes where is the block number in the request
	ParserFunc   PARSER_FUNC // how to parse the request
	DefaultValue string      // the expected default value
	Encoding     string      // number encoding (base64|Hex)
}
```

ParserFunc instructs how to parse the request to fetch the block number.

```go
const (
	PARSER_FUNC_EMPTY                       PARSER_FUNC = 0 
	PARSER_FUNC_PARSE_BY_ARG                PARSER_FUNC = 1 
	PARSER_FUNC_PARSE_CANONICAL             PARSER_FUNC = 2 
	PARSER_FUNC_PARSE_DICTIONARY            PARSER_FUNC = 3 
	PARSER_FUNC_PARSE_DICTIONARY_OR_ORDERED PARSER_FUNC = 4
	PARSER_FUNC_DEFAULT PARSER_FUNC = 6
)
```

### ParseDirective

ParseDirective is a struct that defines for the provider in a generic way how to fetch specific data from the node (for example: latest block height, block hash, ctv...). it describes for the api collection how to get information from the node. 

```go
type ParseDirective struct {
	FunctionTag      FUNCTION_TAG // defines what the function this serves for
	FunctionTemplate string       // api template to fill and send to the node
	ResultParsing    BlockParser  // description for parsing the result of the api
	ApiName          string       // the api name
}
```

`FunctionTag` can be one of the following :
```
const (
	FUNCTION_TAG_GET_BLOCKNUM           FUNCTION_TAG = 1 // get latest block number
	FUNCTION_TAG_GET_BLOCK_BY_NUM       FUNCTION_TAG = 2 // get a specific block by block number
	FUNCTION_TAG_SET_LATEST_IN_METADATA FUNCTION_TAG = 3
	FUNCTION_TAG_SET_LATEST_IN_BODY     FUNCTION_TAG = 4
	FUNCTION_TAG_VERIFICATION           FUNCTION_TAG = 5
)
```

### Verification

The Verification struct defines how to verify a specific property of the API collection. For example, it can be used to verify the chain ID of the node.

```go
type Verification struct {
	Name           string                               // verification name
	ParseDirective *ParseDirective                      // ParseDirective to get the the value to verify from the node
	Values         []*ParseValue                        // expected value we want from the result
	Severity       Verification_VerificationSeverity    // instructions for what to do if a verification fails
}
```

### Headers

This struct defines for the provider what action to take on the headers of the relayed message.

```go
type Header struct {
	Name        string              // name of the header
	Kind        Header_HeaderType   // what action to take
	FunctionTag FUNCTION_TAG        // what is the function of the header
}
```

### Import

Specs can import from other specs, which allows for easy creation and maintenance of specs. Specs import the api collections of the base specs.

A good example of this is the cosmos base specs. All cosmos chains support queries/transactions of the bank module, and these are defined in a cosmos spec (which is disabled since it is a base spec). When creating a cosmos based spec, we can import the cosmos spec and obtain all the APIs of the bank module. This means that specs only need to define the APIs that are unique to the new chain.

Rules:
* Import cycles of specs are prohibited.
* Specs can override/disable/add APIs from the imported API collection.
* Specs also import the verifications, which can be overridden when necessary (for example, the chain-id value must be overwritten).


## Parameters

The Spec module does not contain parameters.

## Queries

The Spec module supports the following queries:

| Query             | Arguments         | What it does                                  |
| ----------        | ---------------   | ----------------------------------------------|
| `list-spec`       | none              | shows all the specs                           |
| `params`          | none              | show the params of the module                 |
| `show-all-chains` | none              | shows all the specs with minimal info         |
| `show-chain-info` | chainid           | shows a spec with minimal info                |
| `show-spec`       | chainid           | shows a full spec                             |

## Transactions

The Spec module does not support any transactions.

## Proposals

The Spec module provides a proposal to add/overwrite a spec to the chain

```
lavad tx gov submit-legacy-proposal spec-add <spec_json_1>,<spec_json_2> --from alice <gas-flags>

```

A valid `add_spec_json_1` JSON proposal format:

```json
{
    "proposal": {
        "title": "Add Specs: Lava",
        "description": "Adding new specification support for relaying Lava data on Lava",
        "specs": [
            {
                "index": "LAV1",
                "name": "lava testnet",
                "enabled": true,
                "imports": [
                    "COSMOSSDK"
                ],
                "providers_types": 1,
                "reliability_threshold": 268435455,
                "data_reliability_enabled": true,
                "block_distance_for_finalized_data": 0,
                "blocks_in_finalization_proof": 1,
                "average_block_time": 30000,
                "allowed_block_lag_for_qos_sync": 2,
                                "shares" : 1,
                "min_stake_provider": {
                    "denom": "ulava",
                    "amount": "50000000000"
                },
                "api_collections": [
                    {
                        "enabled": true,
                        "collection_data": {
                            "api_interface": "rest",
                            "internal_path": "",
                            "type": "GET",
                            "add_on": ""
                        },
                        "apis": [
                            {
                                "name": "/lavanet/lava/spec/show_all_chains",
                                "block_parsing": {
                                    "parser_arg": [
                                        "latest"
                                    ],
                                    "parser_func": "DEFAULT"
                                },
                                "compute_units": 10,
                                "enabled": true,
                                "category": {
                                    "deterministic": true,
                                    "local": false,
                                    "subscription": false,
                                    "stateful": 0
                                },
                                "extra_compute_units": 0
                            }
                        ],
                        "headers": [],
                        "inheritance_apis": [],
                        "parse_directives": [],
                        "verifications": [
                            {
                                "name": "chain-id",
                                "values": [
                                    {
                                        "expected_value": "lava-testnet"
                                    }
                                ]
                            }
                        ]
                    }
                ]
            }
        ]
    },
    "deposit": "10000000ulava"
}
```

### Events


The plans module has the following events:
| Event             | When it happens       |
| ----------        | --------------- |
| `spec_add`        | a successful addition of a spec  |
| `spec_modify`     | a successful modification of an existing spec   |
| `spec_refresh`    | a spec was rereshed since it had a imported spec modified|