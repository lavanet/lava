package types

import (
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

func IbcIprpcReceiverAddress() (receiverName string, receiverAddress sdk.AccAddress) {
	receiverName = "iprpc"
	return receiverName, authtypes.NewModuleAddress(receiverName)
}

const (
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
