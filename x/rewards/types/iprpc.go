package types

import "cosmossdk.io/collections"

const (
	// IprpcSubscriptionPrefix is the prefix to retrieve all IprpcSubscription
	IprpcSubscriptionPrefix = "IprpcSubscription/"

	// MinIprpcCostPrefix is the prefix to retrieve all MinIprpcCost
	MinIprpcCostPrefix = "MinIprpcCost/"

	// IprpcRewardPrefix is the prefix to retrieve all IprpcReward
	IprpcRewardPrefix = "IprpcReward/"

	// IprpcRewardsCurrentPrefix is the prefix to retrieve all IprpcRewardsCurrent
	IprpcRewardsCurrentPrefix = "IprpcRewardsCurrent/"
)

var LastRewardsBlockPrefix = collections.NewPrefix([]byte("LastRewardsBlock/"))
