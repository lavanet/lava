package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgStakeUser = "stake_user"

var _ sdk.Msg = &MsgStakeUser{}

func NewMsgStakeUser(creator string, spec *SpecName, amount sdk.Coin, deadline BlockNum) *MsgStakeUser {
	return &MsgStakeUser{
		Creator:  creator,
		Spec:     spec,
		Amount:   amount,
		Deadline: deadline,
	}
}

func (msg *MsgStakeUser) Route() string {
	return RouterKey
}

func (msg *MsgStakeUser) Type() string {
	return TypeMsgStakeUser
}

func (msg *MsgStakeUser) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgStakeUser) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStakeUser) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
