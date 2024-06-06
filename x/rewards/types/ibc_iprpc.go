package types

import (
	"encoding/json"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type IprpcMemo struct {
	Creator  string `json:"creator"`
	Spec     string `json:"spec"`
	Duration uint64 `json:"duration"`
}

func (im IprpcMemo) IsEqual(other IprpcMemo) bool {
	return im.Creator == other.Creator && im.Duration == other.Duration && im.Spec == other.Spec
}

func CreateIprpcMemo(creator string, spec string, duration uint64) (memo string, err error) {
	durationStr := strconv.FormatUint(duration, 10)

	data := map[string]interface{}{
		"iprpc": map[string]interface{}{
			"creator":  creator,
			"spec":     spec,
			"duration": durationStr,
		},
	}

	memoBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(memoBytes), nil
}

// IbcIprpcReceiverAddress returns a Bech32 address for the string "iprpc"
// Note, the NewModuleAddress() function is used for convenience. The IbcIprpcReceiver is not a module account
func IbcIprpcReceiverAddress() sdk.AccAddress {
	return authtypes.NewModuleAddress(IbcIprpcReceiver)
}

const (
	IbcIprpcReceiver          = "iprpc"
	PendingIbcIprpcFundPrefix = "PendingIbcIprpcFund/"
)

func (piif PendingIbcIprpcFund) IsEqual(other PendingIbcIprpcFund) bool {
	return piif.Index == other.Index && piif.Creator == other.Creator && piif.Spec == other.Spec &&
		piif.Duration == other.Duration && piif.Expiry == other.Expiry && piif.Fund.IsEqual(other.Fund)
}

func (piif PendingIbcIprpcFund) IsEmpty() bool {
	return piif.IsEqual(PendingIbcIprpcFund{})
}

func (piif PendingIbcIprpcFund) IsValid() bool {
	return piif.Expiry > 0 && piif.Fund.IsValid() && piif.Fund.Amount.IsPositive() && piif.Duration > 0
}

func (piif PendingIbcIprpcFund) IsExpired(ctx sdk.Context) bool {
	return uint64(ctx.BlockTime().UTC().Unix()) >= piif.Expiry
}

const (
	NewPendingIbcIprpcFundEventName            = "pending_ibc_iprpc_fund_created"
	ExpiredPendingIbcIprpcFundRemovedEventName = "expired_pending_ibc_iprpc_fund_removed"
	CoverIbcIprpcFundCostEventName             = "cover_ibc_iprpc_fund_cost"
)
