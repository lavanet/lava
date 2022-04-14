package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgUnstakeClient = "unstake_client"

var _ sdk.Msg = &MsgUnstakeClient{}

func NewMsgUnstakeClient(creator string, chainID string) *MsgUnstakeClient {
	return &MsgUnstakeClient{
		Creator: creator,
		ChainID: chainID,
	}
}

func (msg *MsgUnstakeClient) Route() string {
	return RouterKey
}

func (msg *MsgUnstakeClient) Type() string {
	return TypeMsgUnstakeClient
}

func (msg *MsgUnstakeClient) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgUnstakeClient) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgUnstakeClient) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
