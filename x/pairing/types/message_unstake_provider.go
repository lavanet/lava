package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgUnstakeProvider = "unstake_provider"

var _ sdk.Msg = &MsgUnstakeProvider{}

func NewMsgUnstakeProvider(creator string, chainID string) *MsgUnstakeProvider {
	return &MsgUnstakeProvider{
		Creator: creator,
		ChainID: chainID,
	}
}

func (msg *MsgUnstakeProvider) Route() string {
	return RouterKey
}

func (msg *MsgUnstakeProvider) Type() string {
	return TypeMsgUnstakeProvider
}

func (msg *MsgUnstakeProvider) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgUnstakeProvider) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgUnstakeProvider) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
