package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgAutoRenewal = "add_project"

var _ sdk.Msg = &MsgAutoRenewal{}

func NewMsgAutoRenewal(creator string, enable bool) *MsgAutoRenewal {
	return &MsgAutoRenewal{
		Creator: creator,
		Enable:  enable,
	}
}

func (msg *MsgAutoRenewal) Route() string {
	return RouterKey
}

func (msg *MsgAutoRenewal) Type() string {
	return TypeMsgAutoRenewal
}

func (msg *MsgAutoRenewal) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgAutoRenewal) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAutoRenewal) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	return nil
}
