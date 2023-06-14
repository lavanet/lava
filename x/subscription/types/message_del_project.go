package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgDelProject = "del_project"

var _ sdk.Msg = &MsgDelProject{}

func NewMsgDelProject(creator string, name string) *MsgDelProject {
	return &MsgDelProject{
		Creator: creator,
		Name:    name,
	}
}

func (msg *MsgDelProject) Route() string {
	return RouterKey
}

func (msg *MsgDelProject) Type() string {
	return TypeMsgDelProject
}

func (msg *MsgDelProject) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgDelProject) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgDelProject) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	return nil
}
