package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgProofOfWork = "proof_of_work"

var _ sdk.Msg = &MsgProofOfWork{}

func NewMsgProofOfWork() *MsgProofOfWork {
	return &MsgProofOfWork{}
}

func (msg *MsgProofOfWork) Route() string {
	return RouterKey
}

func (msg *MsgProofOfWork) Type() string {
	return TypeMsgProofOfWork
}

func (msg *MsgProofOfWork) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{}
}

func (msg *MsgProofOfWork) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgProofOfWork) ValidateBasic() error {
	return nil
}
