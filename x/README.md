# `x`

## Abstract

This document specifies the Top Architecture of the Lava Blockchain.

It explains how the mechanisms work together to create the modular data access layer that is lava

## Contents

* [Roles](#roles)
  * [Validator](#validator)
  * [Consumer](#consumer)
  * [Provider](#provider)
  * [Contributor](#contributor)
* [Concepts](#concepts)
  * [Fixed Supply](#fixed-supply)
  * [Dual Staking](#dual-staking)
  * [Adding Specifications](#adding-specifications)
  * [Plans & Policies](#plans-&-policies)
  * [Pairing](#pairing)
  * [Lava Protocol P2P](#lava-protocol-p2p)
    * [Sessions](#sessions)
    * [Communication](#communication)
    * [Badges](#badges)
  * [Claiming Rewards](#claiming-rewards)
  * [Reports](#reports)
  * [Boosts](#boosts)
  * [Conflicts](#conflicts)
* [Modules](#modules)

## Roles

### Validator

classic cosmos-sdk validator role, they produce blocks for transactions fees and block creation rewards. validators also participate in subscription rewards, so increased subscriptions directly affect validator incentives.

### Consumer

a role for users that want to consume apis, they can do so with a valid periodic subscription purchased by them or a different user with tokens. having an active subscription allows consumers to get a list of providers to consume apis from and grant rewards. subscription owners can delegate usage in their subscription to developers via projects

### Provider

a role for users that want to serve apis for rewards. they stake tokens to ensure incentive alignment and accountability. when staking providers choose the specifications they want to serve and define the endpoints to consume these specifications. after serving apis off chain via the p2p protocol, providers claim rewards for their service by sending relay payment transactions containing aggregated consumer sessions with signatures.

### Contributor

a role selected by governance, they are participating in subscription rewards for the spec they have been selected, by a governance controlled amount. this role is designed to incentivize spec maintenance/creation or reward successful efforts increasing consumption

## Concepts

### Fixed Supply

in lava, minting is disabled and the amount of tokens is finite. bootstrap rewards are already locked in reward pools and are gradually emitted. this includes rewards for block creation and bootstrap boosts for providers rewards, until the ecosystem matures and consumption sustains it.

excess rewards not given are burned, as well as fraud slashes, causing the lava token supply to diminish over time. to take that into account and adapt to fluctuations in the token demand, provider rewards are directly proportional to consumer purchases, whatever consumers buy is given after participation to providers. it is expected governance will adjust subscription plan prices according to supply and demand

### Dual Staking

in lava both providers and validators stake in order to align incentives with the ecosystem. in order to avoid having one staking pool bigger than the other, resulting in unbalanced security (too low, or over paying for it) and capital inefficiency to maintain both pools, lava introduces a novel dual staking approach. in dual staking the same lava token delegation/stake is exposed to risk and provides rewards in both a validator and a provider at the same time.

when delegating to a validator it is optional to select a provider, if a provider is not selected, the funds are available for future delegation to a provider in a bucket called "empty_provider". this will result in the validators not maximizing their staking effects and rewards, but this is remediable with a delegation tx. cosmos basic staking messages are still supported, and using hooks we are adapting dualstaking entries to adapt. validator self staking is considered a self delegation and can enjoy the benefits of provider delegation as well

when staking a provider it is mandatory to also select a validator to delegate to. the validator delegation is a regular cosmos-sdk delegation, where delegators participate in validator rewards post commission. provider delegations are also participating in rewards, and are also increasing the provider effective stake causing it to get more consumers, and increase both load and rewards. these provider delegations are affected by a provider commission and a delegation limit. if a validator is not selected, our cli selects the biggest validator or a validator with existing delegations.

### Adding Specifications

governance can vote on adding a new api specification to lava. this is done with a spec add proposal. a spec is a configuration json that defines the service in detail.
a spec contains parameters such as allowed apis, their cost, how to parse them, how to keep track of freshness, and how to check for fraud. specifications can also inherit from other specification, reducing the need for repetitions.
once a specification is added, providers can offer services for that spec by staking to it.

### Plans & Policies

when consumers want to purchase a subscription, they have a governance defined selection of options. each options is called a plan. plans have a built in configuration set by governance. plans define the amount of consumption a subscription will get per period of time (usually a month), various restrictions such as the number of providers, geolocations, advanced options and more. these options that define the way a consumer will be able to interact with provider is called a policy

there are 3 levels of policies, where the strictest one applies. the plan defines the top level policy, for example the number of providers per pairing. a subscription can define it's own policy, define for example a bigger number of providers, the stricter of the two will apply. this is especially useful to determine specific geolocation preferences, selecting addons or managing/limiting projects under your subscription

### Pairing

consumers with a valid subscription or project can communicate with providers to consume apis. it is not possible though, to communicate with the entirety of lava. lava implements a pseudo random mix and match algorithm between consumers and providers. the algorithm takes various factors into consideration such as the providers' stake, their geolocation, their Excellence Quality of Service metric, their supported addons, and more and the specific consumer, to pair them in a specific epoch. every epoch this selection can change to create a correct and healthy distribution, providing privacy by distributing traffic and disallowing censorship or spear phishing.

only providers within a consumer's pairing can claim rewards, and the amount of CU they can claim in a specific plan, is lower the more providers there are. the pairing function isn't calculated on chain until rewards are claimed, consumers run it off chain to see the providers they can use, providers run it to verify a consumer's request for the first time, and the chain runs it on the aggregated payment request.

### Lava Protocol P2P

lava defines a protocol between consumers and providers.

#### Communication

this protocol uses grpc communication based on protobufs wrapping all of the supported api interfaces. any api interface can be wrapped, currently supporting jsonrpc, grpc, websocket, rest, uri-rpc. each consumer request contains the api interface it uses so listening for requests in a provider can be done with the same endpoint routing the traffic to the sub service. each consumer request in that protocol is cryptographically signed with a developer key (also the subscription key counts). these signatures can be used in a relay payment transaction to claim rewards.
provider responses are signed as well, adding accountability to the response and it's data. a request or response that is not signed is ignored.

#### Sessions

in order for relay payment requests not to scale in size, we want a succinct proof. a way to obtain this without complexity is use aggregation. each relay that is sent contains a signature for the amount of compute units (CU) so far adding to it the cu on the latest request. each subsequent relay increases the amount that can be claimed. these aggregations from sessions. a provider only needs the last message in a session to claim rewards for it. in order to avoid the provider claiming too much using several proofs from the same session, the blockchain has a double spend protection, a session that was claimed for in a specific epoch blocks all further claims on that session. this causes reward claims to be sent with a delay, to allow closing up all the aggregation of the current epoch, and claim later after it is done.

#### badges

in order to support frontend usage, that is considered as compromising private keys, lava adds a solution called badges. these are on the fly generated private keys, that are granted clearance "badge" to use on behalf of the subscription/developer in a specific epoch for a limited amount of cu. the risk of a badge being compromised is negligible, it has no means to cause hard to the subscription long term

### Claiming Rewards

since provider send claim transaction for "work" done outside of the blockchain, lava supports claiming rewards with a delay. this works well with the aggregation of session described before. lava has a moving time window for claiming rewards. data for pairing and claimed rewards are saved up to that moving window of memory, this allows providers to claim rewards in a timely manner without rushing it just as the epoch ends. it also allows tolerance if the consumer is un-synced or if the provider was unable to claim due to network issues, tx congestion or downtime.

providers save locally (backed up by a db) the latest relay request holding a cryptographic signature from the consumer developer or subscription key as proof. when an epoch ends, and after a backoff these rewards are aggregated into a transaction called relay_payment, that is being automatically triggered by the provider service. this tx is passes verifications accumulates compute units (CU) on behalf of the provider.

at the end of a subscription, all provided CU (that are capped totally and per epoch by the plan policy) are considered when dividing the rewards between providers. each providers gets his respective part out of the subscription's cost minus participation of validators and contributors. these rewards are then divided post commission with delegators. this means that provider rewards are awarded monthly per subscription, depending when it was purchased. given rewards for the provider and its delegators are claimable in dualstaking via a transaction

### Reports

during a session consumers rank the provider they are communicating with. these are called Quality of service reports. providers must include these reports in their claims for rewards or forfeit the rewards. these scores can reduce accumulated CU by a provider up to a value set by governance. this incentivizes providers to provide good enough service to not be penalized.

providers not responding at all to a consumer either via a connection refusal or by repeated errors are considered offline. the consumer then reports them via other provider relay_payments by including them in the requests metadata. when these requests are claimed the blockchain learns of the reports and considers them against the provider. each epoch start all claims against a provider are considered against service it provided, if more compute units usage reported on the provider than it claimed by a factor the provider is jailed. this means the provider can't get anymore pairing until he un-jails.

### Boosts

during the bootstrap of the blockchain it is assumed demand might not sustain the ecosystem right away. in order to encourage providers to join early and bootstrap high quality service form the start, the rewards module uses a preallocated pool of tokens to boost provider rewards. these rewards are scaling linearly with demand, meaning the more demand there is the bigger the boost, until the cap is reached. once the cap of the boost (emitted monthly) is reached it means rewards are sufficient. if organic rewards keep increasing, the boost will decrease as it is no longer needed and get burned.

### Conflicts

while using the services, consumers use the protocol data reliability feature to compare deterministic responses for the same query from several providers. if a discrepancy was found, and since it is cryptographically signed by both providers, the consumer can create a conflict transaction to report the fraudulent provider. this triggers an on chain resolution process of a commit and reveal vote by all of the providers in the spec that are chosen (jury mechanism). providers chosen are required to participate or they will get jailed. after getting sufficient votes, the losing provider will get jailed. after battle testing the feature, and on specs that have enough stake to trust the majority, we are planning it to have more teeth.

## Modules

### utility

#### epochstorage

a utility module to provide a snapshot of a storage upon epoch start and clean older snapshots. all of them are saved and accessible on chain for claiming rewards and running pairings

#### fixationstore

a utility module to provide a ref-counted differential storage. entries are stored as long as a reference is held. another entry is created only when a change is made. when querying for information at a specific height it fetches the entry with a smaller equal height if it exists. this is a more efficient storage wise handling of epoch data, and will replace epochstorage as well in the future.

#### timerstore

a utility module to provide block time based callbacks to be triggered on begin block. this allows periodic checks, timers for subscription expiry, monthly payouts and more.
