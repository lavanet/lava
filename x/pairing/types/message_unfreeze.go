package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgUnfreeze = "unfreeze"

var _ sdk.Msg = &MsgUnfreezeProvider{}

func NewMsgUnfreeze(creator string, chainIds []string) *MsgUnfreezeProvider {
	return &MsgUnfreezeProvider{
		Creator:  creator,
		ChainIds: chainIds,
	}
}

func (msg *MsgUnfreezeProvider) Route() string {
	return RouterKey
}

func (msg *MsgUnfreezeProvider) Type() string {
	return TypeMsgUnfreeze
}

func (msg *MsgUnfreezeProvider) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgUnfreezeProvider) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgUnfreezeProvider) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
