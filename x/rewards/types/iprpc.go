package types

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

func (im IprpcMemo) IsEqual(other IprpcMemo) bool {
	return im.Creator == other.Creator && im.Duration == other.Duration && im.Spec == other.Spec
}
