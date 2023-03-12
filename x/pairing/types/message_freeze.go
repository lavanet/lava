package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgFreeze = "freeze"

var _ sdk.Msg = &MsgFreeze{}

func NewMsgFreeze(creator string, chainIds []string, reason string) *MsgFreeze {
	return &MsgFreeze{
		Creator:  creator,
		ChainIds: chainIds,
		Reason:   reason,
	}
}

func (msg *MsgFreeze) Route() string {
	return RouterKey
}

func (msg *MsgFreeze) Type() string {
	return TypeMsgFreeze
}

func (msg *MsgFreeze) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgFreeze) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgFreeze) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
