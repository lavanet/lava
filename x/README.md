# `x`

## Abstract

This document specifies the Top Architecture of the Lava Blockchain.

It explains how the mechanisms work together to create the modular data access layer that is Lava.

## Contents

* [Roles](#roles)
  * [Validator](#validator)
  * [Consumer](#consumer)
  * [Provider](#provider)
  * [Contributor](#contributor)
  * [Delegator](#delegator)
* [Concepts](#concepts)
  * [Fixed Supply](#fixed-supply)
  * [Epochs](#epochs)
  * [Dual Staking](#dual-staking)
  * [Adding Specifications](#adding-specifications)
  * [Plans & Policies](#plans--policies)
  * [Subscriptions & Projects](#subscriptions--projects)
  * [Pairing](#pairing)
  * [Lava Protocol P2P](#lava-protocol-p2p)
    * [Sessions](#sessions)
    * [Communication](#communication)
    * [Badges](#badges)
  * [Claiming Rewards](#claiming-rewards)
  * [Reports](#reports)
  * [Boosts](#boosts)
  * [Conflicts](#conflicts)
  * [Constant Availability](#constant-availability)
* [Modules](#modules)
  * [Utility](#utility)
    * [Epoch Storage](#epochstorage)
    * [Fixation Store](#fixationstore)
    * [Timer Store](#timerstore)
    * [Downtime](#downtime)
  * [Governance](#governance)
    * [Spec](#spec)
    * [Plans](#plans)
    * [Protocol](#protocol)
  * [Consumers & Providers](#consumers--providers)
    * [Subscription](#subscription)
    * [Projects](#projects)
    * [Pairing](#pairing-module)
  * [Tokenomics](#tokenomics)
    * [Dual Staking](#dualstaking)
    * [Rewards](#rewards)

## Roles

### Validator

Validators play a pivotal role in the Lava Blockchain, akin to the classic cosmos-sdk validator role. Their primary function involves producing blocks, validating transactions, and earning rewards from transaction fees and block creation. Beyond this foundational role, validators also actively participate in subscription rewards. As subscriptions increase in number or activity, validator incentives are directly impacted, creating a symbiotic relationship between network growth and validator rewards.

### Consumer

Consumers within the Lava ecosystem access APIs by purchasing subscriptions. Additionally, users associated with these consumers can gain API access by utilizing badges created by the consumers. Active subscriptions provide a curated list of available providers offering APIs and allow consumers to allocate rewards. Subscription owners have the authority to delegate usage within their subscription, enabling developers' access via designated projects. Providers offer APIs in exchange for rewards facilitated by these subscriptions.

### Provider

Providers in the Lava network are individuals aiming to offer APIs in exchange for rewards. They stake tokens to establish alignment of incentives and ensure accountability within the ecosystem. Upon staking, providers select the specifications they intend to serve and establish the endpoints for consuming these specifications. Post serving APIs off-chain through the P2P protocol, providers claim rewards for their services by initiating relay payment transactions that encompass aggregated consumer sessions accompanied by signatures.

### Contributor

Contributors within the Lava network are appointed by governance to engage in subscription rewards tied to a specified specification, regulated by governance-controlled allocations. This role is tailored to incentivize activities such as specification maintenance/creation or rewarding successful endeavors that enhance consumption within the network.

### Delegator

Delegators serve as essential contributors to the Lava network, playing a crucial part in maintaining its integrity and functionality. Their primary function revolves around token delegation, where they stake tokens to support network security and operational efficiency. By delegating tokens to validators and providers, delegators actively engage in securing the network and ensuring service provision within the ecosystem. This delegation enables them to participate in reward distribution. Delegators receive rewards proportional to their delegation efforts, earning validator rewards when delegating to validators and sharing in provider rewards when staking providers. Additionally, their tokens generate entries in the Dual Staking module, ensuring synchronization between validator and provider staking pools. Delegators can claim their accumulated rewards from their participation in token delegation through dedicated transactions in the Dual Staking module, further incentivizing their active engagement within the network. Overall, delegators' involvement contributes significantly to the stability, security, and sustainability of the Lava network while being rewarded for their participation.

## Concepts

### Fixed Supply

In Lava, minting is disabled, ensuring a finite token supply. Bootstrap rewards are securely locked in reward pools and gradually released, encompassing incentives for block creation and bootstrap boosts for provider rewards, sustaining until the ecosystem matures.

Unclaimed rewards and fraud slashes result in a reduction of the Lava token supply over time. Provider rewards are dynamically tied to consumer purchases — whatever consumers buy is proportionally distributed to providers after their participation. Governance is expected to adjust subscription plan prices in response to fluctuations in token demand and supply.

### Epochs

In Lava, an epoch represents a set of blocks with a fixed size (modifiable by governance). Many mechanisms within Lava are activated only at the transition between epochs. Lava maintains both current storage and fixated storage. Upon an epoch change, the current storage is preserved in the fixated storage, providing a stable reference for all mechanisms to depend on throughout the epoch. Notably, this principle applies to most parameter changes as well.

### Dual Staking

In Lava, both providers and validators engage in staking to align incentives within the ecosystem. To prevent one staking pool from becoming disproportionately larger than the other — leading to unbalanced security or capital inefficiency — Lava introduces a unique dual staking mechanism. With dual staking, the same Lava token delegation or stake carries risk exposure and yields rewards simultaneously in both validator and provider roles.

When delegating to a validator, it's optional to select a provider. If no provider is chosen, the funds remain available for future delegation to a provider in a designated "empty_provider" bucket. While this might not maximize validators' staking effects and rewards, it can be remedied with a delegation transaction. Lava supports cosmos basic staking messages, and through hooks, dual staking entries are adapted accordingly. Validator self-staking is considered self-delegation and can benefit from provider delegation.

When staking a provider, it's mandatory to simultaneously select a validator for delegation. Validator delegation follows the regular cosmos-sdk delegation process, where delegators participate in validator rewards post-commission. Provider delegations also contribute to rewards and boost the effective stake of the provider, attracting more consumers and increasing both load and rewards. These provider delegations are influenced by a provider commission and a delegation limit. In the absence of a chosen validator, our CLI automatically selects the largest validator or a validator with existing delegations.

### Adding Specifications

In Lava, governance has the authority to vote on the addition of a new API specification through a spec-add proposal. A specification, represented by a detailed configuration JSON, defines a service comprehensively.

Specifications encompass parameters like permitted APIs, their associated costs, parsing methodologies, freshness maintenance, and fraud detection protocols. To minimize redundancy, specifications can inherit from other specifications, reducing the necessity for repetition.

Once a specification is successfully added, providers can offer services aligned with that specification by staking tokens to it.

Specifications can delineate contributors and their involvement in reward distribution, as previously noted

### Plans & Policies

When consumers opt for a subscription, they encounter a range of governance-defined options, known as plans. Each plan encompasses preconfigured settings mandated by governance, specifying the volume of consumption a subscription receives over a given period (typically a month). These plans encompass various restrictions such as provider count, geolocations, advanced features like addons, and more. The guidelines within plans that govern consumer-provider interactions are referred to as policies.

There exist three tiers of policies, where the strictest policy takes precedence. While a subscription cannot surpass the constraints outlined by the plan, it can lower certain limits, such as the provider count, within the given policy framework.

### Subscriptions & Projects

Subscriptions in Lava are like passes that consumers buy to access different APIs. They let users reward providers and manage their access. Projects, on the other hand, are spaces within subscriptions where developers can organize and control resources. They help developers manage how they use computing power. Together, subscriptions and projects give users control and provide a flexible system for developers to work efficiently in the Lava ecosystem

### Pairing

Consumers possessing a valid subscription or project can engage with providers to access APIs. However, full communication across the entire Lava network is restricted. Lava employs a pseudo-random matching algorithm between consumers and providers, considering factors such as providers' stake, geolocation, Excellence Quality of Service metric, supported addons, and more, along with specific consumer parameters. This algorithm pairs them within a particular epoch. Every epoch, this selection undergoes changes to maintain a fair and balanced distribution, ensuring privacy by diversifying traffic and preventing censorship or spear phishing attempts.

Only providers within a consumer's designated pairing can claim rewards. The maximum amount of Compute Units (CU) they can claim within a specific plan decreases with an increasing number of providers. This design ensures a fair distribution of the workload among providers, preventing an overwhelming concentration of requests on a single provider.

The pairing mechanism is not computed on-chain until rewards are claimed. Consumers execute it off-chain to view available providers, while providers run it to validate a consumer's request for the first time. The chain executes it upon aggregated payment requests, ensuring accurate validation and reward distribution based on the established pairing.

### Lava Protocol P2P

Lava defines a protocol between consumers and providers.

#### Communication

This protocol utilizes grpc communication based on protobufs encapsulating all supported API interfaces. It can wrap any API interface, currently supporting jsonrpc, grpc, websocket, rest, and uri-rpc. Each consumer request contains the API interface it employs, allowing providers to listen for requests using the same endpoint, directing traffic to the appropriate sub service.

In this protocol, each consumer request is cryptographically signed with a developer key (subscription keys are also valid). These signatures serve as proof in relay payment transactions, enabling the claim of rewards.

Provider responses are also signed, ensuring accountability for the response and its data. Any request or response lacking a signature is disregarded.

#### Sessions

To prevent relay payment requests from scaling in size, the system aims for a concise proof. Aggregation serves as a solution, allowing each relay sent to include a signature for the cumulative compute units (CU) collected thus far, adding the CU from the latest request. Each subsequent relay further augments the claimable amount. These aggregations form sessions, where a provider requires only the last message in a session to claim rewards.

To prevent providers from excessively claiming using multiple proofs from the same session, the blockchain implements double spend protection. Once a session is claimed in a specific epoch, all subsequent claims on that session are blocked. Consequently, reward claims are delayed to ensure the closure of all current epoch aggregations before claiming.

#### Badges

Badges are a protective measure addressing the risk to private keys during frontend usage within Lava. These dynamically generated private keys, termed 'badges,' are authorized to operate on behalf of the subscription/developer within specific epochs, with usage limited to predetermined compute units (CU). Transmitted to providers alongside dynamically generated signatures, these badges grant authorization for service provision and subsequent reward claims on the blockchain.

### Claiming Rewards

Given that providers submit claim transactions for off-chain work, Lava allows delayed claiming of rewards, aligning with the earlier described session aggregation process. The system employs a moving time window for reward claims, preserving pairing data and claimed rewards within this temporal memory. This strategy enables deliberate reward claiming without the pressure of doing so precisely at the epoch's end. It also accommodates scenarios where consumers may lack synchronization or providers face network issues, transaction congestion, or downtime.

Providers retain the latest relay request locally, containing a cryptographic signature from the consumer, developer, or subscription key as evidence. At the conclusion of an epoch, after a delay, these accrued rewards aggregate into a transaction named 'relay_payment,' automatically initiated by the provider service. This transaction undergoes verifications and accumulates compute units (CU) on the provider's behalf.

Upon termination of a subscription, all provided CUs (conforming to the plan policy, both total and per epoch) contribute to dividing rewards among providers. Each provider receives their portion from the subscription's cost after deductions for validators' and contributors' participation. Subsequently, these rewards are divided post-commission with delegators. Provider rewards are thus distributed monthly per subscription, contingent upon its purchase date. Providers and their delegators can claim rewards through a dualstaking transaction.

It's crucial to clarify that provider rewards undergo a deliberate accumulation process on-chain as compute units throughout the month. These rewards are disbursed upon the conclusion of the subscription period, ensuring a systematic and intentional distribution mechanism that depends on the total usage within the subscription. This deliberate approach ensures that rewards accurately reflect the cumulative usage and allows for a fair distribution among providers based on their contributions throughout the subscription period

### Reports

During a session, consumers rate the provider they are interacting with through Quality of Service reports. Providers are required to include these reports in their claims for rewards; failing to do so leads to forfeiting rewards. These scores have the potential to reduce a provider's accumulated CU up to a value set by governance. This mechanism serves as an incentive for providers to maintain a high level of service to avoid penalties.

Providers failing to respond entirely to a consumer — either through connection refusal or repeated errors — are deemed offline. Consumers can report such instances by including these cases in the metadata of other provider relay_payment requests. When these requests are claimed, the blockchain becomes aware of these reports and evaluates them against the provider. At the start of each epoch, all claims against a provider are assessed in relation to the service it provided. If more compute units usage is reported for the provider than claimed by a certain factor, the provider is effectively "jailed," resulting in the provider being unable to receive further pairings until they are "un-jailed."

### Boosts

During the blockchain's bootstrap phase, it anticipates that the demand may not immediately sustain the ecosystem. To incentivize early provider participation and ensure high-quality service from the outset, the rewards module employs a preallocated pool of tokens to enhance provider rewards. These rewards scale proportionately with demand, meaning they increase as demand grows, up to a specified cap. When the cap on the monthly boost is reached, it indicates that rewards are adequate. If organic rewards continue to rise, the boost diminishes to reflect the diminishing need and is ultimately burned.

### Conflicts

When consumers engage with the services, they rely on the protocol's data reliability feature, ensuring cross-verification of deterministic responses across multiple providers for identical queries. These queries, deliberately unpredictable to providers, are randomly selected by consumers. Users have the option to increase the number of checks performed, enhancing their security measures. In instances of conflicting responses, which are cryptographically signed by both providers, consumers can initiate a conflict transaction to report the potentially fraudulent provider. This action triggers an on-chain resolution process, known as the jury mechanism, involving a commit-and-reveal vote by all selected providers within the specification. Selected providers are obligated to participate; failure to do so may result in potential penalties. After the requisite votes, the identified provider at fault faces repercussions, potentially leading to penalties. As part of our continuous testing and refinement efforts, we aim to bolster this mechanism, particularly for specifications supported by substantial stake backing to ensure majority trust.

### Constant Availability

Lava's blockchain incorporates a distinctive feature to maintain operations during critical consensus failures. When consumers and providers detect downtime separately, they utilize an off-chain communication method based on 'virtual epochs.' These epochs enable both parties to mutually agree upon increased usage without resetting their connections, allowing providers to later claim these additional services as rewards when the blockchain returns to normal operation. The process of claiming rewards after the chain resumes is automated for providers and is automatically acknowledged and approved by the consensus mechanism, surpassing the usual limits set within a single epoch.

## Modules

### Utility

#### [epochstorage](epochstorage/README.md)

This utility module offers a snapshot of storage at the commencement of each epoch while maintaining and managing older snapshots. These snapshots are preserved and made available on-chain, facilitating operations such as claiming rewards and executing pairings.

#### [fixationstore](fixationstore/README.md)

This utility module introduces a ref-counted differential storage mechanism. Entries remain stored as long as a reference is held, generating a new entry only upon modification. When retrieving information at a specified height, it fetches the entry with an equal or smaller height if available. It offers a more storage-efficient approach to managing epoch data and is intended to supersede the epochstorage module in the future.

#### [timerstore](timerstore/README.md)

The timerstore module functions as a utility that facilitates block time - based callbacks triggered at the beginning of a block. This functionality enables various processes such as periodic checks, timers for subscription expiry, monthly payouts, and other time-sensitive operations.

#### [downtime](downtime/README.md)

The downtime utility module serves to offer insights into the block average time and detects occurrences of blockchain downtime.

### Governance

#### [spec](spec/README.md)

The spec module is the framework that enables governance to manage API specifications, encompassing their storage, verification, and inheritance functionalities. All modifications within the spec module are executed through the gov router.

#### [plans](plans/README.md)

The plans module establishes a framework for governance to regulate available subscription options and their associated policies. Governance has the authority to introduce plans for consumers to purchase at predefined token prices with specific configurations. Additionally, it allows for the modification of existing plans that have already been acquired.

#### [protocol](protocol/README.md)

This module is responsible for storing the target and minimum versions for the protocol modules. Any alterations to these versions are possible through a governance parameters proposal.

### Consumers & Providers

#### [subscription](subscription/README.md)

This module enables consumers to purchase and engage with their subscriptions. It generates a subscription object for each consumer, managed with monthly timers. Subscriptions retain references for purchased plans, releasing them upon deletion. Upon creation, a default project is initialized, and additional projects can be added via this interface, stored, and managed within the projects module.

The subscription module is also responsible for compensating providers based on accumulated CUs once a subscription expires (or a month change is triggered). Accumulated CUs per provider per chain are stored and utilized when the expiry timer is triggered.

#### [projects](projects/README.md)

This module manages projects intended for developers' usage. It serves as a tool to handle internal usage within user accounts, allowing different allocations of CUs and policy specifications. Each project can define developer and admin keys, along with its specific policy. Subscriptions hold a higher hierarchy and can define policies for their associated projects.

#### [pairing](pairing/README.md)

This core module serves multiple roles:

* Handles provider staking and their storage, incorporating changes in providers, addition of endpoints, or supported provider specifications within this module. It utilizes epochstorage to snapshot the provider's storage per epoch change.
* Manages the pairing function between consumers and providers based on policies and available providers.
* Manages (Quality of Service) QoS reports and payment requests from providers, storing them to prevent double spending and forwarding accumulated CUs to the subscription module.

### Tokenomics

#### [dualstaking](dualstaking/README.md)

This module handles and stores dual staking entries. It hooks into the cosmos-sdk staking module delegation changes to adapt provider entries, ensuring they are identical (up to shares rounding).

Every validator delegation creates an entry in dual staking under a provider and a chain, or a placeholder "empty_provider" if none was specified. Any validator changes, whether via dualstaking transactions or cosmos-sdk staking module transactions, prompt the code to synchronize the entries.

Delegator rewards from subscriptions are also stored in the module, available to be claimed by delegators through a transaction.

#### [rewards](rewards/README.md)

This module manages preallocated pools for distributing incentives, emitted periodically through callbacks. Validator rewards are released block by block from the allocated rewards for the month, decreasing when an excessive amount of stake is bonded, and the surplus is burned as the month passes.

Provider rewards scale with organic consumer usage, incentivizing active and high-performing providers. Payouts are higher during the ecosystem's demand bootstrap phase.
