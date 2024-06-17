package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgAddSpecs = "cover_ibc_iprpc_fund_cost"

var _ sdk.Msg = &MsgAddSpecs{}

func NewMsgAddSpecs(creator string, specs []Spec) *MsgAddSpecs {
	return &MsgAddSpecs{
		Creator: creator,
		Specs:   specs,
	}
}

func (msg *MsgAddSpecs) Route() string {
	return RouterKey
}

func (msg *MsgAddSpecs) Type() string {
	return TypeMsgAddSpecs
}

func (msg *MsgAddSpecs) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgAddSpecs) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAddSpecs) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	for _, v := range msg.Specs {
		err := v.ValidateBasic()
		if err != nil {
			return err
		}
	}
	return nil
}
