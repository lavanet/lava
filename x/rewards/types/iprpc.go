package types

import (
	"encoding/json"

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
)

func (im IprpcMemo) IsEqual(other IprpcMemo) bool {
	return im.Creator == other.Creator && im.Duration == other.Duration && im.Spec == other.Spec
}

func CreateIprpcMemo(creator string, spec string, duration uint64) (memoStr string, err error) {
	memo := IprpcMemo{
		Creator:  creator,
		Spec:     spec,
		Duration: duration,
	}

	// memo wrapper allows marshaling the memo as a nested JSON with a primary key "iprpc"
	memoWrapper := struct {
		Iprpc IprpcMemo `json:"iprpc"`
	}{
		Iprpc: memo,
	}

	bz, err := json.Marshal(memoWrapper)
	if err != nil {
		return "", err
	}

	return string(bz), nil
}

func IbcIprpcReceiverAddress() string {
	return authtypes.NewModuleAddress("iprpc").String()
}
