package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgStakeServicer = "stake_servicer"

var _ sdk.Msg = &MsgStakeServicer{}

func NewMsgStakeServicer(creator string, spec *SpecName, amount sdk.Coin, deadline *BlockNum) *MsgStakeServicer {
	return &MsgStakeServicer{
		Creator:  creator,
		Spec:     spec,
		Amount:   amount,
		Deadline: deadline,
	}
}

func (msg *MsgStakeServicer) Route() string {
	return RouterKey
}

func (msg *MsgStakeServicer) Type() string {
	return TypeMsgStakeServicer
}

func (msg *MsgStakeServicer) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgStakeServicer) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStakeServicer) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
