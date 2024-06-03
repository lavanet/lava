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

// IbcIprpcReceiverAddress returns a Bech32 address for the string "iprpc"
// Note, the NewModuleAddress() function is used for convenience. The IbcIprpcReceiver is not a module account
func IbcIprpcReceiverAddress() sdk.AccAddress {
	return authtypes.NewModuleAddress(IbcIprpcReceiver)
}

const (
	IbcIprpcReceiver = "iprpc"
)

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
