package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSetProjectPolicy = "set_project_policy"

var _ sdk.Msg = &MsgSetProjectPolicy{}

func NewMsgSetProjectPolicy(creator string) *MsgSetProjectPolicy {
	return &MsgSetProjectPolicy{
		Creator: creator,
	}
}

func (msg *MsgSetProjectPolicy) Route() string {
	return RouterKey
}

func (msg *MsgSetProjectPolicy) Type() string {
	return TypeMsgSetProjectPolicy
}

func (msg *MsgSetProjectPolicy) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSetProjectPolicy) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSetProjectPolicy) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
