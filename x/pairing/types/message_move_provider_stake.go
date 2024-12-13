package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgMoveProviderStake = "MoveProviderStake"

var _ sdk.Msg = &MsgMoveProviderStake{}

func NewMsgMoveProviderStake(creator string, srcChain, dstChain string, amount sdk.Coin) *MsgMoveProviderStake {
	return &MsgMoveProviderStake{
		Creator:  creator,
		SrcChain: srcChain,
		DstChain: dstChain,
		Amount:   amount,
	}
}

func (msg *MsgMoveProviderStake) Route() string {
	return RouterKey
}

func (msg *MsgMoveProviderStake) Type() string {
	return TypeMsgMoveProviderStake
}

func (msg *MsgMoveProviderStake) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(legacyerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
