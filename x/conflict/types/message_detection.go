package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgDetection = "detection"

var _ sdk.Msg = &MsgDetection{}

func NewMsgDetection(creator string, finalizationConflict *sdk.Coin, responseConflict *sdk.Coin) *MsgDetection {
	return &MsgDetection{
		Creator:              creator,
		FinalizationConflict: finalizationConflict,
		ResponseConflict:     responseConflict,
	}
}

func (msg *MsgDetection) Route() string {
	return RouterKey
}

func (msg *MsgDetection) Type() string {
	return TypeMsgDetection
}

func (msg *MsgDetection) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgDetection) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgDetection) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
