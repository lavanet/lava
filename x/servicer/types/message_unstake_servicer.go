package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgUnstakeServicer = "unstake_servicer"

var _ sdk.Msg = &MsgUnstakeServicer{}

func NewMsgUnstakeServicer(creator string, spec *SpecName, deadline *BlockNum) *MsgUnstakeServicer {
	return &MsgUnstakeServicer{
		Creator:  creator,
		Spec:     spec,
		Deadline: deadline,
	}
}

func (msg *MsgUnstakeServicer) Route() string {
	return RouterKey
}

func (msg *MsgUnstakeServicer) Type() string {
	return TypeMsgUnstakeServicer
}

func (msg *MsgUnstakeServicer) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgUnstakeServicer) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgUnstakeServicer) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
