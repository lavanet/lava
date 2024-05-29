package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const (
	// IprpcSubscriptionPrefix is the prefix to retrieve all IprpcSubscription
	IprpcSubscriptionPrefix = "IprpcSubscription/"

	// MinIprpcCostPrefix is the prefix to retrieve all MinIprpcCost
	MinIprpcCostPrefix = "MinIprpcCost/"

	// IprpcRewardPrefix is the prefix to retrieve all IprpcReward
	IprpcRewardPrefix = "IprpcReward/"

	// IprpcRewardsCurrentPrefix is the prefix to retrieve all IprpcRewardsCurrent
	IprpcRewardsCurrentPrefix = "IprpcRewardsCurrent/"

	PendingIprpcFundPrefix = "PendingIprpcFund/"
)

type IprpcMemo struct {
	Creator  string `json:"creator"`
	Spec     string `json:"spec"`
	Duration uint64 `json:"duration"`
}

func (im IprpcMemo) IsEqual(other IprpcMemo) bool {
	return im.Creator == other.Creator && im.Duration == other.Duration && im.Spec == other.Spec
}

func IbcIprpcMemoReceiverAddress() string {
	return authtypes.NewModuleAddress("iprpc").String()
}

func (pif PendingIprpcFund) IsEqual(other PendingIprpcFund) bool {
	return pif.Index == other.Index && pif.Creator == other.Creator && pif.Spec == other.Spec &&
		pif.Month == other.Month && pif.Expiry == other.Expiry && pif.Funds.IsEqual(other.Funds) && pif.CostCovered.IsEqual(other.CostCovered)
}

func (pif PendingIprpcFund) IsEmpty() bool {
	return pif.IsEqual(PendingIprpcFund{})
}

func (pif PendingIprpcFund) IsValid() bool {
	return pif.Expiry > 0 && pif.Funds.IsValid() && pif.CostCovered.IsValid()
}

func (pif PendingIprpcFund) IsExpired(ctx sdk.Context) bool {
	return uint64(ctx.BlockTime().UTC().Unix()) >= pif.Expiry
}
