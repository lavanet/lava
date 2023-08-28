package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgUnbond = "unbond"

var _ sdk.Msg = &MsgUnbond{}

func NewMsgUnbond(delegator string, provider string, chainID string, amount sdk.Coin) *MsgUnbond {
	return &MsgUnbond{
		Delegator: delegator,
		Provider:  provider,
		ChainID:   chainID,
		Amount:    amount,
	}
}

func (msg *MsgUnbond) Route() string {
	return RouterKey
}

func (msg *MsgUnbond) Type() string {
	return TypeMsgUnbond
}

func (msg *MsgUnbond) GetSigners() []sdk.AccAddress {
	delegator, err := sdk.AccAddressFromBech32(msg.Delegator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{delegator}
}

func (msg *MsgUnbond) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgUnbond) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Delegator)
	if err != nil {
		return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid delegator address (%s)", err)
	}

	if !msg.Amount.IsValid() {
		return legacyerrors.ErrInvalidCoins
	}

	return nil
}
