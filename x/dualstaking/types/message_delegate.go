package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
)

const TypeMsgDelegate = "delegate"

var _ sdk.Msg = &MsgDelegate{}

func NewMsgDelegate(delegator string, validator string, provider string, chainID string, amount sdk.Coin) *MsgDelegate {
	return &MsgDelegate{
		Creator:   delegator,
		Validator: validator,
		Provider:  provider,
		ChainID:   chainID,
		Amount:    amount,
	}
}

func (msg *MsgDelegate) Route() string {
	return RouterKey
}

func (msg *MsgDelegate) Type() string {
	return TypeMsgDelegate
}

func (msg *MsgDelegate) GetSigners() []sdk.AccAddress {
	delegator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{delegator}
}

func (msg *MsgDelegate) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgDelegate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid delegator address (%s)", err)
	}

	if msg.Provider != commontypes.EMPTY_PROVIDER {
		_, err = sdk.AccAddressFromBech32(msg.Provider)
		if err != nil {
			return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid provider address (%s)", err)
		}
	}

	if !msg.Amount.IsValid() {
		return legacyerrors.ErrInvalidCoins
	}

	_, err = sdk.ValAddressFromBech32(msg.Validator)
	if err != nil {
		return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid validator address (%s)", err)
	}

	return nil
}
