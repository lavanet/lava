package types

import (
	"strings"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgAutoRenewal = "add_project"

var _ sdk.Msg = &MsgAutoRenewal{}

func NewMsgAutoRenewal(creator, consumer, planIndex string, enable bool) *MsgAutoRenewal {
	return &MsgAutoRenewal{
		Consumer: consumer,
		Creator:  creator,
		Enable:   enable,
		Index:    planIndex,
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

	_, err = sdk.AccAddressFromBech32(msg.Consumer)
	if err != nil {
		return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid consumer address (%s)", err)
	}

	if msg.Enable && strings.TrimSpace(msg.Index) == "" {
		return sdkerrors.Wrapf(ErrBlankParameter, "invalid plan index (%s)", msg.Index)
	}

	if !msg.Enable && strings.TrimSpace(msg.Index) != "" {
		return sdkerrors.Wrapf(ErrBlankParameter, "can't use plan index when `--disable` flag is passed")
	}

	return nil
}
