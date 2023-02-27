package types

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSubscribe = "subscribe"

var _ sdk.Msg = &MsgSubscribe{}

func NewMsgSubscribe(creator string, consumer string, index string, isYearly bool) *MsgSubscribe {
	return &MsgSubscribe{
		Creator:  creator,
		Consumer: consumer,
		Index:    index,
		IsYearly: isYearly,
	}
}

func (msg *MsgSubscribe) Route() string {
	return RouterKey
}

func (msg *MsgSubscribe) Type() string {
	return TypeMsgSubscribe
}

func (msg *MsgSubscribe) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSubscribe) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSubscribe) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	_, err = sdk.AccAddressFromBech32(msg.Consumer)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid consumer address (%s)", err)
	}
	if strings.TrimSpace(msg.Index) == "" {
		return sdkerrors.Wrapf(ErrBlankParameter, "invalid plan index (%s)", msg.Index)
	}

	return nil
}
