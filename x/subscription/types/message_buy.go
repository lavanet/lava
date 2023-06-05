package types

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgBuy = "buy"

var _ sdk.Msg = &MsgBuy{}

func NewMsgBuy(creator string, consumer string, index string, duration uint64) *MsgBuy {
	return &MsgBuy{
		Creator:  creator,
		Consumer: consumer,
		Index:    index,
		Duration: duration,
	}
}

func (msg *MsgBuy) Route() string {
	return RouterKey
}

func (msg *MsgBuy) Type() string {
	return TypeMsgBuy
}

func (msg *MsgBuy) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgBuy) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgBuy) ValidateBasic() error {
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
	if msg.Duration == 0 || msg.Duration > MAX_SUBSCRIPTION_DURATION {
		return sdkerrors.Wrapf(ErrInvalidParameter, "invalid subscription duration (%s)", msg.Index)
	}

	return nil
}
