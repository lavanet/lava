package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgStakeClient = "stake_client"

var _ sdk.Msg = &MsgStakeClient{}

func NewMsgStakeClient(creator string, chainID string, amount sdk.Coin, geolocation uint64, vrfpk string) *MsgStakeClient {
	return &MsgStakeClient{
		Creator:     creator,
		ChainID:     chainID,
		Amount:      amount,
		Geolocation: geolocation,
		Vrfpk:       vrfpk,
	}
}

func (msg *MsgStakeClient) Route() string {
	return RouterKey
}

func (msg *MsgStakeClient) Type() string {
	return TypeMsgStakeClient
}

func (msg *MsgStakeClient) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgStakeClient) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStakeClient) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
