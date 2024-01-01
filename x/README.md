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

consumers with a valid subscription or project can communicate with providers to consume apis. it is not possible though, to communicate with the entirety of lava. lava implements a pseudo random mix and match algorithm between consumers and providers. the algorithm takes various factors into consideration such as the providers' stake, their geolocation, their Excellence Quality of Service metric, their supported addons, and more

### Lava Protocol P2P

#### Sessions

#### Communication

### Claiming Rewards

### Reports

### Boosts

### Conflicts

## Modules
