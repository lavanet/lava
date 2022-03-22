package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgUnstakeUser = "unstake_user"

var _ sdk.Msg = &MsgUnstakeUser{}

func NewMsgUnstakeUser(creator string, spec *SpecName, deadline *BlockNum) *MsgUnstakeUser {
	return &MsgUnstakeUser{
		Creator:  creator,
		Spec:     spec,
		Deadline: deadline,
	}
}

func (msg *MsgUnstakeUser) Route() string {
	return RouterKey
}

func (msg *MsgUnstakeUser) Type() string {
	return TypeMsgUnstakeUser
}

func (msg *MsgUnstakeUser) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgUnstakeUser) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgUnstakeUser) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
