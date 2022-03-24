package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgProofOfWork = "proof_of_work"

var _ sdk.Msg = &MsgProofOfWork{}

func NewMsgProofOfWork(creator string, relays []*RelayRequest) *MsgProofOfWork {
	return &MsgProofOfWork{
		Creator: creator,
		Relays:  relays,
	}
}

func (msg *MsgProofOfWork) Route() string {
	return RouterKey
}

func (msg *MsgProofOfWork) Type() string {
	return TypeMsgProofOfWork
}

func (msg *MsgProofOfWork) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgProofOfWork) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgProofOfWork) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
