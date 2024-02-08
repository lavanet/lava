package types

import (
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgSetIprpcData = "set_iprpc_data"

var _ sdk.Msg = &MsgSetIprpcData{}

func NewMsgSetIprpcData(authority string, cost sdk.Coin, subs []string) *MsgSetIprpcData {
	return &MsgSetIprpcData{
		Authority:          authority,
		MinIprpcCost:       cost,
		IprpcSubscriptions: subs,
	}
}

func (msg *MsgSetIprpcData) Route() string {
	return RouterKey
}

func (msg *MsgSetIprpcData) Type() string {
	return TypeMsgSetIprpcData
}

func (msg *MsgSetIprpcData) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}

func (msg *MsgSetIprpcData) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSetIprpcData) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.Wrap(err, "invalid authority address")
	}

	if !msg.MinIprpcCost.IsValid() {
		return sdkerrors.Wrap(fmt.Errorf("invalid min iprpc cost; invalid denom or negative amount. coin: %s", msg.MinIprpcCost.String()), "")
	}

	for _, sub := range msg.IprpcSubscriptions {
		if _, err := sdk.AccAddressFromBech32(sub); err != nil {
			return sdkerrors.Wrap(err, "invalid subscription address")
		}
	}

	return nil
}
