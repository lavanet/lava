# `x/subscription`

## Abstract

This document specifies the subscription module of Lava Protocol.

The subscription module is responsible for managing Lava's consumer subscription.  
To use Lava, consumers must purchase a subscription.  
The subscription operates on a monthly basis, starting from the next block after purchase and ending one month later.  
If a user purchases a subscription for more than one month, it will reset at the end of each month until its expiration.  
When a subscription is reset, the monthly left CUs is set to the plan's monthly CU.

A subscription can be renewed, manually or automatically.  
Additionally, a subscription plan can be bought in advance for months ahead.

There are 2 other concepts that the subscription is connected to:

- [Plans](https://github.com/lavanet/lava/blob/main/x/plans/README.md)
- [Projects](https://github.com/lavanet/lava/blob/main/x/projects/README.md)

To fully understand this module, it is highly recommended to read Plans and Projects READMEs as well.

## Contents

- [Concepts](#concepts)
  - [Subscription](#subscription)
  - [Advance Month](#advance-month)
  - [Subscription Upgrade](#subscription-upgrade)
  - [Subscription Renewal](#subscription-renewal)
  - [Advance Purchase](#advance-purchase)
- [Parameters](#parameters)
- [Queries](#queries)
- [Transactions](#transactions)
- [Proposals](#proposals)

## Concepts

### Subscription

In order for a consumer to purchase a subscription, they first must choose a plan to use.  
After choosing a plan, a consumer can perform the purchase using the `buy` transaction command:

```bash
lavad tx subscription buy [plan-index] [optional: consumer] [optional: duration(months)] [flags]
```

Once a consumer bought a subscription, a new instance of Subscription object will be saved into the state:

```go
struct Subscription {
	Creator            string              // creator pays for the subscription
	Consumer           string              // consumer uses the subscription
	Block              uint64              // when the subscription was last recharged
	PlanIndex          string              // index (name) of plan
	PlanBlock          uint64              // when the plan was created
	DurationBought     uint64              // total requested duration in months
	DurationLeft       uint64              // remaining duration in months
	MonthExpiryTime    uint64              // expiry time of current month
	MonthCuTotal       uint64              // CU allowance during current month
	MonthCuLeft        uint64              // CU remaining during current month
	Cluster            string              // cluster key
	DurationTotal      uint64              // continuous subscription usage in months
	AutoRenewal        bool                // automatic renewal when the subscription expires
	FutureSubscription *FutureSubscription // future subscription made with buy --advance-purchase
}

struct FutureSubscription {
	Creator        string // creator pays for the future subscription. Will replace the original one once activated
	PlanIndex      string // index (name) of plan
	PlanBlock      uint64 // when the plan was created
	DurationBought uint64 // total requested duration in months
}
```

When a consumer buys a subscription, an admin project is automatically generated for them.  
When the creator and consumer are different individuals, indicating that someone is purchasing a subscription for another user, the tokens will be taken out of the creator's account.

Subscriptions are saved in a fixation store (see [FixationStore](https://github.com/lavanet/lava/blob/main/x/fixationstore/README.md) module for reference).

There could only be a single active subscription for a consumer.  
When a consumer purchase a subscription, the tokens are transferred from their account immediately.  
Those tokens are later on distributed among the providers that served that consumer, and the delegators that staked those providers.

### Advance Month

One of the primary processes within this module involves the function `advanceMonth`, which is invoked upon the expiration of a timer set by the subscription. This function holds significance as it manages the logic that occurs at the conclusion of each month of a subscription.

The advanceMonth function plays a crucial role in the subscription lifecycle, performing several important tasks:

1. **CU Tracker Timer Update**: It updates the CU tracker timer, ensuring accurate tracking of Compute Units (CUs) for the subscription.

2. **Subscription Duration Management**: The function checks the subscription's remaining duration. If there is time left, it deducts one month from the remaining duration and updates the total duration count, reflecting the passage of time. The details of the subscription, including CU allocations, are reset for the upcoming month.  
   In cases where the subscription has reached its end (no more remaining months), different scenarios are addressed:

   - Activation of a future subscription: If a future subscription is queued, it becomes active, renewing the service.
   - Auto-renewal: If auto-renewal is enabled, an attempt is made to renew the subscription for another month. If this fails, the expired subscription is removed.
   - No future subscription or auto-renewal: The expired subscription is removed from the system.

3. **Special Cases Handling**: The function also handles special cases, such as the transition to a future subscription or the implementation of auto-renewal, ensuring continuous service or appropriate termination of the subscription.

### Subscription Upgrade

A subscription can be upgraded to a more expensive plan.

While direct downgrading of a subscription plan is not officially supported, consumers seeking a lower-tier plan can still achieve a similar effect.  
This can be done by purchasing additional months under a less expensive plan. It's important to note that the new plan's `plan-index` must differ from the currently active one, and the total payment for the extended period under the cheaper plan should exceed the amount of the original, higher-priced plan.  
This workaround allows consumers to effectively 'downgrade' their subscription within the system's constraints.

Tokens are deducted from the creator's account immediately, and the new plan becomes effective in the next epoch.

More ways to upgrade are by making an [Advance Purchase](#advance-purchase) or enabling [Auto Renewal](#auto-renewal).

### Subscription Renewal

Users have the option to renew their subscription either manually or automatically. To renew manually, users can utilize the same command as demonstrated above. They simply need to adjust the 'plan-index' to match the plan of their currently active subscription. Additionally, they can set the duration, allowing the new duration to be added to the remaining duration of the active subscription.

To renew automatically, users can use the subscription `auto-renewal` transaction command:

```bash
lavad tx subscription auto-renewal [true/false] [optional: plan-index] [optional: consumer] [flags]
```

With auto renewal, users have the flexibility to renew their subscription to any plan, regardless of whether it's cheaper or more expensive than the current active plan. The necessary tokens for renewal are deducted from the user's account at the end of each month, covering the renewal for just one month at a time. If sufficient funds are not available, the subscription will expire.

To disable auto-renewal, users can simply set the `auto-renewal` command to `false`. This will stop the subscription from renewing automatically at the end of its current cycle.

Furthermore, if a user has configured a future subscription and also set auto renewal, the auto renewal will only take effect after the future subscription expires.

### Advance Purchase

Users can purchase several months of subscription in advance, using the subscription `buy` transaction command like so:

```bash
lavad tx subscription buy --advance-purchase [plan-index] [optional: consumer] [optional: duration(months)] [flags]
```

This will create the `FutureSubscription` object inside the `Subscription` object, as can be seen above.
The new subscription will be triggered once the current active subscription expires.

Users can decide on the duration of this future subscription, and they'll pay the full amount for the entire period immediately.
If the plan changes during that period, users who bought this plan before it changed using the `--advance-purchase` flag, they won't be affected by that change.

If a user tries to replace the future subscription with another, the new plan's price must be higher, considering the amount of days bought.
Meaning, if user originally bought X days of a plan with price A, and now wants to advance purchase Y days of a different plan with price B, than the following must be suffice:

$$
Y * B > X * A
$$

## Parameters

The subscription module does not contain parameters.

## Queries

The subscription module supports the following queries:

| Query                  | Arguments             | What it does                                                   |
| ---------------------- | --------------------- | -------------------------------------------------------------- |
| `current`              | consumer (string)     | Shows the current subscription of a consumer to a service plan |
| `list`                 | subscription (string) | Shows all current subscriptions                                |
| `list-projects`        | none                  | Shows all the subscription's projects                          |
| `next-to-month-expiry` | none                  | Shows the subscriptions with the closest month expiry          |
| `params`               | none                  | Shows the parameters of the module                             |

## Transactions

All the transactions below require setting the `--from` flag and gas related flags.

The subscription module supports the following transactions:

| Transaction    | Arguments                                                                               | What it does                                  | Effective in                                                                                                  |
| -------------- | --------------------------------------------------------------------------------------- | --------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| `add-project`  | project-name (string)                                                                   | Add a new project to a subscription           | next block                                                                                                    |
| `auto-renewal` | [true, false] (bool), plan-index (string, optional), consumer (optional)                | Enable/Disable auto-renewal to a subscription | next block                                                                                                    |
| `buy`          | plan-index (string), consumer (string, optional), duration (in months) (int , optional) | Buy a service plan                            | _new subscription_ - next block; <br>_upgrade subscription_ - next epoch;<br>_advance purchase_ - next block; |
| `del-project`  | project-name (string)                                                                   | Delete a project from a subscription          | next epoch                                                                                                    |

Note that the `buy` transaction also support advance purchase and immediate upgrade. Refer to the help section of the commands for more details.

## Proposals

The subscription module does not support any proposals.
